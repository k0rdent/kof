package filewatcher

import (
	"flag"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// newFS returns a FlagSet that returns errors instead of calling os.Exit,
// with usage output suppressed.
func newFS() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Usage = func() {}
	return fs
}

var _ = Describe("parseFrom", func() {
	It("returns a Config populated from the provided args", func() {
		cfg, err := parseFrom(newFS(), []string{
			"--watch-path", "/tmp/a",
			"--watch-path", "/tmp/b",
			"--recursive=false",
			"--metrics-addr", ":8080",
			"--debounce", "200ms",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.WatchPaths).To(ConsistOf("/tmp/a", "/tmp/b"))
		Expect(cfg.Recursive).To(BeFalse())
		Expect(cfg.MetricsAddr).To(Equal(":8080"))
		Expect(cfg.DebounceDuration).To(Equal(200 * time.Millisecond))
	})

	It("applies defaults when optional flags are omitted", func() {
		cfg, err := parseFrom(newFS(), []string{"--watch-path", "/tmp/x"})
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Recursive).To(BeTrue())
		Expect(cfg.MetricsAddr).To(Equal(":9090"))
		Expect(cfg.DebounceDuration).To(Equal(100 * time.Millisecond))
	})

	It("returns an error when no --watch-path is supplied", func() {
		_, err := parseFrom(newFS(), []string{"--recursive=false"})
		Expect(err).To(MatchError(ContainSubstring("--watch-path is required")))
	})

	It("returns an error for an invalid --debounce value", func() {
		_, err := parseFrom(newFS(), []string{
			"--watch-path", "/tmp/x",
			"--debounce", "not-a-duration",
		})
		Expect(err).To(MatchError(ContainSubstring("invalid --debounce")))
	})

	It("returns an error for an unknown flag", func() {
		_, err := parseFrom(newFS(), []string{"--unknown-flag", "value"})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("pathsFlag", func() {
	It("Set appends each value", func() {
		var p pathsFlag
		Expect(p.Set("/a")).To(Succeed())
		Expect(p.Set("/b")).To(Succeed())
		Expect([]string(p)).To(Equal([]string{"/a", "/b"}))
	})

	It("String returns a comma-separated list", func() {
		p := pathsFlag{"/a", "/b", "/c"}
		Expect(p.String()).To(Equal("/a, /b, /c"))
	})

	It("String on an empty flag returns an empty string", func() {
		var p pathsFlag
		Expect(p.String()).To(BeEmpty())
	})
})
