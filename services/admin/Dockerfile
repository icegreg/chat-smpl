FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY services/admin/go.mod services/admin/go.sum* ./services/admin/
COPY pkg/ ./pkg/
COPY proto/ ./proto/

# Download dependencies
WORKDIR /app/services/admin
RUN go mod download

# Copy source code
WORKDIR /app
COPY services/admin/ ./services/admin/

# Build
WORKDIR /app/services/admin
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/admin-server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/admin-server .

EXPOSE 8085

CMD ["./admin-server"]
