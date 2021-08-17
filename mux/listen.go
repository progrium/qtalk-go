package mux

// A Listener is similar to a net.Listener but returns connections wrapped as mux sessions.
type Listener interface {
	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Accept waits for and returns the next incoming session.
	Accept() (*Session, error)
}
