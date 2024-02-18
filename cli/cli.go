package cli

// Stores our application state
type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}
