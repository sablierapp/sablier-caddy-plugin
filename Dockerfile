FROM caddy:2.11.2-builder AS builder

COPY . .

RUN xcaddy build \
    --with github.com/sablierapp/sablier-caddy-plugin=.

FROM caddy:2.11.2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy