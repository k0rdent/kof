package metrics

func (c ClusterMetrics) Add(m *Metric) {
	if cluster := c[m.Cluster]; cluster == nil {
		c[m.Cluster] = make(PodMetrics)
	}
	c[m.Cluster].Add(m)
}

func (p PodMetrics) Add(m *Metric) {
	if pod := p[m.Pod]; pod == nil {
		p[m.Pod] = make(Metrics)
	}
	p[m.Pod].Add(m.Name, m.Value)
}

func (m Metrics) Add(name string, value any) {
	m[name] = value
}
