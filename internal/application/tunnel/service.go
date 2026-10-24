package tunnel

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	d "podux/internal/domain/tunnel"
	"regexp"
	"strings"
	"sync"
)

// Service serializes configuration writes and runtime commands per profile.
// Long process stops never hold the shared configuration/state mutex.
type operation struct {
	mu    sync.Mutex
	users int
}
type Service struct {
	operations map[string]*operation
	failures   map[string]string
	mu         sync.Mutex
	repo       d.Repository
	vault      d.Vault
	runner     d.Runner
	closed     bool
}

func NewService(repo d.Repository, vault d.Vault, runner d.Runner) *Service {
	return &Service{repo: repo, vault: vault, runner: runner, failures: map[string]string{}, operations: map[string]*operation{}}
}
func (s *Service) lockProfile(id string) func() {
	s.mu.Lock()
	op := s.operations[id]
	if op == nil {
		op = &operation{}
		s.operations[id] = op
	}
	op.users++
	s.mu.Unlock()
	op.mu.Lock()
	return func() {
		op.mu.Unlock()
		s.mu.Lock()
		op.users--
		if op.users == 0 {
			delete(s.operations, id)
		}
		s.mu.Unlock()
	}
}
func (s *Service) LogEpoch() string             { return s.runner.LogEpoch() }
func (s *Service) Capabilities() d.Capabilities { return s.runner.Capabilities() }
func (s *Service) List() ([]d.Record, error)    { s.mu.Lock(); defer s.mu.Unlock(); return s.repo.List() }
func (s *Service) Get(id string) (d.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.Get(id)
}

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_+/=-]+$`)

func validToken(value string) bool {
	if len(value) < 32 || len(value) > 8192 || !tokenPattern.MatchString(value) {
		return false
	}
	// Validate only the locally documented token structure. Never call Cloudflare.
	var raw []byte
	var e error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		raw, e = enc.DecodeString(value)
		if e == nil {
			break
		}
	}
	if e != nil {
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return false
	}
	for _, k := range []string{"a", "t", "s"} {
		var v string
		if json.Unmarshal(fields[k], &v) != nil || v == "" {
			return false
		}
	}
	return true
}
func newID() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, 15)
	for i := range out {
		for {
			var b [1]byte
			if _, e := rand.Read(b[:]); e != nil {
				return "", e
			}
			if b[0] < 252 {
				out[i] = letters[int(b[0])%36]
				break
			}
		}
	}
	return string(out), nil
}
func (s *Service) Create(w d.Write) (d.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return d.Record{}, d.ErrConflict
	}
	if w.Config.Validate() != nil || w.Token == nil || !validToken(strings.TrimSpace(*w.Token)) {
		return d.Record{}, d.ErrInvalid
	}
	id, e := newID()
	if e != nil {
		return d.Record{}, d.ErrCredentials
	}
	if s.vault == nil {
		return d.Record{}, d.ErrCredentials
	}
	cipher, e := s.vault.Encrypt(id, strings.TrimSpace(*w.Token))
	if e != nil {
		return d.Record{}, d.ErrCredentials
	}
	return s.repo.Create(d.Record{ID: id, Config: w.Config, Ciphertext: cipher, HasToken: true, Revision: 1})
}
func (s *Service) Update(id string, w d.Write) (d.Record, error) {
	unlock := s.lockProfile(id)
	defer unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return d.Record{}, d.ErrConflict
	}
	if w.Config.Validate() != nil || w.Revision < 1 {
		return d.Record{}, d.ErrInvalid
	}
	r, e := s.repo.Get(id)
	if e != nil {
		return d.Record{}, e
	}
	if w.Revision != r.Revision {
		return d.Record{}, d.ErrConflict
	}
	r.Config = w.Config
	if w.Token != nil {
		token := strings.TrimSpace(*w.Token)
		if !validToken(token) {
			return d.Record{}, d.ErrInvalid
		}
		if s.vault == nil {
			return d.Record{}, d.ErrCredentials
		}
		r.Ciphertext, e = s.vault.Encrypt(id, token)
		if e != nil {
			return d.Record{}, d.ErrCredentials
		}
	}
	return s.repo.Update(r, w.Revision)
}
func (s *Service) Delete(id string, revision int) error {
	unlock := s.lockProfile(id)
	defer unlock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return d.ErrConflict
	}
	r, err := s.repo.Get(id)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if r.Revision != revision {
		return d.ErrConflict
	}
	if err = s.runner.Stop(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return d.ErrConflict
	}
	if err = s.repo.Delete(id, revision); err == nil {
		s.runner.Forget(id)
		delete(s.failures, id)
	}
	return err
}
func (s *Service) command(id, action string) (err error) {
	unlock := s.lockProfile(id)
	defer unlock()
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err == d.ErrCredentials || err == d.ErrUnavailable {
			s.failures[id] = err.Error()
		} else if err == nil {
			delete(s.failures, id)
		}
	}()
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return d.ErrConflict
	}
	s.mu.Lock()
	r, e := s.repo.Get(id)
	s.mu.Unlock()
	if e != nil {
		return e
	}
	if action == "stop" {
		return s.runner.Stop(id)
	}
	if s.vault == nil || !r.HasToken || r.Ciphertext == "" {
		return d.ErrCredentials
	}
	token, e := s.vault.Decrypt(id, r.Ciphertext)
	if e != nil || !validToken(strings.TrimSpace(token)) {
		return d.ErrCredentials
	}
	switch action {
	case "start":
		return s.runner.Start(r, token)
	case "restart":
		return s.runner.Restart(r, token)
	}
	return d.ErrInvalid
}
func (s *Service) Start(id string) error   { return s.command(id, "start") }
func (s *Service) Stop(id string) error    { return s.command(id, "stop") }
func (s *Service) Restart(id string) error { return s.command(id, "restart") }
func (s *Service) Status(id string) (d.Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := s.repo.Get(id); e != nil {
		return d.Status{}, e
	}
	status := s.runner.Status(id)
	if message := s.failures[id]; message != "" && (status.Phase == "stopped" || status.Phase == "failed") {
		status.Phase = "failed"
		status.LastError = message
	}
	return status, nil
}
func (s *Service) Logs(id string, after uint64) ([]d.Log, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := s.repo.Get(id); e != nil {
		return nil, e
	}
	return s.runner.Logs(id, after), nil
}

// Recover starts only autoStart records. Old runtime state and retry budgets are
// never persisted, adopted or inferred from a stored PID.
func (s *Service) Recover() {
	rs, e := s.List()
	if e != nil {
		return
	}
	for _, r := range rs {
		if r.AutoStart {
			_ = s.Start(r.ID)
		}
	}
}
func (s *Service) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return s.runner.Shutdown(ctx)
}
