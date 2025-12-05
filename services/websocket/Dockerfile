FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o websocket-server ./services/websocket/cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/websocket-server .

RUN adduser -D -g '' appuser
USER appuser

CMD ["./websocket-server"]
