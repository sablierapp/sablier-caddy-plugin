<!-- omit in toc -->
# Caddy Sablier Plugin

[![Go Report Card](https://goreportcard.com/badge/github.com/sablierapp/sablier-caddy-plugin)](https://goreportcard.com/report/github.com/sablierapp/sablier-caddy-plugin) 

Start your containers on demand, shut them down automatically when there's no activity using Caddy.

- [Installation](#installation)
- [Configuration](#configuration)
	- [Exemple with a minimal configuration](#exemple-with-a-minimal-configuration)

See the [official module page](https://caddyserver.com/docs/modules/http.handlers.sablier#github.com/sablierapp/sablier-caddy-plugin).

## Installation

This plugin does not come with a pre-built version of Caddy (see [#10](https://github.com/sablierapp/sablier-caddy-plugin/issues/10)).

You must build a custom version of Caddy with this plugin. See [](https://caddyserver.com/docs/build#xcaddy) for more information.


```Dockerfile
FROM caddy:2.10.2-builder AS builder

RUN xcaddy build \
    --with github.com/sablierapp/sablier-caddy-plugin:v1.1.0 # x-release-please-version

FROM caddy:2.10.2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

## Configuration

You can have the following configuration:

```Caddyfile
:80 {
	route /my/route {
    sablier [<sablierURL>=http://sablier:10000] {
			[names container1,container2,...]
			[group mygroup]
			[session_duration 30m]
			dynamic {
				[display_name This is my display name]
				[show_details yes|true|on]
				[theme hacker-terminal]
				[refresh_frequency 2s]
			}
			blocking {
				[timeout 1m]
			}
		}
    reverse_proxy myservice:port
  }
}
```

### Exemple with a minimal configuration

Almost all options are optional and you can setup very simple rules to use the server default values.

```Caddyfile
:80 {
	route /my/route {
    sablier {
			group mygroup
			dynamic
		}
    reverse_proxy myservice:port
  }
}
```