package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Make secret data", func() {
	It("should produces a valid yaml for promxy secret config", func() {
		config := &PromxyConfig{
			RemoteWriteUrl: "http://vminsert-cluster:8480/insert/0/prometheus/api/v1/write",
			ServerGroups: []*PromxyConfigServerGroup{
				{
					Targets:               []string{"vmauth.storage0.example.net:443"},
					PathPrefix:            "/vm/select/0/prometheus/",
					Scheme:                "https",
					DialTimeout:           "1s",
					Username:              "u",
					Password:              "p",
					ClusterName:           "test-cluster",
					TlsInsecureSkipVerify: true,
					BasicAuthEnabled:      true,
				},
			},
		}
		data, err := RenderPromxySecretTemplate(config)
		Expect(err).ToNot(HaveOccurred())
		Expect("\n" + data).To(Equal(`
global:
  evaluation_interval: 5s
  external_labels:
    source: promxy
rule_files:
  - /etc/promxy/rules/*.yaml
alerting:
  alertmanagers:
    - scheme: http
      static_configs:
        - targets:
          - vmalertmanager-cluster:9093
remote_write:
  - url: "http://vminsert-cluster:8480/insert/0/prometheus/api/v1/write"
promxy:
  server_groups:
    - static_configs:
        - targets:
          - "vmauth.storage0.example.net:443"
      path_prefix: "/vm/select/0/prometheus/"
      scheme: "https"
      http_client:
        dial_timeout: "1s"
        tls_config:
          insecure_skip_verify: true
        basic_auth:
          username: "u"
          password: "p"
      labels:
        promxyCluster: "test-cluster"
      ignore_error: true
`))
	})
})
