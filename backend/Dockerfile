# Multi-stage build for the digital-twin-community backend
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Build migrate CLI (statically linked; reused by the K8s init-container)
RUN CGO_ENABLED=0 go install -tags 'postgres' \
    github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev}" \
    -o /bin/server \
    ./cmd/server

# Stage 2: Runtime (minimal distroless image)
FROM gcr.io/distroless/static-debian12:nonroot

# Server binary
COPY --from=builder /bin/server /server
# migrate binary + SQL migration files (used by the K8s init-container)
COPY --from=builder /go/bin/migrate /migrate
COPY --from=builder /app/db/migrations /migrations
# TLS certs + timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Non-root user (distroless default)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
