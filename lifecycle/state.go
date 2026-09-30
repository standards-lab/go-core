package lifecycle

type state int

// stateWaiting is the zero value, so a zero Coordinator accepts
// registrations.
const (
	stateWaiting state = iota
	stateStarting
	stateRunning
	stateDraining
	stateStopped
)
