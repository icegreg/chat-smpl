FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY services/api-gateway/web/package*.json ./
RUN npm ci

COPY services/api-gateway/web/ ./
RUN npm run build

FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git protobuf protobuf-dev

COPY go.mod go.sum ./
COPY proto/chat ./proto/chat
COPY proto/voice ./proto/voice
COPY pkg/metrics ./pkg/metrics
COPY pkg/migrate ./pkg/migrate
RUN go mod download

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

COPY . .

# Generate proto files (chat, files, voice)
RUN protoc -I /usr/include -I . \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/chat/*.proto proto/files/*.proto proto/voice/*.proto

# Install swag for swagger generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger docs
RUN cd services/api-gateway && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Verify proto files were generated and add dependencies
RUN ls -la proto/voice/ && \
    go get github.com/swaggo/swag@latest github.com/swaggo/http-swagger/v2@latest

# Build with GOPROXY=direct to skip module proxy
RUN CGO_ENABLED=0 GOOS=linux GOPROXY=direct go build -a -installsuffix cgo -o api-gateway ./services/api-gateway/cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/api-gateway .

# Copy web static files
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["./api-gateway"]
