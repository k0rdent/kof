package target

import (
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

type Targets struct {
	Clusters `json:"clusters"`
}

func (t *Targets) AddPodResponse(clusterName, nodeName, podName string, podResponse *v1.Response) {
	t.FindOrCreate(clusterName).
		FindOrCreate(nodeName).
		Add(podName, podResponse)
}

func (t *Targets) Merge(target *Targets) {
	if target != nil && len(target.Clusters) > 0 {
		for name, cluster := range target.Clusters {
			t.Add(name, cluster)
		}
	}
}
