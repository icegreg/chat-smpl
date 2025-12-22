FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
COPY proto/voice ./proto/voice
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chat-server ./services/chat/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/chat-server .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 50051

CMD ["./chat-server"]
