package filewatcher

import "github.com/prometheus/client_golang/prometheus"

const (
	metricsNamespace = "file_watcher"

	deletedEvent  = "deleted"
	modifiedEvent = "modified"
)

type fileWatcherMetrics struct {
	driftDetected *prometheus.GaugeVec
	watchedPaths  prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *fileWatcherMetrics {
	m := &fileWatcherMetrics{
		driftDetected: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "drift_detected",
				Help:      "1 if the file at the given path has drifted from baseline (modified or deleted), 0 otherwise.",
			},
			[]string{"path", "event"},
		),
		watchedPaths: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "watched_paths",
			Help:      "Number of filesystem paths registered with watcher.",
		}),
	}
	reg.MustRegister(m.driftDetected, m.watchedPaths)
	return m
}

// setDriftDetected sets the file_watcher_path_changed gauge for the given path and event type to 1 if changed is true, or 0 otherwise.
func (m *fileWatcherMetrics) setDriftDetected(path string, event string, changed bool) {
	if changed {
		m.driftDetected.WithLabelValues(path, event).Set(1)
	} else {
		m.driftDetected.WithLabelValues(path, event).Set(0)
	}
}

// incWatchedPaths increments the watchedPaths gauge by 1.
func (m *fileWatcherMetrics) incWatchedPaths() {
	m.watchedPaths.Inc()
}
