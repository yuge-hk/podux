package httphandler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http/httptest"
	"path/filepath"
	a "podux/internal/application/tunnel"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/cloudflared"
	"podux/internal/infrastructure/credentials"
	"podux/internal/infrastructure/persistence"
	_ "podux/migrations"
	"strings"
	"testing"
)

func TestTunnelAPI(t *testing.T) {
	root := t.TempDir()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: filepath.Join(root, "data")})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e := app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	vault, e := credentials.Open(app.DataDir(), filepath.Join(root, "key", "master.key"), false)
	if e != nil {
		t.Fatal(e)
	}
	repo := persistence.NewTunnelRepository(app)
	runner := cloudflared.NewManager(cloudflared.Binary{}, app.DataDir())
	defer runner.Shutdown(context.Background())
	svc := a.NewService(repo, vault, runner)
	router, e := apis.NewRouter(app)
	if e != nil {
		t.Fatal(e)
	}
	NewTunnelHandler(svc).RegisterHandlers(&core.ServeEvent{App: app, Router: router})
	mux, e := router.BuildMux()
	if e != nil {
		t.Fatal(e)
	}
	c, e := app.FindCollectionByNameOrId("fh_users")
	if e != nil {
		t.Fatal(e)
	}
	user := core.NewRecord(c)
	user.SetEmail("backend-test@example.invalid")
	user.SetPassword("test-only-password")
	if e = app.Save(user); e != nil {
		t.Fatal(e)
	}
	auth, e := user.NewAuthToken()
	if e != nil {
		t.Fatal(e)
	}
	token := base64.StdEncoding.EncodeToString([]byte(`{"a":"test-account","t":"00000000-0000-0000-0000-000000000000","s":"dGVzdC1vbmx5"}`))
	request := func(method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		var b []byte
		if body != nil {
			b, _ = json.Marshal(body)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if authenticated {
			r.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), token) || strings.Contains(w.Body.String(), "v1:") || strings.Contains(w.Body.String(), root) {
			t.Fatal("sensitive response", w.Body.String())
		}
		return w
	}
	config := d.Config{Name: "first", Protocol: "auto", EdgeIPVersion: "auto", ConnectionRetries: 5, GracePeriodSeconds: 1, MaxRestartAttempts: 2}
	for _, path := range []string{"capabilities", "tunnels", "tunnels/aaaaaaaaaaaaaaa/status", "tunnels/aaaaaaaaaaaaaaa/logs"} {
		if w := request("GET", "/api/cloudflared/"+path, nil, false); w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	for _, action := range []string{"start", "stop", "restart"} {
		if w := request("POST", "/api/cloudflared/tunnels/aaaaaaaaaaaaaaa/"+action, nil, false); w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	w := request("POST", "/api/cloudflared/tunnels", d.Write{Config: config, Token: &token}, true)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var created d.Record
	if e = json.Unmarshal(w.Body.Bytes(), &created); e != nil {
		t.Fatal(e)
	}
	if !created.HasToken || created.Revision != 1 {
		t.Fatal(created)
	}
	id := created.ID
	stored, e := repo.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	oldCipher := stored.Ciphertext
	w = request("PUT", "/api/cloudflared/tunnels/"+id, d.Write{Config: config, Revision: 1}, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	stored, _ = repo.Get(id)
	if stored.Ciphertext != oldCipher || stored.Revision != 2 {
		t.Fatal("token changed on omission")
	}
	if w = request("PUT", "/api/cloudflared/tunnels/"+id, d.Write{Config: config, Revision: 1}, true); w.Code != 409 {
		t.Fatal("stale revision", w.Code)
	}
	empty := ""
	if w = request("PUT", "/api/cloudflared/tunnels/"+id, d.Write{Config: config, Revision: 2, Token: &empty}, true); w.Code != 400 {
		t.Fatal("empty token accepted", w.Code)
	}
	if w = request("GET", "/api/cloudflared/tunnels/"+id+"/status", nil, true); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = request("POST", "/api/cloudflared/tunnels/"+id+"/start", nil, true); w.Code != 503 {
		t.Fatal("binary unavailable", w.Code)
	}
	if w = request("POST", "/api/cloudflared/tunnels/"+id+"/stop", nil, true); w.Code != 202 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/api/collections/fh_cloudflare_tunnels/records", "/api/collections/fh_cloudflare_tunnels/records/" + id} {
		w = request("GET", path, nil, true)
		if w.Code == 200 {
			t.Fatal("raw collection accessible")
		}
	}
	if w = request("GET", "/api/cloudflared/tunnels/bad/status", nil, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = request("GET", "/api/cloudflared/tunnels/aaaaaaaaaaaaaaa", nil, true); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w = request("GET", "/api/cloudflared/tunnels/"+id+"/logs?after=-1", nil, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = request("GET", "/api/cloudflared/tunnels", nil, true); w.Code != 200 || !strings.Contains(w.Body.String(), id) {
		t.Fatal(w.Code)
	}
	w = request("GET", "/api/cloudflared/tunnels/"+id+"/logs", nil, true)
	var logResponse struct {
		Epoch string `json:"epoch"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &logResponse); e != nil || logResponse.Epoch == "" {
		t.Fatal("missing log epoch", e)
	}
	// Invalid/oversized/injected input must never reach the runner or leak errors.
	for _, body := range []any{map[string]any{"name": strings.Repeat("x", 40000)}, map[string]any{"noTlsVerify": true}, d.Write{Config: d.Config{Name: "bad", Protocol: "auto", EdgeIPVersion: "auto", GracePeriodSeconds: 1, ExtraArgs: []d.ExtraArg{{Name: "token-file", Value: "/outside"}}}, Revision: 2}} {
		if bad := request("PUT", "/api/cloudflared/tunnels/"+id, body, true); bad.Code != 400 {
			t.Fatal("unsafe input accepted", bad.Code)
		}
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		path := "/api/cloudflared/tunnels"
		if method != "POST" {
			path += "/" + id + "?revision=2"
		}
		if w = request(method, path, map[string]any{}, false); w.Code != 401 {
			t.Fatal("unauthenticated mutation", method, w.Code)
		}
	}
	// Explicit rotation changes ciphertext and remains write-only.
	if w = request("PUT", "/api/cloudflared/tunnels/"+id, d.Write{Config: config, Revision: 2, Token: &token}, true); w.Code != 200 {
		t.Fatal(w.Code)
	}
	stored, _ = repo.Get(id)
	if stored.Ciphertext == oldCipher {
		t.Fatal("nonce reused")
	}
	if w = request("DELETE", "/api/cloudflared/tunnels/"+id+"?revision=2", nil, true); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w = request("DELETE", "/api/cloudflared/tunnels/"+id+"?revision=3", nil, true); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	// Authentication cannot be moved to the query string.
	if w = request("GET", "/api/cloudflared/capabilities?token="+auth, nil, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
}
