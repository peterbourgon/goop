package main

import (
	"fmt"
)

type Field map[string]Node

func NewField() Field {
	return map[string]Node{}
}

func (f Field) Add(n Node) error {
	name := n.Name()
	if n, _ := f.Get(name); n != nil {
		return fmt.Errorf("already exists")
	}
	f[name] = n
	return nil
}

func (f Field) Get(name string) (Node, error) {
	if n, ok := f[name]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("not found")
}

func (f Field) Delete(name string) error {
	n, err := f.Get(name)
	if err != nil {
		return fmt.Errorf("not found")
	}

	for _, parent := range n.Parents() {
		if err := f.Disconnect(parent.Name(), name); err != nil {
			panic(fmt.Errorf("delete(%s): %s", name, err))
		}
	}
	for _, child := range n.Children() {
		if err := f.Disconnect(name, child.Name()); err != nil {
			panic(fmt.Errorf("delete(%s): %s", name, err))
		}
	}

	delete(f, name)
	return nil
}

func (f Field) Connect(src, dst string) error {
	parent, err := f.Get(src)
	if err != nil {
		return err
	}

	child, err := f.Get(dst)
	if err != nil {
		return err
	}

	if reachable(child, src) {
		return fmt.Errorf("cycle detected")
	}

	parent.Events() <- ConnectEvent(child)
	child.Events() <- ConnectionEvent(parent)

	return nil
}

func (f Field) Disconnect(src, dst string) error {
	parent, err := f.Get(src)
	if err != nil {
		return err
	}

	var child Node
	for _, n := range parent.Children() {
		if n.Name() == dst {
			child = n
			break
		}
	}
	if child == nil {
		return fmt.Errorf("'%s' not a child of '%s'", dst, src)
	}

	parent.Events() <- DisconnectEvent(child)
	child.Events() <- DisconnectionEvent(parent)

	return nil
}

type Node interface {
	Name() string
	Parents() []Node
	Children() []Node
	EventReceiver
}

func reachable(n Node, name string) bool {
	for _, child := range n.Children() {
		if child.Name() == name {
			return true
		}
		if reachable(child, name) {
			return true
		}
	}
	return false
}
