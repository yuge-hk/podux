package persistence

import (
	"github.com/pocketbase/pocketbase/core"
	d "podux/internal/domain/tunnel"
	_ "podux/migrations"
	"testing"
)

func TestTunnelRepository(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e := app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	repo := NewTunnelRepository(app)
	r := d.Record{ID: "aaaaaaaaaaaaaaa", Config: d.Config{Name: "test"}, Ciphertext: "encrypted-value"}
	a, e := repo.Create(r)
	if e != nil {
		t.Fatal(e)
	}
	if a.Revision != 1 {
		t.Fatal(a)
	}
	a.Name = "changed"
	b, e := repo.Update(a, 1)
	if e != nil || b.Revision != 2 {
		t.Fatal(b, e)
	}
	if _, e = repo.Update(a, 1); e != d.ErrConflict {
		t.Fatal("stale update", e)
	}
	rs, e := repo.List()
	if e != nil || len(rs) != 1 || rs[0].Name != "changed" {
		t.Fatal(rs, e)
	}
	if e = repo.Delete(a.ID, 1); e != d.ErrConflict {
		t.Fatal("stale delete", e)
	}
	if e = repo.Delete(a.ID, 2); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.Get(a.ID); e != d.ErrNotFound {
		t.Fatal(e)
	}
	c, e := app.FindCollectionByNameOrId(d.Collection)
	if e != nil {
		t.Fatal(e)
	}
	if c.ListRule != nil || c.ViewRule != nil || c.CreateRule != nil || c.UpdateRule != nil || c.DeleteRule != nil {
		t.Fatal("collection unlocked")
	}
	if !c.Fields.GetByName("ciphertext").GetHidden() {
		t.Fatal("ciphertext visible")
	}
}
