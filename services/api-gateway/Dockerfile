FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./services/api-gateway/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/api-gateway .

# Copy web static files
COPY --from=builder /app/services/api-gateway/web/dist ./web/dist

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["./api-gateway"]
