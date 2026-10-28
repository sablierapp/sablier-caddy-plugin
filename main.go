package caddy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

	sablierURL := m.Config.SablierURL
	if !strings.Contains(sablierURL, "://") {
		// Accept host:port, as the proxy-wasm plugin does: the client needs a
		// scheme, and Sablier is usually reached over plain http.
		sablierURL = "http://" + sablierURL
	}

	sablier, err := client.New(sablierURL)
	if err != nil {
		return err
	}
	m.sablier = sablier

	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (sm SablierMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	repl, _ := req.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	resp, err := sm.Config.start(context.TODO(), sm.sablier, repl)
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
	_, err := rw.Write(resp.Body)
	return err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SablierMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*SablierMiddleware)(nil)
)
