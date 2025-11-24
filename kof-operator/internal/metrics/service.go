package metrics

func New(cfg *CollectorServiceConfig) *CollectorService {
	return &CollectorService{config: cfg}
}

func (s *CollectorService) CollectAll() {
	s.CollectHealth()
	s.CollectResources()
	s.CollectInternal()
}
