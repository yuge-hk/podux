package httphandler

import (
	"encoding/json"
	"errors"
	"github.com/pocketbase/pocketbase/core"
	"io"
	"net/http"
	a "podux/internal/application/tunnel"
	d "podux/internal/domain/tunnel"
	"strconv"
)

type TunnelHandler struct{ service *a.Service }

func NewTunnelHandler(s *a.Service) *TunnelHandler { return &TunnelHandler{s} }
func tunnelError(e *core.RequestEvent, err error) error {
	code, message := 500, "tunnel operation failed"
	switch {
	case errors.Is(err, d.ErrInvalid):
		code, message = 400, d.ErrInvalid.Error()
	case errors.Is(err, d.ErrNotFound):
		code, message = 404, d.ErrNotFound.Error()
	case errors.Is(err, d.ErrConflict):
		code, message = 409, d.ErrConflict.Error()
	case errors.Is(err, d.ErrCredentials):
		code, message = 503, d.ErrCredentials.Error()
	case errors.Is(err, d.ErrUnavailable):
		code, message = 503, d.ErrUnavailable.Error()
	}
	return e.JSON(code, map[string]string{"error": message})
}
func tunnelBody(e *core.RequestEvent, v any) error {
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, 32768)
	dec := json.NewDecoder(e.Request.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil {
		return d.ErrInvalid
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return d.ErrInvalid
	}
	return nil
}
func (h *TunnelHandler) RegisterHandlers(e *core.ServeEvent) {
	const root = "/api/cloudflared"
	e.Router.GET(root+"/capabilities", requireAuth(func(e *core.RequestEvent) error { return e.JSON(200, h.service.Capabilities()) }))
	e.Router.GET(root+"/tunnels", requireAuth(func(e *core.RequestEvent) error {
		rs, err := h.service.List()
		if err != nil {
			return tunnelError(e, err)
		}
		return e.JSON(200, map[string]any{"items": rs})
	}))
	e.Router.POST(root+"/tunnels", requireAuth(func(e *core.RequestEvent) error {
		var w d.Write
		if err := tunnelBody(e, &w); err != nil {
			return tunnelError(e, err)
		}
		r, err := h.service.Create(w)
		if err != nil {
			return tunnelError(e, err)
		}
		return e.JSON(201, r)
	}))
	wrap := func(fn func(*core.RequestEvent, string) error) func(*core.RequestEvent) error {
		return requireAuth(func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			if !validID.MatchString(id) {
				return tunnelError(e, d.ErrInvalid)
			}
			return fn(e, id)
		})
	}
	e.Router.GET(root+"/tunnels/{id}", wrap(func(e *core.RequestEvent, id string) error {
		r, err := h.service.Get(id)
		if err != nil {
			return tunnelError(e, err)
		}
		return e.JSON(200, r)
	}))
	e.Router.PUT(root+"/tunnels/{id}", wrap(func(e *core.RequestEvent, id string) error {
		var w d.Write
		if err := tunnelBody(e, &w); err != nil {
			return tunnelError(e, err)
		}
		r, err := h.service.Update(id, w)
		if err != nil {
			return tunnelError(e, err)
		}
		status, _ := h.service.Status(id)
		return e.JSON(200, map[string]any{"tunnel": r, "restartRequired": status.Revision != 0 && status.Revision != r.Revision && status.Phase != "stopped" && status.Phase != "failed"})
	}))
	e.Router.DELETE(root+"/tunnels/{id}", wrap(func(e *core.RequestEvent, id string) error {
		revision, err := strconv.Atoi(e.Request.URL.Query().Get("revision"))
		if err != nil || revision < 1 {
			return tunnelError(e, d.ErrInvalid)
		}
		if err = h.service.Delete(id, revision); err != nil {
			return tunnelError(e, err)
		}
		return e.NoContent(204)
	}))
	for _, action := range []string{"start", "stop", "restart"} {
		e.Router.POST(root+"/tunnels/{id}/"+action, wrap(func(e *core.RequestEvent, id string) error {
			var err error
			switch action {
			case "start":
				err = h.service.Start(id)
			case "stop":
				err = h.service.Stop(id)
			case "restart":
				err = h.service.Restart(id)
			}
			if err != nil {
				return tunnelError(e, err)
			}
			st, err := h.service.Status(id)
			if err != nil {
				return tunnelError(e, err)
			}
			return e.JSON(202, st)
		}))
	}
	e.Router.GET(root+"/tunnels/{id}/status", wrap(func(e *core.RequestEvent, id string) error {
		st, err := h.service.Status(id)
		if err != nil {
			return tunnelError(e, err)
		}
		return e.JSON(200, st)
	}))
	e.Router.GET(root+"/tunnels/{id}/logs", wrap(func(e *core.RequestEvent, id string) error {
		var cursor uint64
		if v := e.Request.URL.Query().Get("after"); v != "" {
			var err error
			cursor, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return tunnelError(e, d.ErrInvalid)
			}
		}
		ls, err := h.service.Logs(id, cursor)
		if err != nil {
			return tunnelError(e, err)
		}
		return e.JSON(200, map[string]any{"items": ls, "epoch": h.service.LogEpoch()})
	}))
}
