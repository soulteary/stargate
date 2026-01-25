package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metricskit "github.com/soulteary/metrics-kit"
)

var (
	// Registry is the Prometheus registry for Stargate metrics
	Registry *metricskit.Registry

	// Auth holds authentication-related metrics
	Auth *metricskit.AuthMetrics

	// Herald holds metrics for Herald service calls
	Herald *metricskit.ExternalServiceMetrics

	// Warden holds metrics for Warden service calls
	Warden *metricskit.ExternalServiceMetrics

	// AuthRefreshTotal counts auth refresh operations
	AuthRefreshTotal *prometheus.CounterVec

	// AuthRefreshDuration measures auth refresh operation duration
	AuthRefreshDuration *prometheus.HistogramVec
)

func init() {
	Init()
}

// Init initializes all Stargate metrics using metrics-kit
func Init() {
	Registry = metricskit.NewRegistry("stargate")
	cm := metricskit.NewCommonMetrics(Registry)

	Auth = cm.NewAuthMetrics()
	Herald = cm.NewExternalServiceMetrics("herald")
	Warden = cm.NewExternalServiceMetrics("warden")

	// Auth refresh metrics using builder pattern
	AuthRefreshTotal = Registry.Counter("auth_refresh_total").
		Help("Total number of auth refresh operations").
		Labels("result").
		BuildVec()

	AuthRefreshDuration = Registry.Histogram("auth_refresh_duration_seconds").
		Help("Auth refresh operation duration in seconds").
		Labels("result").
		Buckets(metricskit.HTTPDurationBuckets()).
		BuildVec()
}

// RecordAuthRequest records an authentication request
func RecordAuthRequest(method, result string) {
	Auth.RecordAuthRequest(method, result)
}

// RecordHeraldCall records a Herald service call
func RecordHeraldCall(operation, result string, duration time.Duration) {
	Herald.RecordCall(operation, result, duration)
}

// RecordWardenCall records a Warden service call
func RecordWardenCall(operation, result string, duration time.Duration) {
	Warden.RecordCall(operation, result, duration)
}

// RecordSessionCreated records a session creation
func RecordSessionCreated() {
	Auth.RecordSessionCreated()
}

// RecordSessionDestroyed records a session destruction
func RecordSessionDestroyed() {
	Auth.RecordSessionDestroyed()
}

// RecordAuthRefresh records an auth refresh operation
func RecordAuthRefresh(result string, duration time.Duration) {
	AuthRefreshTotal.WithLabelValues(result).Inc()
	AuthRefreshDuration.WithLabelValues(result).Observe(duration.Seconds())
}
