package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Registry is the dedicated Prometheus registry for this service.
var Registry = prometheus.NewRegistry()

var (
	OEERecordsTotal     *prometheus.CounterVec
	OEEBatchDuration    *prometheus.HistogramVec
	StopsTotal          *prometheus.CounterVec
	ProductionRunsTotal *prometheus.CounterVec
	IngestErrorsTotal   *prometheus.CounterVec
)

func init() {
	OEERecordsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "ingest",
			Name:      "oee_records_total",
			Help:      "Total OEE records processed.",
		},
		[]string{"device_id"},
	)
	OEEBatchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mentor",
			Subsystem: "ingest",
			Name:      "oee_batch_duration_seconds",
			Help:      "OEE batch processing latency.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"device_id"},
	)
	StopsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "ingest",
			Name:      "stops_total",
			Help:      "Total stop records processed.",
		},
		[]string{"device_id"},
	)
	ProductionRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "ingest",
			Name:      "production_runs_total",
			Help:      "Total production run records processed.",
		},
		[]string{"device_id"},
	)
	IngestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mentor",
			Subsystem: "ingest",
			Name:      "errors_total",
			Help:      "Total ingest errors by endpoint.",
		},
		[]string{"endpoint"},
	)

	Registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		OEERecordsTotal,
		OEEBatchDuration,
		StopsTotal,
		ProductionRunsTotal,
		IngestErrorsTotal,
	)
}

// Init ensures the package init() has run.
func Init() {}
