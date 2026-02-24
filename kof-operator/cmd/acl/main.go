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
	var vlogxyHost string
	var vlogxyScheme string
	var adminEmail string

	flag.StringVar(&httpServerPort, "http-server-port", "9091", "The port for the ACL server.")
	flag.StringVar(&adminEmail, "admin-email", "", "The email address of the admin user.")
	flag.StringVar(&issuer, "issuer", "https://dex.example.com:32000", "The OIDC issuer URL.")
	flag.StringVar(&clientId, "client-id", "grafana-id", "The OIDC client ID.")
	flag.StringVar(&promxyHost, "promxy-host", "kof-mothership-promxy:8082", "The Promxy host.")
	flag.StringVar(&promxyScheme, "promxy-scheme", "http", "The scheme to use when connecting to Promxy (http or https).")
	flag.StringVar(&vlogxyHost, "vlogxy-host", "kof-mothership-vlogxy:8085", "The Vlogxy host.")
	flag.StringVar(&vlogxyScheme, "vlogxy-scheme", "http", "The scheme to use when connecting to Vlogxy (http or https).")
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

	promxyHandler := handlers.NewHandler(handlers.Config{
		Host:       promxyHost,
		Scheme:     promxyScheme,
		DevMode:    developmentMode,
		AdminEmail: adminEmail,
	})

	vlogxyHandler := handlers.NewVlogxyHandler(handlers.Config{
		Host:       vlogxyHost,
		Scheme:     vlogxyScheme,
		DevMode:    developmentMode,
		AdminEmail: adminEmail,
	})

	httpServer := server.NewServer(fmt.Sprintf(":%s", httpServerPort), &serverLog)
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

	httpServer.Router.GET("/api/v1/query_exemplars/*", promxyHandler.ProxyQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/format_query/*", promxyHandler.ProxyQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/parse_query/*", promxyHandler.ProxyQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/query_range/*", promxyHandler.ProxyQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/query/*", promxyHandler.ProxyQueryWithTenantInjection)

	httpServer.Router.GET("/api/v1/series/*", promxyHandler.ProxyMatchQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/labels/*", promxyHandler.ProxyMatchQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/label/*", promxyHandler.ProxyMatchQueryWithTenantInjection)
	httpServer.Router.GET("/api/v1/rules/*", promxyHandler.ProxyMatchQueryWithTenantInjection)

	httpServer.Router.GET("/api/v1/status/*", promxyHandler.HandleProxyBypass)

	httpServer.Router.GET("/vlogxy/*", vlogxyHandler.ProxyLogsWithTenantInjection)
	httpServer.Router.POST("/vlogxy/*", vlogxyHandler.ProxyLogsWithTenantInjection)

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
