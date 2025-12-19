FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git protobuf protobuf-dev

# Copy go mod files
COPY go.mod go.sum ./
COPY services/voice/go.mod ./services/voice/
COPY pkg/ ./pkg/
COPY proto/ ./proto/

# Download root module dependencies
RUN go mod download

# Generate proto files (use versions compatible with grpc v1.62.1)
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/voice/*.proto

# Copy source code
COPY services/voice/ ./services/voice/

# Download voice module dependencies and tidy
WORKDIR /app/services/voice
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /voice-service ./cmd/server

# Final image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /voice-service .

EXPOSE 50054
EXPOSE 8084

ENTRYPOINT ["./voice-service"]
