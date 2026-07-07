package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/k0rdent/kof/kof-operator/internal/filewatcher"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
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

	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Error(err, "unable to create kube client")
		os.Exit(1)
	}

	if cfg.BaselineEnabled {
		store := filewatcher.NewSecretBaselineStore(k8sClient.Client, cfg.BaselineSecretName, cfg.BaselineSecretNamespace)
		watcher.WithBaselineStore(store)
		log.Info("baseline persistence enabled", "secret", cfg.BaselineSecretName, "namespace", cfg.BaselineSecretNamespace)
	}

	metricsHandler := promhttp.Handler()
	srv := server.NewServer(cfg.MetricsAddr, &log)
	srv.Router.GET("/metrics", func(res *server.Response, r *http.Request) {
		metricsHandler.ServeHTTP(res.Writer, r)
	})

	go func() {
		log.Info("starting metrics server", "addr", cfg.MetricsAddr)
		if err := srv.Run(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "metrics server failed")
		}
	}()

	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			log.Error(err, "failed to shutdown metrics server")
		}
	}()

	if err := watcher.Run(ctx); err != nil {
		log.Error(err, "watcher failed")
		os.Exit(1)
	}
}
