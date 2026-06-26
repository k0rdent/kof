package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const defaultPodHTTPTimeout = 5 * time.Second

// PostPodHTTP sends an HTTP POST to a container port on the pod IP.
// This avoids the apiserver pod proxy subresource, which does not reliably
// handle POST requests (HTTP/2 stream INTERNAL_ERROR).
func PostPodHTTP(
	ctx context.Context,
	pod corev1.Pod,
	containerName, portName, path string,
	body io.Reader,
	timeout time.Duration,
) error {
	if pod.Status.PodIP == "" {
		return fmt.Errorf("pod %s has no IP assigned", pod.Name)
	}

	port, err := ExtractContainerPort(&pod, containerName, portName)
	if err != nil {
		return fmt.Errorf("pod %s: %w", pod.Name, err)
	}
	if timeout <= 0 {
		timeout = defaultPodHTTPTimeout
	}

	url := fmt.Sprintf("http://%s:%s%s", pod.Status.PodIP, port, path)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("pod %s: %w", pod.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("pod %s: %w", pod.Name, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("pod %s: unexpected status %s", pod.Name, res.Status)
	}

	return nil
}

// PostPodHTTPEmpty sends an HTTP POST with an empty body to a container port on the pod IP.
func PostPodHTTPEmpty(
	ctx context.Context,
	pod corev1.Pod,
	containerName, portName, path string,
	timeout time.Duration,
) error {
	return PostPodHTTP(ctx, pod, containerName, portName, path, strings.NewReader(""), timeout)
}
