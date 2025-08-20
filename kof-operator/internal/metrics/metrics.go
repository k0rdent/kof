package metrics

func (c Cluster) Add(m *Metric) {
	if cluster := c[m.Cluster]; cluster == nil {
		c[m.Cluster] = make(Pod)
	}
	c[m.Cluster].Add(m)
}

func (p Pod) Add(m *Metric) {
	if pod := p[m.Pod]; pod == nil {
		p[m.Pod] = make(Metrics)
	}
	p[m.Pod].Add(m.Name, m.Data)
}

func (m Metrics) Add(name string, labels *MetricValue) {
	m[name] = append(m[name], labels)
}
