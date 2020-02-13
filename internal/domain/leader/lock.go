package leader

// Lock describes the algebra for a leader lock
type Lock interface {

	// IsLeader returns true if this instance has the leader lock
	IsLeader() bool

	// Start runs the leader lock loop
	Start()

	// Stop stops the leader lock loop
	Stop()
}
