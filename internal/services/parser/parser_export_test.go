package parser

//nolint:gochecknoglobals // test-only exports of unexported functions
var (
	NormalizePorts     = normalizePorts
	NormalizeShortPort = normalizeShortPort
	NormalizeLongPort  = normalizeLongPort
	SplitByColon       = splitByColon
	FindSlash          = findSlash
)

// SetReadFileFn sets the readFileFn on a Service for testing.
func (s *Service) SetReadFileFn(fn func(string) ([]byte, error)) {
	s.readFileFn = fn
}
