package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"podux/internal/application/dashboard"
	"podux/internal/application/frpc"
	proxyapp "podux/internal/application/proxy"
	serverapp "podux/internal/application/server"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/cloudflared"
	"podux/internal/infrastructure/persistence"
	_ "podux/migrations"
)

func eventually(t *testing.T, label string, test func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if test() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal(label)
}
func tcpPort(t *testing.T) int {
	t.Helper()
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func udpPort(t *testing.T) int {
	t.Helper()
	l, e := net.ListenPacket("udp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.LocalAddr().(*net.UDPAddr).Port
}
func tcpEcho(addr string) bool {
	c, e := net.DialTimeout("tcp4", addr, 300*time.Millisecond)
	if e != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(500 * time.Millisecond))
	value := []byte("local-frpc-regression")
	if _, e = c.Write(value); e != nil {
		return false
	}
	buf := make([]byte, len(value))
	_, e = io.ReadFull(c, buf)
	return e == nil && string(buf) == string(value)
}
func udpEcho(addr string) bool {
	c, e := net.DialTimeout("udp4", addr, 300*time.Millisecond)
	if e != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(500 * time.Millisecond))
	if _, e = c.Write([]byte("local-udp")); e != nil {
		return false
	}
	buf := make([]byte, 64)
	n, e := c.Read(buf)
	return e == nil && string(buf[:n]) == "local-udp"
}

