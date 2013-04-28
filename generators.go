package goop

type KillEvent struct{}
type GainEvent struct{ Gain float32 }

//
//
//

type eventProcessor interface {
	processEvent(Event) bool // processed
}

//
//
//

type generatorChannels struct {
	events chan Event
	audio  chan []float32
}

func makeGeneratorChannels() generatorChannels {
	return generatorChannels{
		events: make(chan Event),
		audio:  make(chan []float32),
	}
}

func (gc *generatorChannels) Events() chan<- Event {
	return gc.events
}

func (gc *generatorChannels) AudioOut() <-chan []float32 {
	return gc.audio
}

func (gc *generatorChannels) loop(ep eventProcessor, vp valueProvider) {
	for {
		select {
		case ev := <-gc.events:
			switch ev.(type) {
			case KillEvent:
				return
			default:
				ep.processEvent(ev)
			}

		case gc.audio <- nextBuffer(vp):
			break
		}
	}
}

//
//
//

type simpleParameters struct {
	gain float32
}

func makeSimpleParameters() simpleParameters {
	return simpleParameters{
		gain: 1.0,
	}
}

func (sp *simpleParameters) processEvent(ev Event) bool {
	switch e := ev.(type) {
	case GainEvent:
		sp.gain = e.Gain
		return true
	}
	return false
}

//
//
//

type valueProvider interface {
	nextValue() float32
}

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

type WavGenerator struct {
	generatorChannels
	simpleParameters
	filename string
	data     []float32
	pos      int
}

func NewWavGenerator(filename string) (*WavGenerator, error) {
	data, err := ReadWavData(filename)
	if err != nil {
		return nil, err
	}

	g := &WavGenerator{
		generatorChannels: makeGeneratorChannels(),
		simpleParameters:  makeSimpleParameters(),
		filename:          filename,
		data:              btof32(data.data),
		pos:               0,
	}
	go g.generatorChannels.loop(g, g)
	return g, nil
}

func (g *WavGenerator) Name() string { return g.filename }

func (g *WavGenerator) processEvent(ev Event) bool {
	if g.simpleParameters.processEvent(ev) {
		return true
	}
	switch e := ev.(type) {
	// TODO?
	}
	return false
}

func (g *WavGenerator) nextValue() float32 {
	// this would need synchronization if we ever mutated g.data
	f := g.data[g.pos] * g.gain
	g.pos++
	if g.pos >= len(g.data) {
		g.pos = 0
	}
	return f
}
