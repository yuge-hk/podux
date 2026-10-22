package tunnel

import (
	"context"
	"encoding/base64"
	"github.com/pocketbase/pocketbase/core"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/credentials"
	"podux/internal/infrastructure/persistence"
	_ "podux/migrations"
	"testing"
	"time"
)

type fakeRunner struct {
	started []string
	stopped []string
}

func (f *fakeRunner) LogEpoch() string             { return "test-epoch" }
func (f *fakeRunner) Capabilities() d.Capabilities { return d.Capabilities{} }
func (f *fakeRunner) Start(r d.Record, _ string) error {
	f.started = append(f.started, r.ID)
	return nil
}
func (f *fakeRunner) Stop(id string) error               { f.stopped = append(f.stopped, id); return nil }
func (f *fakeRunner) Restart(r d.Record, s string) error { _ = f.Stop(r.ID); return f.Start(r, s) }
func (f *fakeRunner) Status(string) d.Status             { return d.Status{Phase: "stopped"} }
func (f *fakeRunner) Logs(string, uint64) []d.Log        { return nil }
func (f *fakeRunner) Forget(string)                      {}
func (f *fakeRunner) Shutdown(context.Context) error     { return nil }
func TestRecoveryAndSafeUpdates(t *testing.T) {
	dir := t.TempDir()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: filepath.Join(dir, "data")})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e := app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	repo := persistence.NewTunnelRepository(app)
	v, e := credentials.Open(app.DataDir(), filepath.Join(dir, "secret", "key"), false)
	if e != nil {
		t.Fatal(e)
	}
	f := &fakeRunner{}
	s := NewService(repo, v, f)
	token := base64.StdEncoding.EncodeToString([]byte(`{"a":"test-account","t":"00000000-0000-0000-0000-000000000000","s":"dGVzdA=="}`))
	c := d.Config{Name: "test", Protocol: "auto", EdgeIPVersion: "auto", GracePeriodSeconds: 1, ConnectionRetries: 5}
	a, e := s.Create(d.Write{Config: c, Token: &token})
	if e != nil {
		t.Fatal(e)
	}
	c.AutoStart = true
	b, e := s.Create(d.Write{Config: c, Token: &token})
	if e != nil {
		t.Fatal(e)
	}
	// A manually started non-autoStart tunnel is not restored.
	if e = s.Start(a.ID); e != nil {
		t.Fatal(e)
	}
	f.started = nil
	s.Recover()
	if len(f.started) != 1 || f.started[0] != b.ID {
		t.Fatal(f.started)
	}
	// User stop leaves autoStart unchanged and takes effect until next recovery.
	if e = s.Stop(b.ID); e != nil {
		t.Fatal(e)
	}
	s.Recover()
	if len(f.started) != 2 {
		t.Fatal(f.started)
	}
	// Reopen using the persisted key, simulating a fresh service process.
	v2, e := credentials.Open(app.DataDir(), filepath.Join(dir, "secret", "key"), true)
	if e != nil {
		t.Fatal(e)
	}
	f2 := &fakeRunner{}
	s2 := NewService(repo, v2, f2)
	s2.Recover()
	if len(f2.started) != 1 {
		t.Fatal(f2.started)
	}
	if e = s2.Shutdown(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e = s2.Start(a.ID); e != d.ErrConflict {
		t.Fatal(e)
	}
	noKey := NewService(repo, nil, &fakeRunner{})
	if e = noKey.Start(a.ID); e != d.ErrCredentials {
		t.Fatal(e)
	}
}

type blockingRunner struct {
	fakeRunner
	entered chan struct{}
	release chan struct{}
}

func (f *blockingRunner) Stop(string) error { close(f.entered); <-f.release; return nil }
func TestStopDoesNotBlockOtherProfile(t *testing.T) {
	dir := t.TempDir()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: filepath.Join(dir, "data")})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e := app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	repo := persistence.NewTunnelRepository(app)
	v, e := credentials.Open(app.DataDir(), filepath.Join(dir, "key"), false)
	if e != nil {
		t.Fatal(e)
	}
	runner := &blockingRunner{entered: make(chan struct{}), release: make(chan struct{})}
	s := NewService(repo, v, runner)
	token := base64.StdEncoding.EncodeToString([]byte(`{"a":"test","t":"00000000-0000-0000-0000-000000000000","s":"dGVzdA=="}`))
	config := d.Config{Name: "test", Protocol: "auto", EdgeIPVersion: "auto", GracePeriodSeconds: 1}
	a, e := s.Create(d.Write{Config: config, Token: &token})
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Create(d.Write{Config: config, Token: &token})
	if e != nil {
		t.Fatal(e)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- s.Stop(a.ID) }()
	<-runner.entered
	other := make(chan error, 1)
	go func() {
		if _, e := s.Status(b.ID); e != nil {
			other <- e
			return
		}
		other <- s.Start(b.ID)
	}()
	select {
	case e := <-other:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		close(runner.release)
		t.Fatal("other profile blocked by stop")
	}
	close(runner.release)
	if e := <-stopped; e != nil {
		t.Fatal(e)
	}
}
