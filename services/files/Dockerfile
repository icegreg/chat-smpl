FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
COPY proto/voice ./proto/voice
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o files-server ./services/files/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/files-server .

# Create user and storage directory with correct permissions
RUN adduser -D -g '' appuser && \
    mkdir -p /app/storage && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 8082

CMD ["./files-server"]
