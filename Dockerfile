# ── Stage 1: Build Go binary ───────────────────────────────────────────────────
FROM golang:1.26-alpine AS go-builder
ENV CGO_ENABLED=0
RUN apk add --no-cache git make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build-linux

# ── Stage 2: CA certs ─────────────────────────────────────────────────────────
FROM alpine:3.21 AS certs
RUN apk add -U --no-cache ca-certificates

# ── Stage 3: Final image ───────────────────────────────────────────────────────
FROM alpine:3.21 AS release
LABEL org.opencontainers.image.source="https://github.com/dathan/go-jira-jwt-stats"

COPY --from=certs      /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /app/bin/go-jira-jwt-stats         /app/go-jira-jwt-stats

WORKDIR /app
ENTRYPOINT ["/app/go-jira-jwt-stats"]
