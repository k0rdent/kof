/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller"
	"github.com/k0rdent/kof/kof-operator/internal/controller/istio/cert"
	remotesecret "github.com/k0rdent/kof/kof-operator/internal/controller/istio/remote-secret"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"

	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"

	// +kubebuilder:scaffold:imports
	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var (
	scheme        = runtime.NewScheme()
	setupLog      = ctrl.Log.WithName("setup")
	shutdownLog   = ctrl.Log.WithName("shutdown")
	httpServerLog = ctrl.Log.WithName("http-server")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(grafanav1beta1.AddToScheme(scheme))
	utilruntime.Must(kofv1beta1.AddToScheme(scheme))
	utilruntime.Must(kcmv1beta1.AddToScheme(scheme))
	utilruntime.Must(cmv1.AddToScheme(scheme))
	utilruntime.Must(sveltosv1beta1.AddToScheme(scheme))
	utilruntime.Must(promv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var runController bool
	var remoteWriteUrl string
	var promxyReloadEnpoint string
	var enableServerCORS bool
	var httpServerAddr string
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&httpServerAddr, "http-server-address", ":9090", "The address the http server endpoint binds to.")
	flag.StringVar(
		&k8s.PrometheusReceiverPort,
		"prometheus-receiver-port",
		":9090",
		"Port to query Prometheus receiver server",
	)
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(
		&remoteWriteUrl,
		"remote-write-url",
		"http://vminsert-cluster:8480/insert/0/prometheus/api/v1/write",
		"The promxy remote_write_url address",
	)
	flag.StringVar(
		&promxyReloadEnpoint,
		"promxy-reload-endpoint",
		"http://localhost:8082/-/reload",
		"The promxy config reload endpoint",
	)
	flag.BoolVar(&enableServerCORS, "enable-cors", true, "Enable CORS for local development (allows all origins)")
	flag.BoolVar(&runController, "run-controller", true, "Run controller manager")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	if endpoint, ok := os.LookupEnv("PROMXY_RELOAD_ENDPOINT"); ok {
		promxyReloadEnpoint = endpoint
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	httpServer := server.NewServer(httpServerAddr, &httpServerLog)
	httpServer.Use(server.RecoveryMiddleware)
	httpServer.Use(server.LoggingMiddleware)

	if enableServerCORS {
		httpServer.Use(server.CORSMiddleware(nil))
	}

	httpServer.Router.GET("/*", handlers.ReactAppHandler)
	httpServer.Router.GET("/assets/*", handlers.ReactAppHandler)
	httpServer.Router.GET("/api/targets", handlers.PrometheusHandler)
	httpServer.Router.NotFound(handlers.NotFoundHandler)
	setupLog.Info(fmt.Sprintf("Starting http server on %s", httpServerAddr))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Run(); err != nil {
			if err != http.ErrServerClosed {
				setupLog.Error(err, "Error starting http server")
				os.Exit(1)
			}
		}
	}()

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		// TODO(user): TLSOpts is used to allow configuring the TLS config used for the server. If certificates are
		// not provided, self-signed certificates will be generated by default. This option is not recommended for
		// production environments as self-signed certificates do not offer the same level of trust and security
		// as certificates issued by a trusted Certificate Authority (CA). The primary risk is potentially allowing
		// unauthorized access to sensitive metrics data. Consider replacing with CertDir, CertName, and KeyName
		// to provide certificates, ensuring the server communicates using trusted and secure certificates.
		TLSOpts: tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "37260c18.k0rdent.mirantis.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	record.InitFromRecorder(mgr.GetEventRecorderFor("kof-operator"))

	if err = (&controller.PromxyServerGroupReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		RemoteWriteUrl:     remoteWriteUrl,
		PromxyConfigReload: func() error { return controller.ReloadPromxyConfig(promxyReloadEnpoint) },
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PromxyServerGroup")
		os.Exit(1)
	}
	if err = (&controller.ClusterDeploymentReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		IstioCertManager:    cert.New(mgr.GetClient()),
		RemoteSecretManager: remotesecret.New(mgr.GetClient()),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterDeployment")
		os.Exit(1)
	}
	if err = (&controller.ConfigMapReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigMap")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if runController {
		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	} else {
		wg.Wait()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		shutdownLog.Error(err, "Http server forced to shutdown")
		os.Exit(1)
	}
	wg.Wait()
}
