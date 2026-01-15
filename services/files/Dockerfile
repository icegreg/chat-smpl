FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git protobuf protobuf-dev

COPY go.mod go.sum ./
COPY proto/ ./proto/
COPY pkg/ ./pkg/
RUN go mod download

# Generate proto files
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/files/*.proto proto/chat/*.proto

COPY services/ ./services/

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
