package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	WebhooksReceived   prometheus.Counter
	WebhooksProcessed  *prometheus.CounterVec
	ForwardingDuration *prometheus.HistogramVec
	ForwardingTotal    *prometheus.CounterVec
	RoutesMatched      *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	m := &Metrics{
		WebhooksReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webhook_middleman_webhooks_received_total",
			Help: "Total number of webhook requests received",
		}),
		WebhooksProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webhook_middleman_webhooks_processed_total",
			Help: "Total number of webhook requests processed",
		}, []string{"status"}),
		ForwardingDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "webhook_middleman_forwarding_duration_seconds",
			Help:    "Time taken to forward webhook to destination",
			Buckets: prometheus.DefBuckets,
		}, []string{"destination", "status"}),
		ForwardingTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webhook_middleman_forwarding_total",
			Help: "Total number of forwarding attempts",
		}, []string{"destination", "status"}),
		RoutesMatched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webhook_middleman_routes_matched_total",
			Help: "Total number of routes matched per webhook",
		}, []string{"method", "path"}),
	}

	// Register metrics
	prometheus.MustRegister(
		m.WebhooksReceived,
		m.WebhooksProcessed,
		m.ForwardingDuration,
		m.ForwardingTotal,
		m.RoutesMatched,
	)

	return m
}
