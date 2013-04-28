package goop

import (
	"github.com/peterbourgon/field"
	"sync"
	"time"
)

type Clock struct {
	sync.RWMutex
	events   chan Event
	bpm      float32
	children map[string]EventReceiver
}

type BPMChangeEvent struct{ NewBPM float32 }
type ClockTickEvent struct{}

func NewClock(f *field.Field) *Clock {
	c := &Clock{
		events:   make(chan Event),
		bpm:      60,
		children: map[string]EventReceiver{},
	}
	go c.loop()
	return c
}

func (c *Clock) Name() string { return "clock" }

func (c *Clock) Attributes() map[string]interface{} {
	return map[string]interface{}{"shape": "box"}
}

func (c *Clock) Events() chan<- Event { return c.events }

func (c *Clock) DownstreamConnect(n field.Node) {
	if receiver, ok := n.(EventReceiver); ok {
		c.Lock()
		defer c.Unlock()
		// TODO safety check
		c.children[n.Name()] = receiver
	}
}

func (c *Clock) DownstreamDisconnect(n field.Node) {
	if _, ok := n.(EventReceiver); ok {
		c.Lock()
		defer c.Unlock()
		// TODO safety check
		delete(c.children, n.Name())
	}
}

func (c *Clock) loop() {
	t := c.updateBPM(60) // TODO fix
	for {
		select {
		case ev := <-c.events:
			switch e := ev.(type) {
			case BPMChangeEvent:
				t = c.updateBPM(e.NewBPM)
			}
		case <-t:
			c.broadcast()
		}
	}
}

func (c *Clock) updateBPM(bpm float32) <-chan time.Time {
	c.Lock()
	defer c.Unlock()
	c.bpm = bpm
	return time.Tick(time.Duration(60.0/bpm) * time.Second)
}

func (c *Clock) broadcast() {
	c.RLock()
	defer c.RUnlock()
	for name, receiver := range c.children {
		select {
		case receiver.Events() <- ClockTickEvent{}:
			break
		default:
			D("Clock: tick to %s BLOCKED", name)
		}
	}
}
