FROM caddy:2-builder AS builder

COPY . /module
WORKDIR /module

RUN xcaddy build --with github.com/jarv/caddy-goatcounter=./

FROM caddy:2-alpine

COPY --from=builder /module/caddy /usr/bin/caddy

EXPOSE 80 443

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]