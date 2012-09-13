package main

type AudioSender interface {
	AudioOut() <-chan []float32
	Reset() // stop playback by breaking downstream connections
}
