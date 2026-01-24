package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AuthRequestsTotal counts authentication requests by method and result
	AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stargate_auth_requests_total",
			Help: "Total number of authentication requests",
		},
		[]string{"method", "result"}, // method: password, warden, warden_otp; result: success, failure
	)

	// HeraldCallsTotal counts Herald service calls by operation and result
	HeraldCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stargate_herald_calls_total",
			Help: "Total number of Herald service calls",
		},
		[]string{"operation", "result"}, // operation: create_challenge, verify_challenge; result: success, failure
	)

	// WardenCallsTotal counts Warden service calls by operation and result
	WardenCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stargate_warden_calls_total",
			Help: "Total number of Warden service calls",
		},
		[]string{"operation", "result"}, // operation: check_user, get_user_info; result: success, failure
	)

	// SessionCreatedTotal counts session creations
	SessionCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "stargate_session_created_total",
			Help: "Total number of sessions created",
		},
	)

	// SessionDestroyedTotal counts session destructions
	SessionDestroyedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "stargate_session_destroyed_total",
			Help: "Total number of sessions destroyed",
		},
	)

	// HeraldLatencySeconds measures Herald service call latency
	HeraldLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stargate_herald_latency_seconds",
			Help:    "Herald service call latency in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"operation"}, // operation: create_challenge, verify_challenge
	)

	// WardenLatencySeconds measures Warden service call latency
	WardenLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stargate_warden_latency_seconds",
			Help:    "Warden service call latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"}, // operation: check_user, get_user_info
	)

	// AuthRefreshTotal counts auth refresh operations
	AuthRefreshTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stargate_auth_refresh_total",
			Help: "Total number of auth refresh operations",
		},
		[]string{"result"}, // result: success, failure
	)

	// AuthRefreshDuration measures auth refresh operation duration
	AuthRefreshDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stargate_auth_refresh_duration_seconds",
			Help:    "Auth refresh operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"}, // result: success, failure
	)
)

// RecordAuthRequest records an authentication request
func RecordAuthRequest(method, result string) {
	AuthRequestsTotal.WithLabelValues(method, result).Inc()
}

// RecordHeraldCall records a Herald service call
func RecordHeraldCall(operation, result string, duration time.Duration) {
	HeraldCallsTotal.WithLabelValues(operation, result).Inc()
	HeraldLatencySeconds.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordWardenCall records a Warden service call
func RecordWardenCall(operation, result string, duration time.Duration) {
	WardenCallsTotal.WithLabelValues(operation, result).Inc()
	WardenLatencySeconds.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordSessionCreated records a session creation
func RecordSessionCreated() {
	SessionCreatedTotal.Inc()
}

// RecordSessionDestroyed records a session destruction
func RecordSessionDestroyed() {
	SessionDestroyedTotal.Inc()
}

// RecordAuthRefresh records an auth refresh operation
func RecordAuthRefresh(result string, duration time.Duration) {
	AuthRefreshTotal.WithLabelValues(result).Inc()
	AuthRefreshDuration.WithLabelValues(result).Observe(duration.Seconds())
}
