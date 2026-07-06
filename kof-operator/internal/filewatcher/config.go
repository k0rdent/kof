package filewatcher

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// pathsFlag implements flag.Value for a repeatable --watch-path flag.
type pathsFlag []string

func (p *pathsFlag) String() string {
	return strings.Join(*p, ", ")
}

func (p *pathsFlag) Set(v string) error {
	*p = append(*p, v)
	return nil
}

// Config holds all file-watcher configuration.
type Config struct {
	// WatchPaths is the list of filesystem paths to watch.
	// Set via --watch-path (repeatable).
	WatchPaths []string

	// Recursive controls whether subdirectories are watched recursively.
	// Set via --recursive (default: true).
	Recursive bool

	// MetricsAddr is the address on which the Prometheus metrics HTTP server listens.
	// Set via --metrics-addr (default: ":9090").
	MetricsAddr string

	// DebounceDuration is the minimum interval between two events on the same path
	// before a new metric/log is emitted. Rapid bursts within this window are
	// collapsed into a single event.
	// Set via --debounce (default: 100ms).
	DebounceDuration time.Duration

	// BaselineSecretNamespace is the namespace of the baseline Secret.
	// Defaults to the value of the POD_NAMESPACE environment variable.
	// Set via --baseline-namespace (default: "kof").
	BaselineSecretNamespace string

	// BaselineSecretName is the name of the baseline Secret.
	// Set via --baseline-secret-name (default: "file-watcher-baseline").
	BaselineSecretName string

	// BaselineEnabled controls whether baseline persistence is enabled.
	// Set via --baseline-enabled (default: false).
	BaselineEnabled bool
}

// ParseFlags registers CLI flags on flag.CommandLine, calls flag.Parse(),
// validates the result, and returns the populated Config. Callers should
// register any additional flags (e.g. zap flags) before invoking this function.
func ParseFlags() (*Config, error) {
	return parseFrom(flag.CommandLine, os.Args[1:])
}

// parseFrom is the testable core of ParseFlags. It registers flags on fs,
// parses args, and returns the validated Config.
func parseFrom(fs *flag.FlagSet, args []string) (*Config, error) {
	var paths pathsFlag
	var recursive bool
	var metricsAddr string
	var debounceStr string
	var baselineSecretName string
	var baselineNamespace string
	var baselineEnabled bool

	fs.Var(&paths, "watch-path", "Filesystem `path` to watch (repeatable, at least one required).")
	fs.BoolVar(&recursive, "recursive", true, "Watch subdirectories recursively.")
	fs.StringVar(&metricsAddr, "metrics-addr", ":9090", "`address` for the Prometheus /metrics HTTP endpoint.")
	fs.StringVar(&debounceStr, "debounce", "100ms", "Minimum `duration` between two events on the same path (Go duration string).")
	fs.BoolVar(&baselineEnabled, "baseline-enabled", false, "Enable baseline persistence.")
	fs.StringVar(&baselineSecretName, "baseline-secret-name", "file-watcher-baseline", "Name of the baseline Secret.")
	fs.StringVar(&baselineNamespace, "baseline-namespace", "kof", "Namespace for the baseline Secret (defaults to $POD_NAMESPACE; empty disables baseline persistence).")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one --watch-path is required")
	}

	debounce, err := time.ParseDuration(debounceStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --debounce %q: %w", debounceStr, err)
	}

	return &Config{
		WatchPaths:              []string(paths),
		Recursive:               recursive,
		MetricsAddr:             metricsAddr,
		DebounceDuration:        debounce,
		BaselineSecretNamespace: baselineNamespace,
		BaselineSecretName:      baselineSecretName,
		BaselineEnabled:         baselineEnabled,
	}, nil
}
