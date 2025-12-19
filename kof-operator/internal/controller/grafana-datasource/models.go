package datasource

import (
	"encoding/json"
	"fmt"
	"time"

	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config holds all configuration options for creating a Grafana datasource.
type Config struct {
	ClusterName      string
	ClusterNamespace string
	Type             string
	Category         string
	Access           string
	URL              string
	IsDefault        bool
	BasicAuth        bool
	BasicAuthUser    string
	OwnerReference   metav1.OwnerReference
	ResyncPeriod     metav1.Duration
	JSONData         json.RawMessage
	SecureJSONData   json.RawMessage
	ValuesFrom       []grafanav1beta1.ValueFrom
}

const (
	TypePrometheus   = "prometheus"
	TypeVictoriaLogs = "victoriametrics-logs-datasource"
	TypeJaeger       = "jaeger"
)

const (
	AccessProxy  = "proxy"
	AccessDirect = "direct"
)

const (
	CategoryMetrics = "metrics"
	CategoryLogs    = "logs"
	CategoryTraces  = "traces"
)

var (
	DefaultResyncPeriod            = metav1.Duration{Duration: 5 * time.Minute}
	DefaultGrafanaInstanceSelector = map[string]string{"dashboards": "grafana"}
)

func BuildBasicAuthValuesFrom(secretName, usernameKey, passwordKey string) []grafanav1beta1.ValueFrom {
	return []grafanav1beta1.ValueFrom{
		{
			TargetPath: "basicAuthUser",
			ValueFrom: grafanav1beta1.ValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: usernameKey,
				},
			},
		},
		{
			TargetPath: "secureJsonData.basicAuthPassword",
			ValueFrom: grafanav1beta1.ValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: passwordKey,
				},
			},
		},
	}
}

func BuildJSONDataWithTimeout(tlsSkipVerify bool, timeoutSeconds int) json.RawMessage {
	return json.RawMessage(
		fmt.Sprintf(`{"tlsSkipVerify": %t, "timeout": "%d"}`, tlsSkipVerify, timeoutSeconds),
	)
}
