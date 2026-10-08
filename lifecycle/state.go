package lifecycle

type state int

// stateWaiting is the state [New] returns a Coordinator in: it accepts
// registrations and one [Coordinator.Exec] or [Coordinator.Run].
const (
	stateWaiting state = iota
	stateStarting
	stateRunning
	stateDraining
	stateStopped
)
