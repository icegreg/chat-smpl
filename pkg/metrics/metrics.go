package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics contains HTTP request metrics
type HTTPMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestSize     *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
}

// GRPCMetrics contains gRPC request metrics
type GRPCMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	StreamsTotal    *prometheus.CounterVec
	ActiveStreams   prometheus.Gauge
}

// RabbitMQMetrics contains RabbitMQ message metrics
type RabbitMQMetrics struct {
	MessagesPublished *prometheus.CounterVec
	MessagesConsumed  *prometheus.CounterVec
	PublishDuration   *prometheus.HistogramVec
	PublishErrors     *prometheus.CounterVec
}

// NewHTTPMetrics creates HTTP metrics for a service
func NewHTTPMetrics(serviceName string) *HTTPMetrics {
	return &HTTPMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceName + "_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		RequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceName + "_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"method", "path"},
		),
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceName + "_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"method", "path"},
		),
		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: serviceName + "_http_active_requests",
				Help: "Number of active HTTP requests",
			},
		),
	}
}

// NewGRPCMetrics creates gRPC metrics for a service
func NewGRPCMetrics(serviceName string) *GRPCMetrics {
	return &GRPCMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceName + "_grpc_request_duration_seconds",
				Help:    "gRPC request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		StreamsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_grpc_streams_total",
				Help: "Total number of gRPC streams",
			},
			[]string{"method", "type"},
		),
		ActiveStreams: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: serviceName + "_grpc_active_streams",
				Help: "Number of active gRPC streams",
			},
		),
	}
}

// NewRabbitMQMetrics creates RabbitMQ metrics for a service
func NewRabbitMQMetrics(serviceName string) *RabbitMQMetrics {
	return &RabbitMQMetrics{
		MessagesPublished: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_rabbitmq_messages_published_total",
				Help: "Total number of messages published to RabbitMQ",
			},
			[]string{"exchange", "routing_key"},
		),
		MessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_rabbitmq_messages_consumed_total",
				Help: "Total number of messages consumed from RabbitMQ",
			},
			[]string{"queue"},
		),
		PublishDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceName + "_rabbitmq_publish_duration_seconds",
				Help:    "RabbitMQ publish duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"exchange"},
		),
		PublishErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceName + "_rabbitmq_publish_errors_total",
				Help: "Total number of RabbitMQ publish errors",
			},
			[]string{"exchange", "error_type"},
		),
	}
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// Custom application metrics
var (
	// Chat-specific metrics
	ChatMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_messages_total",
			Help: "Total number of chat messages",
		},
		[]string{"chat_id", "type"},
	)

	ChatParticipantsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chat_participants",
			Help: "Number of participants in a chat",
		},
		[]string{"chat_id"},
	)

	// WebSocket/Centrifugo metrics
	WebSocketConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_connections_total",
			Help: "Total number of WebSocket connections",
		},
		[]string{"status"},
	)

	WebSocketActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	// File upload metrics
	FilesUploadedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "files_uploaded_total",
			Help: "Total number of files uploaded",
		},
		[]string{"content_type"},
	)

	FileUploadSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "file_upload_size_bytes",
			Help:    "File upload size in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to 16MB
		},
		[]string{"content_type"},
	)

	FileDownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "file_downloads_total",
			Help: "Total number of file downloads",
		},
		[]string{"file_id"},
	)

	// Auth metrics
	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"type", "status"},
	)

	ActiveSessionsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "auth_active_sessions",
			Help: "Number of active user sessions",
		},
	)
)
