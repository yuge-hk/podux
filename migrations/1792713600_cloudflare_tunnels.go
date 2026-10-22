package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("fh_cloudflare_tunnels")
		minimumRevision := 1.0
		// All collection API rules remain nil (superuser only). Use authenticated custom APIs.
		c.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120},
			&core.JSONField{Name: "config", Required: true, MaxSize: 16384},
			&core.TextField{Name: "ciphertext", Hidden: true, Required: true, Max: 16384},
			&core.NumberField{Name: "revision", Required: true, OnlyInt: true, Min: &minimumRevision})
		return app.Save(c)
	}, func(app core.App) error {
		c, e := app.FindCollectionByNameOrId("fh_cloudflare_tunnels")
		if e != nil {
			return e
		}
		return app.Delete(c)
	})
}
