package goop

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type EventDelay struct {
	ev    Event
	delay int64 // ticks
}

func newEventDelay(s string) (EventDelay, error) {
	// format: "<string: event name> <float32: value> <int64: ticks to delay>"
	toks := strings.Split(strings.TrimSpace(s), " ")
	if len(toks) != 3 {
		return EventDelay{}, errors.New(fmt.Sprintf("%s: bad token count\n", s))
	}
	name := toks[0]
	val64, valErr := strconv.ParseFloat(toks[1], 32)
	if valErr != nil {
		return EventDelay{}, errors.New(fmt.Sprintf("bad event value: %s", valErr))
	}
	delay64, delayErr := strconv.ParseInt(toks[2], 10, 64)
	if delayErr != nil {
		return EventDelay{}, errors.New(fmt.Sprintf("bad delay value: %s", delayErr))
	}
	return EventDelay{Event{name, float32(val64), nil}, delay64}, nil
}

func (evd EventDelay) String() string {
	return fmt.Sprintf("%s %.3f %d", evd.ev.Name, evd.ev.Val, evd.delay)
}

type Pattern struct {
	eventIn     chan Event
	eventDelays []EventDelay
}

func (p *Pattern) Events() chan<- Event { return p.eventIn }

func NewPattern(s string) *Pattern {
	eventIn := make(chan Event, OTHER_CHAN_BUFFER)
	eventDelays := make([]EventDelay, 0)
	toks := strings.Split(s, "/")
	for i, tok := range toks {
		if evd, err := newEventDelay(tok); err == nil {
			eventDelays = append(eventDelays, evd)
		} else {
			println("pattern:", i, err)
		}
	}
	p := &Pattern{eventIn, eventDelays}
	go p.patternLoop()
	return p
}

func (p *Pattern) patternLoop() {
	if len(p.eventDelays) == 0 {
		return
	}
	var target EventReceiver = nil
	pos, wait := 0, p.eventDelays[0].delay
	for {
		select {
		case ev := <-p.eventIn:
			switch ev.Name {
			case "kill":
				return
			case "reset":
				target = nil
				pos = 0
			case "fire":
				if r, ok := ev.Arg.(EventReceiver); ok {
					target = r
				}
			case "tick":
				if target != nil {
					if wait <= 0 {
						target.Events() <- p.eventDelays[pos].ev
						pos++
						if pos < len(p.eventDelays) {
							// next event in the pattern, please
							wait = p.eventDelays[pos].delay
						} else {
							// the end of the pattern; done
							pos, wait, target = 0, 0, nil
						}
					}
					wait--
				}
			}
		}
	}
}

func (p *Pattern) String() string {
	s := ""
	for i, evd := range p.eventDelays {
		if i > 0 {
			s = fmt.Sprintf("%s / ", s)
		}
		s = fmt.Sprintf("%s%s", s, evd)
	}
	return s
}
