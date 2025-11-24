package metrics

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (s *CollectorService) CollectHealth() {
	condition := getReadyCondition(s.config.Pod.Status.Conditions)

	if condition == nil {
		s.sendMetric(ConditionReadyHealthy, &MetricValue{Value: "unhealthy"})
		s.sendMetric(ConditionReadyReason, &MetricValue{Value: "MissingReadyCondition"})
		s.sendMetric(ConditionReadyMessage, &MetricValue{Value: "Pod status does not contain Ready condition"})
		s.error(fmt.Errorf("the Ready condition is not found in pod status"))
		return
	}

	if condition.Status == corev1.ConditionTrue {
		s.sendMetric(ConditionReadyHealthy, &MetricValue{Value: "healthy"})
	} else {
		s.sendMetric(ConditionReadyHealthy, &MetricValue{Value: "unhealthy"})
		s.sendMetric(ConditionReadyReason, &MetricValue{Value: condition.Reason})
		s.sendMetric(ConditionReadyMessage, &MetricValue{Value: condition.Message})
	}
}
