package main

type AudioSender interface {
	AudioOut() <-chan []float32
}