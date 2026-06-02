package controller

import (
	"context"
	"os"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	"github.com/k0rdent/kof/kof-operator/internal/strutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Regionless ConfigMap bootstrap", func() {
	ctx := context.Background()

	const managementClusterName = "mothership"
	const regionlessDomain = "mothership.example.com"
	const regionlessHTTPConfig = `{"tls_config": {"insecure_skip_verify": true}}`

	regionlessConfigMapNamespacedName := types.NamespacedName{
		Name:      GetRegionalClusterConfigMapName(managementClusterName),
		Namespace: k8s.DefaultSystemNamespace,
	}

	BeforeEach(func() {
		Expect(os.Setenv("KOF_REGIONLESS_ENABLED", strutil.True)).To(Succeed())
		Expect(os.Setenv("KOF_REGIONLESS_DOMAIN", regionlessDomain)).To(Succeed())
		Expect(os.Setenv("KOF_REGIONLESS_HTTP_CONFIG", regionlessHTTPConfig)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Setenv("KOF_REGIONLESS_ENABLED", strutil.False)).To(Succeed())
		setKofNamespaceIstioInjection(ctx, false)
	})

	It("creates a regional ConfigMap with endpoints from the regionless domain", func() {
		setKofNamespaceIstioInjection(ctx, false)

		Expect(CreateOrUpdateRegionlessConfigMap(ctx, k8sClient, managementClusterName)).To(Succeed())

		configMap := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, regionlessConfigMapNamespacedName, configMap)).To(Succeed())
		Expect(configMap.Labels[labels.KofGeneratedLabel]).To(Equal(strutil.True))
		Expect(configMap.Labels[KofClusterRoleLabel]).To(Equal(KofRoleRegional))
		Expect(configMap.Labels[KofRegionlessLabel]).To(Equal(strutil.True))
		Expect(configMap.Data[RegionalClusterNameKey]).To(Equal(managementClusterName))
		Expect(configMap.Data[RegionalClusterNamespaceKey]).To(Equal(k8s.DefaultSystemNamespace))
		Expect(configMap.Data[RegionalKofHTTPConfigKey]).To(Equal(regionlessHTTPConfig))
		Expect(configMap.Data[ReadMetricsKey]).To(Equal("https://vmauth.mothership.example.com/vm/select/0/prometheus"))
		Expect(configMap.Data[WriteMetricsKey]).
			To(Equal("https://vmauth.mothership.example.com/vm/insert/0/prometheus/api/v1/write"))
		Expect(configMap.Data[WriteLogsKey]).
			To(Equal("https://vmauth.mothership.example.com/vli/insert/opentelemetry/v1/logs"))
	})

	It("uses Istio read endpoints when the KOF namespace has Istio injection enabled", func() {
		setKofNamespaceIstioInjection(ctx, true)

		Expect(CreateOrUpdateRegionlessConfigMap(ctx, k8sClient, managementClusterName)).To(Succeed())

		configMap := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, regionlessConfigMapNamespacedName, configMap)).To(Succeed())
		Expect(configMap.Data[RegionalIstioRoleKey]).To(Equal("member"))
		Expect(configMap.Data[ReadMetricsKey]).To(Equal("http://mothership-vmauth:8427/vm/select/0/prometheus"))
		Expect(configMap.Data[ReadLogsKey]).To(Equal("http://mothership-vmauth:8427/vls"))
		Expect(configMap.Data[WriteMetricsKey]).To(BeEmpty())
		Expect(configMap.Data[WriteLogsKey]).To(BeEmpty())
	})
})

func setKofNamespaceIstioInjection(ctx context.Context, enabled bool) {
	namespace := &corev1.Namespace{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: k8s.KofNamespace}, namespace)).To(Succeed())
	if namespace.Labels == nil {
		namespace.Labels = map[string]string{}
	}
	if enabled {
		namespace.Labels["istio-injection"] = "enabled"
	} else {
		delete(namespace.Labels, "istio-injection")
	}
	Expect(k8sClient.Update(ctx, namespace)).To(Succeed())
}
