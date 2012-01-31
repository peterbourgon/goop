package goop

import (
	"errors"
	"fmt"
	"sync"
)

type synchronizedMap struct {
	m   map[string]interface{}
	mtx sync.Mutex
}

func newSynchronizedMap() *synchronizedMap {
	m := &synchronizedMap{make(map[string]interface{}), sync.Mutex{}}
	return m
}

func (m *synchronizedMap) get(key string) (interface{}, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if item, ok := m.m[key]; ok {
		return item, nil
	}
	return nil, errors.New(fmt.Sprintf("%s: no such key", key))
}

func (m *synchronizedMap) set(key string, val interface{}) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, exists := m.m[key]; exists {
		return errors.New(fmt.Sprintf("%s: already exists", key))
	}
	m.m[key] = val
	return nil
}

func (m *synchronizedMap) del(key string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, exists := m.m[key]; !exists {
		return errors.New(fmt.Sprintf("%s: no such key", key))
	}
	delete(m.m, key)
	return nil
}

// A goop Network is a collection of arbitrary Modules, plus a few
// singletons which help it model a patch-bay of audio "equipment".
type Network struct {
	container *synchronizedMap
	mixer     *Mixer
	clock     *Clock
	autocron  int64
}

func NewNetwork() *Network {
	container := newSynchronizedMap()
	mixer := NewMixer()
	clock := NewClock()
	g := &Network{container, mixer, clock, 0}
	return g
}

func (n *Network) Add(name string, module interface{}) error {
	return n.container.set(name, module)
}

func (n *Network) Del(name string) error {
	item, err := n.container.get(name)
	if err != nil {
		return err
	}
	if r, ok := item.(EventReceiver); ok {
		r.Events() <- Event{"kill", 0, nil}
	}
	n.container.del(name)
	return nil
}

func (n *Network) Connect(from, to string) error {
	return errors.New("not yet implemented")
}

func (n *Network) Disconnect(from string) error {
	return errors.New("not yet implemented")
}

func (n *Network) Fire(to string, ev Event) error {
	r, err := n.getEventReceiver(to)
	if err != nil {
		return errors.New(fmt.Sprintf("fire: %s", err))
	}
	r.Events() <- ev
	return nil
}

func (n *Network) QueueFire(to string, ev Event) error {
	r, err := n.getEventReceiver(to)
	if err != nil {
		return errors.New(fmt.Sprintf("fire: %s", err))
	}
	n.clock.Queue(TargetAndEvent{r.Events(), ev})
	return nil
}

func (n *Network) getEventReceiver(name string) (EventReceiver, error) {
	item, itemErr := n.container.get(name)
	if itemErr != nil {
		return nil, itemErr
	}
	r, rOk := item.(EventReceiver)
	if !rOk {
		return nil, errors.New(fmt.Sprintf("%s: can't receive events", name))
	}
	return r, nil
}