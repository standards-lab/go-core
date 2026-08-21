package lifecycle

type state string

const (
	stateWaiting  state = "WAITING"
	stateStarting state = "STARTING"
	stateRunning  state = "RUNNING"
	stateDraining state = "DRAINING"
	stateStopped  state = "STOPPED"
)
