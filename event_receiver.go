package goop

type Event interface{}

type EventReceiver interface {
	Events() chan<- Event
}
