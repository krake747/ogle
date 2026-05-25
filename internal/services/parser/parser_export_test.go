package parser

// SetReadFileFn sets the readFileFn on a Service for testing.
func (s *Service) SetReadFileFn(fn func(string) ([]byte, error)) {
	s.readFileFn = fn
}
