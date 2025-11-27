package metrics

func New(cfg *MetricCollectorServiceConfig) *MetricCollectorService {
	return &MetricCollectorService{config: cfg}
}

func (s *MetricCollectorService) CollectAll() {
	s.CollectHealth()
	s.CollectResources()
	s.CollectInternal()
}
