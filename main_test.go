package caddy_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestSablierMiddleware_ServeHTTP_NegotiatedErrorResponse(t *testing.T) {
	sablierMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := []string{"application/json;q=0.5", "text/html;q=0.9"}
		if got := r.Header.Values("Accept"); !slices.Equal(got, want) {
			t.Errorf("expected Accept %v, got %v", want, got)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html>Error Page</html>"))
	}))
	defer sablierMockServer.Close()

	sm := &plugin.SablierMiddleware{
		Config: plugin.Config{
			SablierURL:      sablierMockServer.URL,
			Names:           []string{"test-service"},
			SessionDuration: &oneMinute,
			Dynamic:         &plugin.DynamicConfiguration{},
		},
	}
	if err := sm.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Add("Accept", "application/json;q=0.5")
	req.Header.Add("Accept", "text/html;q=0.9")
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		t.Error("non-ready response must not reach the backend")
		return nil
	})

	w := httptest.NewRecorder()
	if err := sm.ServeHTTP(w, req, next); err != nil {
		t.Fatal(err)
	}

	res := w.Result()
	//nolint:errcheck
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html; charset=utf-8, got %q", got)
	}
	if got := w.Body.String(); got != "<html>Error Page</html>" {
		t.Errorf("expected body <html>Error Page</html>, got %q", got)
	}

	t.Run("canceled request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		if err := sm.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx), next); !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}
