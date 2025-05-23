package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	static "github.com/k0rdent/kof/kof-operator/webapp/collector"
)

const MothershipClusterName = "mothership"

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	_, err := fmt.Fprintln(res.Writer, "404 - Page not found")
	if err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}

func ReactAppHandler(res *server.Response, req *http.Request) {
	if serveStaticFile(res, req, static.ReactFS) {
		return
	}
	NotFoundHandler(res, req)
}

func PrometheusHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()
	targets := &target.PrometheusTargets{}

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create client")
		return
	}

	cdList, err := k8s.GetClusterDeployments(ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get cluster deployments")
	}

	clusters := make([]*k8s.Cluster, 0, len(cdList.Items))
	for _, cd := range cdList.Items {
		secret, err := k8s.GetKubeconfigSecret(ctx, kubeClient.Client, cd.Name, cd.Namespace)
		if err != nil {
			res.Logger.Error(err, "Failed to get secret", "clusterName", cd.Name)
			continue
		}

		clusters = append(clusters, &k8s.Cluster{
			Name:   cd.Name,
			Secret: secret,
		})
	}

	localTargets, err := k8s.CollectPrometheusTargets(ctx, kubeClient, MothershipClusterName)
	if err != nil {
		res.Logger.Error(err, "Failed to collect the Prometheus target from the mothership")
	}

	targets.Merge(localTargets)

	for _, cluster := range clusters {
		client, err := k8s.NewKubeClientFromKubeconfig(cluster.GetKubeconfig())
		if err != nil {
			res.Logger.Error(err, "Failed to create client", "clusterName", cluster.Name)
			continue
		}

		newTargets, err := k8s.CollectPrometheusTargets(ctx, client, cluster.Name)
		if err != nil {
			res.Logger.Error(err, "Failed to collect prometheus target", "clusterName", cluster.Name)
			continue
		}

		targets.Merge(newTargets)
	}

	jsonResponse, err := json.Marshal(targets)
	if err != nil {
		res.Logger.Error(err, "Failed to general response")
	}

	res.Writer.Header().Set("Content-Type", "application/json")
	res.SetStatus(http.StatusOK)

	_, err = fmt.Fprintln(res.Writer, string(jsonResponse))
	if err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
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
	defer func() {
		err := file.Close()
		if err != nil {
			res.Logger.Error(err, "Cannot close file", "path", filePath)
		}
	}()

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
