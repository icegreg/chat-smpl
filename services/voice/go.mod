module github.com/icegreg/chat-smpl/services/voice

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/icegreg/chat-smpl/pkg/metrics v0.0.0
	github.com/icegreg/chat-smpl/proto/chat v0.0.0
	github.com/icegreg/chat-smpl/proto/voice v0.0.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/prometheus/client_golang v1.19.0
	github.com/rabbitmq/amqp091-go v1.9.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
)

replace github.com/icegreg/chat-smpl/proto/voice => ../../proto/voice

replace github.com/icegreg/chat-smpl/proto/chat => ../../proto/chat

replace github.com/icegreg/chat-smpl/pkg/metrics => ../../pkg/metrics
