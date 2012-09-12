package main

type EventReceiver interface {
	Events() chan<- Event
}

type Event struct {
	Type  string
	Value float32
	Arg   interface{}
}

func ConnectEvent(dst Node) Event       { return Event{Connect, 0.0, dst} }
func DisconnectEvent(dst Node) Event    { return Event{Disconnect, 0.0, dst} }
func ConnectionEvent(src Node) Event    { return Event{Connection, 0.0, src} }
func DisconnectionEvent(src Node) Event { return Event{Disconnection, 0.0, src} }
func KillEvent() Event                  { return Event{Kill, 0.0, nil} }

const (
	Connect       = "connect"
	Disconnect    = "disconnect"
	Connection    = "connection"
	Disconnection = "disconnection"
	Kill          = "kill" // stop all processing loops
)
