FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy everything (local replace directives require source files)
COPY . .

# Add centrifuge-go dependency and build
RUN go get github.com/centrifugal/centrifuge-go@latest && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /health-service ./services/health/cmd/server

# Final image
FROM alpine:3.19

RUN apk add --no-cache wget ca-certificates

COPY --from=builder /health-service /health-service

EXPOSE 8085

CMD ["/health-service"]
