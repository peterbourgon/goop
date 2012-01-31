package goop

import (
	"sync"
	"time"
)

type Clock struct {
	eventBuf      chan TargetAndEvent
	bpm           float32
	eventIn       chan Event
	tickReceivers map[string]EventReceiver
	mtx           sync.Mutex
}

func NewClock() *Clock {
	eb := make(chan TargetAndEvent, 10*OTHER_CHAN_BUFFER)
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	tr := make(map[string]EventReceiver)
	c := &Clock{eb, 60, ei, tr, sync.Mutex{}}
	go c.run()
	return c
}

func (c *Clock) Events() chan<- Event {
	return c.eventIn
}

func (c *Clock) Queue(tev TargetAndEvent) {
	c.eventBuf <- tev
}

func (c *Clock) DeferredEvents() chan<- TargetAndEvent { return c.eventBuf }

func (c *Clock) Register(name string, r EventReceiver) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.tickReceivers[name] = r
}

func (c *Clock) Unregister(name string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	delete(c.tickReceivers, name)
}

func (c *Clock) run() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for {
		select {
		case ev := <-c.eventIn:
			switch ev.name {
			case "bpm":
				c.bpm = ev.val
			case "kill":
				return
			}
		default:
			for name, r := range c.tickReceivers {
				select {
				case r.Events() <- Event{"tick", 0.0, nil}:
					break
				default:
					println("tick receiver", name, "wasn't ready")
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
			c.mtx.Unlock()
			<-time.After(time.Duration(int64((60 / c.bpm) * 1e9)))
			c.mtx.Lock()
		}
	}
}
