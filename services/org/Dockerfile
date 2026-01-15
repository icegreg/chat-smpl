FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files and local modules required by replace directives
COPY go.mod go.sum ./
COPY proto/chat ./proto/chat
COPY proto/voice ./proto/voice
COPY pkg/metrics ./pkg/metrics
COPY pkg/migrate ./pkg/migrate
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o org-server ./services/org/cmd/server

# Build the CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o org-cli ./services/org/cmd/cli

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/org-server .
COPY --from=builder /app/org-cli .

# Create non-root user and set ownership
RUN adduser -D -g '' appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 50055

CMD ["./org-server"]
