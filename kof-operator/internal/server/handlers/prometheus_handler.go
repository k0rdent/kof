package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/utils"
	v1 "github.com/prometheus/prometheus/web/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MothershipClusterName    = "mothership"
	PrometheusEndpoint       = "api/v1/targets"
	PrometheusReceiverLabel  = "k0rdent.mirantis.com/kof-prometheus-receiver"
	PrometheusPortAnnotation = "kof.k0rdent.mirantis.com/prometheus-api-server-port"
	PrometheusPortName       = "api-server"
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
	cdList, err := k8s.GetKofClusterDeployments(ctx, h.kubeClient.Client)
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

		newTargets, err := collectPrometheusTargets(ctx, h.logger, client, cd.Name)
		if err != nil {
			h.logger.Error(err, "Failed to collect prometheus target", "clusterName", cd.Name)
			continue
		}

		h.targets.Merge(newTargets)
	}

	return nil
}

func (h *PrometheusTargets) collectLocalTargets(ctx context.Context) error {
	localTargets, err := collectPrometheusTargets(ctx, h.logger, h.kubeClient, MothershipClusterName)
	if err != nil {
		return err
	}

	h.targets.Merge(localTargets)
	return nil
}

func collectPrometheusTargets(ctx context.Context, logger *logr.Logger, kubeClient *k8s.KubeClient, clusterName string) (*target.Targets, error) {
	response := &target.Targets{Clusters: make(target.Clusters)}

	pods, err := k8s.GetCollectorPods(ctx, kubeClient.Client, client.HasLabels{PrometheusReceiverLabel})
	if err != nil {
		return response, fmt.Errorf("failed to list pods: %v", err)
	}

	for _, pod := range pods.Items {
		port, err := getPrometheusPort(&pod)
		if err != nil {
			logger.Error(err, "failed to get prometheus port", "portName", PrometheusPortName)
			continue
		}

		podResponse, err := fetchTargetsFromPod(kubeClient, &pod, port)
		if err != nil {
			logger.Error(err, "unable to fetch targets from pod", "pod", pod.Name, "port", port, "cluster", clusterName)
			continue
		}

		response.AddPodResponse(clusterName, pod.Spec.NodeName, pod.Name, podResponse)
	}

	return response, nil
}

func fetchTargetsFromPod(kubeClient *k8s.KubeClient, pod *corev1.Pod, port int) (*v1.Response, error) {
	localPort, err := utils.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free port: %v", err)
	}

	pf := k8s.NewPortForwarder(kubeClient.RestConfig, pod, port, localPort)

	if err := pf.Run(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %v", err)
	}

	resp, err := pf.DoRequest(PrometheusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed request to %s: %v", PrometheusEndpoint, err)
	}

	pf.Close()

	podResponse := &v1.Response{}
	if err := json.Unmarshal(resp, podResponse); err != nil {
		return nil, fmt.Errorf("invalid response format: %v", err)
	}
	return podResponse, nil
}

func getPrometheusPort(pod *corev1.Pod) (int, error) {
	if strPort, ok := pod.Annotations[PrometheusPortAnnotation]; ok {
		port, err := strconv.Atoi(strPort)
		if err != nil {
			return 0, fmt.Errorf("invalid port annotation %q: %v", strPort, err)
		}
		return port, nil
	}

	port, err := k8s.ExtractContainerPort(pod, DefaultCollectorContainerName, PrometheusPortName)
	if err != nil {
		return 0, fmt.Errorf("failed to extract container port: %v", err)
	}
	return int(port), nil
}
