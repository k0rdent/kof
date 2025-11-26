package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otel "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	defaultNamespace     = "default"
	defaultCollectorName = "test-collector"
	defaultPodName       = "test-pod"
)

var (
	serverResponse server.Response
	expectedLabels map[string]string
)

type FakeResponseWriter struct {
	HeaderMap  http.Header
	StatusCode int
	Body       []byte
}

func NewFakeResponseWriter() *FakeResponseWriter {
	return &FakeResponseWriter{
		HeaderMap: make(http.Header),
	}
}

func (f *FakeResponseWriter) Header() http.Header {
	return f.HeaderMap
}

func (f *FakeResponseWriter) Write(b []byte) (int, error) {
	f.Body = append(f.Body, b...)
	return len(b), nil
}

func (f *FakeResponseWriter) WriteHeader(statusCode int) {
	f.StatusCode = statusCode
}

func createCollector(name, namespace string, status otel.OpenTelemetryCollectorStatus) error { //nolint:unparam
	collector := &otel.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: otel.OpenTelemetryCollectorSpec{
			UpgradeStrategy: "none",
			Config: otel.Config{
				Exporters: otel.AnyConfig{
					Object: map[string]interface{}{},
				},
				Receivers: otel.AnyConfig{
					Object: map[string]interface{}{},
				},
				Processors: &otel.AnyConfig{
					Object: map[string]interface{}{},
				},
				Connectors: &otel.AnyConfig{
					Object: map[string]interface{}{},
				},
				Service: otel.Service{
					Pipelines: map[string]*otel.Pipeline{},
				},
			},
		},
	}

	err := kubeClient.Client.Create(ctx, collector)
	if err != nil {
		return err
	}
	collector.Status = status
	return kubeClient.Client.Status().Update(ctx, collector)
}

func createPod(name, namespace string, labels map[string]string, status corev1.PodStatus) error { //nolint:unparam
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "otc-container",
					Image: "otel/opentelemetry-collector-contrib:latest",
				},
			},
		},
	}

	err := kubeClient.Client.Create(ctx, pod)
	if err != nil {
		return err
	}
	pod.Status = status
	return kubeClient.Client.Status().Update(ctx, pod)
}

