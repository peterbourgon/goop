package main

import (
	"fmt"
	"math"
)

const (
	SRATE             = 44100 // audio sample rate
	BUFSZ             = 2048  // audio buffer size
	AUDIO_CHAN_BUFFER = 0     // unbuffered
	EVENT_CHAN_BUFFER = 10
)

// generatorChannels are designed to be embedded into Generators
// to satisfy the Node and AudioSender interfaces.
type generatorChannels struct {
	eventIn  chan Event
	audioOut chan []float32
}

func (gc *generatorChannels) Events() chan<- Event {
	return gc.eventIn
}

func (gc *generatorChannels) AudioOut() <-chan []float32 {
	return gc.audioOut
}

func (gc *generatorChannels) Reset() {
	close(gc.audioOut)
	gc.audioOut = make(chan []float32, AUDIO_CHAN_BUFFER)
}

func makeGeneratorChannels() generatorChannels {
	ei := make(chan Event, EVENT_CHAN_BUFFER)
	ao := make(chan []float32, AUDIO_CHAN_BUFFER)
	return generatorChannels{ei, ao}
}

//
//
//

const (
	KeyDown = "keydown"
	KeyUp   = "keyup"
	Gain    = "gain"
)

func KeyDownEvent(n Note) Event { return Event{KeyDown, n.Hz(), n} }
func KeyUpEvent(n Note) Event   { return Event{KeyUp, n.Hz(), n} }
func GainEvent(g float32) Event { return Event{Gain, g, nil} }

// simpleParameters are sufficient to control simple,
// single-mode Generators.
type simpleParameters struct {
	hz    float32
	phase float32 // 0..1
	gain  float32 // 0..1
}

// process applies Events which should have an effect on simpleParameters.
func (sp *simpleParameters) process(ev Event) {
	switch ev.Type {
	case KeyDown:
		sp.hz = ev.Value
	case KeyUp:
		sp.hz = 0.0
	case Gain:
		sp.gain = ev.Value
	}
}

func makeSimpleParameters() simpleParameters {
	return simpleParameters{0.0, 0.0, 1.0}
}

//
//
//

// valueProviders implement a method that yields a float32 audio value.
// Those values are stacked together into a buffer by nextBuffer, and
// typically yielded over an AudioSender port.
//
// Probably every Generator should satisfy the valueProvider interface.
type valueProvider interface {
	nextValue() float32
}

// nextBuffer aggregates BUFSZ values from the valueProvider
// into a single buffer, which it then returns.
func nextBuffer(vp valueProvider) []float32 {
	buf := make([]float32, BUFSZ)
	for i := 0; i < BUFSZ; i++ {
		buf[i] = vp.nextValue()
	}
	return buf
}

//
//
//

// A GeneratorFunction should define output for input [0 .. 1].
// We scale that to the range [0 .. 0.25]. Call that scaled output
// 'F'. We generate a waveform based on phase [0 .. 1] as follows:
//
//    phase < 0.25: output = F
//    phase < 0.50: output = F mirrored horizontally
//    phase < 0.75: output = F mirrored vertically
//    phase < 1.00: output = F mirrored horizontally + vertically
//
// (Thanks to Alexander Surma for the idea on this one.)
type GeneratorFunction func(float32) float32

func nextGeneratorFunctionValue(f GeneratorFunction, hz float32, phase *float32) float32 {
	var val, p float32 = 0.0, 0.0
	switch {
	case *phase <= 0.25:
		p = (*phase - 0.00) * 4
		val = f(p) // no mirror
	case *phase <= 0.50:
		p = (*phase - 0.25) * 4
		val = f(1 - p) // horizontal mirror
	case *phase <= 0.75:
		p = (*phase - 0.50) * 4
		val = -f(p) // vertical mirror
	case *phase <= 1.00:
		p = (*phase - 0.75) * 4
		val = -f(1 - p) // horizontal + vertical mirror
	default:
		panic("unreachable")
	}
	*phase += hz / SRATE
	if *phase > 1.0 {
		*phase -= 1.0
	}
	return val
}

func saw(x float32) float32 {
	return x
}

func sine(x float32) float32 {
	// want only 1/4 sine over range [0..1], so need x/4
	return float32(math.Sin(2 * math.Pi * float64(x/4)))
}

func square(x float32) float32 {
	if x < 0.5 {
		return 1.0
	}
	return 0.0
}

//
//
//

// A simpleGenerator is any generator which can provide audio data
// using only simpleParameters. Handily, this describes a large class of
// generators.
//
// A simpleGenerator also claims singleAncestry; that is,
// it has up to 1 parent and 1 child Node in the Field.
type simpleGenerator struct {
	generatorChannels
	simpleParameters
	nodeName
	singleChild
	noParents
}

func (g *simpleGenerator) String() string {
	return fmt.Sprintf("%.2f hz, gain %.2f", g.hz, g.gain)
}

// generatorLoop is the common function that should drive all Generators
// which contain generatorChannels and are driven by simpleParameters.
//
// It processes certain common Event types, and passes the remaining Events
// off to the given simpleParameters.
//
// It uses the passed valueProvider (typically the concrete Generator itself)
// to generate audio buffers which are pushed over the outgoing audio channel.
func (sg *simpleGenerator) loop(vp valueProvider) {
	for {
		select {
		case ev := <-sg.generatorChannels.eventIn:
			switch ev.Type {
			case Connection, Disconnection:
				break // no parents: ignore

			case Connect:
				n, ok := ev.Arg.(Node)
				if !ok {
					panic("simpleGenerator Connect to non-Node")
				}
				sg.singleChild.Node = n
				sg.generatorChannels.Reset()

			case Disconnect:
				sg.singleChild.Node = nil
				sg.generatorChannels.Reset()

			case Kill:
				sg.singleChild.Node = nil
				sg.generatorChannels.Reset()
				return

			default:
				sg.simpleParameters.process(ev)
			}
		case sg.generatorChannels.audioOut <- nextBuffer(vp):
			break
		}
	}
}

//
//
//

// thanks to #go-nuts skelterjohn for this construction idiom
type SineGenerator struct{ simpleGenerator }

func (g *SineGenerator) String() string {
	return fmt.Sprintf("SineGenerator: %s", g.simpleGenerator.String())
}

func NewSineGenerator(name string) *SineGenerator {
	g := SineGenerator{
		simpleGenerator{
			generatorChannels: makeGeneratorChannels(),
			simpleParameters:  makeSimpleParameters(),
			nodeName:          nodeName(name),
		},
	}
	go g.simpleGenerator.loop(&g)
	return &g
}

// nextValue for a SineGenerator will output a pure sine waveform at the
// frequency described by the simpleParameter's hz parameter.
func (g *SineGenerator) nextValue() float32 {
	return nextGeneratorFunctionValue(sine, g.hz, &g.phase) * g.gain
}
