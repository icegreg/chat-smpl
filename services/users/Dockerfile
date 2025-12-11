FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o users-server ./services/users/cmd/server

# Build the CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o users-cli ./services/users/cmd/cli

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/users-server .
COPY --from=builder /app/users-cli .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081

CMD ["./users-server"]
