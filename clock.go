package main

import (
	"time"
)

const (
	Tick = "tick"
	BPM  = "bpm"
)

func TickEvent(i int, c *Clock) Event { return Event{Tick, float32(i), c} }

type Clock struct {
	nodeName
	noParents
	noChildren

	bpm     float32
	f       Field
	i       int
	eventIn chan Event
}

func NewClock(f Field) *Clock {
	c := &Clock{
		nodeName: "clock",
		bpm:      120,
		f:        f,
		i:        0,
		eventIn:  make(chan Event),
	}
	go c.loop()
	return c
}

// Events satisfies the Node interface for Clock
func (c *Clock) Events() chan<- Event { return c.eventIn }

func (c *Clock) loop() {
	d := bpm2duration(c.bpm)
	D("clock operating at %s", d)
	t := time.NewTicker(bpm2duration(c.bpm))
	for {
		select {
		case <-t.C:
			go c.f.Broadcast(TickEvent(c.i, c)) // broadcast includes self
			c.i++

		case ev := <-c.eventIn:
			switch ev.Type {
			case BPM:
				c.bpm = ev.Value
				t.Stop()
				t = time.NewTicker(bpm2duration(c.bpm))
			case Kill:
				t.Stop()
				return
			}
		}
	}
}

func bpm2duration(bpm float32) time.Duration {
	return time.Duration(float32(time.Minute) / bpm)
}
