package caddy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	plugin "github.com/sablierapp/sablier-caddy-plugin"
)

// TestSablierRequest pins the request the middleware sends to Sablier for
// each configuration: endpoint path and exact query parameters.
func TestSablierRequest(t *testing.T) {
	fiveMinutes := 5 * oneMinute
	tests := []struct {
		name      string
		config    plugin.Config
		host      string
		wantPath  string
		wantQuery url.Values
	}{
		{
			name:      "dynamic with names",
			config:    plugin.Config{Names: []string{"nginx", "apache"}, Dynamic: &plugin.DynamicConfiguration{}},
			wantPath:  "/api/strategies/dynamic",
			wantQuery: url.Values{"names": {"nginx", "apache"}},
		},
		{
			name: "dynamic with every option",
			config: plugin.Config{
				Group:           "default",
				SessionDuration: &oneMinute,
				Dynamic: &plugin.DynamicConfiguration{
					DisplayName:      "Hello World!",
					ShowDetails:      &tru,
					Theme:            "hacker-terminal",
					RefreshFrequency: &oneMinute,
				},
			},
			wantPath: "/api/strategies/dynamic",
			wantQuery: url.Values{
				"group":             {"default"},
				"session_duration":  {"1m0s"},
				"display_name":      {"Hello World!"},
				"show_details":      {"true"},
				"theme":             {"hacker-terminal"},
				"refresh_frequency": {"1m0s"},
			},
		},
		{
			name:      "dynamic with show_details off",
			config:    plugin.Config{Group: "default", Dynamic: &plugin.DynamicConfiguration{ShowDetails: &fals}},
			wantPath:  "/api/strategies/dynamic",
			wantQuery: url.Values{"group": {"default"}, "show_details": {"false"}},
		},
		{
			name:      "blocking with every option",
			config:    plugin.Config{Names: []string{"nginx"}, SessionDuration: &oneMinute, Blocking: &plugin.BlockingConfiguration{Timeout: &fiveMinutes}},
			wantPath:  "/api/strategies/blocking",
			wantQuery: url.Values{"names": {"nginx"}, "session_duration": {"1m0s"}, "timeout": {"5m0s"}},
		},
		{
			name:      "blocking without options",
			config:    plugin.Config{Group: "default", Blocking: &plugin.BlockingConfiguration{}},
			wantPath:  "/api/strategies/blocking",
			wantQuery: url.Values{"group": {"default"}},
		},
		{
			name:      "placeholders are expanded per request, empty names are dropped",
			config:    plugin.Config{Names: []string{"{http.request.host.labels.3}", "{http.request.header.X-Missing}"}, Dynamic: &plugin.DynamicConfiguration{}},
			host:      "myapp.sub.example.com",
			wantPath:  "/api/strategies/dynamic",
			wantQuery: url.Values{"names": {"myapp"}},
		},
		{
			name:      "group placeholder is expanded per request",
			config:    plugin.Config{Group: "{http.request.host.labels.3}", Blocking: &plugin.BlockingConfiguration{}},
			host:      "myapp.sub.example.com",
			wantPath:  "/api/strategies/blocking",
			wantQuery: url.Values{"group": {"myapp"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *http.Request
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r
				w.Header().Set("X-Sablier-Session-Status", "ready")
			}))
			defer srv.Close()

			sm := &plugin.SablierMiddleware{Config: tt.config}
			sm.Config.SablierURL = srv.URL
			if err := sm.Provision(caddy.Context{}); err != nil {
				t.Fatal(err)
			}

			host := tt.host
			if host == "" {
				host = "example.com"
			}
			req := httptest.NewRequest(http.MethodGet, "http://"+host+"/", nil)
			repl := caddy.NewReplacer()
			req = req.WithContext(context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl))
			caddyhttp.PrepareRequest(req, repl, nil, nil)

			next := caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error { return nil })
			if err := sm.ServeHTTP(httptest.NewRecorder(), req, next); err != nil {
				t.Fatal(err)
			}

			if got == nil {
				t.Fatal("no request reached Sablier")
			}
			if got.URL.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", got.URL.Path, tt.wantPath)
			}
			if got.URL.Query().Encode() != tt.wantQuery.Encode() {
				t.Errorf("query = %s, want %s", got.URL.Query().Encode(), tt.wantQuery.Encode())
			}
		})
	}
}

// TestSablierResponseIsForwarded pins what the client receives while the
// session is not ready: Sablier's status code, body and Content-Type, so an
// error such as an unknown group or a blocking timeout is not served as 200.
func TestSablierResponseIsForwarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Group not found"}`))
	}))
	defer srv.Close()

	sm := &plugin.SablierMiddleware{Config: plugin.Config{SablierURL: srv.URL, Group: "nope", Dynamic: &plugin.DynamicConfiguration{}}}
	if err := sm.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	next := caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		t.Error("the request reached the backend while the session is not ready")
		return nil
	})
	if err := sm.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil), next); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want Sablier's %d", rec.Code, http.StatusNotFound)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	if got := rec.Body.String(); got != `{"title":"Group not found"}` {
		t.Errorf("body = %q, want Sablier's body", got)
	}
}

// TestSablierURLWithoutScheme checks that a Sablier URL given as host:port,
// without http://, is reached over http.
func TestSablierURLWithoutScheme(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("X-Sablier-Session-Status", "ready")
	}))
	defer srv.Close()
	hostPort := strings.TrimPrefix(srv.URL, "http://")
	_, port, _ := strings.Cut(hostPort, ":")

	for _, sablierURL := range []string{hostPort, "localhost:" + port} {
		sm := &plugin.SablierMiddleware{Config: plugin.Config{SablierURL: sablierURL, Group: "demo", Dynamic: &plugin.DynamicConfiguration{}}}
		if err := sm.Provision(caddy.Context{}); err != nil {
			t.Errorf("SablierURL %q: %v", sablierURL, err)
			continue
		}
		path = ""
		next := caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error { return nil })
		if err := sm.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), next); err != nil {
			t.Errorf("SablierURL %q: %v", sablierURL, err)
			continue
		}
		if path != "/api/strategies/dynamic" {
			t.Errorf("SablierURL %q: Sablier received path %q, want /api/strategies/dynamic", sablierURL, path)
		}
	}
}
