package caddy

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
	"net/http"
)

var _ caddy.Provisioner = (*Sablier)(nil)

// Provision implements [caddy.Provisioner].
func (sm *Sablier) Provision(ctx caddy.Context) error {
	sm.logger = ctx.Logger(sm)

	sm.logger.Info("sablier middleware configuration", zap.Any("Config", sm.Config))
	req, err := sm.Config.BuildRequest()

	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	sm.request = req
	sm.client = &http.Client{}

	sm.logger.Info("sablier middleware provisioned",
		zap.String("url", sm.Config.SablierURL),
		zap.Strings("names", sm.Config.Names),
		zap.String("group", sm.Config.Group))
	return nil
}
