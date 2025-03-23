package caddy

import (
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
	"net/http"
)

func init() {
	caddy.RegisterModule(Sablier{})
}

// Sablier is a middleware that contacts a Sablier instance to check if a session is ready.
type Sablier struct {
	Config  Config
	client  *http.Client
	request *http.Request
	logger  *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Sablier) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.sablier",
		New: func() caddy.Module { return new(Sablier) },
	}
}
