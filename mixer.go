package main

import (
	"code.google.com/p/portaudio-go/portaudio"
	"fmt"
	"sync"
)

// A Mixer multiplexes audio data channels from AudioSenders into a single
// stream, which it passes to the audio subsystem.
type Mixer struct {
	multipleParents
	noChildren

	gain    float32
	on      bool
	chans   []<-chan []float32
	eventIn chan Event

	sync.Mutex
	cond *sync.Cond
}

func (m *Mixer) String() string {
	return fmt.Sprintf("Mixer: %d connections, gain %.2f", len(m.chans), m.gain)
}

// NewMixer returns a new Mixer, ready to use.
func NewMixer() *Mixer {
	m := &Mixer{
		gain:    0.1,
		on:      false,
		chans:   []<-chan []float32{},
		eventIn: make(chan Event, EVENT_CHAN_BUFFER),
		cond:    nil,
	}
	m.cond = sync.NewCond(m)
	go m.eventLoop()
	go m.Play()
	return m
}

// Name satisfies the Name() method in the Node interface.
func (m *Mixer) Name() string { return "mixer" }

// Events satisfies the Events() method in the Node interface.
func (m *Mixer) Events() chan<- Event { return m.eventIn }

func (m *Mixer) eventLoop() {
	for {
		select {
		case ev := <-m.eventIn:
			switch ev.Type {

			case Kill:
				m.dropAll()
				m.Stop()
				return

			case Connection:
				D("Mixer got connection: %v", ev.Arg)
				sender, senderOk := ev.Arg.(AudioSender)
				if !senderOk {
					return
				}
				node, nodeOk := ev.Arg.(Node)
				if !nodeOk {
					return
				}
				func() {
					m.Lock()
					defer m.Unlock()
					m.chans = append(m.chans, sender.AudioOut())
					m.multipleParents.Add(node)
				}()

			case Disconnection:
				sender, senderOk := ev.Arg.(AudioSender)
				if !senderOk {
					return
				}
				node, nodeOk := ev.Arg.(Node)
				if !nodeOk {
					return
				}
				func() {
					m.Lock()
					defer m.Unlock()
					for i, ch := range m.chans {
						if ch == sender.AudioOut() {
							m.chans = append(m.chans[:i], m.chans[i+1:]...)
							return
						}
					}
				}()
				func() {
					m.Lock()
					defer m.Unlock()
					m.multipleParents.Delete(node.Name())
				}()

			}
		}
	}
}

// dropAll removes all audio channels from the Mixer's internal map,
// effectively stopping all audio playback. This is meant only to be called
// from a Kill event.
func (m *Mixer) dropAll() {
	m.Lock()
	defer m.Unlock()
	m.chans = make([]<-chan []float32, 0)
	m.multipleParents = multipleParents{}
}

// Play is a blocking call which initializes the audio subsystem. It should
// be called on a separate goroutine. Calling Stop will trigger Play to
// return.
func (m *Mixer) Play() {
	const (
		ICHAN = 1
		OCHAN = 1
	)
	m.Lock()
	defer m.Unlock()
	m.on = true
	stream, err := portaudio.OpenDefaultStream(ICHAN, OCHAN, SRATE, BUFSZ, m)
	if err != nil {
		panic(fmt.Sprintf("open: %s", err))
	}
	defer stream.Close()
	if err = stream.Start(); err != nil {
		panic(fmt.Sprintf("start: %s", err))
	}
	D("Mixer playing")
	m.cond.Wait()
	if err = stream.Stop(); err != nil {
		panic(fmt.Sprintf("stop: %s", err))
	}
	m.on = false
	m.cond.Broadcast()
}

// Stop triggers the Play function to break from its blocking state and tear
// down the audio subsystem.
func (m *Mixer) Stop() {
	m.Lock()
	defer m.Unlock()
	m.cond.Broadcast()
}

// Join blocks until the Play function has successfully torn down the audio
// subsystem.
func (m *Mixer) Join() {
	m.Lock()
	defer m.Unlock()
	for m.on {
		m.cond.Wait()
	}
}

// ProcessAudio is the callback function provided to the PortAudio subsystem
// which is called on a regular basis to provide audio data.
func (m *Mixer) ProcessAudio(in, out []float32) {
	for i := 0; i < len(out); i++ {
		out[i] = 0.0
	}
	m.Lock()
	defer m.Unlock()
	mux(&m.chans, m.gain, out)
}

// mux multiplexes all the given channels into the output buffer,
// scaling each audio datapoint by the gain parameter.
//
// mux also handles removal of closed channels from the passed slice.
func mux(chans *[]<-chan []float32, gain float32, out []float32) {
	good, bad := 0, 0
	for i, c := range *chans {
		buf, ok := <-c
		if ok {
			good++
			for j := range buf {
				out[j] += gain * buf[j]
			}
		} else {
			bad++
			(*chans)[i] = nil
		}
	}
	newChans, idx := make([]<-chan []float32, good), 0
	for _, c := range *chans {
		if c != nil {
			newChans[idx] = c
			idx++
		}
	}
	(*chans) = newChans
}
