FROM node:24-alpine AS deps
WORKDIR /deps
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=deps /deps/node_modules ./node_modules
RUN cp node_modules/htmx.org/dist/htmx.min.js static/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/invoice-app ./cmd/server

FROM alpine:3.23 AS runtime

RUN apk add --no-cache wget font-noto

RUN wget -O - 'https://github.com/typst/typst/releases/download/v0.13.1/typst-x86_64-unknown-linux-musl.tar.xz' 2>/dev/null \
    | tar xJ -C /usr/local/bin --strip-components=1 typst-x86_64-unknown-linux-musl/typst \
    && chmod +x /usr/local/bin/typst

RUN adduser -D -H -h /app -u 1000 app \
    && mkdir -p /app/data /app/migrations /app/auth-migrations \
    && chown app:app /app /app/data

COPY --from=build /out/invoice-app /app/
COPY migrations /app/migrations
COPY auth-migrations /app/auth-migrations

FROM scratch

COPY --from=runtime /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=runtime /etc/passwd /etc/passwd
COPY --from=runtime /etc/group /etc/group
COPY --from=runtime --chown=1000:1000 /tmp /tmp
COPY --from=runtime /usr/local/bin/typst /usr/local/bin/typst
COPY --from=runtime /usr/share/fonts /usr/share/fonts
COPY --from=runtime --chown=1000:1000 /app /app

WORKDIR /app

ENV ADDRESS=:8080 \
    SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

USER 1000:1000
EXPOSE 8080
CMD ["/app/invoice-app"]