// The test uses Podux's unchanged FRPC service and pinned real FRPS on loopback.
// No production ports, public server, credentials, firewall or routing changes.
func TestFRPCTCPUDPReconnectAndCloudflaredCoexistence(t *testing.T) {
	frpsPath := os.Getenv("PODUX_TEST_FRPS")
	if frpsPath == "" {
		t.Skip("set PODUX_TEST_FRPS to the isolated pinned FRPS binary; native CI does so explicitly")
	}
	frpsPath, e := filepath.Abs(frpsPath)
	if e != nil {
		t.Fatal(e)
	}
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	tmp := t.TempDir()
	fixture := filepath.Join(tmp, "connector-fixture")
	if runtime.GOOS == "windows" {
		fixture += ".exe"
	}
	build := exec.Command("go", "build", "-o", fixture, filepath.Join(repoRoot, "site/qa/testdata/cloudflared/main.go"))
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatalf("fixture build: %v: %s", e, out)
	}
	// The existing FRPC runner uses relative pb_data paths; isolate its working
	// directory rather than changing production behavior for this test.
	t.Chdir(tmp)
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: filepath.Join(tmp, "pb_data")})
	if e = app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e = app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	tcpOrigin, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer tcpOrigin.Close()
	go func() {
		for {
			c, e := tcpOrigin.Accept()
			if e != nil {
				return
			}
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }()
		}
	}()
	udpOrigin, e := net.ListenPacket("udp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer udpOrigin.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, e := udpOrigin.ReadFrom(buf)
			if e != nil {
				return
			}
			_, _ = udpOrigin.WriteTo(buf[:n], addr)
		}
	}()
	bind, remoteTCP, remoteUDP := tcpPort(t), tcpPort(t), udpPort(t)
	for remoteTCP == bind {
		remoteTCP = tcpPort(t)
	}
	configPath := filepath.Join(tmp, "frps.toml")
	content := fmt.Sprintf("bindAddr = '127.0.0.1'\nbindPort = %d\nproxyBindAddr = '127.0.0.1'\nauth.method = 'token'\nauth.token = 'local-test-only'\nlog.to = '%s'\nlog.level = 'error'\n", bind, filepath.ToSlash(filepath.Join(tmp, "frps.log")))
	if e = os.WriteFile(configPath, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
	var child *exec.Cmd
	var waited chan error
	startFRPS := func() {
		t.Helper()
		child = exec.Command(frpsPath, "-c", configPath)
		if e := child.Start(); e != nil {
			t.Fatal(e)
		}
		waited = make(chan error, 1)
		current := child
		go func() { waited <- current.Wait() }()
		eventually(t, "local FRPS did not listen", func() bool {
			c, e := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", bind), 100*time.Millisecond)
			if e == nil {
				c.Close()
			}
			return e == nil
		})
	}
	stopFRPS := func() {
		if child == nil {
			return
		}
		_ = child.Process.Kill()
		<-waited
		child = nil
	}
	startFRPS()
	defer stopFRPS()
	servers := persistence.NewServerRepository(app)
	proxies := persistence.NewProxyRepository(app)
	client := frpc.NewService(app, servers, proxies)
	collection, e := app.FindCollectionByNameOrId("fh_servers")
	if e != nil {
		t.Fatal(e)
	}
	server := core.NewRecord(collection)
	server.Id = "aaaaaaaaaaaaaaa"
	server.Set("serverName", "local-regression")
	server.Set("serverAddr", "127.0.0.1")
	server.Set("serverPort", strconv.Itoa(bind))
	server.Set("auth", map[string]any{"method": "token", "token": "local-test-only"})
	server.Set("transport", map[string]any{"protocol": "tcp", "tls": map[string]any{"enable": false}, "heartbeatInterval": 1, "heartbeatTimeout": 3})
	server.Set("log", map[string]any{"level": "error", "maxDays": 1})
	if e = app.Save(server); e != nil {
		t.Fatal(e)
	}
	collection, e = app.FindCollectionByNameOrId("fh_proxies")
	if e != nil {
		t.Fatal(e)
	}
	addProxy := func(id, kind string, origin, remote int) *core.Record {
		r := core.NewRecord(collection)
		r.Id = id
		r.Set("name", "local-"+kind)
		r.Set("proxyType", kind)
		r.Set("serverId", server.Id)
		r.Set("localIP", "127.0.0.1")
		r.Set("localPort", strconv.Itoa(origin))
		r.Set("remotePort", strconv.Itoa(remote))
		r.Set("status", "enabled")
		r.Set("bootStatus", "offline")
		if e = app.Save(r); e != nil {
			t.Fatal(e)
		}
		return r
	}
	tcpProxy := addProxy("bbbbbbbbbbbbbbb", "tcp", tcpOrigin.Addr().(*net.TCPAddr).Port, remoteTCP)
	addProxy("ccccccccccccccc", "udp", udpOrigin.LocalAddr().(*net.UDPAddr).Port, remoteUDP)
	id := server.Id
	if e = client.LaunchFrpc(&id); e != nil {
		t.Fatal(e)
	}
	defer func() {
		_ = client.TerminateFrpc(&id)
		eventually(t, "FRPC monitor did not finish", func() bool { return !client.IsServerRunning(id) })
	}()
	tcpAddr, udpAddr := fmt.Sprintf("127.0.0.1:%d", remoteTCP), fmt.Sprintf("127.0.0.1:%d", remoteUDP)
	eventually(t, "TCP proxy traffic failed", func() bool { return tcpEcho(tcpAddr) })
	eventually(t, "UDP proxy traffic failed", func() bool { return udpEcho(udpAddr) })
	t.Log("Passed: real loopback TCP and UDP traffic through unchanged Podux FRPC service")
	cf := cloudflared.NewManager(cloudflared.Probe(fixture), app.DataDir())
	defer cf.Shutdown(context.Background())
	tunnel := d.Record{ID: "ddddddddddddddd", Revision: 1, Config: d.Config{Name: "coexistence", Protocol: "quic", EdgeIPVersion: "auto", GracePeriodSeconds: 1}}
	if e = cf.Start(tunnel, "synthetic-fixture-only"); e != nil {
		t.Fatal(e)
	}
	eventually(t, "test connector not ready", func() bool { return cf.Status(tunnel.ID).Ready })
	if !tcpEcho(tcpAddr) || !udpEcho(udpAddr) {
		t.Fatal("connector disturbed FRPC")
	}
	if e = cf.Stop(tunnel.ID); e != nil {
		t.Fatal(e)
	}
	if !tcpEcho(tcpAddr) || !udpEcho(udpAddr) {
		t.Fatal("stopping connector disturbed FRPC")
	}
	// Disable one proxy using the existing configuration reload path.
	tcpProxy.Set("status", "disabled")
	if e = app.Save(tcpProxy); e != nil {
		t.Fatal(e)
	}
	if e = client.ReloadFrpc(&id); e != nil {
		t.Fatal(e)
	}
	eventually(t, "disabled TCP proxy still forwards", func() bool { return !tcpEcho(tcpAddr) })
	eventually(t, "UDP did not recover after FRPC reload", func() bool { return udpEcho(udpAddr) })
	tcpProxy.Set("status", "enabled")
	if e = app.Save(tcpProxy); e != nil {
		t.Fatal(e)
	}
	if e = client.ReloadFrpc(&id); e != nil {
		t.Fatal(e)
	}
	eventually(t, "TCP proxy not restored", func() bool { return tcpEcho(tcpAddr) })
	stopFRPS()
	eventually(t, "FRPS outage did not interrupt traffic", func() bool { return !tcpEcho(tcpAddr) })
	startFRPS()
	eventually(t, "FRPC did not reconnect", func() bool { return tcpEcho(tcpAddr) && udpEcho(udpAddr) })
	topo := dashboard.NewService(app, serverapp.NewService(app, servers), proxyapp.NewService(app, proxies))
	data, e := topo.GetTopology()
	if e != nil || len(data.Servers) != 1 || len(data.Proxies) != 2 {
		t.Fatal("original topology regression", e)
	}
	if e = cf.Start(tunnel, "synthetic-fixture-only"); e != nil {
		t.Fatal(e)
	}
	eventually(t, "coexistence restart", func() bool { return cf.Status(tunnel.ID).Ready })
	if e = client.TerminateFrpc(&id); e != nil {
		t.Fatal(e)
	}
	if e = cf.Shutdown(context.Background()); e != nil {
		t.Fatal(e)
	}
	eventually(t, "FRPC did not stop", func() bool { return !client.IsServerRunning(id) })
	eventually(t, "TCP port not released", func() bool {
		l, e := net.Listen("tcp4", tcpAddr)
		if e == nil {
			l.Close()
		}
		return e == nil
	})
	eventually(t, "UDP port not released", func() bool {
		l, e := net.ListenPacket("udp4", udpAddr)
		if e == nil {
			l.Close()
		}
		return e == nil
	})
	entries, e := os.ReadDir(filepath.Join(app.DataDir(), "cloudflared"))
	if e != nil || len(entries) != 0 {
		t.Fatal("connector credentials left behind", e)
	}
	t.Log("Passed: single-proxy reload, FRPS outage/reconnect, original topology, joint service shutdown and port/credential cleanup")
}
