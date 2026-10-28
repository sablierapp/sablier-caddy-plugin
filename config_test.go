package caddy_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/sablierapp/sablier-caddy-plugin"
)

var fals bool = false
var tru bool = true
var oneMinute = 1 * time.Minute

func TestConfig_UnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         caddy.Config
		wantErr      bool
		wantErrValue string
	}{
		{
			name: "default sablier URL",
			input: `sablier {
				group mygroup
				dynamic
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
				Group:      "mygroup",
				Dynamic:    &caddy.DynamicConfiguration{},
			},
			wantErr: false,
		},
		{
			name: "specific sablier URL",
			input: `sablier http://mysablier:3000 {
				names container1 container2 container3
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://mysablier:3000",
				Names:      []string{"container1", "container2", "container3"},
				Blocking:   &caddy.BlockingConfiguration{},
			},
			wantErr: false,
		},
		{
			name: "parse valid names dynamic",
			input: `sablier {
				names container1 container2 container3
				session_duration 1m
				dynamic {
					display_name This is a display name!
					show_details on
					theme hacker-terminal
					refresh_frequency 1m
				}
			}`,
			want: caddy.Config{
				SablierURL:      "http://sablier:10000",
				Names:           []string{"container1", "container2", "container3"},
				SessionDuration: &oneMinute,
				Dynamic: &caddy.DynamicConfiguration{
					DisplayName:      "This is a display name!",
					ShowDetails:      &tru,
					Theme:            "hacker-terminal",
					RefreshFrequency: &oneMinute,
				},
			},
			wantErr: false,
		},
		{
			name: "parse valid group dynamic",
			input: `sablier {
				group mygroup
				session_duration 1m
				dynamic {
					display_name This is a display name!
					show_details on
					theme hacker-terminal
					refresh_frequency 1m
				}
			}`,
			want: caddy.Config{
				SablierURL:      "http://sablier:10000",
				Group:           "mygroup",
				SessionDuration: &oneMinute,
				Dynamic: &caddy.DynamicConfiguration{
					DisplayName:      "This is a display name!",
					ShowDetails:      &tru,
					Theme:            "hacker-terminal",
					RefreshFrequency: &oneMinute,
				},
			},
			wantErr: false,
		},
		{
			name: "parse valid names blocking",
			input: `sablier {
				group mygroup
				session_duration 1m
				blocking {
					timeout 1m
				}
			}`,
			want: caddy.Config{
				SablierURL:      "http://sablier:10000",
				Group:           "mygroup",
				SessionDuration: &oneMinute,
				Blocking: &caddy.BlockingConfiguration{
					Timeout: &oneMinute,
				},
			},
			wantErr: false,
		},
		{
			name:  "parse invalid no strategies",
			input: `sablier`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify one strategy (dynamic or blocking)",
		},
		{
			name: "parse invalid two strategies",
			input: `sablier {
				blocking 
				dynamic
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify only one strategy",
		},
		{
			name: "parse invalid no names or group",
			input: `sablier {
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify names or group",
		},
		{
			name: "parse empty names",
			input: `sablier {
				names
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify names or group",
		},
		{
			name: "parse empty group",
			input: `sablier {
				group
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify names or group",
		},
		{
			name: "parse invalid names and group",
			input: `sablier {
				names container1 container2
				group mygroup
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "you must specify either names or group",
		},
		{
			name: "parse invalid session_duration",
			input: `sablier {
				group mygroup
				session_duration invalid
				blocking
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "time: invalid duration \"invalid\"",
		},
		{
			name: "parse invalid refresh_frequency",
			input: `sablier {
				group mygroup
				dynamic {
					refresh_frequency invalid
				}
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "time: invalid duration \"invalid\"",
		},
		{
			name: "parse invalid timeout",
			input: `sablier {
				group mygroup
				blocking {
					timeout invalid
				}
			}`,
			want: caddy.Config{
				SablierURL: "http://sablier:10000",
			},
			wantErr:      true,
			wantErrValue: "time: invalid duration \"invalid\"",
		},
	}

	for _, tt := range tests {
		h := httpcaddyfile.Helper{
			Dispenser: caddyfile.NewTestDispenser(tt.input),
		}
		got := caddy.Config{}
		err := got.UnmarshalCaddyfile(h.Dispenser)

		if tt.wantErr {
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: UnmarshalCaddyfile() error = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}
			if err.Error() != tt.wantErrValue {
				t.Errorf("%s: UnmarshalCaddyfile() error = %v, wantErrValue = %v", tt.name, err.Error(), tt.wantErrValue)
			}
		} else if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("%s: UnmarshalCaddyfile() = %v, want %v", tt.name, got, tt.want)
		}

	}
}
