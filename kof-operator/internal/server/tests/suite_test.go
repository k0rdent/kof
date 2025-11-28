/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otel "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	// +kubebuilder:scaffold:imports
)

var (
	ctx        context.Context
	kubeClient *k8s.KubeClient
	testEnv    *envtest.Environment
)

func TestServerHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Handlers Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = otel.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kcmv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	kubeClient, err = k8s.NewClientFromRestConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	k8s.LocalKubeClient = kubeClient
})

var _ = AfterEach(func() {
	By("Cleanup all objects")
	objects := []client.Object{
		&corev1.Pod{},
		&otel.OpenTelemetryCollector{},
	}

	namespaces := []string{defaultNamespace}
	for _, obj := range objects {
		for _, ns := range namespaces {
			err := kubeClient.Client.DeleteAllOf(ctx, obj, client.InNamespace(ns))
			Expect(err).To(Succeed())
		}
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
