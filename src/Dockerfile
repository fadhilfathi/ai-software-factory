# =============================================================================
# Stage 1: Build the Go binary
# =============================================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Cache module downloads first (layer stays cached unless go.mod changes)
COPY go.mod ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /build/api ./cmd/main.go

# =============================================================================
# Stage 2: Minimal runtime image
# =============================================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /build/api /app/api
COPY --from=builder /build/db/migrations /app/db/migrations

WORKDIR /app
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/v1/healthz || exit 1

ENTRYPOINT ["/app/api"]
