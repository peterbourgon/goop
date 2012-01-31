package goop

import ()

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
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	var ai <-chan []float32 = nil
	ao := make(chan []float32, AUDIO_CHAN_BUFFER)
	return effectChannels{ei, ai, ao}
}

func (ec *effectChannels) Events() chan<- Event {
	return ec.eventIn
}

func (ec *effectChannels) AudioOut() <-chan []float32 {
	return ec.audioOut
}

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

// effectLoop manages the effectChannels.
func (ec *effectChannels) effectLoop(ep eventProcessor, ap audioProcessor) {
	var buf []float32 = nil
	for {
		var ok bool = false
		if ec.audioIn != nil && buf == nil {
			if buf, ok = <-ec.audioIn; ok {
				ap.processAudio(buf)
			} else {
				ec.audioIn = nil // closed
			}
		}
		select {
		case ev := <-ec.eventIn:
			switch ev.Name {
			case "receivefrom":
				if sender, ok := ev.Arg.(AudioSender); ok {
					ec.audioIn = sender.AudioOut()
				}
			case "disconnect":
				ec.Reset()
			case "kill":
				ec.Reset()
				return
			default:
				ep.processEvent(ev)
			}
		case ec.audioOut <- buf:
			buf = nil // need a new one, now
		}
	}
}

// The eventProcessor interface is designed to be implemented by concrete
// effects. The (initial) effectChannels goroutine, which is responsible for
// (among other things) processing a shared sub-set of Event types directly,
// will pass off all non-handled Event types to this function.
type eventProcessor interface {
	processEvent(e Event)
}

// The audioProcessor interface is designed to be implemented by concrete
// effects. The processAudio method should manipulate the passed audio
// buffer in-place.
type audioProcessor interface {
	processAudio(buf []float32)
}

// The GainLFO is an effect which cycles the gain of the audio signal
// from min to max at a rate of hz.
type GainLFO struct {
	effectChannels
	min   float32
	max   float32
	hz    float32
	phase float32
}

func NewGainLFO() *GainLFO {
	ec := makeEffectChannels()
	e := &GainLFO{ec, 0.0, 1.0, 1.0, 0.0}
	go e.effectLoop(e, e)
	return e
}

// GainLFO's processEvent manages changes to min, max and hz values.
func (e *GainLFO) processEvent(ev Event) {
	switch ev.Name {
	case "min":
		e.min = ev.Val
	case "max":
		e.max = ev.Val
	case "hz":
		e.hz = ev.Val
	}
}

func (e *GainLFO) processAudio(buf []float32) {
	for i, v := range buf {
		raw := nextSineValue(e.hz, &e.phase)
		mod := ((e.max - e.min) * raw) + e.min
		buf[i] = mod * v
	}
}

// The Delay is an effect which buffers incoming audio data for (approximately)
// delay seconds before sending it out the outgoing audio channel.
type Delay struct {
	effectChannels
	history chan []float32
	delay   float32
}

func NewDelay() *Delay {
	ec := makeEffectChannels()
	e := &Delay{ec, nil, 0.0}
	e.resetDelay(1.0)
	go e.effectLoop(e, e)
	return e
}

// resetDelay takes a new delay parameter, and resets the internal state of
// the Delay effect so that the new delay is applied.
func (e *Delay) resetDelay(d float32) {
	e.delay = d
	depth := int64((SRATE * e.delay) / BUFSZ)
	e.history = make(chan []float32, depth)
}

// Delay's processEvent manages changes to the delay parameter.
func (e *Delay) processEvent(ev Event) {
	switch ev.Name {
	case "delay":
		e.resetDelay(ev.Val)
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
