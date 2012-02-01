package goop

import (
	"time"
)

type Clock struct {
	eventBuf      chan TargetAndEvent
	bpm           float32
	eventIn       chan Event
	tickReceivers []EventReceiver
}

func NewClock() *Clock {
	eb := make(chan TargetAndEvent, 10*OTHER_CHAN_BUFFER)
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	tr := make([]EventReceiver, 0)
	c := &Clock{eb, 60, ei, tr}
	go c.run()
	return c
}

func (c *Clock) Events() chan<- Event { return c.eventIn }

func (c *Clock) DeferredEvents() chan<- TargetAndEvent { return c.eventBuf }

func (c *Clock) register(r EventReceiver) {
	for _, existing := range c.tickReceivers {
		if existing == r {
			return
		}
	}
	c.tickReceivers = append(c.tickReceivers, r)
}

func (c *Clock) unregister(r EventReceiver) {
	for i, existing := range c.tickReceivers {
		if existing == r {
			c.tickReceivers = append(c.tickReceivers[:i], c.tickReceivers[i+1:]...)
			return
		}
	}
}

func (c *Clock) run() {
	for {
		select {
		case ev := <-c.eventIn:
			switch ev.Name {
			case "bpm":
				c.bpm = ev.Val
			case "kill":
				return
			case "register":
				if r, ok := ev.Arg.(EventReceiver); ok {
					c.register(r)
				}
			case "unregister":
				if r, ok := ev.Arg.(EventReceiver); ok {
					c.unregister(r)
				}
			}
		default:
			//println("clock ticking to", len(c.tickReceivers))
			for i, r := range c.tickReceivers {
				select {
				case r.Events() <- Event{"tick", 0.0, nil}:
					break
				default:
					println("tick receiver", i, "wasn't ready")
					break
				}
			}
			func() {
				for {
					select {
					case et := <-c.eventBuf:
						et.target.Events() <- et.event
					default:
						return
					}
				}
			}()
			<-time.After(time.Duration(int64((60 / c.bpm) * 1e9)))
		}
	}
}
