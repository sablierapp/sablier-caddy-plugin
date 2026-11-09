FROM caddy:2.11.9-builder AS builder

COPY . .

RUN xcaddy build \
    --with github.com/sablierapp/sablier-caddy-plugin=.

FROM caddy:2.11.9

COPY --from=builder /usr/bin/caddy /usr/bin/caddy