package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type EventDelay struct {
	ev    Event
	delay float32 // sec
}

func newEventDelay(s string) (EventDelay, error) {
	toks := strings.Split(strings.TrimSpace(s), " ")
	if len(toks) != 3 {
		return EventDelay{}, errors.New(fmt.Sprintf("%s: not enough tokens\n", s))
	}
	name := toks[0]
	val64, valErr := strconv.ParseFloat(toks[1], 32)
	if valErr != nil {
		return EventDelay{}, errors.New(fmt.Sprintf("bad event value: %s", valErr))
	}
	delay64, delayErr := strconv.ParseFloat(toks[2], 32)
	if delayErr != nil {
		return EventDelay{}, errors.New(fmt.Sprintf("bad delay value: %s", delayErr))
	}
	return EventDelay{Event{name, float32(val64), nil}, float32(delay64)}, nil
}

func (evd EventDelay) String() string {
	return fmt.Sprintf("%s %.3f %.3f", evd.ev.name, evd.ev.val, evd.delay)
}

type Pattern []EventDelay

func NewPattern(s string) *Pattern {
	p := make(Pattern, 0)
	toks := strings.Split(s, "/")
	for i, tok := range toks {
		if evd, err := newEventDelay(tok); err == nil {
			p = append(p, evd)
		} else {
			fmt.Printf("pattern: %d: %s", i, err)
		}
	}
	return &p
}

func (p *Pattern) Fire(r EventReceiver) {
	go func() {
		for _, evd := range *p {
			r.Events() <- evd.ev
			<-time.After(time.Duration(evd.delay * 1e9))
		}
	}()
}

func (p *Pattern) String() string {
	s := ""
	for i, evd := range *p {
		if i > 0 {
			s = fmt.Sprintf("%s / ", s)
		}
		s = fmt.Sprintf("%s%s", s, evd)
	}
	return s
}
