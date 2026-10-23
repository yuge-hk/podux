package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	d "podux/internal/domain/tunnel"
	_ "podux/migrations"
)

func TestTunnelWritesDoNotReachSQLLoggers(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer app.ResetBootstrapState()
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	var logs []string
	for _, builder := range []dbx.Builder{app.ConcurrentDB(), app.NonconcurrentDB()} {
		db := builder.(*dbx.DB)
		db.QueryLogFunc = func(_ context.Context, _ time.Duration, statement string, _ *sql.Rows, _ error) {
			logs = append(logs, statement)
		}
		db.ExecLogFunc = func(_ context.Context, _ time.Duration, statement string, _ sql.Result, _ error) {
			logs = append(logs, statement)
		}
		db.LogFunc = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }
		db.PerfFunc = func(_ int64, statement string, _ bool) { logs = append(logs, statement) }
	}
	ProtectTunnelQueryLogs(app)
	repo := NewTunnelRepository(app)
	const secret = "encrypted-test-value-not-for-logs"
	r, err := repo.Create(d.Record{ID: "aaaaaaaaaaaaaaa", Config: d.Config{Name: "test"}, Ciphertext: secret})
	if err != nil {
		t.Fatal(err)
	}
	r.Name = "updated"
	if _, err = repo.Update(r, r.Revision); err != nil {
		t.Fatal(err)
	}
	for _, message := range logs {
		if strings.Contains(message, secret) || strings.Contains(strings.ToLower(message), "into `"+d.Collection) || strings.Contains(strings.ToLower(message), "update `"+d.Collection) {
			t.Fatal("credential query reached a logger")
		}
	}
	// The filter changes observability only for the new collection.
	logs = nil
	var one int
	if err = app.DB().NewQuery("SELECT 1").Row(&one); err != nil || one != 1 {
		t.Fatal("unrelated query changed", err)
	}
	if len(logs) == 0 {
		t.Fatal("unrelated SQL logging was disabled")
	}
	if err = app.RunInTransaction(func(tx core.App) error {
		rec, err := tx.FindRecordById(d.Collection, r.ID)
		if err != nil {
			return err
		}
		rec.Set("ciphertext", secret+"-transaction")
		return tx.Save(rec)
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(logs, "\n"), secret) {
		t.Fatal("transaction credential query reached a logger")
	}
}
