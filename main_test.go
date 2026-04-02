package caddy_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	plugin "github.com/sablierapp/sablier-caddy-plugin"
)

func TestSablierMiddleware_ServeHTTP(t *testing.T) {
	type fields struct {
		Next              caddyhttp.Handler
		SablierMiddleware *plugin.SablierMiddleware
	}
	type sablier struct {
		headers map[string]string
		body    string
	}
	tests := []struct {
		name     string
		fields   fields
		sablier  sablier
		expected string
	}{
		{
			name: "sablier service is ready",
			sablier: sablier{
				headers: map[string]string{
					"X-Sablier-Session-Status": "ready",
				},
				body: "response from sablier",
			},
			fields: fields{
				Next: caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					_, err := fmt.Fprint(w, "response from service")
					return err
				}),
				SablierMiddleware: &plugin.SablierMiddleware{
					Config: plugin.Config{
						SessionDuration: &oneMinute,
						Dynamic:         &plugin.DynamicConfiguration{},
					},
				},
			},
			expected: "response from service",
		},
		{
			name: "sablier service is not ready",
			sablier: sablier{
				headers: map[string]string{
					"X-Sablier-Session-Status": "not-ready",
				},
				body: "response from sablier",
			},
			fields: fields{
				Next: caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					_, err := fmt.Fprint(w, "response from service")
					return err
				}),
				SablierMiddleware: &plugin.SablierMiddleware{
					Config: plugin.Config{
						SessionDuration: &oneMinute,
						Dynamic:         &plugin.DynamicConfiguration{},
					},
				},
			},
			expected: "response from sablier",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sablierMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tt.sablier.headers {
					w.Header().Add(key, value)
				}
				_, _ = w.Write([]byte(tt.sablier.body))
			}))
			//nolint:errcheck
			defer sablierMockServer.Close()

			tt.fields.SablierMiddleware.Config.SablierURL = sablierMockServer.URL

			err := tt.fields.SablierMiddleware.Provision(caddy.Context{})
			if err != nil {
				panic(err)
			}

			req := httptest.NewRequest(http.MethodGet, "/my-nginx", nil)
			w := httptest.NewRecorder()

			err = tt.fields.SablierMiddleware.ServeHTTP(w, req, tt.fields.Next)
			if err != nil {
				panic(err)
			}

			res := w.Result()
			//nolint:errcheck
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("expected error to be nil got %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s got %v", tt.expected, string(data))
			}
		})
	}
}

func TestSablierMiddleware_ServeHTTP_PlaceholderExpansion(t *testing.T) {
	var receivedNames []string
	sablierMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedNames = r.URL.Query()["names"]
		_, _ = w.Write([]byte("ok"))
	}))
	defer sablierMockServer.Close()

	sm := &plugin.SablierMiddleware{
		Config: plugin.Config{
			SablierURL:      sablierMockServer.URL,
			Names:           []string{"nginx", "{http.request.host.labels.3}"},
			SessionDuration: &oneMinute,
			Dynamic:         &plugin.DynamicConfiguration{},
		},
	}

	err := sm.Provision(caddy.Context{})
	if err != nil {
		t.Fatal(err)
	}

	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Request to "myapp.sub.example.com" => labels.3 = "myapp"
	req := httptest.NewRequest(http.MethodGet, "http://myapp.sub.example.com/", nil)
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)
	caddyhttp.PrepareRequest(req, repl, nil, nil)

	w := httptest.NewRecorder()
	err = sm.ServeHTTP(w, req, next)
	if err != nil {
		t.Fatal(err)
	}

	if len(receivedNames) != 2 || receivedNames[0] != "nginx" || receivedNames[1] != "myapp" {
		t.Errorf("expected names=[nginx myapp], got names=%v", receivedNames)
	}
}
