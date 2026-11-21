package lifecycle

type state int

const (
	stateWaiting state = iota
	stateStarting
	stateRunning
	stateDraining
	stateStopped
)