var _ = Describe("Collector Metrics Handler", func() {
	Context("Test Collector Handler", func() {
		BeforeEach(func() {
			fakeLogger := logr.Discard()
			serverResponse = server.Response{
				Writer: &FakeResponseWriter{
					HeaderMap: make(http.Header),
				},
				Logger:   &fakeLogger,
				Duration: 100 * time.Millisecond,
				Status:   200,
			}

			expectedLabels = map[string]string{
				"app.kubernetes.io/component":  "opentelemetry-collector",
				"app.kubernetes.io/instance":   "default.collectors-daemon",
				handlers.CollectorMetricsLabel: "true",
			}
		})

		It("should return error when OpenTelemetryCollector has no status replicas", func() {
			err := createCollector(
				defaultCollectorName,
				defaultNamespace,
				otel.OpenTelemetryCollectorStatus{},
			)
			Expect(err).NotTo(HaveOccurred())

			handlers.CollectorHandler(&serverResponse, &http.Request{})

			response := &handlers.Response{}
			err = json.Unmarshal(serverResponse.Writer.(*FakeResponseWriter).Body, &response)
			Expect(err).NotTo(HaveOccurred())

			cluster := response.Clusters[handlers.MothershipClusterName]
			Expect(cluster).NotTo(BeNil())

			customResources := cluster.CustomResources
			Expect(customResources).NotTo(BeNil())

			expectedCustomResource := customResources[defaultCollectorName]
			Expect(expectedCustomResource).NotTo(BeNil())

			Expect(expectedCustomResource.MessageType).To(Equal(metrics.MessageTypeError))
			Expect(expectedCustomResource.Message).To(Equal(handlers.CollectorNoReplicasMessage))
		})

		It("should return error when all collector pods are down", func() {
			selector := labels.SelectorFromSet(expectedLabels)

			err := createCollector(
				defaultCollectorName,
				defaultNamespace,
				otel.OpenTelemetryCollectorStatus{
					Scale: otel.ScaleSubresourceStatus{
						Selector:       selector.String(),
						Replicas:       2,
						StatusReplicas: "0/2",
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			handlers.CollectorHandler(&serverResponse, &http.Request{})

			response := &handlers.Response{}
			err = json.Unmarshal(serverResponse.Writer.(*FakeResponseWriter).Body, &response)
			Expect(err).NotTo(HaveOccurred())

			cluster := response.Clusters[handlers.MothershipClusterName]
			Expect(cluster).NotTo(BeNil())

			customResources := cluster.CustomResources
			Expect(customResources).NotTo(BeNil())

			expectedCustomResource := customResources[defaultCollectorName]
			Expect(expectedCustomResource).NotTo(BeNil())

			Expect(expectedCustomResource.MessageType).To(Equal(metrics.MessageTypeError))
			Expect(expectedCustomResource.Message).To(Equal(fmt.Sprintf(handlers.CollectorAllReplicasDownMessage, 2)))

			Expect(expectedCustomResource.Pods).To(BeEmpty())
		})

		It("should return healthy status when all collector pods are running", func() {
			selector := labels.SelectorFromSet(expectedLabels)

			err := createCollector(
				defaultCollectorName,
				defaultNamespace,
				otel.OpenTelemetryCollectorStatus{
					Scale: otel.ScaleSubresourceStatus{
						Selector:       selector.String(),
						Replicas:       2,
						StatusReplicas: "2/2",
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			podName1 := defaultPodName + "-1"
			err = createPod(
				podName1,
				defaultNamespace,
				expectedLabels,
				corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			podName2 := defaultPodName + "-2"
			err = createPod(
				podName2,
				defaultNamespace,
				expectedLabels,
				corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			handlers.CollectorHandler(&serverResponse, &http.Request{})

			response := &handlers.Response{}
			err = json.Unmarshal(serverResponse.Writer.(*FakeResponseWriter).Body, &response)
			Expect(err).NotTo(HaveOccurred())

			cluster := response.Clusters[handlers.MothershipClusterName]
			Expect(cluster).NotTo(BeNil())

			customResources := cluster.CustomResources
			Expect(customResources).NotTo(BeNil())

			expectedCustomResource := customResources[defaultCollectorName]
			Expect(expectedCustomResource).NotTo(BeNil())

			Expect(expectedCustomResource.MessageType).To(BeEmpty())
			Expect(expectedCustomResource.Message).To(BeEmpty())

			Expect(expectedCustomResource.Pods).To(HaveLen(2))

			for podName, pod := range expectedCustomResource.Pods {
				Expect(podName).To(SatisfyAny(Equal(podName1), Equal(podName2)))
				Expect(pod.Metrics[metrics.ConditionReadyHealthy][0].Value).To(Equal("healthy"))
			}
		})

		It("should return pod status metrics when some collector pods are unhealthy", func() {
			selector := labels.SelectorFromSet(expectedLabels)

			err := createCollector(
				defaultCollectorName,
				defaultNamespace,
				otel.OpenTelemetryCollectorStatus{
					Scale: otel.ScaleSubresourceStatus{
						Selector:       selector.String(),
						Replicas:       2,
						StatusReplicas: "1/2",
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			healthyPodName := defaultPodName + "-healthy"
			err = createPod(
				healthyPodName,
				defaultNamespace,
				expectedLabels,
				corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			unhealthyPodName := defaultPodName + "-unhealthy"
			err = createPod(
				unhealthyPodName,
				defaultNamespace,
				expectedLabels,
				corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:    corev1.PodReady,
							Status:  corev1.ConditionFalse,
							Reason:  "ContainerCreating",
							Message: "Container is being created",
						},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			handlers.CollectorHandler(&serverResponse, &http.Request{})

			response := &handlers.Response{}
			err = json.Unmarshal(serverResponse.Writer.(*FakeResponseWriter).Body, &response)
			Expect(err).NotTo(HaveOccurred())

			cluster := response.Clusters[handlers.MothershipClusterName]
			Expect(cluster).NotTo(BeNil())

			customResources := cluster.CustomResources
			Expect(customResources).NotTo(BeNil())

			expectedCustomResource := customResources[defaultCollectorName]
			Expect(expectedCustomResource).NotTo(BeNil())

			Expect(expectedCustomResource.MessageType).To(Equal(metrics.MessageTypeError))
			Expect(expectedCustomResource.Message).To(Equal(fmt.Sprintf(handlers.CollectorSomeReplicasDownMessage, 1, 2)))

			Expect(expectedCustomResource.Pods).To(HaveLen(2))

			healthyPod := expectedCustomResource.Pods[healthyPodName]
			Expect(healthyPod).NotTo(BeNil())
			Expect(healthyPod.Metrics[metrics.ConditionReadyHealthy][0].Value).To(Equal("healthy"))

			unhealthyPod := expectedCustomResource.Pods[unhealthyPodName]
			Expect(unhealthyPod).NotTo(BeNil())
			Expect(unhealthyPod.Metrics[metrics.ConditionReadyHealthy][0].Value).To(Equal("unhealthy"))
			Expect(unhealthyPod.Metrics[metrics.ConditionReadyReason][0].Value).To(Equal("ContainerCreating"))
		})

		It("should return metrics for a single pod when only one collector pod is running", func() {
			selector := labels.SelectorFromSet(expectedLabels)

			err := createCollector(
				defaultCollectorName,
				defaultNamespace,
				otel.OpenTelemetryCollectorStatus{
					Scale: otel.ScaleSubresourceStatus{
						Selector:       selector.String(),
						Replicas:       2,
						StatusReplicas: "1/2",
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = createPod(
				defaultPodName,
				defaultNamespace,
				expectedLabels,
				corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())

			handlers.CollectorHandler(&serverResponse, &http.Request{})

			response := &handlers.Response{}
			err = json.Unmarshal(serverResponse.Writer.(*FakeResponseWriter).Body, &response)
			Expect(err).NotTo(HaveOccurred())

			cluster := response.Clusters[handlers.MothershipClusterName]
			Expect(cluster).NotTo(BeNil())

			customResources := cluster.CustomResources
			Expect(customResources).NotTo(BeNil())

			expectedCustomResource := customResources[defaultCollectorName]
			Expect(expectedCustomResource).NotTo(BeNil())

			Expect(expectedCustomResource.MessageType).To(Equal(metrics.MessageTypeError))
			Expect(expectedCustomResource.Message).To(Equal(fmt.Sprintf(handlers.CollectorSomeReplicasDownMessage, 1, 2)))

			Expect(expectedCustomResource.Pods).To(HaveLen(1))

			healthyPod := expectedCustomResource.Pods[defaultPodName]
			Expect(healthyPod).NotTo(BeNil())
			Expect(healthyPod.Metrics[metrics.ConditionReadyHealthy][0].Value).To(Equal("healthy"))
		})
	})
})
