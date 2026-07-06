package filewatcher

import "github.com/prometheus/client_golang/prometheus"

const metricsNamespace = "file_watcher"

type fileWatcherMetrics struct {
	eventsTotal  *prometheus.CounterVec
	watchedPaths prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *fileWatcherMetrics {
	m := &fileWatcherMetrics{
		eventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "events_total",
				Help:      "Total number of file system events observed, by path and event type (modified or deleted).",
			},
			[]string{"path", "event"},
		),
		watchedPaths: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "watched_paths",
			Help:      "Number of filesystem paths registered with watcher.",
		}),
	}
	reg.MustRegister(m.eventsTotal, m.watchedPaths)
	return m
}
