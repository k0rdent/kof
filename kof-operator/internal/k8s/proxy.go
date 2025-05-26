package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func Proxy(ctx context.Context, clientset *kubernetes.Clientset, pod corev1.Pod, port, path string) ([]byte, error) {
	return clientset.CoreV1().
		RESTClient().
		Get().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(fmt.Sprintf("%s%s", pod.Name, port)).
		SubResource("proxy").
		Suffix(path).
		Do(ctx).
		Raw()
}
