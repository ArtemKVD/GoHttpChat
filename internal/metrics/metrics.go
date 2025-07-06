package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MessagesSend = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_messages_sent_total",
	}, []string{"sender", "receiver"})

	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_active_users",
	})

	HttpDurations = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Buckets: []float64{0.1, 0.3, 0.5},
	}, []string{"path", "method"})
)
