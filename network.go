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

func (m *synchronizedMap) names() []string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	n, i := make([]string, len(m.m)), 0
	for name, _ := range m.m {
		n[i] = name
		i++
	}
	return n
}

// A goop Network is a collection of arbitrary Modules,
// plus some helper interfaces.
type Network struct {
	container *synchronizedMap
	der       DeferredEventReceiver
}

func NewNetwork(der DeferredEventReceiver) *Network {
	container := newSynchronizedMap()
	g := &Network{container, der}
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

func (n *Network) Get(name string) (interface{}, error) {
	return n.container.get(name)
}

func (n *Network) Names() []string {
	return n.container.names()
}

func (n *Network) Connect(from, to string) error {
	fromItem, fromErr := n.container.get(from)
	if fromErr != nil {
		return errors.New(fmt.Sprintf("connect: %s", fromErr))
	}
	_, fromSenderOk := fromItem.(AudioSender)
	if !fromSenderOk {
		return errors.New(fmt.Sprintf("connect: %s: doesn't send audio", from))
	}
	toItem, toErr := n.container.get(to)
	if toErr != nil {
		return errors.New(fmt.Sprintf("connect: %s", toErr))
	}
	toReceiver, toReceiverOk := toItem.(EventReceiver)
	if !toReceiverOk {
		return errors.New(fmt.Sprintf("connect: %s: can't receive events", to))
	}
	// Should be buffer this one?
	toReceiver.Events() <- Event{"receivefrom", 0.0, fromItem}
	return nil
}

func (n *Network) Disconnect(from string) error {
	fromItem, fromErr := n.container.get(from)
	if fromErr != nil {
		return errors.New(fmt.Sprintf("disconnect: %s", fromErr))
	}
	r, ok := fromItem.(EventReceiver)
	if !ok {
		return errors.New(fmt.Sprintf("connect: %s: can't receive events", from))
	}
	r.Events() <- Event{"disconnect", 0.0, nil}
	return nil
}

const (
	Immediately = iota
	Deferred
)

func (n *Network) Fire(to string, ev Event, when int) error {
	r, err := n.getEventReceiver(to)
	if err != nil {
		return errors.New(fmt.Sprintf("fire: %s", err))
	}
	switch when {
	case Immediately:
		r.Events() <- ev
	case Deferred:
		n.der.DeferredEvents() <- TargetAndEvent{r, ev}
	default:
		panic("unreachable")
	}
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