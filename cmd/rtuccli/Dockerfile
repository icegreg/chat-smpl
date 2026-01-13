FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY cmd/rtuccli/go.mod cmd/rtuccli/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/rtuccli/ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o rtuccli

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/rtuccli /usr/local/bin/rtuccli

ENTRYPOINT ["rtuccli"]
