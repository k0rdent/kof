package metrics

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (s *Service) CollectHealth() {
	condition := getReadyCondition(s.config.Pod.Status.Conditions)

	if condition == nil {
		s.send(ConditionReadyHealthy, &MetricValue{Value: "unhealthy"})
		s.send(ConditionReadyReason, &MetricValue{Value: "MissingReadyCondition"})
		s.send(ConditionReadyMessage, &MetricValue{Value: "Pod status does not contain Ready condition"})
		s.error(fmt.Errorf("the Ready condition is not found in pod status"))
		return
	}

	if condition.Status == corev1.ConditionTrue {
		s.send(ConditionReadyHealthy, &MetricValue{Value: "healthy"})
	} else {
		s.send(ConditionReadyHealthy, &MetricValue{Value: "unhealthy"})
		s.send(ConditionReadyReason, &MetricValue{Value: condition.Reason})
		s.send(ConditionReadyMessage, &MetricValue{Value: condition.Message})
	}
}
