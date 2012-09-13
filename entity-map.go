package main

import (
	"fmt"
)

type EntityMap map[string]createFunction

type createFunction func(string) Node

var (
	entityMap EntityMap
)

func init() {
	entityMap = EntityMap{
		"sine":           NewSineGeneratorNode,
		"sine-generator": NewSineGeneratorNode,

		"gainlfo": NewGainLFONode,
		"lfo":     NewGainLFONode,
	}
}

func (m EntityMap) CreateInstance(kind, name string) (Node, error) {
	f, ok := m[kind]
	if !ok {
		return nil, fmt.Errorf("'%s' unrecognized", kind)
	}
	return f(name), nil
}
