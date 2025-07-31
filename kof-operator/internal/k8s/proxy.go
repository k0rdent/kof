package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const ProxyTimeout = 5 * time.Second

func Proxy(ctx context.Context, clientset *kubernetes.Clientset, pod corev1.Pod, port, path string) ([]byte, error) {
	proxyCtx, cancel := context.WithTimeout(ctx, ProxyTimeout)
	defer cancel()

	return clientset.CoreV1().
		RESTClient().
		Get().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s:%s", pod.Name, port)).
		SubResource("proxy").
		Suffix(path).
		Do(proxyCtx).
		Raw()
}
