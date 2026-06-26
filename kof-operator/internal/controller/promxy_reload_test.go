package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
)

func TestReloadPromxyConfig_InvalidSelector(t *testing.T) {
	t.Parallel()

	kubeClient := &k8s.KubeClient{
		Client: fake.NewClientBuilder().Build(),
	}

	err := ReloadPromxyConfig(context.Background(), kubeClient, "kof", "not a valid selector ==")
	if err == nil {
		t.Fatal("expected error for invalid selector")
	}
}

func TestReloadPromxyConfig_NoPods(t *testing.T) {
	t.Parallel()

	kubeClient := &k8s.KubeClient{
		Client: fake.NewClientBuilder().Build(),
	}

	err := ReloadPromxyConfig(
		context.Background(),
		kubeClient,
		"kof",
		"app.kubernetes.io/name=kof-mothership-promxy",
	)
	if err == nil {
		t.Fatal("expected error when no promxy pods exist")
	}
}

func TestReloadPromxyConfig_NoRunningPods(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kof-mothership-promxy-abc",
			Namespace: "kof",
			Labels: map[string]string{
				"app.kubernetes.io/name":     "kof-mothership-promxy",
				"app.kubernetes.io/instance": "kof-mothership",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	kubeClient := &k8s.KubeClient{
		Client: fake.NewClientBuilder().
			WithObjects(pod).
			Build(),
	}

	err := ReloadPromxyConfig(
		context.Background(),
		kubeClient,
		"kof",
		"app.kubernetes.io/name=kof-mothership-promxy,app.kubernetes.io/instance=kof-mothership",
	)
	if err == nil {
		t.Fatal("expected error when no running promxy pods exist")
	}
}
