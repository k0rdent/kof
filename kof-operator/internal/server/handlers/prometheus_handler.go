package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

const (
	AdoptedClusterSecretSuffix = "kubeconf"
	ClusterSecretSuffix        = "kubeconfig"
	MothershipClusterName      = "mothership"
)

type PrometheusTargetHandler struct {
	targets    *target.PrometheusTargets
	kubeClient *k8s.KubeClient
	logger     *logr.Logger
}

func newPrometheusTargetHandler(res *server.Response, req *http.Request) (*PrometheusTargetHandler, error) {
	kubeClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	return &PrometheusTargetHandler{
		targets:    &target.PrometheusTargets{},
		kubeClient: kubeClient,
		logger:     res.Logger,
	}, nil
}

func PrometheusHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	h, err := newPrometheusTargetHandler(res, req)
	if err != nil {
		res.Logger.Error(err, "Failed to create prometheus handler")
		internalError(res, BasicInternalErrorMessage)
		return
	}

	if err := h.collectClusterDeploymentsTargets(ctx); err != nil {
		res.Logger.Error(err, "Failed to get cluster deployment")
	}

	if err := h.collectLocalTargets(ctx); err != nil {
		res.Logger.Error(err, fmt.Sprintf("Failed to collect the Prometheus target from the %s", MothershipClusterName))
	}

	sendResponse(res, h.targets)
}

func (h *PrometheusTargetHandler) collectClusterDeploymentsTargets(ctx context.Context) error {
	cdList, err := k8s.GetClusterDeployments(ctx, h.kubeClient.Client)
	if err != nil {
		return err
	}

	if len(cdList.Items) == 0 {
		h.logger.Info("Cluster deployments not found")
		return nil
	}

	clusters := make([]*k8s.Cluster, 0, len(cdList.Items))
	for _, cd := range cdList.Items {
		var secretName string

		if strings.Contains(cd.Spec.Template, "adopted") {
			secretName = fmt.Sprintf("%s-%s", cd.Name, AdoptedClusterSecretSuffix)
		} else {
			secretName = fmt.Sprintf("%s-%s", cd.Name, ClusterSecretSuffix)
		}

		secret, err := k8s.GetSecret(ctx, h.kubeClient.Client, secretName, cd.Namespace)
		if err != nil {
			h.logger.Error(err, "Failed to get secret", "clusterName", cd.Name)
			continue
		}

		clusters = append(clusters, &k8s.Cluster{
			Name:   cd.Name,
			Secret: secret,
		})
	}

	for _, cluster := range clusters {
		client, err := k8s.NewKubeClientFromKubeconfig(cluster.GetKubeconfig())
		if err != nil {
			h.logger.Error(err, "Failed to create client", "clusterName", cluster.Name)
			continue
		}

		newTargets, err := k8s.CollectPrometheusTargets(ctx, h.logger, client, cluster.Name)
		if err != nil {
			h.logger.Error(err, "Failed to collect prometheus target", "clusterName", cluster.Name)
			continue
		}

		h.targets.Merge(newTargets)
	}

	return nil
}

func (h *PrometheusTargetHandler) collectLocalTargets(ctx context.Context) error {
	localTargets, err := k8s.CollectPrometheusTargets(ctx, h.logger, h.kubeClient, MothershipClusterName)
	if err != nil {
		return err
	}

	h.targets.Merge(localTargets)
	return nil
}
