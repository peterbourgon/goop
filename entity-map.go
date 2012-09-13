package main

import (
	"fmt"
)

type CreateInstanceMap map[string]createFunction

type createFunction func(string) Node

var (
	createInstanceMap CreateInstanceMap
)

func init() {
	createInstanceMap = CreateInstanceMap{
		"sine":           NewSineGeneratorNode,
		"sine-generator": NewSineGeneratorNode,

		"gainlfo":  NewGainLFONode,
		"gain-lfo": NewGainLFONode,
		"lfo":      NewGainLFONode,
	}

}

func (m CreateInstanceMap) CreateInstance(kind, name string) (Node, error) {
	f, ok := m[kind]
	if !ok {
		return nil, fmt.Errorf("'%s' unrecognized", kind)
	}
	return f(name), nil
}
