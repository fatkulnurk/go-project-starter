# Multi-stage build: one image with every application binary, so all compose
# services share a single build. Fully static (CGO_ENABLED=0): all drivers
# (pgx, go-sql-driver/mysql, go-redis, amqp091-go, franz-go, AWS SDK) are pure Go.
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN set -eux; \
    mkdir -p /out; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/subscriber ./cmd/subscriber; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

# The migrate binary reads migrations from ./migrations (relative to the
# working directory); public/ is served statically at /assets/*.
COPY --from=build /out/* /usr/local/bin/
COPY migrations /app/migrations
COPY public /app/public

WORKDIR /app
CMD ["api"]