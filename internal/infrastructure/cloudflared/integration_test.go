package cloudflared

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"strings"
	"testing"
	"time"
)

// Opt in with an already downloaded official binary; CI unit tests stay offline.
func TestOfficialBinary(t *testing.T) {
	path := os.Getenv("PODUX_TEST_CLOUDFLARED")
	if path == "" {
		t.Skip("set PODUX_TEST_CLOUDFLARED for official binary compatibility checks")
	}
	b := Probe(path)
	if !b.Caps.Available || !b.Caps.TokenFile || b.Caps.TLSLocalOverride {
		t.Fatalf("capabilities %+v", b.Caps)
	}
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	cfg := filepath.Join(dir, "config.yml")
	if e := os.WriteFile(tokenFile, []byte("invalid-test-token-never-a-real-credential"), 0600); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(cfg, []byte("{}\n"), 0600)
	for _, protocol := range []string{"auto", "quic", "http2"} {
		c := config()
		c.Protocol = protocol
		c.ExtraArgs = []d.ExtraArg{{Name: "ha-connections", Value: "2"}}
		c.Region = "us"
		c.EdgeIPVersion = "4"
		c.EdgeBindAddress = "127.0.0.1"
		c.ConnectionRetries = 0
		c.GracePeriodSeconds = 1
		c.PostQuantum = protocol == "quic"
		args, e := BuildArgs(c, tokenFile, cfg, "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, b.Path, args...)
		cmd.Env = cleanEnv()
		out, e := cmd.CombinedOutput()
		cancel()
		if e == nil {
			t.Fatal("invalid token accepted")
		}
		s := string(out)
		if strings.Contains(s, "flag provided but not defined") || strings.Contains(s, "Incorrect Usage") || !strings.Contains(strings.ToLower(s), "token") {
			t.Fatalf("unexpected parse result: %s", s)
		}
	}
	// Real child lifecycle with file credential input, failure reporting, cleanup.
	m := NewManager(b, t.TempDir())
	r := record("aaaaaaaaaaaaaaa")
	if e := m.Start(r, "invalid-test-token-never-a-real-credential"); e != nil {
		t.Fatal(e)
	}
	await(t, m, r.ID, "failed")
	if e := m.Shutdown(context.Background()); e != nil {
		t.Fatal(e)
	}
	for _, l := range m.Logs(r.ID, 0) {
		if strings.Contains(l.Message, "invalid-test-token") {
			t.Fatal("token leaked")
		}
	}
}
