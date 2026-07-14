package observability

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProbeFunc func(context.Context) error

type requestKey struct {
	service string
	handler string
	method  string
	status  string
}

type dependencyKey struct {
	service    string
	dependency string
}

type histogram struct {
	buckets []uint64
	count   uint64
	sum     float64
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

var (
	metricsMu sync.RWMutex

	httpRequestsTotal = map[requestKey]uint64{}
	httpDurations     = map[requestKey]*histogram{}
	dependencyUp      = map[dependencyKey]float64{}

	durationBucketBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5}
)

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(service, handler string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		startedAt := time.Now()
		next.ServeHTTP(recorder, r)

		key := requestKey{
			service: service,
			handler: handler,
			method:  r.Method,
			status:  strconv.Itoa(recorder.status),
		}
		observeRequest(key, time.Since(startedAt).Seconds())
	})
}

func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, renderMetrics())
	})
}

func SetDependencyStatus(service, dependency string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1
	}

	metricsMu.Lock()
	dependencyUp[dependencyKey{
		service:    service,
		dependency: dependency,
	}] = value
	metricsMu.Unlock()
}

func StartDependencyMonitor(ctx context.Context, service string, interval time.Duration, probes map[string]ProbeFunc) {
	runOnce := func() {
		for dependency, probe := range probes {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := probe(probeCtx)
			cancel()
			SetDependencyStatus(service, dependency, err == nil)
		}
	}

	runOnce()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
}

func observeRequest(key requestKey, seconds float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	httpRequestsTotal[key]++

	h, exists := httpDurations[key]
	if !exists {
		h = &histogram{
			buckets: make([]uint64, len(durationBucketBounds)),
		}
		httpDurations[key] = h
	}

	for index, bound := range durationBucketBounds {
		if seconds <= bound {
			h.buckets[index]++
		}
	}
	h.count++
	h.sum += seconds
}

func renderMetrics() string {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	var builder strings.Builder

	builder.WriteString("# HELP antifraud_http_requests_total Total HTTP requests handled by Anti-Fraud services.\n")
	builder.WriteString("# TYPE antifraud_http_requests_total counter\n")

	requestKeys := sortedRequestKeys(httpRequestsTotal)
	for _, key := range requestKeys {
		fmt.Fprintf(
			&builder,
			"antifraud_http_requests_total{service=%q,handler=%q,method=%q,status=%q} %d\n",
			key.service, key.handler, key.method, key.status, httpRequestsTotal[key],
		)
	}

	builder.WriteString("# HELP antifraud_http_request_duration_seconds HTTP request duration in seconds for Anti-Fraud services.\n")
	builder.WriteString("# TYPE antifraud_http_request_duration_seconds histogram\n")

	durationKeys := sortedRequestKeys(httpDurations)
	for _, key := range durationKeys {
		h := httpDurations[key]
		for index, bound := range durationBucketBounds {
			fmt.Fprintf(
				&builder,
				"antifraud_http_request_duration_seconds_bucket{service=%q,handler=%q,method=%q,status=%q,le=%q} %d\n",
				key.service, key.handler, key.method, key.status, formatBound(bound), h.buckets[index],
			)
		}
		fmt.Fprintf(
			&builder,
			"antifraud_http_request_duration_seconds_bucket{service=%q,handler=%q,method=%q,status=%q,le=%q} %d\n",
			key.service, key.handler, key.method, key.status, "+Inf", h.count,
		)
		fmt.Fprintf(
			&builder,
			"antifraud_http_request_duration_seconds_sum{service=%q,handler=%q,method=%q,status=%q} %.6f\n",
			key.service, key.handler, key.method, key.status, h.sum,
		)
		fmt.Fprintf(
			&builder,
			"antifraud_http_request_duration_seconds_count{service=%q,handler=%q,method=%q,status=%q} %d\n",
			key.service, key.handler, key.method, key.status, h.count,
		)
	}

	builder.WriteString("# HELP antifraud_dependency_up Whether a service can reach a dependency right now (1=healthy, 0=unhealthy).\n")
	builder.WriteString("# TYPE antifraud_dependency_up gauge\n")

	dependencyKeys := sortedDependencyKeys(dependencyUp)
	for _, key := range dependencyKeys {
		fmt.Fprintf(
			&builder,
			"antifraud_dependency_up{service=%q,dependency=%q} %.0f\n",
			key.service, key.dependency, dependencyUp[key],
		)
	}

	builder.WriteString("# HELP go_goroutines Number of goroutines that currently exist.\n")
	builder.WriteString("# TYPE go_goroutines gauge\n")
	fmt.Fprintf(&builder, "go_goroutines %d\n", runtime.NumGoroutine())

	return builder.String()
}

func sortedRequestKeys[T any](source map[requestKey]T) []requestKey {
	keys := make([]requestKey, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].service != keys[j].service {
			return keys[i].service < keys[j].service
		}
		if keys[i].handler != keys[j].handler {
			return keys[i].handler < keys[j].handler
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	return keys
}

func sortedDependencyKeys(source map[dependencyKey]float64) []dependencyKey {
	keys := make([]dependencyKey, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].service != keys[j].service {
			return keys[i].service < keys[j].service
		}
		return keys[i].dependency < keys[j].dependency
	})
	return keys
}

func formatBound(bound float64) string {
	return strconv.FormatFloat(bound, 'f', -1, 64)
}
