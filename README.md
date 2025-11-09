<!-- omit in toc -->
# Caddy Sablier Plugin

[![Go Report Card](https://goreportcard.com/badge/github.com/sablierapp/sablier-caddy-plugin)](https://goreportcard.com/report/github.com/sablierapp/sablier-caddy-plugin)
[![Discord](https://img.shields.io/discord/1298488955947454464?logo=discord&logoColor=5865F2&cacheSeconds=1&link=http%3A%2F%2F)](https://discord.gg/6TXtfeWqx3)

Start your containers on demand, shut them down automatically when there's no activity using [Caddy](https://caddyserver.com).

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
	- [Minimal configuration](#minimal-configuration)
- [Other Plugins](#other-plugins)
- [Community](#community)
- [Support](#support)

See the [official module page](https://caddyserver.com/docs/modules/http.handlers.sablier#github.com/sablierapp/sablier-caddy-plugin).

## Installation

This plugin does not come with a pre-built version of [Caddy](https://caddyserver.com) (see [#10](https://github.com/sablierapp/sablier-caddy-plugin/issues/10)).

You must build a custom version of [Caddy](https://caddyserver.com) with this plugin. See [building from source](https://caddyserver.com/docs/build#xcaddy) for more information.


**Dockerfile Example**:
```Dockerfile
FROM caddy:2.10.2-builder AS builder

RUN xcaddy build \
    --with github.com/sablierapp/sablier-caddy-plugin:v1.0.1 # x-release-please-version

FROM caddy:2.10.2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

## Usage

See the [docker example](./examples/docker/) on how to use the plugin.

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

### Minimal configuration

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

## Other Plugins

- [sablier-traefik-plugin](https://github.com/sablierapp/sablier-traefik-plugin)
- [sablier-proxywasm-plugin](https://github.com/sablierapp/sablier-proxywasm-plugin)

## Community

Join our Discord server to discuss and get support!

[![Discord](https://img.shields.io/discord/1298488955947454464?logo=discord&logoColor=5865F2&cacheSeconds=1&link=http%3A%2F%2F)](https://discord.gg/6TXtfeWqx3)

## Support

See [SUPPORT.md](SUPPORT.md)