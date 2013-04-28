package goop

type AudioSender interface {
	AudioOut() <-chan []float32
}
