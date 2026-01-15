FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files and local dependencies
COPY go.mod go.sum ./
COPY pkg/ ./pkg/
COPY proto/ ./proto/

# Download dependencies
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o websocket-server ./services/websocket/cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/websocket-server .

RUN adduser -D -g '' appuser
USER appuser

CMD ["./websocket-server"]
