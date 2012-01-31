package main

import (
	"errors"
	"fmt"
	"github.com/bobappleyard/readline"
	//"github.com/nsf/termbox-go"
	"strconv"
	"strings"
	"time"
)

var (
	X        map[string]interface{}
	AUTOCRON int64
	CLOCK    *Clock
)

// In general, UI commands should only generate Events, which
// should be sent to items in the X, or sub-interfaces thereof.
// If you're reflecting something you get out of X to anything
// other than an EventReceiver, you're doing something wrong!

func init() {
	X = make(map[string]interface{})
	CLOCK = NewClock()
	add("clock", CLOCK)
	add("mixer", MIXER)
	AUTOCRON = 1
}

func uiParse(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if s[0] == '#' {
		return true
	}
	for _, cmd := range strings.Split(s, ";") {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		toks := strings.Split(cmd, " ")
		cmd, args := toks[0], toks[1:]
		switch cmd {
		case "quit":
			return false
		case "add":
			doAdd(args)
		case "del", "delete":
			doDel(args)
		case "every":
			doEvery(args) // just an alias for "add cron <autogen ID> ..."
		case "connect":
			doConnect(args)
		case "disconnect":
			doDisconnect(args)
		case "fire":
			doFire(args)
		case "firepattern", "firep", "fp":
			doFirePattern(args)
		case "stopall":
			doStopall(args)
		case "sleep":
			doSleep(args)
		case "info", "i":
			doInfo(args)
		default:
			fmt.Printf("%s: ?\n", cmd)
		}
	}
	return true
}

func uiParseHistory(s string) bool {
	rc := uiParse(s)
	readline.AddHistory(s)
	return rc
}

func uiLoop() {
	for {
		line := readline.String("> ")
		rc := uiParseHistory(line)
		if !rc {
			break
		}
	}
}

func doAdd(args []string) {
	if len(args) < 2 {
		fmt.Printf("add <what> <name>\n")
		return
	}
	switch args[0] {
	case "sin", "sine", "sinegenerator":
		add(args[1], NewSineGenerator())
	case "square", "sq":
		add(args[1], NewSquareGenerator())
	case "saw":
		add(args[1], NewSawGenerator())
	case "wav":
		if len(args) < 3 {
			fmt.Printf("add wav <name> <filename>\n")
			return
		}
		if g := NewWavGenerator(args[2]); g != nil {
			add(args[1], g)
		} else {
			fmt.Printf("add: wav: failed: probably bad file\n")
		}
	case "lfo", "gainlfo":
		add(args[1], NewGainLFO())
	case "delay":
		add(args[1], NewDelay())
	case "cron":
		if len(args) < 4 {
			fmt.Printf("add cron <name> <delay> <cmd...>\n")
			return
		}
		delay64, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Printf("%s: invalid delay: %s\n", args[2], err)
			return
		}
		c := NewCron(delay64, uiParse, strings.Join(args[3:], " "))
		add(args[1], c)
	case "pattern":
		if len(args) < 3 {
			fmt.Printf("add pattern <name> <event-delay> / ...\n")
			return
		}
		p := NewPattern(strings.Join(args[2:], " "))
		add(args[1], p)
	default:
		fmt.Printf("add: what?\n")
	}
}

func add(name string, item interface{}) bool {
	if _, exists := X[name]; exists {
		fmt.Printf("add: %s: exists\n", name)
		return false
	}
	X[name] = item
	if cron, ok := item.(*Cron); ok {
		CLOCK.Register(name, cron)
	}
	/* temporarily disable registering of things to the Clock
	if r, ok := item.(EventReceiver); ok {
		CLOCK.Register(name, r)
	}
	*/
	fmt.Printf("add: %s: OK\n", name)
	return true
}

func doDel(args []string) {
	if len(args) < 1 {
		fmt.Printf("del <name>\n")
		return
	}
	name := args[0]
	item, ok := X[name]
	if !ok {
		fmt.Printf("del: %s: no such thing\n", name)
	}
	if r, ok := item.(EventReceiver); ok {
		CLOCK.Unregister(name) // just in case
		r.Events() <- Event{"kill", 0.0, nil}
	}
	delete(X, name)
	fmt.Printf("del: %s: OK\n", name)
}

func doEvery(args []string) {
	if len(args) < 3 {
		fmt.Printf("every <delay> <cmd...>\n")
		return
	}
	cronName := fmt.Sprintf("c%d", AUTOCRON)
	AUTOCRON++
	args = append([]string{"cron", cronName}, args...)
	doAdd(args)
}

func toEventReceiver(name string) (EventReceiver, error) {
	item, itemOk := X[name]
	if !itemOk {
		return nil, errors.New(fmt.Sprintf("%s doesn't exist", name))
	}
	receiver, receiverOk := item.(EventReceiver)
	if !receiverOk {
		return nil, errors.New(fmt.Sprintf("%s doesn't receive events", name))
	}
	return receiver, nil
}

