package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

const (
	MothershipClusterName = "mothership"
)

type PrometheusTargets struct {
	targets    *target.Targets
	kubeClient *k8s.KubeClient
	logger     *logr.Logger
}

func newPrometheusTargets(res *server.Response) (*PrometheusTargets, error) {
	kubeClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	return &PrometheusTargets{
		targets:    &target.Targets{Clusters: make(target.Clusters)},
		kubeClient: kubeClient,
		logger:     res.Logger,
	}, nil
}

func PrometheusHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	h, err := newPrometheusTargets(res)
	if err != nil {
		res.Logger.Error(err, "Failed to create prometheus handler")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	if err := h.collectClusterDeploymentsTargets(ctx); err != nil {
		res.Logger.Error(err, "Failed to get cluster deployment")
	}

	if err := h.collectLocalTargets(ctx); err != nil {
		res.Logger.Error(err, fmt.Sprintf("Failed to collect the Prometheus target from the %s", MothershipClusterName))
	}

	res.Send(h.targets, http.StatusOK)
}

func (h *PrometheusTargets) collectClusterDeploymentsTargets(ctx context.Context) error {
	cdList, err := k8s.GetClusterDeployments(ctx, h.kubeClient.Client)
	if err != nil {
		return err
	}

	if len(cdList.Items) == 0 {
		h.logger.Info("Cluster deployments not found")
		return nil
	}

	for _, cd := range cdList.Items {
		secretName := k8s.GetSecretName(&cd)
		secret, err := k8s.GetSecret(ctx, h.kubeClient.Client, secretName, cd.Namespace)
		if err != nil {
			h.logger.Error(err, "Failed to get secret", "clusterName", cd.Name)
			continue
		}

		kubeconfig := k8s.GetSecretValue(secret)
		if kubeconfig == nil {
			h.logger.Error(fmt.Errorf("no value"), "failed to get secret value")
			continue
		}

		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			h.logger.Error(err, "Failed to create client", "clusterName", cd.Name)
			continue
		}

		newTargets, err := k8s.CollectPrometheusTargets(ctx, h.logger, client, cd.Name)
		if err != nil {
			h.logger.Error(err, "Failed to collect prometheus target", "clusterName", cd.Name)
			continue
		}

		h.targets.Merge(newTargets)
	}

	return nil
}

func (h *PrometheusTargets) collectLocalTargets(ctx context.Context) error {
	localTargets, err := k8s.CollectPrometheusTargets(ctx, h.logger, h.kubeClient, MothershipClusterName)
	if err != nil {
		return err
	}

	h.targets.Merge(localTargets)
	return nil
}
