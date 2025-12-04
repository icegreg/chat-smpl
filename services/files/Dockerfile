FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o files-server ./services/files/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/files-server .

# Create uploads directory
RUN mkdir -p /app/uploads

RUN adduser -D -g '' appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8082

CMD ["./files-server"]
