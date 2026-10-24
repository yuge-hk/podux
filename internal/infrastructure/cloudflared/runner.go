package cloudflared

import (
	"bufio"
	"context"
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/credentials"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager owns only cloudflared children; it never touches FRPC services.
type Manager struct {
	epoch     string
	mu        sync.Mutex
	instances map[string]*instance
	ports     map[int]string
	closed    bool
	binary    Binary
	root      string
	command   func(string, ...string) *exec.Cmd
	backoff   time.Duration
	stable    time.Duration
	client    *http.Client
}
type instance struct {
	op     sync.Mutex
	mu     sync.Mutex
	status d.Status
	logs   []d.Log
	cursor uint64
	cancel context.CancelFunc
	done   chan struct{}
}

func NewManager(b Binary, dataDir string) *Manager {
	absolute, err := filepath.Abs(dataDir)
	if err == nil {
		dataDir = absolute
	}
	return &Manager{epoch: rand.Text(), instances: map[string]*instance{}, ports: map[int]string{}, binary: b, root: filepath.Join(dataDir, "cloudflared"), command: exec.Command, backoff: time.Second, stable: 5 * time.Minute, client: &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (m *Manager) LogEpoch() string             { return m.epoch }
func (m *Manager) Capabilities() d.Capabilities { return m.binary.Caps }
func (m *Manager) get(id string) *instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.instances[id]
	if i == nil {
		i = &instance{status: d.Status{Phase: "stopped", Protocol: "unknown"}}
		m.instances[id] = i
	}
	return i
}
func (i *instance) event(message string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.cursor++
	i.logs = append(i.logs, d.Log{Cursor: i.cursor, Message: message})
	if len(i.logs) > 500 {
		i.logs = append([]d.Log{}, i.logs[len(i.logs)-500:]...)
	}
}
func (m *Manager) Status(id string) d.Status {
	i := m.get(id)
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.status
}
func (m *Manager) Logs(id string, after uint64) []d.Log {
	i := m.get(id)
	i.mu.Lock()
	defer i.mu.Unlock()
	out := []d.Log{}
	for _, l := range i.logs {
		if l.Cursor > after {
			out = append(out, l)
		}
	}
	return out
}
func (m *Manager) Start(r d.Record, token string) error {
	i := m.get(r.ID)
	i.op.Lock()
	defer i.op.Unlock()
	return m.start(i, r, token)
}
func (m *Manager) start(i *instance, r d.Record, token string) error {
	if !d.ValidID.MatchString(r.ID) || r.Config.Validate() != nil || token == "" {
		return d.ErrInvalid
	}
	if !m.binary.Caps.Available {
		return d.ErrUnavailable
	}
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return d.ErrConflict
	}
	i.mu.Lock()
	if i.done != nil {
		select {
		case <-i.done:
		default:
			i.mu.Unlock()
			return d.ErrConflict
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel
	i.done = make(chan struct{})
	done := i.done
	i.status = d.Status{Phase: "starting", Protocol: "unknown", Generation: i.status.Generation + 1, Revision: r.Revision}
	i.mu.Unlock()
	go m.supervise(ctx, i, r, token, done)
	return nil
}
func (m *Manager) Stop(id string) error {
	i := m.get(id)
	i.op.Lock()
	defer i.op.Unlock()
	return m.stop(i)
}
func (m *Manager) stop(i *instance) error {
	i.mu.Lock()
	cancel, done := i.cancel, i.done
	if cancel != nil {
		i.status.Phase = "stopping"
		cancel()
	}
	i.mu.Unlock()
	if done != nil {
		<-done
	}
	i.mu.Lock()
	i.status.Phase = "stopped"
	i.status.Ready = false
	i.status.Connections = 0
	i.status.NextRetry = ""
	i.mu.Unlock()
	return nil
}
func (m *Manager) Restart(r d.Record, token string) error {
	i := m.get(r.ID)
	i.op.Lock()
	defer i.op.Unlock()
	if e := m.stop(i); e != nil {
		return e
	}
	return m.start(i, r, token)
}
func (m *Manager) Forget(id string) { m.mu.Lock(); defer m.mu.Unlock(); delete(m.instances, id) }
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	ids := make([]string, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) { defer wg.Done(); _ = m.Stop(id) }(id)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (m *Manager) reserve(id string, port int) (string, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if port != 0 && m.ports[port] != "" {
		return "", nil, d.ErrConflict
	}
	l, e := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if e != nil {
		return "", nil, d.ErrConflict
	}
	actual := l.Addr().(*net.TCPAddr).Port
	if m.ports[actual] != "" {
		l.Close()
		return "", nil, d.ErrConflict
	}
	m.ports[actual] = id
	addr := l.Addr().String()
	l.Close()
	return addr, func() { m.mu.Lock(); delete(m.ports, actual); m.mu.Unlock() }, nil
}
func (m *Manager) supervise(ctx context.Context, i *instance, r d.Record, token string, done chan struct{}) {
	defer close(done)
	attempts := 0
	for {
		if ctx.Err() != nil {
			m.finish(i, "stopped", "")
			return
		}
		started := time.Now()
		failed := m.run(ctx, i, r, token)
		if ctx.Err() != nil {
			m.finish(i, "stopped", "")
			return
		}
		if !failed {
			m.finish(i, "stopped", "")
			return
		}
		if time.Since(started) >= m.stable {
			attempts = 0
		}
		if !r.AutoRestart || attempts >= r.MaxRestartAttempts {
			m.finish(i, "failed", "cloudflared exited or could not start")
			return
		}
		delay := m.backoff * time.Duration(1<<min(attempts, 6))
		if delay > 60*time.Second {
			delay = 60 * time.Second
		}
		attempts++
		i.mu.Lock()
		i.status.Phase = "backoff"
		i.status.Ready = false
		i.status.RestartCount = attempts
		i.status.NextRetry = time.Now().Add(delay).UTC().Format(time.RFC3339Nano)
		i.mu.Unlock()
		i.event("automatic restart scheduled")
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			m.finish(i, "stopped", "")
			return
		case <-timer.C:
		}
	}
}
func (m *Manager) finish(i *instance, phase, last string) {
	i.mu.Lock()
	i.status.Phase = phase
	i.status.Ready = false
	i.status.Connections = 0
	i.status.Protocol = "unknown"
	i.status.MetricsAddress = ""
	i.status.NextRetry = ""
	if last != "" || phase == "stopped" {
		i.status.LastError = last
	}
	i.mu.Unlock()
}
func (m *Manager) run(ctx context.Context, i *instance, r d.Record, token string) bool {
	i.mu.Lock()
	i.status.Phase = "starting"
	i.status.Protocol = "unknown"
	i.status.Ready = false
	i.status.Connections = 0
	i.status.NextRetry = ""
	i.status.MetricsAddress = ""
	i.mu.Unlock()
	addr, release, e := m.reserve(r.ID, r.MetricsPort)
	if e != nil {
		i.event("metrics address unavailable")
		return true
	}
	defer release()
	if e = credentials.PrivateDir(m.root); e != nil {
		i.event("runtime directory unavailable")
		return true
	}
	dir, e := os.MkdirTemp(m.root, r.ID+"-")
	if e != nil {
		return true
	}
	defer os.RemoveAll(dir)
	if credentials.PrivateDir(dir) != nil {
		return true
	}
	tokenPath := filepath.Join(dir, "token")
	configPath := filepath.Join(dir, "config.yml")
	if e = credentials.PrivateFile(tokenPath, []byte(token)); e != nil {
		return true
	}
	// Explicit empty config prevents implicit ~/.cloudflared/config.yml loading.
	if e = credentials.PrivateFile(configPath, []byte("{}\n")); e != nil {
		return true
	}
	args, e := BuildArgs(r.Config, tokenPath, configPath, addr)
	if e != nil {
		return true
	}
	cmd := m.command(m.binary.Path, args...)
	cmd.Env = cleanEnv()
	cmd.Dir = dir
	prepareProcess(cmd)
	sink := &outputSink{instance: i, metrics: addr}
	cmd.Stdout = sink
	cmd.Stderr = sink
	// WaitDelay bounds inherited pipe handles even when a child spawns descendants.
	cmd.WaitDelay = 2 * time.Second
	if e = cmd.Start(); e != nil {
		i.event("cloudflared launch failed")
		return true
	}
	i.event("cloudflared process started")
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }() // exactly one Wait per child
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case e = <-exited:
			sink.flush()
			killProcess(cmd)
			if e != nil {
				i.event("cloudflared process exited with error")
			} else {
				i.event("cloudflared process exited normally")
			}
			return e != nil
		case <-ctx.Done():
			signalProcess(cmd)
			timer := time.NewTimer(time.Duration(r.GracePeriodSeconds)*time.Second + time.Second)
			select {
			case <-exited:
				timer.Stop()
			case <-timer.C:
				killProcess(cmd)
				<-exited
			}
			killProcess(cmd)
			sink.flush()
			i.event("cloudflared process stopped")
			return false
		case <-ticker.C:
			i.mu.Lock()
			listening := i.status.MetricsAddress == addr
			i.mu.Unlock()
			if listening {
				req, e := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/ready", nil)
				if e != nil {
					continue
				}
				res, e := m.client.Do(req)
				ready := e == nil && res.StatusCode == 200
				if res != nil {
					res.Body.Close()
				}
				i.mu.Lock()
				i.status.Ready = ready
				if ready {
					i.status.Phase = "running"
					i.status.LastError = ""
				} else {
					i.status.Phase = "starting"
				}
				i.mu.Unlock()
				if ready {
					m.collectMetrics(ctx, i, addr)
				}
			}
		}
	}
}

