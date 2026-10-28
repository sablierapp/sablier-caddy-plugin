package caddy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/sablierapp/sablier/client"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("sablier", parseCaddyfile)

}

type DynamicConfiguration struct {
	DisplayName      string
	ShowDetails      *bool
	Theme            string
	RefreshFrequency *time.Duration
}

type BlockingConfiguration struct {
	Timeout *time.Duration
}

type Config struct {
	SablierURL      string
	Names           []string
	Group           string
	SessionDuration *time.Duration
	Dynamic         *DynamicConfiguration
	Blocking        *BlockingConfiguration
}

func CreateConfig() *Config {
	return &Config{
		SablierURL:      "http://sablier:10000",
		Names:           []string{},
		SessionDuration: nil,
		Dynamic:         nil,
		Blocking:        nil,
	}
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
//		sablier [<sablierURL>] {
//			[names container1,container2,...]
//			[group mygroup]
//			[session_duration 30m]
//			dynamic {
//				[display_name This is my display name]
//				[show_details yes|true|on]
//				[theme hacker-terminal]
//				[refresh_frequency 2s]
//			}
//			blocking {
//				[timeout 1m]
//			}
//		}
//
func (c *Config) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			c.SablierURL = d.Val()
		} else {
			c.SablierURL = "http://sablier:10000"
		}
		if d.NextArg() {
			return d.ArgErr()
		}
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			subdirective := d.Val()
			args := strings.Join(d.RemainingArgs(), " ")
			switch subdirective {
			case "names":
				c.Names = parseNames(args)
			case "group":
				c.Group = args
			case "session_duration":
				duration, err := time.ParseDuration(args)
				if err != nil {
					return err
				}
				c.SessionDuration = &duration
			case "dynamic":
				dynamic, err := parseDynamic(d)
				if err != nil {
					return err
				}
				c.Dynamic = dynamic
			case "blocking":
				blocking, err := parseBlocking(d)
				if err != nil {
					return err
				}
				c.Blocking = blocking
			}
		}
	}

	if c.Blocking == nil && c.Dynamic == nil {
		return fmt.Errorf("you must specify one strategy (dynamic or blocking)")
	}

	if c.Blocking != nil && c.Dynamic != nil {
		return fmt.Errorf("you must specify only one strategy")
	}

	if len(c.Names) == 0 && len(c.Group) == 0 {
		return fmt.Errorf("you must specify names or group")
	}

	if len(c.Names) > 0 && len(c.Group) > 0 {
		return fmt.Errorf("you must specify either names or group")
	}

	return nil
}

func parseNames(value string) []string {
	names := strings.Split(value, " ")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}

	if len(names) == 1 && names[0] == "" {
		return make([]string, 0)
	}

	return names
}

func parseDynamic(d *caddyfile.Dispenser) (*DynamicConfiguration, error) {
	conf := &DynamicConfiguration{}
	for nesting := d.Nesting(); d.NextBlock(nesting); {
		subdirective := d.Val()
		args := strings.Join(d.RemainingArgs(), " ")
		switch subdirective {
		case "display_name":
			conf.DisplayName = args
		case "show_details":
			shouldShow := isEnabledArg(args)
			conf.ShowDetails = &shouldShow
		case "theme":
			conf.Theme = args
		case "refresh_frequency":
			duration, err := time.ParseDuration(args)
			if err != nil {
				return nil, err
			}
			conf.RefreshFrequency = &duration
		}
	}
	return conf, nil
}

func parseBlocking(d *caddyfile.Dispenser) (*BlockingConfiguration, error) {
	conf := &BlockingConfiguration{}
	for nesting := d.Nesting(); d.NextBlock(nesting); {
		subdirective := d.Val()
		args := strings.Join(d.RemainingArgs(), " ")
		switch subdirective {
		case "timeout":
			duration, err := time.ParseDuration(args)
			if err != nil {
				return nil, err
			}
			conf.Timeout = &duration
		}
	}
	return conf, nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var c Config
	err := c.UnmarshalCaddyfile(h.Dispenser)
	return SablierMiddleware{Config: c}, err
}

func isEnabledArg(s string) bool {
	if s == "yes" || s == "true" || s == "on" {
		return true
	}
	return false
}

// start calls the configured strategy for one incoming request. Caddy
// placeholders (e.g. {labels.3}) in names and group are expanded with repl,
// and names that expand to nothing are dropped.
func (c *Config) start(ctx context.Context, sablier *client.Client, repl *caddy.Replacer) (*client.Response, error) {
	names, group := c.Names, c.Group
	if repl != nil {
		names = make([]string, 0, len(c.Names))
		for _, name := range c.Names {
			if expanded := repl.ReplaceAll(name, ""); expanded != "" {
				names = append(names, expanded)
			}
		}
		group = repl.ReplaceAll(c.Group, "")
	}

	var sessionDuration *string
	if c.SessionDuration != nil {
		sessionDuration = ptr(c.SessionDuration.String())
	}

	if c.Dynamic != nil {
		params := client.StartDynamicParams{
			Names:           nonEmpty(names),
			Group:           nonEmptyString(group),
			SessionDuration: sessionDuration,
			DisplayName:     nonEmptyString(c.Dynamic.DisplayName),
			Theme:           nonEmptyString(c.Dynamic.Theme),
			ShowDetails:     c.Dynamic.ShowDetails,
		}
		if c.Dynamic.RefreshFrequency != nil {
			params.RefreshFrequency = ptr(c.Dynamic.RefreshFrequency.String())
		}
		return sablier.StartDynamic(ctx, params)
	}

	params := client.StartBlockingParams{
		Names:           nonEmpty(names),
		Group:           nonEmptyString(group),
		SessionDuration: sessionDuration,
	}
	if c.Blocking.Timeout != nil {
		params.Timeout = ptr(c.Blocking.Timeout.String())
	}
	return sablier.StartBlocking(ctx, params)
}

func ptr[T any](v T) *T { return &v }

// nonEmpty and nonEmptyString leave unset parameters out of the request, so
// Sablier applies its defaults.
func nonEmpty(values []string) *[]string {
	if len(values) == 0 {
		return nil
	}
	return &values
}

func nonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
