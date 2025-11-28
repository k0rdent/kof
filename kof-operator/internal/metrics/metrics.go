package metrics

type (
	ClusterMap        map[string]*ClusterInfo
	CustomResourceMap map[string]*CustomResourceInfo
	PodMap            map[string]*PodInfo
	MetricsMap        map[string][]*MetricValue
)

func (cm ClusterMap) AddMetric(m *MetricData) {
	if m == nil || m.Cluster == "" {
		return
	}

	if cluster := cm[m.Cluster]; cluster == nil {
		cm[m.Cluster] = &ClusterInfo{
			BaseResourceStatus: BaseResourceStatus{Name: m.Cluster},
			CustomResources:    make(CustomResourceMap),
		}
	}
	cm[m.Cluster].CustomResources.AddMetric(m)
}

func (crm CustomResourceMap) AddMetric(m *MetricData) {
	if m == nil || m.CustomResource == "" {
		return
	}

	if cr := crm[m.CustomResource]; cr == nil {
		crm[m.CustomResource] = &CustomResourceInfo{
			BaseResourceStatus: BaseResourceStatus{Name: m.CustomResource},
			Pods:               make(PodMap),
		}
	}
	crm[m.CustomResource].Pods.AddMetric(m)
}

func (pm PodMap) AddMetric(m *MetricData) {
	if m == nil || m.Pod == "" {
		return
	}

	if pod := pm[m.Pod]; pod == nil {
		pm[m.Pod] = &PodInfo{
			BaseResourceStatus: BaseResourceStatus{Name: m.Pod},
			Metrics:            make(MetricsMap),
		}
	}
	pm[m.Pod].Metrics.Add(m)
}

func (m MetricsMap) Add(data *MetricData) {
	if data == nil || data.Name == "" || data.Value == nil {
		return
	}

	m[data.Name] = append(m[data.Name], data.Value)
}

func (cm ClusterMap) AddStatus(s *StatusMessage) {
	if s == nil || s.Cluster == "" {
		return
	}

	if cluster := cm[s.Cluster]; cluster == nil {
		cm[s.Cluster] = &ClusterInfo{
			BaseResourceStatus: BaseResourceStatus{Name: s.Cluster},
			CustomResources:    make(CustomResourceMap),
		}
	}

	if s.CustomResource == "" {
		cm[s.Cluster].Message = s.Message
		cm[s.Cluster].MessageType = s.Type
		return
	}

	cm[s.Cluster].CustomResources.AddStatus(s)
}

func (crm CustomResourceMap) AddStatus(s *StatusMessage) {
	if s == nil || s.CustomResource == "" {
		return
	}

	if cr := crm[s.CustomResource]; cr == nil {
		crm[s.CustomResource] = &CustomResourceInfo{
			BaseResourceStatus: BaseResourceStatus{Name: s.CustomResource},
			Pods:               make(PodMap),
		}
	}

	if s.Pod == "" {
		crm[s.CustomResource].Message = s.Message
		crm[s.CustomResource].MessageType = s.Type
		return
	}

	crm[s.CustomResource].Pods.AddStatus(s)
}

func (pm PodMap) AddStatus(s *StatusMessage) {
	if s == nil || s.Pod == "" {
		return
	}

	if pod := pm[s.Pod]; pod == nil {
		pm[s.Pod] = &PodInfo{
			BaseResourceStatus: BaseResourceStatus{Name: s.Pod},
			Metrics:            make(MetricsMap),
		}
	}

	pm[s.Pod].Message = s.Message
	pm[s.Pod].MessageType = s.Type
}
