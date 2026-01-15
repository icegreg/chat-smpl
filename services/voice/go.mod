module github.com/icegreg/chat-smpl/services/voice

go 1.24.0

toolchain go1.24.11

require (
	github.com/google/uuid v1.6.0
	github.com/icegreg/chat-smpl/pkg/metrics v0.0.0
	github.com/icegreg/chat-smpl/pkg/migrate v0.0.0
	github.com/icegreg/chat-smpl/proto/chat v0.0.0
	github.com/icegreg/chat-smpl/proto/voice v0.0.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/rabbitmq/amqp091-go v1.9.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.7
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-chi/chi/v5 v5.1.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/prometheus/client_golang v1.19.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
)

replace github.com/icegreg/chat-smpl/proto/voice => ../../proto/voice

replace github.com/icegreg/chat-smpl/proto/chat => ../../proto/chat

replace github.com/icegreg/chat-smpl/pkg/metrics => ../../pkg/metrics

replace github.com/icegreg/chat-smpl/pkg/migrate => ../../pkg/migrate
