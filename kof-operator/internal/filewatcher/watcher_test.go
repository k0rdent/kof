package filewatcher

import (
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestFileWatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FileWatcher Suite")
}

// newTestWatcher builds a Watcher with a fresh registry and a near-zero
// debounce so unit tests are not sensitive to timing.
func newTestWatcher(cfg *Config) (*Watcher, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	w, err := newWatcherWithRegistry(cfg, logr.Discard(), reg)
	Expect(err).NotTo(HaveOccurred())
	return w, reg
}

// gatherGauge reads the file_watcher_drift_detected gauge value from reg with the given labels.
func gatherGauge(reg *prometheus.Registry, labels map[string]string) float64 {
	mfs, err := reg.Gather()
	if err != nil {
		return 0
	}
	for _, mf := range mfs {
		if mf.GetName() != "file_watcher_drift_detected" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsMatch(m.GetLabel(), labels) {
				if g := m.GetGauge(); g != nil {
					return g.GetValue()
				}
			}
		}
	}
	return 0
}

func labelsMatch(got []*dto.LabelPair, want map[string]string) bool {
	matched := 0
	for _, lp := range got {
		if v, ok := want[lp.GetName()]; ok && v == lp.GetValue() {
			matched++
		}
	}
	return matched == len(want)
}

var _ = Describe("Watcher debounce tests", func() {
	var w *Watcher

	BeforeEach(func() {
		w, _ = newTestWatcher(&Config{
			WatchPaths:       []string{"."},
			DebounceDuration: 100 * time.Millisecond,
		})
	})

	AfterEach(func() {
		err := w.fw.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	It("passes the first event for a path", func() {
		Expect(w.debounce("/tmp/test.txt")).To(BeTrue())
	})

	It("suppresses a second event within the debounce window", func() {
		w.debounce("/tmp/test.txt")
		Expect(w.debounce("/tmp/test.txt")).To(BeFalse())
	})

	It("allows an event after the debounce window has elapsed", func() {
		w.cfg.DebounceDuration = 1 * time.Millisecond
		w.debounce("/tmp/test.txt")
		time.Sleep(5 * time.Millisecond)
		Expect(w.debounce("/tmp/test.txt")).To(BeTrue())
	})

	It("tracks different paths independently", func() {
		Expect(w.debounce("/tmp/a.txt")).To(BeTrue())
		Expect(w.debounce("/tmp/b.txt")).To(BeTrue())
	})
})

var _ = Describe("Watcher handleEvent tests", func() {
	var (
		w   *Watcher
		reg *prometheus.Registry
	)

	BeforeEach(func() {
		w, reg = newTestWatcher(&Config{
			WatchPaths:       []string{"."},
			Recursive:        false,
			DebounceDuration: 0,
		})
	})

	AfterEach(func() {
		err := w.fw.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("sets path_modified gauge to 1",
		func(op fsnotify.Op, expectedEvent string) {
			path := "/nonexistent/unit-test-path"
			w.handleEvent(fsnotify.Event{Name: path, Op: op})
			Expect(testutil.ToFloat64(
				w.metrics.driftDetected.WithLabelValues(path, expectedEvent),
			)).To(Equal(1.0))
		},
		Entry("Write → modified", fsnotify.Write, "modified"),
		Entry("Create → modified", fsnotify.Create, "modified"),
		Entry("Remove → deleted", fsnotify.Remove, "deleted"),
		Entry("Rename → deleted", fsnotify.Rename, "deleted"),
	)

	It("ignores Chmod events", func() {
		path := "/nonexistent/chmod-path"
		w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Chmod})

		// Verify neither label was set.
		Expect(gatherGauge(reg, map[string]string{
			"path": path, "event": "modified",
		})).To(Equal(0.0))
		Expect(gatherGauge(reg, map[string]string{
			"path": path, "event": "deleted",
		})).To(Equal(0.0))
	})

	It("does not set the gauge for a debounced event", func() {
		w.cfg.DebounceDuration = time.Hour
		path := "/nonexistent/debounce-test"

		w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Write})
		w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Write})

		Expect(testutil.ToFloat64(
			w.metrics.driftDetected.WithLabelValues(path, "modified"),
		)).To(Equal(1.0))
	})
})
