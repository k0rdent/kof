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

	flag.StringVar(&httpServerPort, "http-server-port", "9091", "The port for the ACL server.")
	flag.StringVar(&issuer, "issuer", "https://dex.example.com:32000", "The OIDC issuer URL.")
	flag.StringVar(&clientId, "client-id", "grafana-id", "The OIDC client ID.")
	flag.StringVar(&promxyHost, "promxy-host", "kof-mothership-promxy:8082", "The Promxy host.")
	flag.DurationVar(
		&shutdownTimeout,
		"shutdown-timeout",
		defaultShutdownTimeout,
		"The shutdown timeout for the http server.",
	)
	flag.BoolVar(
		&developmentMode,
		"development-mode",
		true,
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

	handlers.DevMode = developmentMode
	handlers.PromxyHost = promxyHost

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

	httpServer.Router.GET("/api/v1/query_exemplars/*", handlers.HandleQueryWithTenant)
	httpServer.Router.GET("/api/v1/format_query/*", handlers.HandleQueryWithTenant)
	httpServer.Router.GET("/api/v1/parse_query/*", handlers.HandleQueryWithTenant)
	httpServer.Router.GET("/api/v1/query_range/*", handlers.HandleQueryWithTenant)
	httpServer.Router.GET("/api/v1/query/*", handlers.HandleQueryWithTenant)

	httpServer.Router.GET("/api/v1/series/*", handlers.HandleMatchWithTenant)
	httpServer.Router.GET("/api/v1/labels/*", handlers.HandleMatchWithTenant)
	httpServer.Router.GET("/api/v1/label/*", handlers.HandleMatchWithTenant)
	httpServer.Router.GET("/api/v1/rules/*", handlers.HandleMatchWithTenant)

	httpServer.Router.GET("/api/v1/status/*", handlers.HandleProxyBypass)

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
