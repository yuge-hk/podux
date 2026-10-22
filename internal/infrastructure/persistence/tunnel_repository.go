package persistence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	d "podux/internal/domain/tunnel"
)

type TunnelRepository struct{ app core.App }

func NewTunnelRepository(app core.App) *TunnelRepository { return &TunnelRepository{app} }
func tunnelRecord(r *core.Record) (d.Record, error) {
	var c d.Config
	if e := json.Unmarshal([]byte(r.GetString("config")), &c); e != nil {
		return d.Record{}, d.ErrInvalid
	}
	cipher := r.GetString("ciphertext")
	return d.Record{ID: r.Id, Config: c, Revision: r.GetInt("revision"), HasToken: cipher != "", Ciphertext: cipher}, nil
}
func (r *TunnelRepository) List() ([]d.Record, error) {
	rs, e := r.app.FindAllRecords(d.Collection)
	if e != nil {
		return nil, e
	}
	out := make([]d.Record, 0, len(rs))
	for _, v := range rs {
		x, e := tunnelRecord(v)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, nil
}
func (r *TunnelRepository) Get(id string) (d.Record, error) {
	v, e := r.app.FindRecordById(d.Collection, id)
	if errors.Is(e, sql.ErrNoRows) {
		return d.Record{}, d.ErrNotFound
	}
	if e != nil {
		return d.Record{}, e
	}
	return tunnelRecord(v)
}
func (r *TunnelRepository) Create(v d.Record) (d.Record, error) {
	c, e := r.app.FindCollectionByNameOrId(d.Collection)
	if e != nil {
		return d.Record{}, e
	}
	rec := core.NewRecord(c)
	rec.Id = v.ID
	rec.Set("name", v.Name)
	rec.Set("config", v.Config)
	rec.Set("ciphertext", v.Ciphertext)
	rec.Set("revision", 1)
	if e = r.app.Save(rec); e != nil {
		return d.Record{}, e
	}
	return tunnelRecord(rec)
}
func (r *TunnelRepository) Update(v d.Record, revision int) (d.Record, error) {
	cfg, e := json.Marshal(v.Config)
	if e != nil {
		return d.Record{}, e
	}
	result, e := r.app.DB().Update(d.Collection, dbx.Params{"name": v.Name, "config": string(cfg), "ciphertext": v.Ciphertext, "revision": revision + 1}, dbx.HashExp{"id": v.ID, "revision": revision}).Execute()
	if e != nil {
		return d.Record{}, e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return d.Record{}, e
	}
	if n != 1 {
		return d.Record{}, d.ErrConflict
	}
	return r.Get(v.ID)
}
func (r *TunnelRepository) Delete(id string, revision int) error {
	result, e := r.app.DB().Delete(d.Collection, dbx.HashExp{"id": id, "revision": revision}).Execute()
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return d.ErrConflict
	}
	return nil
}
