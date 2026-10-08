package lifecycle

// Monitored is a value that reports runtime failures on a channel. A
// [graph.Dependency] whose Value implements it is watched by the
// [Coordinator] that [New] returns for its System, as a channel registered
// with [Coordinator.Monitor] is: the first non-nil error ends the run. Err
// is called once startup has completed, so a value may create its channel
// in Start. A nil error is ignored, a closed channel retires quietly, and a
// nil channel, which never yields, is not watched.
type Monitored interface {
	Err() <-chan error
}
