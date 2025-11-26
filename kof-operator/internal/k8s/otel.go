package k8s

import (
	"context"

	otel "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOpenTelemetryCollectors(ctx context.Context, client client.Client) (*otel.OpenTelemetryCollectorList, error) {
	optlcList := new(otel.OpenTelemetryCollectorList)
	err := client.List(ctx, optlcList)
	return optlcList, err
}

func ExtractPodSelectorsFromOTelCollector(otelc *otel.OpenTelemetryCollector) (labels.Selector, error) {
	if otelc.Status.Scale.Selector == "" {
		return nil, nil
	}
	return labels.Parse(otelc.Status.Scale.Selector)
}
