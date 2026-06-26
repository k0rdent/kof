package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPostPodHTTPEmpty_NoPodIP(t *testing.T) {
	t.Parallel()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "promxy-1"},
	}

	err := PostPodHTTPEmpty(context.Background(), pod, "promxy", "http", "/-/reload", 0)
	if err == nil {
		t.Fatal("expected error when pod has no IP")
	}
	if !strings.Contains(err.Error(), "no IP assigned") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostPodHTTPEmpty_MissingPort(t *testing.T) {
	t.Parallel()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "promxy-1"},
		Status:     corev1.PodStatus{PodIP: "10.0.0.1"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "promxy",
			}},
		},
	}

	err := PostPodHTTPEmpty(context.Background(), pod, "promxy", "http", "/-/reload", 0)
	if err == nil {
		t.Fatal("expected error when container port is missing")
	}
}
