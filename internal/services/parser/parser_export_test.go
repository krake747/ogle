package parser

//nolint:gochecknoglobals // test-only exports of unexported functions
var (
	NormalizePorts     = normalizePorts
	NormalizeShortPort = normalizeShortPort
	NormalizeLongPort  = normalizeLongPort
	SplitByColon       = splitByColon
	FindSlash          = findSlash
)
