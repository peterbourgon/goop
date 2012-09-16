package main

import (
	"fmt"
)

type Field map[string]Node

func NewField() Field {
	return map[string]Node{}
}

func (f Field) Add(n Node) error {
	defer writeDotfile(f)
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
	defer writeDotfile(f)
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

	n.Events() <- KillEvent()
	delete(f, name)
	return nil
}

func (f Field) Connect(src, dst string) error {
	D("Connect(%s, %s)", src, dst)
	defer writeDotfile(f)
	parent, err := f.Get(src)
	if err != nil {
		return err
	}

	child, err := f.Get(dst)
	if err != nil {
		return err
	}

	if reachable(child, parent) {
		return fmt.Errorf("cycle detected")
	}

	parent.Events() <- ConnectEvent(child)
	child.Events() <- ConnectionEvent(parent)

	return nil
}

func (f Field) Disconnect(src, dst string) error {
	defer writeDotfile(f)
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

func (f Field) DisconnectAll(src string) error {
	defer writeDotfile(f)
	parent, err := f.Get(src)
	if err != nil {
		return err
	}

	for _, child := range parent.Children() {
		parent.Events() <- DisconnectEvent(child)
		child.Events() <- DisconnectionEvent(parent)
	}

	return nil
}

func (f Field) Broadcast(ev Event) {
	for _, n := range f {
		n.Events() <- ev
	}
}

func (f *Field) Dot() string {
	s := "digraph G {\n"

	// nodes
	for _, n := range *f {
		s += fmt.Sprintf(
			"\t%s [shape=box,label=\"%s\"];\n",
			n.Name(),
			NodeLabel(n),
			//n,
		)
	}
	s += "\n"

	// edges
	for _, n := range *f {
		D("Dot: adding edges for %d children of %s", len(n.Children()), n.Name())
		for _, child := range n.Children() {
			s += fmt.Sprintf("\t%s -> %s;\n", n.Name(), child.Name())
		}
	}

	s += "}"
	return s
}

//
//
//

type Node interface {
	Name() string
	Parents() []Node
	Children() []Node
	EventReceiver
}

var nilNode Node

type Typed interface {
	Kind() string
}

func NodeLabel(n Node) string {
	if typed, ok := n.(Typed); ok {
		return fmt.Sprintf("%s '%s'", typed.Kind(), n.Name())
	}
	return n.Name()
}

func reachable(n, tgt Node) bool {
	D("reachable(\n\t%6s %v,\n\t%6s %v\n)", n.Name(), n, tgt.Name(), tgt)
	if n == tgt {
		D(" reachable because %v == %v", n, tgt)
		return true
	}
	for i, child := range n.Children() {
		if child == n {
			D(" reachable because %s Child[%d] == %v", n.Name(), i, tgt)
			return true
		}
		if reachable(child, tgt) {
			D(" recursive return reachable")
			return true
		}
	}
	D(" not reachable!")
	return false
}

// A nodeName may be embedded into any type to satisfy
// the Name() method of the Node interface.
type nodeName string

func (nn nodeName) Name() string { return string(nn) }

//
//
//

// singleParent may be embedded into any type to satisfy
// the Parents() method of the Node interface, with arity=1.
//
// To set, do myStruct.ParentNode = n.
// To clear, do myStruct.ParentNode = nilNode.
type singleParent struct{ ParentNode Node }

func (sp singleParent) Parents() []Node {
	if sp.ParentNode == nilNode {
		return []Node{}
	}
	return []Node{sp.ParentNode}
}

func (sp *singleParent) processEvent(ev Event) {
	switch ev.Type {
	case Connection: // upstream
		node, nodeOk := ev.Arg.(Node)
		if !nodeOk {
			break
		}
		sp.ParentNode = node

	case Disconnection: // upstream
		sp.ParentNode = nilNode

	}
}

// singleChild may be embedded into any type to satisfy
// the Children() method of the Node interface, with arity=1.
//
// To set, do myStruct.ChildNode = n.
// To clear, do myStruct.ChildNode = nilNode.
type singleChild struct{ ChildNode Node }

func (sc singleChild) Children() []Node {
	if sc.ChildNode == nilNode {
		return []Node{}
	}
	return []Node{sc.ChildNode}
}

func (sc *singleChild) processEvent(ev Event) {
	switch ev.Type {
	case Connect: // downstream
		node, nodeOk := ev.Arg.(Node)
		if !nodeOk {
			break
		}
		sc.ChildNode = node

	case Disconnect: // downstream
		sc.ChildNode = nilNode // TODO could do more thorough checking
	}
}

// singleAncestry combines singleParent + singleChild.
// It should be embedded into a concrete struct.
// It requires no explicit initialization.
type singleAncestry struct {
	singleParent
	singleChild
}

func (sa *singleAncestry) processEvent(ev Event) {
	switch ev.Type {
	case Connect, Disconnect: // downstream
		sa.singleChild.processEvent(ev)

	case Connection, Disconnection: // upstream
		sa.singleParent.processEvent(ev)

	case Kill:
		sa.singleChild.ChildNode = nilNode
		sa.singleParent.ParentNode = nilNode
	}
}

// multipleParents may be embedded into any type to satisfy
// the Parents() method of the Node interface, with arity=N.
type multipleParents struct{ m map[string]Node }

func newMultipleParents() *multipleParents {
	return &multipleParents{
		m: map[string]Node{},
	}
}

func (mp *multipleParents) Parents() []Node {
	parents := []Node{}
	for _, n := range mp.m {
		parents = append(parents, n)
	}
	return parents
}

func (mp *multipleParents) AddParent(n Node) {
	mp.m[n.Name()] = n
}

func (mp *multipleParents) DeleteParent(name string) {
	delete(mp.m, name)
}

// noParents may be embedded into any type to satisfy
// the Parents() method of the Node interface, with arity=0.
type noParents struct{}

func (np noParents) Parents() []Node { return []Node{} }

// noChildren may be embedded into any type to satisfy
// the Children() method of the Node interface, with arity=0.
type noChildren struct{}

func (nc noChildren) Children() []Node { return []Node{} }