func doConnect(args []string) {
	if len(args) < 2 {
		fmt.Printf("connect <from> <to>\n")
		return
	}
	fromName, toName := args[0], args[1]
	fromItem, fromOk := X[fromName]
	if !fromOk {
		fmt.Printf("%s doesn't exist\n", fromName)
		return
	}
	_, fromSenderOk := fromItem.(AudioSender)
	if !fromSenderOk {
		fmt.Printf("%s doesn't send audio\n", fromName)
		return
	}
	toItem, toOk := X[toName]
	if !toOk {
		fmt.Printf("%s doesn't exist\n", toName)
		return
	}
	toReceiver, toReceiverOk := toItem.(EventReceiver)
	if !toReceiverOk {
		fmt.Printf("%s can't receive (connection) events\n", toName)
		return
	}
	// Should be buffer this one?
	toReceiver.Events() <- Event{"receivefrom", 0.0, fromItem}
}

func doDisconnect(args []string) {
	if len(args) < 1 {
		fmt.Printf("disconnect <from>\n")
		return
	}
	fromName := args[0]
	item, itemOk := X[fromName]
	if !itemOk {
		fmt.Printf("disconnect: %s: doesn't exist\n", fromName)
		return
	}
	r, ok := item.(EventReceiver)
	if !ok {
		fmt.Printf("disconnect: %s: can't receive events\n", fromName)
		return
	}
	// send it a Reset event (note: don't Reset directly!!)
	r.Events() <- Event{"disconnect", 0.0, nil}
}

func doFire(args []string) {
	if len(args) < 3 {
		fmt.Printf("fire <name> <val> <where>\n")
		return
	}
	name := args[0]
	val64, err := strconv.ParseFloat(args[1], 32)
	if err != nil {
		fmt.Printf("fire: %s: invalid value\n", args[1])
		return
	}
	receiverName := args[2]
	receiverItem, receiverItemOk := X[receiverName]
	if !receiverItemOk {
		fmt.Printf("fire: %s: invalid\n", receiverName)
		return
	}
	receiver, receiverOk := receiverItem.(EventReceiver)
	if !receiverOk {
		fmt.Printf("fire: %s: can't receive events", receiverName)
		return
	}
	fire(Event{name, float32(val64), nil}, receiver)
}

func doFirePattern(args []string) {
	if len(args) < 2 {
		fmt.Printf("firepattern <name> <target>\n")
		return
	}
	patternName, receiverName := args[0], args[1]
	patternItem, patternItemOk := X[patternName]
	if !patternItemOk {
		fmt.Printf("firepattern: %s: invalid\n", patternName)
		return
	}
	pattern, patternOk := patternItem.(*Pattern)
	if !patternOk {
		fmt.Printf("firepattern: %s: not a pattern\n", patternName)
		return
	}
	receiverItem, receiverItemOk := X[receiverName]
	if !receiverItemOk {
		fmt.Printf("firepattern: %s: invalid\n", receiverName)
		return
	}
	receiver, receiverOk := receiverItem.(EventReceiver)
	if !receiverOk {
		fmt.Printf("firepattern: %s: can't receive events\n", receiverName)
		return
	}
	pattern.Fire(receiver)
}

func fire(ev Event, r EventReceiver) {
	CLOCK.Queue(TargetAndEvent{r.Events(), ev})
}

func doStopall(args []string) {
	MIXER.DropAll()
}

func doSleep(args []string) {
	if len(args) < 1 {
		fmt.Printf("sleep <sec>\n")
		return
	}
	delay64, err := strconv.ParseFloat(args[0], 32)
	if err != nil {
		fmt.Printf("sleep: %s: invalid delay\n", args[0])
		return
	}
	<-time.After(time.Duration(int64(delay64 * 1e9)))
}

func doInfo(args []string) {
	for name, o := range X {
		what, details := "unknown", ""
		switch x := o.(type) {
		case *Mixer:
			what = "the mixer"
			details = fmt.Sprintf("%d connections", len(x.chans))
		case *Clock:
			what = "the clock"
			details = fmt.Sprintf("at %.2f BPM", x.bpm)
		case *SineGenerator:
			what = "sine generator"
			details = fmt.Sprintf("%.2f hz", x.hz)
		case *SquareGenerator:
			what = "square generator"
			details = fmt.Sprintf("%.2f hz", x.hz)
		case *WavGenerator:
			what = "wav generator"
		case *GainLFO:
			what = "gain LFO"
			details = fmt.Sprintf("%.2f-%.2f @%.2f hz", x.min, x.max, x.hz)
		case *Delay:
			what = "delay"
			details = fmt.Sprintf("%.2fs", x.delay)
		case *Cron:
			what = "cron"
			details = fmt.Sprintf("every %d ticks, %s", x.delay, x.cmd)
		case *Pattern:
			what = "pattern"
			details = fmt.Sprintf("%s", x)
		}
		msg := fmt.Sprintf(" %s - %s", name, what)
		if details != "" {
			msg = fmt.Sprintf("%s (%s)", msg, details)
		}
		fmt.Printf("%s\n", msg)
	}
}
