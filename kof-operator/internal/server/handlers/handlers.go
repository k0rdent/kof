package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	static "github.com/k0rdent/kof/kof-operator/webapp/collector"
	corev1 "k8s.io/api/core/v1"
)

var handlerMutex sync.Mutex

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	fmt.Fprintln(res.Writer, "404 - Page not found")
}

func ReactAppHandler(res *server.Response, req *http.Request) {
	if serveStaticFile(res, req, static.ReactFS) {
		return
	}
	NotFoundHandler(res, req)
}

func PrometheusHandler(res *server.Response, req *http.Request) {
	handlerMutex.Lock()
	defer handlerMutex.Unlock()

	ctx := req.Context()
	targets := &target.PrometheusTargets{}

	kubeClient, err := k8s.NewClient()
	if err != nil {
		log.Printf("failed to create client: %v", err)
		return
	}

	cdList, err := k8s.GetClusterDeployments(ctx, kubeClient.Client)
	if err != nil {
		log.Printf("failed to get cluster deployments: %v", err)
	}

	secrets := make([]*corev1.Secret, 0, len(cdList.Items))
	for _, cd := range cdList.Items {
		secretName := fmt.Sprintf("%s-kubeconfig", cd.Name)
		secret, err := k8s.GetKubeconfigSecret(ctx, kubeClient.Client, secretName, cd.Namespace)
		if err != nil {
			log.Printf("failed to get secret: %v", err)
			return
		}

		secrets = append(secrets, secret)
	}

	localTargets, err := k8s.CollectPrometheusTargets(ctx, kubeClient)
	if err != nil {
		log.Println("failed to collect prometheus target: ", err)
	}

	targets.Merge(localTargets)

	kubeconfigs := k8s.GetKubeconfigFromSecretList(secrets)

	for _, kubeconfig := range kubeconfigs {
		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			log.Println("failed to create client:", err)
			continue
		}

		newTargets, err := k8s.CollectPrometheusTargets(ctx, client)
		if err != nil {
			log.Println("failed to collect prometheus target: ", err)
			continue
		}

		targets.Merge(newTargets)
	}

	jsonResponse, err := json.Marshal(targets)
	if err != nil {
		log.Printf("failed to general response: %v", err)
	}

	res.Writer.Header().Set("Content-Type", "application/json")
	res.SetStatus(http.StatusOK)
	fmt.Fprintln(res.Writer, string(jsonResponse))
}

func serveStaticFile(res *server.Response, req *http.Request, staticFS fs.FS) bool {
	filePath := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
	if filePath == "" {
		filePath = "index.html"
	}

	file, err := staticFS.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		return false
	}

	contentType := getContentType(filePath)
	res.Writer.Header().Set("Content-Type", contentType)

	http.ServeContent(res.Writer, req, filePath, stat.ModTime(), file.(io.ReadSeeker))
	return true
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "text/plain"
	}
}
