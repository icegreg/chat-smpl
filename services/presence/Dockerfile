# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for go mod
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o presence-server ./services/presence/cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/presence-server .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 50052

CMD ["./presence-server"]
