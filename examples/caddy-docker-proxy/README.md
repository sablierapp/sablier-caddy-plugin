# Caddy Docker Proxy Plugin Example

This example demonstrates how to set up [Caddy](https://caddyserver.com/) with [Sablier](https://github.com/sablierapp/sablier), [Caddy Sablier Plugin](https://github.com/sablierapp/sablier-caddy-plugin) and [Caddy-Docker-Proxy](https://github.com/lucaslorentz/caddy-docker-proxy) so Caddy can be configured with docker labels just like Sablier.

## Getting Started

### Prerequisites

- Caddy built with both the Caddy Sablier Plugin and Caddy docker Proxy. For this you can use the xcaddy builder.

```
ARG CADDY_VERSION=2.10.2
FROM caddy:${CADDY_VERSION}-builder AS builder

RUN xcaddy build \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2 \
    --with github.com/sablierapp/sablier-caddy-plugin@v1.0.1 # x-release-please-version

FROM caddy:${CADDY_VERSION}-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

CMD ["caddy", "docker-proxy"]
```
- Caddy and Sablier installed and properly configured.
- Having Caddy, Sablier and the service you want to manage with them in the same Docker network (or accessible to each other some other way).

### Caddy Docker Proxy Configuration

By default this plugin will only check for labels on running containers which causes an issue because Sablier will stop them making Caddy lose the reverse proxy settings for that container. That makes your service inaccessible so Sablier cannot start it back up when you try to access it.

The fix is simple, add the following env variable to your Caddy container:
```
CADDY_DOCKER_SCAN_STOPPED_CONTAINERS=true
```
Example compose file for this:
```yaml
services:
  caddy:
    image: caddy-plugins
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./site:/srv
      - /var/run/docker.sock:/var/run/docker.sock
      - caddy_data:/data
      - ./logs:/logs
    environment:
      - CADDY_INGRESS_NETWORKS=caddy
      - CADDY_DOCKER_SCAN_STOPPED_CONTAINERS=true
    networks:
      - caddy
    extra_hosts:
      - host.docker.internal:host-gateway
    # This is important to avoid errors later on.
    labels:
      caddy.order: sablier before reverse_proxy


  sablier:
    image: sablierapp/sablier:1.11.1 # x-release-please-version
    container_name: sablier
    restart: unless-stopped
    depends_on:
      - caddy
    command:
        - start
        - --provider.name=docker
    volumes:
      - '/var/run/docker.sock:/var/run/docker.sock'
    networks:
      - caddy

networks:
  caddy:
    external: true

volumes:
  caddy_data:
```
**_NOTE:_**  Since the `caddy` network is external, you need to create it first by running `$ docker network create caddy`.

then start the compose stack with:

```bash
docker compose up -d
```
### Adding a Service

Now that you have Caddy and Sablier running, adding a new service is easy and requires no editing of the Caddyfile. For example:
```yaml
services:
  mimic:
    image: sablierapp/mimic:v0.3.1
    healthcheck:
      test: [ "CMD", "/mimic", "healthcheck"]
      interval: 5s
    labels:
      sablier.enable: true
      sablier.group: mimic
      caddy: mimic.example.com
      caddy.reverse_proxy: "{{upstreams 80}}"
      caddy.sablier: "http://sablier:10000"
      caddy.sablier.group: mimic
      caddy.sablier.session_duration: 10m
      caddy.sablier.dynamic:
    networks:
      - caddy

networks:
  caddy:
    external: true


```

Now you can open your browser and access (no need to reload Caddy):
```
https://mimic.example.com
```
