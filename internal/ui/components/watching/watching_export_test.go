package watching

func SetState(m *Model, s int) {
	m.st = state(s)
}
