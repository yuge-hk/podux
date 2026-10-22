package cloudflared

import (
	d "podux/internal/domain/tunnel"
	"strings"
	"testing"
)

func config() d.Config {
	return d.Config{Name: "test", Protocol: "auto", EdgeIPVersion: "auto", ConnectionRetries: 5, GracePeriodSeconds: 1, MaxRestartAttempts: 2}
}
func TestArgs(t *testing.T) {
	c := config()
	a, e := BuildArgs(c, "/private/token", "/private/config", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	s := strings.Join(a, " ")
	if !strings.Contains(s, "run --token-file /private/token --protocol auto") || strings.Contains(s, "--token ") {
		t.Fatal(s)
	}
	for _, mut := range []func(*d.Config){func(c *d.Config) { c.Protocol = "shell" }, func(c *d.Config) { c.Region = "eu" }, func(c *d.Config) { c.PostQuantum = true }, func(c *d.Config) { c.MetricsPort = 65536 }, func(c *d.Config) { c.ConnectionRetries = -1 }, func(c *d.Config) { c.ExtraArgs = []d.ExtraArg{{Name: "token", Value: "secret"}} }, func(c *d.Config) { c.EdgeBindAddress = "127.0.0.1"; c.EdgeIPVersion = "6" }} {
		c := config()
		mut(&c)
		if _, e = BuildArgs(c, "t", "c", "m"); e == nil {
			t.Fatal("accepted invalid config")
		}
	}
}
func TestEnvironment(t *testing.T) {
	t.Setenv("TUNNEL_TOKEN", "secret")
	t.Setenv("NO_TLS_VERIFY", "true")
	t.Setenv("LD_PRELOAD", "bad")
	for _, v := range cleanEnv() {
		if strings.Contains(v, "secret") || strings.HasPrefix(v, "NO_TLS_VERIFY=") || strings.HasPrefix(v, "LD_PRELOAD=") {
			t.Fatal(v)
		}
	}
}
