package main

import (
	"fmt"
)

// effectChannels are designed to be embedded into simple effects,
// which should accept exactly one input audio channel, manipulate it
// somehow, and provide the manipulated output on exactly one output
// audio channel. Simple effects should also respond to (at minimum)
// a certain subset of Events, so effectChannels also implements the
// EventReceiver interface, and handles a subset of Event types itself.
type effectChannels struct {
	eventIn  chan Event
	audioIn  <-chan []float32
	audioOut chan []float32
}

func makeEffectChannels() effectChannels {
	return effectChannels{
		eventIn: make(chan Event, EVENT_CHAN_BUFFER),
		// audioIn initially nil
		audioOut: make(chan []float32, AUDIO_CHAN_BUFFER),
	}
}

// Events() satisfies the Node interface.
func (ec *effectChannels) Events() chan<- Event {
	return ec.eventIn
}

// AudioOut() satisfies the AudioSender interface.
func (ec *effectChannels) AudioOut() <-chan []float32 {
	return ec.audioOut
}

// Reset() satisfies the AudioSender interface.
func (ec *effectChannels) Reset() {
	close(ec.audioOut)
	ec.audioOut = make(chan []float32, AUDIO_CHAN_BUFFER)
}

// The important thing to consider here is how data flows through the network.
// It's a pull-based system. Audio producers will happily produce as much data
// as they can cram into their output channels. The consumer at the end of the
// data chain (ie. the mixer) is ultimately responsible for draining the
// channels. So, it controls the rate at which audio data should be produced.
//
// How do we handle disconnections? Specifically, when a disconnect signal is
// sent to a module, it closes its audio output channels, and re-creates them.
// How will that be signaled to downstream receivers? Detecting a channel close
// is simple enough, on the receiving end:
//
//     buf, ok := <-audioIn
//     if !ok {
//	       // closed
//     }
//
// But that implies that checking if a channel is closed will yield actual data
// if it's not. We don't want that behavior. So, receivers need to throttle
// their channel-drains to at most one per every available slot in their
// downstream channel-fills. The simple way to do that is to buffer one
// []float32 in each module.
//
//     var buf []float32 = nil
//     for {
//         if buf == nil {
//             buf := <-audioIn
//             process(buf)
//         }
//         select {
//         case audioOut <-buf:
//             buf = nil
//         case ev := <-eventIn:
//             process(ev)
//         }
//     }
//
// Should work...

type simpleEffect struct {
	nodeName
	singleAncestry
	effectChannels
}

func makeSimpleEffect(name string) simpleEffect {
	return simpleEffect{
		nodeName:       nodeName(name),
		effectChannels: makeEffectChannels(),
	}
}

func (se *simpleEffect) loop(ep eventProcessor, ap audioProcessor) {
	var buf []float32 = nil
	for {
		// as described above: buffer exactly 1 audio buffer locally
		var ok bool = false
		if se.effectChannels.audioIn != nil && buf == nil {
			if buf, ok = <-se.effectChannels.audioIn; ok {
				ap.processAudio(buf)
			} else {
				se.effectChannels.audioIn = nil // closed
			}
		}

		select {
		case ev := <-se.effectChannels.eventIn:
			switch ev.Type {
			case Connect: // downstream
				node, nodeOk := ev.Arg.(Node)
				if !nodeOk {
					break
				}
				se.ChildNode = node
				D("simpleEffect got valid Connect: Parents=%d Children=%d", len(se.Parents()), len(se.Children()))
				// nothing to do re: audio channels, really

			case Disconnect: // downstream
				se.ChildNode = nilNode // TODO could do more thorough checking
				se.effectChannels.Reset()

			case Connection: // upstream
				node, nodeOk := ev.Arg.(Node)
				if !nodeOk {
					D("simpleEffect got Connection from non-Node")
					break
				}
				se.ParentNode = node
				sender, senderOk := ev.Arg.(AudioSender)
				if !senderOk {
					D("simpleEffect got Connection from non-AudioSender")
					break
				}
				se.effectChannels.audioIn = sender.AudioOut()
				D("simpleEffect got valid Connection: Parents=%d Children=%d", len(se.Parents()), len(se.Children()))

			case Disconnection: // upstream
				se.ParentNode = nilNode
				// nothing to do re: audio channels, really

			case Kill:
				se.ChildNode = nilNode
				se.ParentNode = nilNode
				se.effectChannels.Reset()
				return

			default:
				ep.processEvent(ev)
			}

		case se.effectChannels.audioOut <- buf:
			buf = nil // need a new one, now
		}
	}
}