var connectedProtocol = regexp.MustCompile(`(?:^|[ \t])protocol=(quic|http2)(?:[ \t]|$)`)

type outputSink struct {
	mu       sync.Mutex
	pending  string
	discard  bool
	instance *instance
	metrics  string
}

func (s *outputSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p)
	for _, b := range p {
		if b == '\n' {
			if !s.discard {
				s.line(s.pending)
			}
			s.pending = ""
			s.discard = false
		} else if !s.discard {
			if len(s.pending) >= 8192 {
				s.pending = ""
				s.discard = true
			} else {
				s.pending += string(b)
			}
		}
	}
	return n, nil
}
func (s *outputSink) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.discard && s.pending != "" {
		s.line(s.pending)
	}
	s.pending = ""
}

// Never relay arbitrary child output: it may contain Token fragments, remote
// origin credentials, file paths or secrets unknown to this process. Emit only
// allowlisted semantic events with fixed text and strictly parsed protocol.
func (s *outputSink) line(line string) {
	if strings.Contains(line, "Starting metrics server on "+s.metrics+"/") {
		s.instance.mu.Lock()
		s.instance.status.MetricsAddress = s.metrics
		s.instance.mu.Unlock()
		s.instance.event("metrics listener started")
		return
	}
	if strings.Contains(line, "Registered tunnel connection") {
		p := connectedProtocol.FindStringSubmatch(line)
		if p != nil {
			s.instance.mu.Lock()
			s.instance.status.Protocol = p[1]
			s.instance.mu.Unlock()
			s.instance.event("tunnel connection registered (" + p[1] + ")")
		}
		return
	}
	if strings.Contains(line, " ERR ") || strings.Contains(line, `"level":"error"`) {
		s.instance.mu.Lock()
		if s.instance.status.Phase == "stopping" || s.instance.status.Phase == "stopped" {
			s.instance.mu.Unlock()
			return
		}
		s.instance.status.LastError = "cloudflared reported a connection error"
		s.instance.mu.Unlock()
		s.instance.event("cloudflared reported a connection error")
	}
}

var _ d.Runner = (*Manager)(nil)

func (m *Manager) collectMetrics(ctx context.Context, i *instance, addr string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/metrics", nil)
	if err != nil {
		return
	}
	res, err := m.client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return
	}
	scan := bufio.NewScanner(io.LimitReader(res.Body, 256*1024))
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) == 2 && fields[0] == "cloudflared_tunnel_ha_connections" {
			v, e := strconv.Atoi(fields[1])
			if e == nil && v >= 0 && v <= 1000 {
				i.mu.Lock()
				i.status.Connections = v
				i.mu.Unlock()
			}
			return
		}
	}
}
