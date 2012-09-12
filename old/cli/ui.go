package main

import (
	"errors"
	"fmt"
	"github.com/bobappleyard/readline"
	"goop"
	//"github.com/nsf/termbox-go"
	"strconv"
	"strings"
	"time"
)

var (
	CLOCK    *goop.Clock
	MIXER    *goop.Mixer
	NETWORK  *goop.Network
	AUTOCRON int64
)

// In general, UI commands should only generate Events, which
// should be sent to items in the X, or sub-interfaces thereof.
// If you're reflecting something you get out of X to anything
// other than an EventReceiver, you're doing something wrong!

func init() {
	CLOCK = goop.NewClock()
	MIXER = goop.NewMixer()
	NETWORK = goop.NewNetwork(CLOCK)
	AUTOCRON = 1
	NETWORK.Add("clock", CLOCK)
	NETWORK.Add("mixer", MIXER)
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
		case "add", "a":
			doAdd(args)
		case "delete", "del":
			doDel(args)
		case "every":
			doEvery(args) // just an alias for "add cron <autogen ID> ..."
		case "connect", "conn", "con", "c":
			doConnect(args)
		case "disconnect", "disconn", "discon", "dis", "d":
			doDisconnect(args)
		case "register", "reg", "r":
			doRegister(args)
		case "unregister", "unreg", "un", "u":
			doUnregister(args)
		case "fire", "f":
			doFire(args, goop.Deferred)
		case "fire!", "f!":
			doFire(args, goop.Immediately)
		case "push", "pu":
			doPush(args)
		case "pop", "po":
			doPop(args)
		case "ramp":
			doRamp(args)
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
		add(args[1], goop.NewSineGenerator())
	case "square", "sq":
		add(args[1], goop.NewSquareGenerator())
	case "saw":
		add(args[1], goop.NewSawGenerator())
	case "wav":
		if len(args) < 3 {
			fmt.Printf("add wav <name> <filename>\n")
			return
		}
		if g := goop.NewWavGenerator(args[2]); g != nil {
			add(args[1], g)
		} else {
			fmt.Printf("add: wav: failed: probably bad file\n")
		}
	case "lfo", "gainlfo":
		add(args[1], goop.NewGainLFO())
	case "delay":
		add(args[1], goop.NewDelay())
	case "echo":
		add(args[1], goop.NewEcho())
	case "sequencer", "seq":
		s := goop.NewSequencer()
		CLOCK.Events() <- goop.Event{"register", 0.0, s}
		add(args[1], s)
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
		c := goop.NewCron(delay64, uiParse, strings.Join(args[3:], " "))
		CLOCK.Events() <- goop.Event{"register", 0.0, c}
		add(args[1], c)
	default:
		fmt.Printf("add: what?\n")
	}
}

func add(name string, item interface{}) bool {
	if item == nil {
		panic("add nil item!")
	}
	if err := NETWORK.Add(name, item); err != nil {
		fmt.Printf("add: %s: %s\n", name, err)
		return false
	}
	fmt.Printf("add: %s: OK\n", name)
	return true
}

