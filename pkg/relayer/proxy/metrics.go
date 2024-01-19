package proxy

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// Register the defined metrics with Prometheus
	// prometheus.MustRegister(httpRequestsTotal, httpRequestDurationSeconds, httpRequestSizeBytes, httpResponseSizeBytes)
}

// relays_total{proxy_name, service_id}
// relays_duration_seconds{proxy_name, service_id}
// relays_size_bytes{proxy_name, service_id}

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "status_code", "proxy_name"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
		},
		[]string{"method", "status_code", "proxy_name"},
	)
	httpRequestSizeBytes = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Size of HTTP requests in bytes",
		},
		[]string{"method", "status_code", "proxy_name"},
	)
	httpResponseSizeBytes = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Size of HTTP responses in bytes",
		},
		[]string{"method", "status_code", "proxy_name"},
	)
)

func (sync *synchronousRPCServer) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		proxyName := sync.proxyConfig.ProxyName

		observer := &responseObserver{ResponseWriter: w, statusCode: http.StatusOK}
		timer := prometheus.NewTimer(httpRequestDurationSeconds.WithLabelValues(method, strconv.Itoa(observer.statusCode), proxyName))

		next.ServeHTTP(observer, r)

		timer.ObserveDuration()
		requestSize := computeApproximateRequestSize(r)

		statusCode := strconv.Itoa(observer.statusCode)
		httpRequestsTotal.WithLabelValues(method, statusCode, proxyName).Inc()
		httpRequestSizeBytes.WithLabelValues(method, statusCode, proxyName).Observe(float64(requestSize))
		httpResponseSizeBytes.WithLabelValues(method, statusCode, proxyName).Observe(float64(observer.bytesWritten))
	})
}

type responseObserver struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (o *responseObserver) WriteHeader(statusCode int) {
	o.statusCode = statusCode
	o.ResponseWriter.WriteHeader(statusCode)
}

func (o *responseObserver) Write(b []byte) (int, error) {
	if o.statusCode == 0 {
		o.statusCode = http.StatusOK
	}
	size, err := o.ResponseWriter.Write(b)
	o.bytesWritten += int64(size)
	return size, err
}

// This is approximation of the size of the request.
func computeApproximateRequestSize(r *http.Request) int {
	size := 0
	if r.ContentLength > 0 {
		size += int(r.ContentLength)
	}
	if r.URL != nil {
		size += len(r.URL.String())
	}
	for name, headers := range r.Header {
		size += len(name)
		for _, h := range headers {
			size += len(h)
		}
	}
	return size
}
