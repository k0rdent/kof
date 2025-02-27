package controller

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed template/secret.tmpl
var promxySecretTemplate string

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
	BasicAuthEnabled      bool
}

func RenderPromxySecretTemplate(config *PromxyConfig) (string, error) {
	t := template.Must(template.New("promxy-secret").Parse(promxySecretTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, config)
	return buf.String(), err
}
