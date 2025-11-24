package k8s

import (
	"context"
	"strings"

	otel "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOpenTelemetryCollectors(ctx context.Context, client client.Client) (*otel.OpenTelemetryCollectorList, error) {
	optlcList := new(otel.OpenTelemetryCollectorList)
	err := client.List(ctx, optlcList)
	return optlcList, err
}

func ExtractPodSelectorsFromOTelCollector(otelc *otel.OpenTelemetryCollector) client.MatchingLabels {
	if otelc.Status.Scale.Selector == "" {
		return nil
	}

	matchingLabels := make(map[string]string)
	keyPairs := strings.SplitSeq(otelc.Status.Scale.Selector, ",")
	for pair := range keyPairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}

		key := kv[0]
		value := kv[1]
		matchingLabels[key] = value
	}

	return client.MatchingLabels(matchingLabels)
}
