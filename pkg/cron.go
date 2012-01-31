package goop

import ()

type Cron struct {
	parser  func(string) bool
	delay   int64 // ticks
	cmd     string
	eventIn chan Event
}

func (c *Cron) Events() chan<- Event {
	return c.eventIn
}

func NewCron(delay int64, parser func(string) bool, cmd string) *Cron {
	c := &Cron{parser, delay, cmd, make(chan Event, OTHER_CHAN_BUFFER)}
	go c.cronLoop()
	return c
}

func (c *Cron) cronLoop() {
	var ticks int64 = 0
	for {
		select {
		case ev := <-c.eventIn:
			switch ev.name {
			case "kill":
				close(c.eventIn)
				return
			case "tick":
				ticks++
				if ticks >= c.delay {
					ticks = 0
					c.parser(c.cmd)
				}
			}
		}
	}
}
