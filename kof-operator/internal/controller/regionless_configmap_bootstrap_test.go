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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Regionless ConfigMap bootstrap", func() {
	ctx := context.Background()

	const managementClusterName = "mothership"
	const regionlessDomain = "mothership.example.com"

	regionlessConfigMapNamespacedName := types.NamespacedName{
		Name:      GetRegionalClusterConfigMapName(managementClusterName),
		Namespace: k8s.DefaultSystemNamespace,
	}

	BeforeEach(func() {
		Expect(os.Setenv("KOF_REGIONLESS_ENABLED", strutil.True)).To(Succeed())
		Expect(os.Setenv("KOF_REGIONLESS_DOMAIN", regionlessDomain)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Setenv("KOF_REGIONLESS_ENABLED", strutil.False)).To(Succeed())
	})

	It("creates a regional ConfigMap with endpoints from the regionless domain", func() {
		Expect(CreateOrUpdateRegionlessConfigMap(ctx, k8sClient, managementClusterName)).To(Succeed())

		configMap := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, regionlessConfigMapNamespacedName, configMap)).To(Succeed())
		Expect(configMap.Labels[labels.KofGeneratedLabel]).To(Equal(strutil.True))
		Expect(configMap.Labels[KofClusterRoleLabel]).To(Equal(KofRoleRegional))
		Expect(configMap.Labels[KofRegionlessLabel]).To(Equal(strutil.True))
		Expect(configMap.Data[RegionalClusterNameKey]).To(Equal(managementClusterName))
		Expect(configMap.Data[RegionalClusterNamespaceKey]).To(Equal(k8s.DefaultSystemNamespace))
		Expect(configMap.Data[RegionalKofHTTPConfigKey]).To(BeEmpty())
		Expect(configMap.Data[ReadMetricsKey]).To(Equal("https://vmauth.mothership.example.com/vm/select/0/prometheus"))
		Expect(configMap.Data[WriteMetricsKey]).
			To(Equal("https://vmauth.mothership.example.com/vm/insert/0/prometheus/api/v1/write"))
		Expect(configMap.Data[WriteLogsKey]).
			To(Equal("https://vmauth.mothership.example.com/vli/insert/opentelemetry/v1/logs"))
	})

	It("uses internal service endpoints for regionless generated resources", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetRegionalClusterConfigMapName(managementClusterName),
				Namespace: k8s.DefaultSystemNamespace,
				Labels: map[string]string{
					KofRegionlessLabel: strutil.True,
				},
			},
		}
		regionalConfigMap := &RegionalClusterConfigMap{
			clusterName:      managementClusterName,
			releaseNamespace: k8s.KofNamespace,
			ctx:              ctx,
			client:           k8sClient,
			configMap:        configMap,
		}

		endpoint := regionalConfigMap.GetReadEndpoint(ReadMetricsAnnotation, "https://vmauth.mothership.example.com/vm/select/0/prometheus")
		Expect(endpoint).To(Equal("http://vmauth-cluster:8427/vm/select/0/prometheus"))
	})
})
