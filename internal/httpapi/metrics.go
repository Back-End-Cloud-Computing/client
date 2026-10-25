package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Each router owns its registry, keeping tests and multiple servers independent.
func instrument(mux *http.ServeMux, next http.Handler) http.Handler {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "client_http_requests_total", Help: "HTTP requests by route and response status."}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "client_http_request_duration_seconds", Help: "HTTP request duration."}, []string{"method", "route"})
	registry.MustRegister(requests, duration, prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		m := httpsnoop.CaptureMetrics(next, w, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		method := r.Method
		if !strings.Contains(" GET POST PUT PATCH DELETE HEAD OPTIONS ", " "+method+" ") {
			method = "OTHER"
		}
		requests.WithLabelValues(method, route, strconv.Itoa(m.Code)).Inc()
		duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	})
}
