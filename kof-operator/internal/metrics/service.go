package metrics

func New(cfg *ServiceConfig) *Service {
	return &Service{config: cfg}
}

func (s *Service) CollectAll() {
	s.CollectHealth()
	s.CollectResources()
	s.CollectInternal()
}
