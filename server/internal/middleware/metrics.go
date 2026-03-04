package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type metricKey struct {
	Path   string
	Method string
	Status int
}

type Metrics struct {
	mu      sync.Mutex
	counts  map[metricKey]int64
	latency map[metricKey]time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{counts: map[metricKey]int64{}, latency: map[metricKey]time.Duration{}}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		k := metricKey{Path: r.URL.Path, Method: r.Method, Status: rec.status}
		m.mu.Lock()
		m.counts[k]++
		m.latency[k] += time.Since(start)
		m.mu.Unlock()
	})
}

func (m *Metrics) Handler(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	keys := make([]metricKey, 0, len(m.counts))
	for k := range m.counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a := fmt.Sprintf("%s|%s|%d", keys[i].Path, keys[i].Method, keys[i].Status)
		b := fmt.Sprintf("%s|%s|%d", keys[j].Path, keys[j].Method, keys[j].Status)
		return strings.Compare(a, b) < 0
	})
	for _, k := range keys {
		c := m.counts[k]
		lat := m.latency[k]
		avg := 0.0
		if c > 0 {
			avg = float64(lat.Milliseconds()) / float64(c)
		}
		_, _ = fmt.Fprintf(w, "qf_http_requests_total{path=\"%s\",method=\"%s\",status=\"%d\"} %d\n", k.Path, k.Method, k.Status, c)
		_, _ = fmt.Fprintf(w, "qf_http_request_avg_ms{path=\"%s\",method=\"%s\",status=\"%d\"} %.2f\n", k.Path, k.Method, k.Status, avg)
	}
}
