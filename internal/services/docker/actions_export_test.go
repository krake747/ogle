package docker

// SetCommander sets the commander on a Service for testing.
func (s *Service) SetCommander(c Commander) {
	s.commander = c
}
