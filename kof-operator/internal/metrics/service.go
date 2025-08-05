package metrics

func New(cfg *ServiceConfig) *Service {
	if cfg.ContainerName == "" && cfg.Pod != nil && len(cfg.Pod.Spec.Containers) > 0 {
		cfg.ContainerName = cfg.Pod.Spec.Containers[0].Name
	}
	return &Service{config: cfg}
}

func (s *Service) CollectAll() {
	s.CollectHealth()
	s.CollectResources()
	s.CollectInternal()
}
