package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var Registry = prometheus.NewRegistry()

var (
	SnapshotsTotal *prometheus.CounterVec
	BatchDuration  *prometheus.HistogramVec
	ErrorsTotal    *prometheus.CounterVec
)

func Init() {
	SnapshotsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "energy",
			Name:      "snapshots_total",
		},
		[]string{"device_id"},
	)
	BatchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mentor",
			Subsystem: "energy",
			Name:      "batch_duration_seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"device_id"},
	)
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "energy",
			Name:      "errors_total",
		},
		[]string{"cause"},
	)

	Registry.MustRegister(SnapshotsTotal, BatchDuration, ErrorsTotal)
}
