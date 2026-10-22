package cloudflared

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Binary struct {
	Path string
	Caps d.Capabilities
}

var versionRE = regexp.MustCompile(`cloudflared version (20[0-9]{2})\.([0-9]+)\.([0-9]+)(?:[^\r\n]*)`)
var verifiedFlags = []string{"token-file", "protocol", "region", "retries", "grace-period", "edge-ip-version", "edge-bind-address", "post-quantum", "metrics", "loglevel", "no-autoupdate", "config", "ha-connections"}

// boundedOutput prevents an invalid executable from exhausting memory during probing.
type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 65536 {
		left := 65536 - b.Len()
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
func Probe(path string) Binary {
	b := Binary{}
	if path == "" {
		path = "cloudflared"
	}
	p, e := exec.LookPath(path)
	if e != nil {
		return b
	}
	p, e = filepath.Abs(p)
	if e != nil {
		return b
	}
	run := func(args ...string) (string, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, args...)
		cmd.Env = cleanEnv()
		var out boundedOutput
		cmd.Stdout = &out
		cmd.Stderr = &out
		e := cmd.Run()
		return out.String(), e == nil
	}
	s, ok := run("--version")
	if !ok {
		return b
	}
	m := versionRE.FindStringSubmatch(s)
	if m == nil {
		return b
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	if year < 2025 || (year == 2025 && month < 4) {
		return b
	}
	parent, ok := run("tunnel", "--help")
	if !ok {
		return b
	}
	sub, ok := run("tunnel", "run", "--help")
	if !ok {
		return b
	}
	// Some supported edge/protocol flags are hidden from help. Test their
	// parser acceptance without reading credentials or starting a connector.
	if !strings.Contains(parent+sub, "--token-file") {
		return b
	}
	probeConfig := d.Config{Name: "probe", Protocol: "quic", Region: "us", PostQuantum: true, ConnectionRetries: 0, GracePeriodSeconds: 1, EdgeIPVersion: "4", EdgeBindAddress: "127.0.0.1", ExtraArgs: []d.ExtraArg{{Name: "loglevel", Value: "info"}, {Name: "ha-connections", Value: "2"}}}
	args, err := BuildArgs(probeConfig, "unused-token", "unused-config", "127.0.0.1:0")
	if err != nil {
		return b
	}
	if _, ok := run(append(args, "--help")...); !ok {
		return b
	}
	b.Path = p
	b.Caps = d.Capabilities{Available: true, Version: strconv.Itoa(year) + "." + strconv.Itoa(month) + "." + strconv.Itoa(patch), Build: buildInfo(m[0]), TokenFile: true, Flags: append([]string{}, verifiedFlags...)}
	return b
}
func BuildArgs(c d.Config, tokenFile, configFile, metrics string) ([]string, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if tokenFile == "" || configFile == "" {
		return nil, d.ErrInvalid
	}
	a := []string{"tunnel", "--config", configFile, "--no-autoupdate", "--metrics", metrics, "--grace-period", strconv.Itoa(c.GracePeriodSeconds) + "s", "--retries", strconv.Itoa(c.ConnectionRetries), "--edge-ip-version", c.EdgeIPVersion}
	if c.Region != "" {
		a = append(a, "--region", c.Region)
	}
	if c.EdgeBindAddress != "" {
		a = append(a, "--edge-bind-address", c.EdgeBindAddress)
	}
	for _, v := range c.ExtraArgs {
		a = append(a, "--"+v.Name, v.Value)
	}
	a = append(a, "run", "--token-file", tokenFile, "--protocol", c.Protocol)
	if c.PostQuantum {
		a = append(a, "--post-quantum")
	}
	return a, nil
}
func cleanEnv() []string {
	// Minimal allowlist also excludes proxy credentials, TUNNEL_*, CLI overrides,
	// preload injection and inherited credentials. No secrets travel via argv/env.
	allowed := map[string]bool{"PATH": true, "SYSTEMROOT": true, "WINDIR": true, "TEMP": true, "TMP": true, "TMPDIR": true, "TZ": true}
	out := []string{}
	for _, s := range os.Environ() {
		k, _, _ := strings.Cut(s, "=")
		if allowed[strings.ToUpper(k)] {
			out = append(out, s)
		}
	}
	return out
}

var buildRE = regexp.MustCompile(`\(built ([0-9: -]+ UTC)\)`)

func buildInfo(s string) string {
	m := buildRE.FindStringSubmatch(s)
	if m == nil {
		return "unknown"
	}
	return m[1]
}
