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
