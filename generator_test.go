package goop

import (
	"math"
	"testing"
)

func near(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.001
}

func TestGeneratorFunction(t *testing.T) {
	saw := func(x float32) float32 { return x }
	var v, hz, phase float32 = 0.0, 0.0, 0.0
	v = nextGeneratorFunctionValue(saw, hz, &phase)
	if !near(v, 0.0) {
		t.Errorf("val: expected %f, got %f", 0.0, v)
	}
	if !near(phase, 0.0) {
		t.Errorf("phase: expected %f, got %f", 0.0, phase)
	}
	hz = 440.0
	v = nextGeneratorFunctionValue(saw, hz, &phase)
	if !near(v, 0.0) {
		t.Errorf("val: expected %f, got %f", 0.0, v)
	}
	if !near(phase, 0.0+(hz/SRATE)) {
		t.Errorf("phase: expected %f, got %f", 0.0, phase)
	}
	v = nextGeneratorFunctionValue(saw, hz, &phase)
	if !near(v, 0.04) {
		t.Errorf("val: expected %f, got %f", 0.0, v)
	}
	if !near(phase, 0.0+(2*(hz/SRATE))) {
		t.Errorf("phase: expected %f, got %f", 0.0, phase)
	}
}
