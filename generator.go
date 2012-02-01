package goop

import (
	"math"
	"fmt"
)

// generatorChannels are designed to be embedded into Generators
// to satisfy relevant interfaces.
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
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	ao := make(chan []float32, AUDIO_CHAN_BUFFER)
	return generatorChannels{ei, ao}
}

// simpleParameters are sufficient to control simple, single-mode
// Generators. They include hz, phase and gain.
type simpleParameters struct {
	hz    float32
	phase float32 // 0..1
	gain  float32 // 0..1
}

// process applies Events which should have an effect on simpleParameters.
func (sp *simpleParameters) process(ev Event) {
	switch ev.Name {
	case "keydown":
		sp.hz = ev.Val
	case "keyup":
		sp.hz = 0.0
	case "gain":
		sp.gain = ev.Val
	}
}

func makeSimpleParameters() simpleParameters {
	return simpleParameters{0.0, 0.0, 1.0}
}

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

// generatorLoop is the common function that should drive all Generators
// which contain generatorChannels and are driven by simpleParameters.
// 
// It processes certain common Event types, and passes the remaining Events
// off to the given simpleParameters.
//
// It uses the passed valueProvider (typically the concrete Generator itself)
// to generate audio buffers which are pushed over the outgoing audio channel.
func (gc *generatorChannels) generatorLoop(sp *simpleParameters, vp valueProvider) {
	for {
		select {
		case ev := <-gc.eventIn:
			switch ev.Name {
			case "disconnect":
				gc.Reset()
			case "kill":
				gc.Reset()
				return
			default:
				sp.process(ev)
			}
		case gc.audioOut <- nextBuffer(vp):
			break
		}
	}
}

// nextSineValue computes the next value in a simple sine wave as defined by
// the hz value, with an offset into the waveform as specified by the phase.
// The function updates phase.
func nextSineValue(hz float32, phase *float32) float32 {
	val := float32(math.Sin(float64(2.0 * *phase * math.Pi)))
	*phase += hz / SRATE
	if *phase > 1.0 {
		*phase -= 1.0
	}
	return val
}

// nextSawValue computes the next value in a simple saw wave.
func nextSawValue(hz float32, phase *float32) float32 {
	var val float32 = 0.0
	switch {
	case *phase < 0.25:
		val = *phase * 4.0
	case *phase < 0.75:
		val = 1 - ((*phase - 0.25) * 4)
	case *phase <= 1.0:
		val = -1 + ((*phase - 0.75) * 4)
	default:
		panic("oh no")
	}
	*phase += hz / SRATE
	if *phase > 1.0 {
		*phase -= 1.0
	}
	return float32(val)
}

// A simpleGenerator is any generator which can provide audio data 
// using only simpleParameters. Handily, this describes a large class of
// Generators.
type simpleGenerator struct {
	generatorChannels
	simpleParameters
}

func (g *simpleGenerator) String() string {
	return fmt.Sprintf("%.2f hz, gain %.2f", g.hz, g.gain)
}

// thanks to #go-nuts skelterjohn for this construction idiom
type SineGenerator struct { simpleGenerator }

func NewSineGenerator() *SineGenerator {
	g := SineGenerator{ simpleGenerator{makeGeneratorChannels(), makeSimpleParameters()} }
	go g.generatorLoop(&g.simpleParameters, &g)
	return &g
}

// nextValue for a SineGenerator will output a pure sine waveform at the
// frequency described by the simpleParameter's hz parameter.
func (g *SineGenerator) nextValue() float32 {
	return nextSineValue(g.hz, &g.phase) * g.gain
}

type SquareGenerator struct { simpleGenerator }

func NewSquareGenerator() *SquareGenerator {
	g := SquareGenerator{ simpleGenerator{makeGeneratorChannels(), makeSimpleParameters()} }
	go g.generatorLoop(&g.simpleParameters, &g)
	return &g
}

// nextValue for a SquareGenerator will output a pleasantly buzzy square
// waveform at the frequency described by the simpleParameter's hz parameter.
func (g *SquareGenerator) nextValue() float32 {
	if nextSineValue(g.hz, &g.phase) > 0.5 {
		return g.gain
	}
	return 0.0
}

type SawGenerator struct { simpleGenerator }

func NewSawGenerator() *SawGenerator {
	g := SawGenerator{ simpleGenerator{makeGeneratorChannels(), makeSimpleParameters()} }
	go g.generatorLoop(&g.simpleParameters, &g)
	return &g
}

// nextValue for a SineGenerator will output a sawtooth waveform at the
// frequency described by the simpleParameter's hz parameter.
func (g *SawGenerator) nextValue() float32 {
	return nextSawValue(g.hz, &g.phase) * g.gain
}

type WavGenerator struct {
	generatorChannels
	data []float32
	pos  int
	gain float32
}

func (g *WavGenerator) String() string {
	return fmt.Sprintf("gain %.2f", g.gain)
}

func NewWavGenerator(file string) *WavGenerator {
	wd, dataErr := ReadWavData(file)
	if dataErr != nil {
		return nil
	}
	// TODO we need to account for differences in the sample rate, probably?
	g := &WavGenerator{makeGeneratorChannels(), btof32(wd.data), 0, 1.0}
	go g.generatorLoop()
	return g
}

// Special case, as we have no (need for) simpleParameters.
func (g *WavGenerator) generatorLoop() {
	for {
		select {
		case ev := <-g.eventIn:
			switch ev.Name {
			case "disconnect":
				g.Reset()
			case "kill":
				g.Reset()
				return
			case "gain":
				g.gain = ev.Val
			}
		case g.audioOut <- nextBuffer(g):
			break
		}
	}
}

func (g *WavGenerator) nextValue() float32 {
	f := g.data[g.pos] * g.gain
	g.pos++
	if g.pos >= len(g.data) {
		g.pos = 0
	}
	return f
}
