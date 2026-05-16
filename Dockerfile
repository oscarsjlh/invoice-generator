FROM golang:1.26 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/invoice-app ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    && curl -fsSL https://github.com/typst/typst/releases/download/v0.13.1/typst-x86_64-unknown-linux-musl.tar.xz \
    | tar xJ -C /usr/local/bin --strip-components=1 typst-x86_64-unknown-linux-musl/typst \
    && chmod +x /usr/local/bin/typst \
    && apt-get remove -y curl && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*

RUN useradd --system --create-home app
COPY --from=build /out/invoice-app /app/invoice-app
COPY migrations /app/migrations
COPY templates /app/templates

ENV ADDRESS=:8080 \
    DATABASE_PATH=/app/data/invoices.db \
    TEMPLATES_DIR=/app/templates \
    MIGRATIONS_DIR=/app/migrations

RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
CMD ["/app/invoice-app"]
