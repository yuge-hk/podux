package cloudflared

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"strings"
	"sync"
	"testing"
	"time"
)

// A real Go child process, never a shell. No production credentials or network
// access are required. Modes cover graceful/forced exits and failed starts.
func TestHelperProcess(t *testing.T) {
	args := os.Args
	pos := -1
	for j, a := range args {
		if a == "--helper" {
			pos = j
			break
		}
	}
	if pos < 0 {
		return
	}
	mode := args[pos+1]
	args = args[pos+2:]
	if mode == "fail" {
		fmt.Fprintln(os.Stderr, " ERR deliberately-private-token-value")
		os.Exit(7)
	}
	if mode == "normal" {
		os.Exit(0)
	}
	var addr, tokenPath string
	for j, a := range args {
		if j+1 < len(args) {
			if a == "--metrics" {
				addr = args[j+1]
			}
			if a == "--token-file" {
				tokenPath = args[j+1]
			}
		}
	}
	token, e := os.ReadFile(tokenPath)
	if e != nil {
		os.Exit(8)
	}
	fmt.Fprintln(os.Stderr, "unknown secret:", string(token))
	l, e := net.Listen("tcp4", addr)
	if e != nil {
		os.Exit(9)
	}
	fmt.Fprintln(os.Stderr, " INF Starting metrics server on "+addr+"/metrics")
	fmt.Fprintln(os.Stderr, " INF Registered tunnel connection connIndex=0 protocol=quic")
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 2") })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	go http.Serve(l, mux)
	if mode == "ignore" {
		signal.Ignore(os.Interrupt)
		ignoreTerm()
		select {}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	notifyTerm(ch)
	<-ch
	fmt.Fprintln(os.Stderr, " ERR connection canceled during graceful shutdown")
	os.Exit(0)
}
func manager(t *testing.T, mode string) *Manager {
	t.Helper()
	m := NewManager(Binary{Path: os.Args[0], Caps: d.Capabilities{Available: true, TokenFile: true}}, t.TempDir())
	m.backoff = 10 * time.Millisecond
	m.command = func(path string, args ...string) *exec.Cmd {
		return exec.Command(path, append([]string{"-test.run=^TestHelperProcess$", "--", "--helper", mode}, args...)...)
	}
	t.Cleanup(func() {
		ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		if e := m.Shutdown(ctx); e != nil {
			t.Error(e)
		}
	})
	return m
}
func record(id string) d.Record { return d.Record{ID: id, Config: config(), Revision: 1} }
func await(t *testing.T, m *Manager, id, phase string) d.Status {
	t.Helper()
	until := time.Now().Add(6 * time.Second)
	for time.Now().Before(until) {
		s := m.Status(id)
		if s.Phase == phase {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("wanted %s, got %+v", phase, m.Status(id))
	return d.Status{}
}
func TestLifecycleIsolation(t *testing.T) {
	m := manager(t, "serve")
	a, b := record("aaaaaaaaaaaaaaa"), record("bbbbbbbbbbbbbbb")
	var wg sync.WaitGroup
	success := make(chan bool, 10)
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); success <- m.Start(a, "deliberately-private-token-value") == nil }()
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("started %d", n)
	}
	sa := await(t, m, a.ID, "running")
	if sa.Protocol != "quic" || !sa.Ready {
		t.Fatal(sa)
	}
	if e := m.Start(b, "other-private-value"); e != nil {
		t.Fatal(e)
	}
	sb := await(t, m, b.ID, "running")
	if sa.MetricsAddress == sb.MetricsAddress {
		t.Fatal("metrics collision")
	}
	if e := m.Stop(a.ID); e != nil {
		t.Fatal(e)
	}
	if m.Status(b.ID).Phase != "running" {
		t.Fatal("other tunnel stopped")
	}
	if e := m.Restart(a, "replacement-private-value"); e != nil {
		t.Fatal(e)
	}
	sa = await(t, m, a.ID, "running")
	if sa.Generation != 2 {
		t.Fatal(sa)
	}
	for _, log := range m.Logs(a.ID, 0) {
		if strings.Contains(log.Message, "private") {
			t.Fatal("secret leaked")
		}
	}
	_ = m.Stop(a.ID)
	if files, _ := os.ReadDir(m.root); len(files) != 1 {
		t.Fatalf("only b runtime should remain: %d", len(files))
	}
}
func TestRestartBudgetAndCancellation(t *testing.T) {
	m := manager(t, "fail")
	r := record("aaaaaaaaaaaaaaa")
	r.AutoRestart = true
	r.MaxRestartAttempts = 2
	if e := m.Start(r, "private-value"); e != nil {
		t.Fatal(e)
	}
	s := await(t, m, r.ID, "failed")
	if s.RestartCount != 2 {
		t.Fatal(s)
	}
	n := 0
	for _, l := range m.Logs(r.ID, 0) {
		if l.Message == "cloudflared process started" {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("want initial + two retries, got %d", n)
	}
	m.backoff = time.Second
	if e := m.Start(r, "private-value"); e != nil {
		t.Fatal(e)
	}
	await(t, m, r.ID, "backoff")
	_ = m.Stop(r.ID)
	time.Sleep(30 * time.Millisecond)
	if s = m.Status(r.ID); s.Phase != "stopped" || s.NextRetry != "" {
		t.Fatal(s)
	}
}
func TestNormalExitDoesNotRestart(t *testing.T) {
	m := manager(t, "normal")
	r := record("aaaaaaaaaaaaaaa")
	r.AutoRestart = true
	_ = m.Start(r, "private")
	s := await(t, m, r.ID, "stopped")
	if s.RestartCount != 0 {
		t.Fatal(s)
	}
}
func TestForcedStop(t *testing.T) {
	m := manager(t, "ignore")
	r := record("aaaaaaaaaaaaaaa")
	_ = m.Start(r, "private")
	await(t, m, r.ID, "running")
	start := time.Now()
	_ = m.Stop(r.ID)
	if time.Since(start) > 4*time.Second {
		t.Fatal("stop unbounded")
	}
	if files, _ := os.ReadDir(m.root); len(files) != 0 {
		t.Fatal("credential file remains")
	}
}
func TestMetricsConflict(t *testing.T) {
	m := manager(t, "serve")
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	r := record("aaaaaaaaaaaaaaa")
	r.MetricsPort = l.Addr().(*net.TCPAddr).Port
	_ = m.Start(r, "private")
	await(t, m, r.ID, "failed")
}
func TestBoundedLogs(t *testing.T) {
	i := &instance{}
	s := &outputSink{instance: i, metrics: "127.0.0.1:1000"}
	for range 600 {
		_, _ = s.Write([]byte(" ERR token=private\n"))
	}
	_, _ = s.Write([]byte(strings.Repeat("x", 100000) + "\n"))
	if len(i.logs) != 500 || len(s.pending) > 8192 {
		t.Fatal("unbounded logs")
	}
	for _, l := range i.logs {
		if strings.Contains(l.Message, "private") {
			t.Fatal("leak")
		}
	}
}
func TestDataDirectoryLock(t *testing.T) {
	dir := t.TempDir()
	release, e := LockDataDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	if other, e := LockDataDir(dir); e == nil {
		other()
		t.Fatal("duplicate lock")
	}
	release()
	release, e = LockDataDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	release()
}
func TestShutdownRejectsStarts(t *testing.T) {
	m := manager(t, "serve")
	r := record("aaaaaaaaaaaaaaa")
	_ = m.Start(r, "private")
	await(t, m, r.ID, "running")
	if e := m.Shutdown(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := m.Start(r, "private"); e != d.ErrConflict {
		t.Fatal(e)
	}
}
func TestMetricsParsingAndStaleFiles(t *testing.T) {
	m := manager(t, "serve")
	r := record("aaaaaaaaaaaaaaa")
	if e := m.Start(r, "private"); e != nil {
		t.Fatal(e)
	}
	s := await(t, m, r.ID, "running")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s = m.Status(r.ID)
		if s.Connections == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.Connections != 2 {
		t.Fatal(s)
	}
	_ = m.Stop(r.ID)
	stale := filepath.Join(m.root, "aaaaaaaaaaaaaaa-12345")
	os.MkdirAll(stale, 0700)
	os.WriteFile(filepath.Join(stale, "token"), []byte("test-only"), 0600)
	if e := PrepareRuntime(filepath.Dir(m.root)); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(stale); !os.IsNotExist(e) {
		t.Fatal("stale credentials remain")
	}
}
func TestInternalFixedMetricsConflict(t *testing.T) {
	m := manager(t, "serve")
	a, b := record("aaaaaaaaaaaaaaa"), record("bbbbbbbbbbbbbbb")
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	a.MetricsPort = port
	b.MetricsPort = port
	if e = m.Start(a, "private"); e != nil {
		t.Fatal(e)
	}
	await(t, m, a.ID, "running")
	if e = m.Start(b, "private"); e != nil {
		t.Fatal(e)
	}
	await(t, m, b.ID, "failed")
	if m.Status(a.ID).Phase != "running" {
		t.Fatal("conflicting start disturbed first tunnel")
	}
	_ = m.Stop(a.ID)
	if e = m.Start(b, "private"); e != nil {
		t.Fatal(e)
	}
	await(t, m, b.ID, "running")
}
func TestLogEpochChangesAcrossManagerRestart(t *testing.T) {
	a := NewManager(Binary{}, t.TempDir())
	b := NewManager(Binary{}, t.TempDir())
	if a.LogEpoch() == "" || a.LogEpoch() == b.LogEpoch() {
		t.Fatal("log epochs must identify manager lifetime")
	}
}

func TestStopDoesNotReportShutdownConnectionErrors(t *testing.T) {
	m := manager(t, "serve")
	r := record("aaaaaaaaaaaaaaa")
	if err := m.Start(r, "synthetic-test-only"); err != nil {
		t.Fatal(err)
	}
	await(t, m, r.ID, "running")
	if err := m.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	if status := m.Status(r.ID); status.Phase != "stopped" || status.LastError != "" {
		t.Fatalf("unexpected stopped status: %+v", status)
	}
	for _, log := range m.Logs(r.ID, 0) {
		if log.Message == "cloudflared reported a connection error" {
			t.Fatal("shutdown reported as connection failure")
		}
	}
	// Real runtime errors must still be recorded.
	i := &instance{status: d.Status{Phase: "running"}}
	sink := &outputSink{instance: i}
	sink.line(" ERR synthetic connection failure")
	if i.status.LastError == "" {
		t.Fatal("runtime error suppressed")
	}
}
