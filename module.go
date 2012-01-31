package goop

// An Event is a thing which is typically fed to a module, in order to
// enact some change of state in that module. 
type Event struct {
	name string
	val  float32
	arg  interface{}
}

// The EventReceiver interface should be implemented by any module which
// intends to receive and process events.
type EventReceiver interface {
	Events() chan<- Event
}

type AudioSender interface {
	AudioOut() <-chan []float32
}
