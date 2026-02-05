# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies (including gcc for CGO/sqlite3 and icu-dev for beads)
RUN apk add --no-cache git ca-certificates tzdata gcc g++ musl-dev openssh-client icu-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Install bd CLI for bead operations (build from source due to replace directives)
RUN git clone --depth 1 https://github.com/steveyegge/beads.git /tmp/beads && \
    cd /tmp/beads && \
    CGO_ENABLED=1 go build -o /go/bin/bd ./cmd/bd && \
    rm -rf /tmp/beads

# Copy source code
COPY . .

# Build the application with CGO enabled for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o agenticorp \
    ./cmd/agenticorp

# Runtime stage
FROM alpine:latest

# Install runtime dependencies including git and openssh for git operations
RUN apk add --no-cache ca-certificates tzdata git openssh-client

# Create non-root user
RUN addgroup -g 1000 agenticorp && \
    adduser -D -u 1000 -G agenticorp agenticorp

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/agenticorp /app/agenticorp

# Copy bd CLI
COPY --from=builder /go/bin/bd /usr/local/bin/bd

# Copy config file
COPY --from=builder /build/config.yaml /app/config.yaml

# Copy personas directory
COPY --from=builder /build/personas /app/personas

# Copy workflows directory
COPY --from=builder /build/workflows /app/workflows

# Copy web static files
COPY --from=builder /build/web/static /app/web/static

# Create SSH directory for mounted keys and set permissions
RUN mkdir -p /home/agenticorp/.ssh && \
    chown -R agenticorp:agenticorp /home/agenticorp/.ssh && \
    chmod 700 /home/agenticorp/.ssh

# Create source mount point
RUN mkdir -p /app/src && chown agenticorp:agenticorp /app/src

# Create data directory for SQLite database persistence
RUN mkdir -p /app/data && chown agenticorp:agenticorp /app/data

# Change ownership
RUN chown -R agenticorp:agenticorp /app

# Switch to non-root user
USER agenticorp

# Configure git to use SSH
RUN git config --global core.sshCommand "ssh -i /home/agenticorp/.ssh/id_ecdsa -o UserKnownHostsFile=/home/agenticorp/.ssh/known_hosts"

# Expose port (if needed in future)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/agenticorp"]
