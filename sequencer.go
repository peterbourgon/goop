package goop

import (
	"fmt"
)

type Slot []Event

// A Sequencer is a set of Events which are fired to a set of 
// EventReceivers on a schedule determined by the Clock.
type Sequencer struct {
	eventIn   chan Event
	receivers []EventReceiver
	slots     []Slot
	pos       int
}

func (s *Sequencer) Events() chan<- Event { return s.eventIn }

func (s *Sequencer) String() string {
	return fmt.Sprintf("%d receivers, %d slots", len(s.receivers), len(s.slots))
}

func NewSequencer() *Sequencer {
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	sl := make([]Slot, 0)
	re := make([]EventReceiver, 0)
	s := &Sequencer{ei, re, sl, 0}
	go s.sequenceLoop()
	return s
}

func (s *Sequencer) register(r EventReceiver) {
	for _, existing := range s.receivers {
		if existing == r {
			return
		}
	}
	s.receivers = append(s.receivers, r)
}

func (s *Sequencer) unregister(r EventReceiver) {
	for i, existing := range s.receivers {
		if r == existing {
			s.receivers = append(s.receivers[:i], s.receivers[:i+1]...)
			return	
		}
	}
}

func (s *Sequencer) sequenceLoop() {
	for {
		select {
		case ev := <-s.eventIn:
			switch ev.Name {
			case "kill":
				return
			case "disconnect":
				s.receivers = make([]EventReceiver, 0)
			case "clear":
				s.pos = 0
				s.slots = make([]Slot, 0)
			case "register":
				if r, ok := ev.Arg.(EventReceiver); ok {
					s.register(r)	
				}
			case "unregister":
				if r, ok := ev.Arg.(EventReceiver); ok {
					s.unregister(r)
				}
			case "push":
				if sl, ok := ev.Arg.(Slot); ok {
					s.slots = append(s.slots, sl)
				}
			case "pop":
				if len(s.slots) > 0 {
					s.slots = s.slots[:len(s.slots)-1]
				}
			case "tick":
				if len(s.slots) > 0 {
					for _, r := range s.receivers {
						for _, ev := range s.slots[s.pos] {
							r.Events() <- ev
						}
					}
				}
				s.pos++
				if s.pos >= len(s.slots) {
					s.pos = 0
				}
			}
		}
	}
}