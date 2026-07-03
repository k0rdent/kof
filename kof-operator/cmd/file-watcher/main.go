package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/k0rdent/kof/kof-operator/internal/filewatcher"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// Register zap flags before flag.Parse() is called inside ParseFlags.
	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := ctrl.Log.WithName("file-watcher")

	cfg, err := filewatcher.ParseFlags()
	if err != nil {
		log.Error(err, "invalid configuration")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	watcher, err := filewatcher.NewWatcher(cfg, log)
	if err != nil {
		log.Error(err, "failed to create watcher")
		os.Exit(1)
	}

	metricsHandler := promhttp.Handler()
	srv := server.NewServer(cfg.MetricsAddr, &log)
	srv.Router.GET("/metrics", func(res *server.Response, r *http.Request) {
		metricsHandler.ServeHTTP(res.Writer, r)
	})

	go func() {
		log.Info("starting metrics server", "addr", cfg.MetricsAddr)
		if serveErr := srv.Run(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Error(serveErr, "metrics server failed")
		}
	}()
	defer srv.Shutdown(ctx)

	if err := watcher.Start(ctx); err != nil {
		log.Error(err, "watcher failed")
		os.Exit(1)
	}
}
