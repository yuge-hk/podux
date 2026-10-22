package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

func TestCloudflareMigrationRoundTrip(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	if e := app.RunAllMigrations(); e != nil {
		t.Fatal(e)
	}
	if _, e := app.FindCollectionByNameOrId("fh_servers"); e != nil {
		t.Fatal(e)
	}
	runner := core.NewMigrationsRunner(app, core.AppMigrations)
	files, e := runner.Down(1)
	if e != nil || len(files) != 1 {
		t.Fatal(files, e)
	}
	if _, e = app.FindCollectionByNameOrId("fh_cloudflare_tunnels"); e == nil {
		t.Fatal("collection remains")
	}
	if _, e = app.FindCollectionByNameOrId("fh_servers"); e != nil {
		t.Fatal("FRPC collection changed", e)
	}
	if _, e = runner.Up(); e != nil {
		t.Fatal(e)
	}
	if _, e = app.FindCollectionByNameOrId("fh_cloudflare_tunnels"); e != nil {
		t.Fatal(e)
	}
}

func TestUpgradePreservesExistingFRPCRecords(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if e := app.Bootstrap(); e != nil {
		t.Fatal(e)
	}
	defer app.ResetBootstrapState()
	// Apply only the pre-feature snapshot, then populate it before the additive migration.
	var old core.MigrationsList
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1774402980_collections_snapshot.go" {
			old.Add(migration)
		}
	}
	if _, e := core.NewMigrationsRunner(app, old).Up(); e != nil {
		t.Fatal(e)
	}
	c, e := app.FindCollectionByNameOrId("fh_servers")
	if e != nil {
		t.Fatal(e)
	}
	server := core.NewRecord(c)
	server.Id = "aaaaaaaaaaaaaaa"
	server.Set("serverName", "preserved-existing-server")
	server.Set("serverAddr", "127.0.0.1")
	server.Set("serverPort", "7000")
	if e = app.Save(server); e != nil {
		t.Fatal(e)
	}
	if e = app.RunAppMigrations(); e != nil {
		t.Fatal(e)
	}
	if _, e = app.FindCollectionByNameOrId("fh_cloudflare_tunnels"); e != nil {
		t.Fatal(e)
	}
	got, e := app.FindRecordById("fh_servers", server.Id)
	if e != nil || got.GetString("serverName") != "preserved-existing-server" {
		t.Fatal("upgrade changed FRPC data", e)
	}
	if _, e = core.NewMigrationsRunner(app, core.AppMigrations).Down(1); e != nil {
		t.Fatal(e)
	}
	if _, e = app.FindRecordById("fh_servers", server.Id); e != nil {
		t.Fatal("rollback removed existing FRPC record", e)
	}
}
