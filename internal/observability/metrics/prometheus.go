// Package metrics provides Prometheus metrics for the prescription engine.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all application metrics
type Metrics struct {
	PrescriptionsCreated  prometheus.Counter
	PrescriptionsRouted   prometheus.Counter
	PrescriptionsFailed   prometheus.Counter
	ProcessingDuration    prometheus.Histogram
	ActivePrescriptions   prometheus.Gauge
	KafkaMessagesProduced prometheus.Counter
	KafkaMessagesConsumed prometheus.Counter
	OutboxPending         prometheus.Gauge
	CircuitBreakerState   *prometheus.GaugeVec
}

// New creates and registers all metrics
func New() *Metrics {
	m := &Metrics{
		PrescriptionsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "prescriptions_created_total",
			Help: "Total prescriptions created",
		}),
		PrescriptionsRouted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "prescriptions_routed_total",
			Help: "Total prescriptions routed to pharmacy",
		}),
		PrescriptionsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "prescriptions_failed_total",
			Help: "Total failed prescriptions",
		}),
		ProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "prescription_processing_duration_seconds",
			Help:    "Prescription processing duration",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),
		ActivePrescriptions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "prescriptions_active",
			Help: "Currently active prescriptions",
		}),
		KafkaMessagesProduced: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total Kafka messages produced",
		}),
		KafkaMessagesConsumed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total Kafka messages consumed",
		}),
		OutboxPending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "outbox_pending_entries",
			Help: "Pending outbox entries",
		}),
		CircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		}, []string{"name"}),
	}

	prometheus.MustRegister(
		m.PrescriptionsCreated,
		m.PrescriptionsRouted,
		m.PrescriptionsFailed,
		m.ProcessingDuration,
		m.ActivePrescriptions,
		m.KafkaMessagesProduced,
		m.KafkaMessagesConsumed,
		m.OutboxPending,
		m.CircuitBreakerState,
	)

	return m
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
