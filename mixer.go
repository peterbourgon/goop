package goop

import (
	"code.google.com/p/portaudio-go/portaudio"
	"github.com/peterbourgon/field"
	"sync"
)

const (
	SRATE = 44100
	BUFSZ = 2048
)

type Mixer struct {
	sync.RWMutex
	playing  bool
	parents  map[string]AudioSender
	gain     float32
	stopCond *sync.Cond
	doneCond *sync.Cond
}

func NewMixer(f *field.Field) *Mixer {
	m := &Mixer{
		playing: false,
		parents: map[string]AudioSender{},
		gain:    0.5,
	}
	m.stopCond = sync.NewCond(m)
	m.doneCond = sync.NewCond(m)
	return m
}

func (m *Mixer) Name() string { return "mixer" }

func (m *Mixer) Attributes() map[string]interface{} {
	return map[string]interface{}{
		"shape": "box",
	}
}

func (m *Mixer) UpstreamConnect(n field.Node) {
	if sender, ok := n.(AudioSender); ok {
		m.Lock()
		defer m.Unlock()
		// TODO safety check
		m.parents[n.Name()] = sender
	}
}

func (m *Mixer) UpstreamDisconnect(n field.Node) {
	if _, ok := n.(AudioSender); ok {
		m.Lock()
		defer m.Unlock()
		// TODO safety check
		delete(m.parents, n.Name())
	}
}

func (m *Mixer) Play() {
	m.Lock()
	defer m.Unlock()

	const (
		ICHAN = 1
		OCHAN = 1
	)
	stream, err := portaudio.OpenDefaultStream(ICHAN, OCHAN, SRATE, BUFSZ, m)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	if err = stream.Start(); err != nil {
		panic(err)
	}
	m.playing = true

	m.stopCond.Wait()

	if err = stream.Stop(); err != nil {
		panic(err)
	}
	m.playing = false

	m.doneCond.Broadcast()
}

func (m *Mixer) Stop() {
	m.Lock()
	defer m.Unlock()
	m.stopCond.Broadcast()
}

func (m *Mixer) Join() {
	m.Lock()
	defer m.Unlock()
	m.doneCond.Wait()
}

func (m *Mixer) ProcessAudio(in, out []float32) {
	for i, _ := range out {
		out[i] = 0.0
	}
	m.mux(out)
}

func (m *Mixer) mux(out []float32) {
	m.RLock()
	defer m.RUnlock()

	for name, sender := range m.parents {
		buf, ok := <-sender.AudioOut()
		if !ok {
			D("Mixer: %s: no audio", name)
			continue
		}

		for j := range buf {
			out[j] += m.gain * buf[j]
		}
	}
}
