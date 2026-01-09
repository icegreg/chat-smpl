FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git protobuf protobuf-dev

COPY go.mod go.sum ./
COPY proto/ ./proto/
COPY pkg/ ./pkg/
RUN go mod download

# Generate proto files AFTER downloading dependencies but BEFORE building
# This ensures we use freshly generated proto files with consistent versions
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/chat/*.proto

# Copy service source code (proto/ is already in place with regenerated files)
COPY services/ ./services/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chat-server ./services/chat/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/chat-server .

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 50051

CMD ["./chat-server"]
