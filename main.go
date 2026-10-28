package caddy

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/sablierapp/sablier/client"
)

func init() {
	caddy.RegisterModule(SablierMiddleware{})
}

type SablierMiddleware struct {
	Config  Config
	sablier *client.Client
}

// CaddyModule returns the Caddy module information.
func (SablierMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.sablier",
		New: func() caddy.Module { return new(SablierMiddleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *SablierMiddleware) Provision(ctx caddy.Context) error {
	if m.Config.Dynamic == nil && m.Config.Blocking == nil {
		return fmt.Errorf("no strategy configured")
	}

	sablier, err := client.New(m.Config.SablierURL)
	if err != nil {
		return err
	}
	m.sablier = sablier

	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (sm SablierMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	repl, _ := req.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	resp, err := sm.Config.start(req.Context(), sm.sablier, repl)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return err
	}

	if resp.Ready() {
		return next.ServeHTTP(rw, req)
	}

	return forward(resp, rw)
}

func forward(resp *client.Response, rw http.ResponseWriter) error {
	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	// The waiting page is served at the user's URL: a stored copy would keep
	// showing once the instances are ready.
	rw.Header().Set("Cache-Control", "no-store")
	rw.WriteHeader(resp.StatusCode)
	_, err := rw.Write(resp.Body)
	return err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SablierMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*SablierMiddleware)(nil)
)
