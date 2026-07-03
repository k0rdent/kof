package filewatcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
)

// Watcher watches configured filesystem paths and emits structured logs and
// Prometheus metrics for every file modification or deletion.
type Watcher struct {
	cfg     *Config
	fw      *fsnotify.Watcher
	log     logr.Logger
	mu      sync.Mutex
	last    map[string]time.Time // debounce: path → last event emission time
	metrics *fileWatcherMetrics
}

// NewWatcher constructs and initialises a Watcher using the default Prometheus registry.
func NewWatcher(cfg *Config, log logr.Logger) (*Watcher, error) {
	return newWatcherWithRegistry(cfg, log, prometheus.DefaultRegisterer)
}

// newWatcherWithRegistry constructs a Watcher that registers its metrics with reg.
// Used in tests to avoid polluting the default registry.
func newWatcherWithRegistry(cfg *Config, log logr.Logger, reg prometheus.Registerer) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}
	return &Watcher{
		cfg:     cfg,
		fw:      fw,
		log:     log,
		last:    make(map[string]time.Time),
		metrics: newMetrics(reg),
	}, nil
}

// Start registers all configured paths with the underlying watcher and begins
// processing events. It blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	defer w.fw.Close() //nolint:errcheck

	for _, root := range w.cfg.WatchPaths {
		if err := w.addPath(root); err != nil {
			return fmt.Errorf("failed to add watch path %q: %w", root, err)
		}
	}

	return w.loop(ctx)
}

// addPath registers a single path (and, when Recursive is true, its full
// directory sub-tree) with the underlying fsnotify watcher.
func (w *Watcher) addPath(root string) error {
	if !w.cfg.Recursive {
		if err := w.fw.Add(root); err != nil {
			return err
		}
		w.metrics.watchedPaths.Inc()
		w.log.Info("watching path", "path", root)
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			w.log.Error(err, "walk error, skipping path", "path", path)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if addErr := w.fw.Add(path); addErr != nil {
			w.log.Error(addErr, "failed to watch directory", "path", path)
			return nil
		}
		w.metrics.watchedPaths.Inc()
		w.log.Info("watching directory", "path", path)
		return nil
	})
}

// loop reads events and errors from the fsnotify watcher until ctx is cancelled.
func (w *Watcher) loop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			w.log.Info("context cancelled, shutting down watcher")
			return nil

		case event, ok := <-w.fw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.fw.Errors:
			if !ok {
				return nil
			}
			w.log.Error(err, "fsnotify watcher error")
		}
	}
}

// handleEvent processes a single fsnotify event, emitting a log line and
// incrementing the appropriate Prometheus counter.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	var eventType string
	switch {
	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		eventType = "deleted"
	case event.Has(fsnotify.Write) || event.Has(fsnotify.Create):
		eventType = "modified"
	default:
		// Chmod or other events are ignored.
		return
	}

	if !w.debounce(event.Name) {
		return
	}

	w.log.Info("file event detected",
		"event", eventType,
		"path", event.Name,
	)
	w.metrics.eventsTotal.WithLabelValues(event.Name, eventType).Inc()

	// When a new directory appears and recursive mode is enabled, register it
	// so that files created within it are also tracked.
	if w.cfg.Recursive && event.Has(fsnotify.Create) {
		if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
			if addErr := w.fw.Add(event.Name); addErr != nil {
				w.log.Error(addErr, "failed to watch new directory", "path", event.Name)
			} else {
				w.metrics.watchedPaths.Inc()
				w.log.Info("watching new directory", "path", event.Name)
			}
		}
	}
}

// debounce returns true only when the given path has not been seen within
// cfg.DebounceDuration. Rapid bursts of events on the same path are collapsed
// into a single emission.
func (w *Watcher) debounce(path string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	if last, ok := w.last[path]; ok && now.Sub(last) < w.cfg.DebounceDuration {
		return false
	}
	w.last[path] = now
	return true
}
