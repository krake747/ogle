package theme

// Changed is emitted by app.Model after a new theme has been loaded.
// Every component updates its stored pointer on receipt.
type Changed struct {
	Theme *Theme
}
