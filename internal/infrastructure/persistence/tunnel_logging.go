package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	d "podux/internal/domain/tunnel"
)

// ProtectTunnelQueryLogs must be installed after bootstrap, before serving
// requests. PocketBase dev mode logs interpolated SQL, including encrypted
// credentials. Suppress only this collection; leave all other query logs intact.
func ProtectTunnelQueryLogs(app core.App) {
	seen := map[*dbx.DB]bool{}
	for _, builder := range []dbx.Builder{app.ConcurrentDB(), app.NonconcurrentDB()} {
		db, ok := builder.(*dbx.DB)
		if !ok || seen[db] {
			continue
		}
		seen[db] = true
		protectTunnelQueryLogger(db)
	}
}

func protectTunnelQueryLogger(db *dbx.DB) {
	private := func(statement string) bool {
		return strings.Contains(strings.ToLower(statement), d.Collection)
	}
	if previous := db.QueryLogFunc; previous != nil {
		db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, statement string, rows *sql.Rows, err error) {
			if !private(statement) {
				previous(ctx, elapsed, statement, rows, err)
			}
		}
	}
	if previous := db.ExecLogFunc; previous != nil {
		db.ExecLogFunc = func(ctx context.Context, elapsed time.Duration, statement string, result sql.Result, err error) {
			if !private(statement) {
				previous(ctx, elapsed, statement, result, err)
			}
		}
	}
	if previous := db.LogFunc; previous != nil {
		db.LogFunc = func(format string, args ...any) {
			if !private(fmt.Sprintf(format, args...)) {
				previous(format, args...)
			}
		}
	}
	if previous := db.PerfFunc; previous != nil {
		db.PerfFunc = func(elapsed int64, statement string, execute bool) {
			if !private(statement) {
				previous(elapsed, statement, execute)
			}
		}
	}
}
