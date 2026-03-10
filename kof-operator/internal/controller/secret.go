package controller

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed template/secret.tmpl
var metricsSecretTemplate string

//go:embed template/logs_secret.tmpl
var logsSecretTemplate string

type PromxyConfig struct {
	RemoteWriteUrl string
	ServerGroups   []*PromxyConfigServerGroup
}

type PromxyConfigServerGroup struct {
	Targets               []string
	PathPrefix            string
	Scheme                string
	DialTimeout           string
	TlsInsecureSkipVerify bool
	Username              string
	Password              string
	ClusterName           string
	ClusterNamespace      string
	BasicAuthEnabled      bool
}

func RenderMetricsSecretTemplate(config *PromxyConfig) (string, error) {
	t := template.Must(template.New("metrics-secret").Parse(metricsSecretTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, config)
	return buf.String(), err
}

func RenderLogsSecretTemplate(config *PromxyConfig) (string, error) {
	t := template.Must(template.New("logs-secret").Parse(logsSecretTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, config)
	return buf.String(), err
}
