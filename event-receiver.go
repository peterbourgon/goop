package main

import (
	"fmt"
	"strconv"
	"strings"
)

// The eventProcessor interface is designed to be implemented by concrete
// nodes in the network, to handle specific Events relevant to their param.
// Often there is a hierarchy of eventProcessors within a single concrete
// type, and Events fall from upper to lower levels.
type eventProcessor interface {
	processEvent(ev Event)
}

// An EventReceiver is capable of receiving and processing Events.
type EventReceiver interface {
	Events() chan<- Event
}

// Event describes any asynchronous thing which may be
// sent to Nodes in the Field.
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

// ParseArbitraryEvents attempts to parse the passed string into an
// arbitrary Event. An arbitrary event has the grammar
// ArbitraryEvent := <string> [ "-" <float32> ]
func ParseArbitraryEvent(s string) (Event, error) {
	toks := strings.Split(s, "-")
	switch len(toks) {
	case 1:
		return Event{toks[0], 0.0, nil}, nil

	case 2:
		val, err := strconv.ParseFloat(toks[1], 32)
		if err != nil {
			return Event{}, fmt.Errorf("bad Value %s", toks[1])
		}
		return Event{toks[0], float32(val), nil}, nil

	default:
		return Event{}, fmt.Errorf("couldn't parse Event")
	}
	panic("unreachable")
}
