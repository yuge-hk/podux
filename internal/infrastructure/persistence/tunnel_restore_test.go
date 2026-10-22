package persistence

import (
	"github.com/pocketbase/pocketbase/core"
	"os"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/credentials"
	"testing"
)

func TestColdDatabaseAndExternalKeyRestore(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "old", "data")
	restored := filepath.Join(root, "restored", "data")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: original})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	if e := app.RunAllMigrations(); e != nil {
		app.ResetBootstrapState()
		t.Fatal(e)
	}
	vault, e := credentials.Open(original, "", false)
	if e != nil {
		t.Fatal(e)
	}
	id := "aaaaaaaaaaaaaaa"
	cipher, e := vault.Encrypt(id, "synthetic-restore-token")
	if e != nil {
		t.Fatal(e)
	}
	repo := NewTunnelRepository(app)
	record, e := repo.Create(d.Record{ID: id, Config: d.Config{Name: "restore-profile", Protocol: "auto", EdgeIPVersion: "auto", GracePeriodSeconds: 1}, Ciphertext: cipher})
	if e != nil {
		t.Fatal(e)
	}
	if e = app.ResetBootstrapState(); e != nil {
		t.Fatal(e)
	}
	// Cold-copy the complete disposable data directory; the key is deliberately
	// not included. No production database or online SQLite file is copied.
	e = filepath.WalkDir(original, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(original, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(restored, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0600)
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = credentials.Open(restored, "", true); e != d.ErrCredentials {
		t.Fatal("database-only backup must not invent a key", e)
	}
	key, e := os.ReadFile(original + ".cloudflared-secrets/master.key")
	if e != nil {
		t.Fatal(e)
	}
	keyDir := restored + ".cloudflared-secrets"
	if e = credentials.PrivateDir(keyDir); e != nil {
		t.Fatal(e)
	}
	if e = credentials.PrivateFile(filepath.Join(keyDir, "master.key"), key); e != nil {
		t.Fatal(e)
	}
	clear(key)
	recovered, e := credentials.Open(restored, "", true)
	if e != nil {
		t.Fatal(e)
	}
	other := core.NewBaseApp(core.BaseAppConfig{DataDir: restored})
	if e = other.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer other.ResetBootstrapState()
	if e = other.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	got, e := NewTunnelRepository(other).Get(id)
	if e != nil || got.Revision != record.Revision || got.Name != record.Name || got.Ciphertext != cipher {
		t.Fatal("restored profile changed", e)
	}
	plaintext, e := recovered.Decrypt(id, got.Ciphertext)
	if e != nil || plaintext != "synthetic-restore-token" {
		t.Fatal("restored key/data association failed")
	}
}