// The audioProcessor interface is designed to be implemented by concrete
// Effects. The processAudio method should manipulate the passed audio
// buffer in-place.
type audioProcessor interface {
	processAudio(buf []float32)
}

//
//
//

// The GainLFO is an Effect which cycles the gain of the audio signal
// from min to max at a rate of hz.
type GainLFO struct {
	simpleEffect

	min   float32
	max   float32
	hz    float32
	phase float32
}

func NewGainLFO(name string) *GainLFO {
	e := &GainLFO{
		simpleEffect: makeSimpleEffect(name),

		min:   0.0,
		max:   1.0,
		hz:    1.0,
		phase: 0.0,
	}
	go e.simpleEffect.loop(e, e)
	return e
}

func NewGainLFONode(name string) Node {
	return Node(NewGainLFO(name))
}

func (e *GainLFO) String() string {
	return fmt.Sprintf("[%s: %.2f-%.2f @ %.2f hz]", NodeLabel(e), e.min, e.max, e.hz)
}

func (e *GainLFO) Kind() string { return "gain LFO" }

// GainLFO's processEvent manages changes to min, max and hz values.
func (e *GainLFO) processEvent(ev Event) {
	switch ev.Type {
	case "min":
		e.min = ev.Value
	case "max":
		e.max = ev.Value
	case "hz":
		e.hz = ev.Value
	}
}

// GainLFO's processAudio changes the amplitude of the buffer.
func (e *GainLFO) processAudio(buf []float32) {
	for i, v := range buf {
		raw := nextGeneratorFunctionValue(sine, e.hz, &e.phase)
		mod := ((e.max - e.min) * raw) + e.min
		buf[i] = mod * v
	}
}

//
//
//

// The Delay is an Effect which buffers incoming audio data for (approximately)
// delay seconds before sending it out the outgoing audio channel.
type Delay struct {
	simpleEffect

	history chan []float32
	delay   float32
}

func NewDelay(name string) *Delay {
	initialDelay := float32(1.0) // sec
	depth := int64((SRATE * initialDelay) / BUFSZ)
	e := &Delay{
		simpleEffect: makeSimpleEffect(name),

		history: make(chan []float32, depth),
		delay:   initialDelay,
	}
	go e.simpleEffect.loop(e, e)
	return e
}

func NewDelayNode(name string) Node {
	return Node(NewDelay(name))
}

func (e *Delay) String() string {
	return fmt.Sprintf("[%s: %.2fs]", NodeLabel(e), e.delay)
}

func (e *Delay) Kind() string { return "Delay" }

// Delay's processEvent manages changes to the delay parameter.
func (e *Delay) processEvent(ev Event) {
	switch ev.Type {
	case "delay":
		e.delay = ev.Value
		depth := int64((SRATE * e.delay) / BUFSZ)
		e.history = make(chan []float32, depth)
	}
}

func (e *Delay) processAudio(buf []float32) {
	select {
	case e.history <- buf:
		// not yet full, so we shouldn't output anything
		for i, _ := range buf {
			buf[i] = 0.0
		}
	default:
		// full, so pop + push
		outBuf := <-e.history
		e.history <- buf
		buf = outBuf
	}
}

//
//
//

// An Echo is just a Delay with different processAudio logic.
type Echo struct {
	Delay
	wet float32 // 0..1
}

func NewEcho(name string) *Echo {
	initialDelay := float32(1.0) // sec
	depth := int64((SRATE * initialDelay) / BUFSZ)
	e := &Echo{
		Delay: Delay{
			simpleEffect: makeSimpleEffect(name),
			history:      make(chan []float32, depth),
			delay:        initialDelay,
		},
		wet: 0.5,
	}
	go e.simpleEffect.loop(e, e)
	return e
}

func NewEchoNode(name string) Node {
	return Node(NewEcho(name))
}

func (e *Echo) String() string {
	return fmt.Sprintf("[%s: %.2fs]", NodeLabel(e), e.delay)
}

func (e *Echo) Kind() string { return "Echo" }

func (e *Echo) processEvent(ev Event) {
	switch ev.Type {
	case "wet":
		if ev.Value >= 0.0 && ev.Value <= 1 {
			e.wet = ev.Value
		}
	default:
		e.Delay.processEvent(ev)
	}
}

func (e *Echo) processAudio(buf []float32) {
	select {
	case e.history <- buf:
		// not yet full, so we shouldn't output anything
		break
	default:
		outBuf := <-e.history // pop
		e.history <- buf      // push
		if len(buf) != len(outBuf) {
			break
		}
		for i, val := range buf {
			buf[i] = (e.wet * val) + (outBuf[i] * (1 - e.wet))
		}
	}
}
