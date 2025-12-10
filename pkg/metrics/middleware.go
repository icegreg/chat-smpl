package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// HTTPMiddleware returns chi middleware for HTTP metrics
func HTTPMiddleware(m *HTTPMetrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.ActiveRequests.Inc()
			defer m.ActiveRequests.Dec()

			// Wrap response writer to capture status code and size
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			path := getRoutePath(r)

			m.RequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(ww.statusCode)).Inc()
			m.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
			m.RequestSize.WithLabelValues(r.Method, path).Observe(float64(r.ContentLength))
			m.ResponseSize.WithLabelValues(r.Method, path).Observe(float64(ww.bytesWritten))
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// getRoutePath returns the route pattern or path
func getRoutePath(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return r.URL.Path
}
