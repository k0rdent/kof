package metrics

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (s *Service) CollectHealth() {
	condition := getReadyCondition(s.config.Pod.Status.Conditions)

	if condition == nil {
		s.send(MetricHealthy, "unhealthy")
		s.send(MetricReadyReason, "MissingReadyCondition")
		s.send(MetricReadyMessage, "Pod status does not contain Ready condition")
		s.error(fmt.Errorf("Ready condition not found in pod status"))
		return
	}

	if condition.Status == corev1.ConditionTrue {
		s.send(MetricHealthy, "healthy")
	} else {
		s.send(MetricHealthy, "unhealthy")
		s.send(MetricReadyReason, condition.Reason)
		s.send(MetricReadyMessage, condition.Message)
	}
}
