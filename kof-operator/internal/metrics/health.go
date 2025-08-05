package metrics

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (s *Service) CollectHealth() {
	condition := getReadyCondition(s.config.Pod.Status.Conditions)

	if condition == nil {
		s.send(ConditionReadyHealthy, "unhealthy")
		s.send(ConditionReadyReason, "MissingReadyCondition")
		s.send(ConditionReadyMessage, "Pod status does not contain Ready condition")
		s.error(fmt.Errorf("Ready condition not found in pod status"))
		return
	}

	if condition.Status == corev1.ConditionTrue {
		s.send(ConditionReadyHealthy, "healthy")
	} else {
		s.send(ConditionReadyHealthy, "unhealthy")
		s.send(ConditionReadyReason, condition.Reason)
		s.send(ConditionReadyMessage, condition.Message)
	}
}
