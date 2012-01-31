package goop

// An Event is a thing which is typically fed to a module, in order to
// enact some change of state in that module. 
type Event struct {
	Name string
	Val  float32
	Arg  interface{}
}

// The EventReceiver interface should be implemented by any module which
// intends to receive and process events.
type EventReceiver interface {
	Events() chan<- Event
}

// A TargetAndEvent is used to store an Event and its Receiver
// for later (deferred) firing.
type TargetAndEvent struct {
	target EventReceiver
	event  Event
}

// The DeferredEventReceiver should be implemented by some module
// which can buffer Events to targets until some later time.
// In practice, this is the Clock, directly.
type DeferredEventReceiver interface {
	DeferredEvents() chan<- TargetAndEvent
}

// The AudioSender interface should be implemented by any module which
// generates and yields audio data.
type AudioSender interface {
	AudioOut() <-chan []float32
}