func doDel(args []string) {
	if len(args) < 1 {
		fmt.Printf("del <name>\n")
		return
	}
	name := args[0]
	item, itemErr := NETWORK.Get(name)
	if itemErr != nil {
		fmt.Printf("del: %s: %s\n", name, itemErr)
		return
	}
	if r, ok := item.(goop.EventReceiver); ok {
		CLOCK.Events() <- goop.Event{"unregister", 0.0, r} // safe
	}
	if err := NETWORK.Del(name); err != nil {
		fmt.Printf("del: %s: %s\n", name, err)
		return
	}
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

func doConnect(args []string) {
	if len(args) < 2 {
		fmt.Printf("connect <from> <to>\n")
		return
	}
	NETWORK.Connect(args[0], args[1])
}

func doDisconnect(args []string) {
	if len(args) < 1 {
		fmt.Printf("disconnect <from>\n")
		return
	}
	NETWORK.Disconnect(args[0])
}

func toEventReceiver(name string) (goop.EventReceiver, error) {
	item, itemErr := NETWORK.Get(name)
	if itemErr != nil {
		return nil, errors.New(fmt.Sprintf("%s: %s", name, itemErr))
	}
	r, ok := item.(goop.EventReceiver)
	if !ok {
		return nil, errors.New(fmt.Sprintf("%s: not an Event Receiver", name))
	}
	return r, nil
}

// Register is just like Connect, except for Events rather than audio data.
func doRegister(args []string) {
	if len(args) < 2 {
		fmt.Printf("register <src> <tgt>\n")
		return
	}
	src, dst := args[0], args[1]
	dstReceiver, dstReceiverErr := toEventReceiver(dst)
	if dstReceiverErr != nil {
		fmt.Printf("register: %s: %s\n", dst, dstReceiverErr)
		return
	}
	ev := goop.Event{"register", 0.0, dstReceiver}
	NETWORK.Fire(src, ev, goop.Immediately)
}

func doUnregister(args []string) {
	if len(args) < 1 {
		fmt.Printf("unregister <src> [<tgt>]\n")
		return
	}
	src := args[0]
	if len(args) >= 2 {
		dst := args[1]
		dstReceiver, dstReceiverErr := toEventReceiver(dst)
		if dstReceiverErr != nil {
			fmt.Printf("register: %s: %s\n", dst, dstReceiverErr)
			return
		}
		ev := goop.Event{"unregister", 0.0, dstReceiver}
		NETWORK.Fire(src, ev, goop.Immediately)
	} else {
		NETWORK.Fire(src, goop.Event{"disconnect", 0.0, nil}, goop.Immediately)
	}
}

func doFire(args []string, when int) {
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
	ev := goop.Event{name, float32(val64), nil}
	NETWORK.Fire(receiverName, ev, when)
}

func stringToEvent(s string) (goop.Event, error) {
	// <name> <value>
	toks := strings.Split(strings.TrimSpace(s), " ")
	if len(toks) != 2 {
		return goop.Event{}, errors.New("invalid Event format")
	}
	val64, convertErr := strconv.ParseFloat(toks[1], 32)
	if convertErr != nil {
		return goop.Event{}, convertErr
	}
	return goop.Event{toks[0], float32(val64), nil}, nil
}

func doPush(args []string) {
	if len(args) < 3 {
		fmt.Printf("push <sequencer> <name> <val> [ + <name> <val> ... ]\n")
		return
	}
	target, args := args[0], args[1:]
	eventStrings := strings.Split(strings.Join(args, " "), "+")
	slot := goop.Slot{}
	for _, eventString := range eventStrings {
		ev, err := stringToEvent(eventString)
		if err != nil {
			fmt.Printf("push: %s: %s\n", eventString, err)
			return
		}
		slot = append(slot, ev)
	}
	ev := goop.Event{"push", 0.0, slot}
	NETWORK.Fire(target, ev, goop.Immediately)
	fmt.Printf("push: %d into next slot of %s\n", len(slot), target)
}

func doPop(args []string) {
	if len(args) < 1 {
		fmt.Printf("pop <sequencer>\n")
		return
	}
	NETWORK.Fire(args[0], goop.Event{"pop", 0.0, nil}, goop.Immediately)
}

func doRamp(args []string) {
	if len(args) < 5 {
		fmt.Printf("ramp <target> <eventname> <begin val> <end val> <sec> [<divisions>]\n")
		return
	}
	target, eventName := args[0], args[1]
	_, rErr := toEventReceiver(target)
	if rErr != nil {
		fmt.Printf("ramp: %s: %s\n", target, rErr)
		return
	}
	begin64, beginErr := strconv.ParseFloat(args[2], 32)
	if beginErr != nil {
		fmt.Printf("ramp: begin: %s\n", beginErr)
		return
	}
	end64, endErr := strconv.ParseFloat(args[3], 32)
	if endErr != nil {
		fmt.Printf("ramp: end: %s\n", endErr)
		return
	}
	sec64, secErr := strconv.ParseFloat(args[4], 32)
	if secErr != nil {
		fmt.Printf("ramp: duration: %s\n", secErr)
		return
	}
	div := 100
	if len(args) >= 6 {
		divTry, divErr := strconv.ParseInt(args[5], 10, 64)
		if divErr != nil {
			fmt.Printf("ramp: divisions: %s\n", divErr)
		}
		div = int(divTry)
	}
	go func() {
		for i := 0; i < div+1; i++ {
			thisVal := float32(begin64 + (float64(i) * ((end64 - begin64) / float64(div))))
			ev := goop.Event{eventName, thisVal, nil}
			NETWORK.Fire(target, ev, goop.Immediately)
			<-time.After(time.Duration(sec64 / float64(div) * 1e9))
		}
	}()
	fmt.Printf("ramping %s %s from %.2f to %.2f over %.2fs in %d steps\n", target, eventName, begin64, end64, sec64, div)
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
	for _, name := range NETWORK.Names() {
		what, details := "unknown", ""
		o, err := NETWORK.Get(name)
		if err != nil {
			continue
		}
		switch x := o.(type) {
		case *goop.Mixer:
			what, details = "the mixer", fmt.Sprintf("%s", x)
		case *goop.Clock:
			what, details = "the clock", fmt.Sprintf("%s", x)
		case *goop.SineGenerator:
			what, details = "sine generator", fmt.Sprintf("%s", x)
		case *goop.SquareGenerator:
			what, details = "square generator", fmt.Sprintf("%s", x)
		case *goop.SawGenerator:
			what, details = "sawtooth generator", fmt.Sprintf("%s", x)
		case *goop.GainLFO:
			what, details = "gain LFO", fmt.Sprintf("%s", x)
		case *goop.Delay:
			what, details = "delay", fmt.Sprintf("%s", x)
		case *goop.Echo:
			what, details = "echo", fmt.Sprintf("%s", x)
		case *goop.Cron:
			what, details = "cron", fmt.Sprintf("%s", x)
		case *goop.Sequencer:
			what, details = "sequencer", fmt.Sprintf("%s", x)
		default:
			what, details = "unknown", fmt.Sprintf("%s", x)
		}
		msg := fmt.Sprintf(" %s - %s", name, what)
		if details != "" {
			msg = fmt.Sprintf("%s (%s)", msg, details)
		}
		fmt.Printf("%s\n", msg)
	}
}
