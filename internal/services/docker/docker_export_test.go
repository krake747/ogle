package docker

//nolint:gochecknoglobals // test-only exports of unexported functions
var (
	ParsePsOutput = parsePsOutput
	ParseState    = parseState
)
