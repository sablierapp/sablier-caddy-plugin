package caddy

import (
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

var _ caddyhttp.MiddlewareHandler = (*Sablier)(nil)

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (sm Sablier) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	sablierRequest := sm.request.Clone(req.Context())

	resp, err := sm.client.Do(sablierRequest)
	if err != nil {
		sm.logger.Error("failed to contact sablier service",
			zap.Error(err))
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Sablier-Session-Status") == "ready" {
		sm.logger.Info("session is ready, continuing to next handler")
		go sm.keepAlive(req)
		return next.ServeHTTP(rw, req)
	}
	sm.logger.Info("forwarding sablier response to client")
	return forward(resp, rw)
}

func (sm Sablier) keepAlive(req *http.Request) {
	sablierRequest := sm.request.Clone(req.Context())

	timer := time.NewTicker(*sm.Config.SessionDuration / 2)
	for {
		select {
		case <-req.Context().Done():
			sm.logger.Debug("keepalive routine stopped, request context done")
			return
		case <-timer.C:
			sm.logger.Debug("sending keepalive request")
			resp, err := sm.client.Do(sablierRequest)
			if err != nil {
				sm.logger.Error("keepalive request failed",
					zap.Error(err))
				continue
			}
			err = resp.Body.Close()
			if err != nil {
				sm.logger.Error("failed to close keepalive response body", zap.Error(err))
				continue
			}
		}
	}
}

func forward(resp *http.Response, rw http.ResponseWriter) error {
	rw.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	rw.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, err := io.Copy(rw, resp.Body)
	return err
}
