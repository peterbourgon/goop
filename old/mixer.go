package goop

import (
	"code.google.com/p/portaudio-go/portaudio"
	"fmt"
	"sync"
)

const (
	SRATE = 44100
	BUFSZ = 2205
	CHBUF = 1
)

const (
	AUDIO_CHAN_BUFFER = 0
	OTHER_CHAN_BUFFER = 10
)

// A Mixer multiplexes audio data channels from AudioSenders into a single
// stream, which it passes to the audio subsystem.
type Mixer struct {
	mtx     sync.Mutex
	cnd     *sync.Cond
	gain    float32
	on      bool
	chans   []<-chan []float32
	eventIn chan Event
}

func (m *Mixer) String() string {
	return fmt.Sprintf("%d connections, gain %.2f", len(m.chans), m.gain)
}

// NewMixer returns a new Mixer, ready to use.
func NewMixer() *Mixer {
	mx := sync.Mutex{}
	ga := float32(0.1)
	on := false
	ch := make([]<-chan []float32, 0)
	ei := make(chan Event, OTHER_CHAN_BUFFER)
	m := Mixer{mtx: mx, gain: ga, on: on, chans: ch, eventIn: ei}
	m.cnd = sync.NewCond(&m.mtx)
	go m.eventLoop()
	go m.Play()
	return &m
}

func (m *Mixer) Events() chan<- Event { return m.eventIn }

func (m *Mixer) eventLoop() {
	for {
		select {
		case ev := <-m.eventIn:
			switch ev.Name {
			case "kill":
				m.DropAll()
				m.Stop()
				return
			case "receivefrom":
				if sender, ok := ev.Arg.(AudioSender); ok {
					func() {
						m.mtx.Lock()
						defer m.mtx.Unlock()
						m.chans = append(m.chans, sender.AudioOut())
					}()
				}
			}
		}
	}
}

// DropAll removes all audio channels from the Mixer's internal map,
// effectively stopping all audio playback.
func (m *Mixer) DropAll() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.chans = make([]<-chan []float32, 0)
}

// Play is a blocking call which initializes the audio subsystem. It should
// be called on a separate goroutine. Calling Stop will trigger Play to 
// return.
func (m *Mixer) Play() {
	const (
		ICHAN = 1
		OCHAN = 1
	)
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.on = true
	stream, err := portaudio.OpenDefaultStream(ICHAN, OCHAN, SRATE, BUFSZ, m)
	if err != nil {
		panic(fmt.Sprintf("open: %s", err))
	}
	defer stream.Close()
	if err = stream.Start(); err != nil {
		panic(fmt.Sprintf("start: %s", err))
	}
	m.cnd.Wait()
	if err = stream.Stop(); err != nil {
		panic(fmt.Sprintf("stop: %s", err))
	}
	m.on = false
	m.cnd.Broadcast()
}

// Stop triggers the Play function to break from its blocking state and tear
// down the audio subsystem.
func (m *Mixer) Stop() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.cnd.Broadcast()
}

// Join blocks until the Play function has successfully torn down the audio
// subsystem.
func (m *Mixer) Join() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for m.on {
		m.cnd.Wait()
	}
}

// ProcessAudio is the callback function provided to the PortAudio subsystem
// which is called on a regular basis to provide audio data.
func (m *Mixer) ProcessAudio(in, out []float32) {
	for i := 0; i < len(out); i++ {
		out[i] = 0.0
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	mux(&m.chans, m.gain, out)
}

// mux multiplexes all the given channels into the output buffer, scaling
// each audio datapoint by the gain parameter.
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
