package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/acl/handlers"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	srvhandlers "github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	"github.com/k0rdent/kof/kof-operator/internal/telemetry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	serverLog = ctrl.Log.WithName("acl-server")
)

const (
	defaultShutdownTimeout = 5 * time.Second
)

func main() {
	var shutdownTimeout time.Duration
	var enableServerCORS bool
	var developmentMode bool
	var httpServerPort string
	var issuer string
	var clientId string
	var promxyHost string
	var promxyScheme string
	var logsHost string
	var logsScheme string
	var adminEmail string
	var tracesHost string
	var tracesScheme string

	flag.StringVar(&httpServerPort, "http-server-port", "9091", "The port for the ACL server.")
	flag.StringVar(&adminEmail, "admin-email", "", "The email address of the admin user.")
	flag.StringVar(&issuer, "issuer", "https://dex.example.com", "The OIDC issuer URL.")
	flag.StringVar(&clientId, "client-id", "grafana-id", "The OIDC client ID.")
	flag.StringVar(&promxyHost, "promxy-host", "kof-mothership-promxy:8082", "The Promxy host.")
	flag.StringVar(&promxyScheme, "promxy-scheme", "http", "The scheme to use when connecting to Promxy (http or https).")
	flag.StringVar(&logsHost, "logs-host", "vlselect-kof-mothership-logs-multilevel-select.kof.svc:9471", "The Logs host.")
	flag.StringVar(&logsScheme, "logs-scheme", "http", "The scheme to use when connecting to Logs (http or https).")
	flag.StringVar(
		&tracesHost,
		"traces-host",
		"vtselect-kof-mothership-multilevel-select.kof.svc:10471",
		"The Traces backend host.",
	)
	flag.StringVar(&tracesScheme, "traces-scheme", "http", "The scheme to use when connecting to Traces (http or https).")
	flag.DurationVar(
		&shutdownTimeout,
		"shutdown-timeout",
		defaultShutdownTimeout,
		"The shutdown timeout for the http server.",
	)
	flag.BoolVar(
		&developmentMode,
		"development-mode",
		false,
		"Allow requests without authentication and tenant injection for testing purposes.",
	)

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	shutdownTelemetry, err := telemetry.Setup(context.Background(), "kof-acl-server")
	if err != nil {
		serverLog.Error(err, "Failed to initialize telemetry")
		os.Exit(1)
	}
	defer func() {
		telCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(telCtx); err != nil {
			serverLog.Error(err, "Failed to shutdown telemetry")
		}
	}()

	oidcCtx := oidc.ClientContext(context.Background(), &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: developmentMode,
			},
		},
	})

	oidcProvider, err := oidc.NewProvider(oidcCtx, issuer)
	if err != nil {
		serverLog.Error(err, "Failed to initialize OIDC provider")
		os.Exit(1)
	}

	promxyConfig := handlers.Config{
		Host:       promxyHost,
		Scheme:     promxyScheme,
		DevMode:    developmentMode,
		AdminEmail: adminEmail,
	}

	promxyQueryHandler := handlers.NewPromxyQueryHandler(promxyConfig)
	promxyAlertsHandler := handlers.NewPromxyAlertsHandler(promxyConfig)
	promxyRulesHandler := handlers.NewPromxyRulesHandler(promxyConfig)

	logsHandler := handlers.NewLogsHandler(handlers.Config{
		Host:       logsHost,
		Scheme:     logsScheme,
		DevMode:    developmentMode,
		AdminEmail: adminEmail,
	})

	tracesConfig := handlers.Config{
		Host:       tracesHost,
		Scheme:     tracesScheme,
		DevMode:    developmentMode,
		AdminEmail: adminEmail,
	}

	jaegerAPITraceHandler := handlers.NewJaegerTraceHandler(tracesConfig)
	jaegerAPITracesHandler := handlers.NewJaegerTracesHandler(tracesConfig)
	jaegerAPIServiceHandler := handlers.NewJaegerServicesHandler(tracesConfig)

	httpServer := server.NewServer(fmt.Sprintf(":%s", httpServerPort), &serverLog)
	httpServer.Use(server.SpanNameMiddleware)
	httpServer.Use(server.RecoveryMiddleware)
	httpServer.Use(server.LoggingMiddleware)
	httpServer.Use(server.AuthenticationMiddleware(server.AuthConfig{
		Provider:         oidcProvider,
		ClientID:         clientId,
		SkipOnEmptyToken: developmentMode,
	}))

	if enableServerCORS {
		httpServer.Use(server.CORSMiddleware(nil))
	}

	httpServer.Router.GET("/metrics/api/v1/query_exemplars/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/query_exemplars/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/format_query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/format_query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/parse_query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/parse_query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/query_range/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/query_range/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/query/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/series/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/series/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/labels/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.POST("/metrics/api/v1/labels/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/label/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/rules/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyRulesHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/alerts/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, promxyAlertsHandler)
	})

	httpServer.Router.GET("/metrics/api/v1/status/buildinfo", func(res *server.Response, req *http.Request) {
		handlers.ProxyBypass(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/status/config", func(res *server.Response, req *http.Request) {
		handlers.AdminProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/status/flags", func(res *server.Response, req *http.Request) {
		handlers.AdminProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/status/runtimeinfo", func(res *server.Response, req *http.Request) {
		handlers.AdminProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/status/tsdb", func(res *server.Response, req *http.Request) {
		handlers.AdminProxy(res, req, promxyQueryHandler)
	})
	httpServer.Router.GET("/metrics/api/v1/status/blocks", func(res *server.Response, req *http.Request) {
		handlers.AdminProxy(res, req, promxyQueryHandler)
	})

	httpServer.Router.GET("/logs/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, logsHandler)
	})
	httpServer.Router.POST("/logs/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, logsHandler)
	})

	httpServer.Router.GET("/traces/select/jaeger/api/services", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, jaegerAPIServiceHandler)
	})
	httpServer.Router.GET("/traces/select/jaeger/api/traces/*", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, jaegerAPITraceHandler)
	})
	httpServer.Router.GET("/traces/select/jaeger/api/traces", func(res *server.Response, req *http.Request) {
		handlers.ACLProxy(res, req, jaegerAPITracesHandler)
	})

	httpServer.Router.NotFound(srvhandlers.NotFoundHandler)

	serverLog.Info(fmt.Sprintf("Starting http server on :%s", httpServerPort))

	if err := httpServer.Run(); err != nil {
		if err != http.ErrServerClosed {
			serverLog.Error(err, "Error starting http server")
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		serverLog.Error(err, "Http server forced to shutdown")
		os.Exit(1)
	}
}
