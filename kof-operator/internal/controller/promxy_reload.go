package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
)

const (
	promxyReloadPath        = "/-/reload"
	promxyContainerName     = "promxy"
	promxyContainerPortName = "http"
	promxyReloadTimeout     = 30 * time.Second
)

func ReloadPromxyConfig(ctx context.Context, kubeClient *k8s.KubeClient, namespace, labelSelector string) error {
	logger := log.FromContext(ctx)

	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return fmt.Errorf("invalid promxy pod label selector %q: %w", labelSelector, err)
	}

	podList, err := k8s.GetPods(
		ctx,
		kubeClient.Client,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	)
	if err != nil {
		return fmt.Errorf("listing promxy pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return fmt.Errorf("no promxy pods found in namespace %q matching %q", namespace, labelSelector)
	}

	var reloadErrors []error
	reloaded := 0
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			logger.Info(
				"Skipping promxy config reload for pod that is not running",
				"pod", pod.Name,
				"phase", pod.Status.Phase,
			)
			continue
		}

		logger.Info("Reloading promxy config", "pod", pod.Name, "podIP", pod.Status.PodIP)
		if err := k8s.PostPodHTTPEmpty(ctx, pod, promxyContainerName, promxyContainerPortName, promxyReloadPath, promxyReloadTimeout); err != nil {
			logger.Error(err, "Failed to reload promxy config", "pod", pod.Name)
			reloadErrors = append(reloadErrors, fmt.Errorf("pod %s: %w", pod.Name, err))
			continue
		}
		logger.Info("Reloaded promxy config", "pod", pod.Name, "podIP", pod.Status.PodIP)
		reloaded++
	}

	if reloaded == 0 {
		if len(reloadErrors) > 0 {
			return fmt.Errorf("failed to reload promxy config on all pods: %w", errors.Join(reloadErrors...))
		}
		return fmt.Errorf("no running promxy pods to reload in namespace %q matching %q", namespace, labelSelector)
	}

	if len(reloadErrors) == 0 {
		logger.Info("Reloaded promxy config on all pods", "reloaded", reloaded)
	}

	return errors.Join(reloadErrors...)
}
