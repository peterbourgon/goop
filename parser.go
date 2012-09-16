package main

import (
	"fmt"
	"strings"
	"time"
)

type Parser interface {
	Parse(s string)
}

//
//
//

type EchoParser struct{ Output }

func NewEchoParser(output Output) EchoParser {
	return EchoParser{output}
}

func (p EchoParser) Parse(s string) {
	p.Print(s)
}

//
//
//

type FieldParser struct {
	f      Field
	output Output
}

func NewFieldParser(f Field, output Output) *FieldParser {
	return &FieldParser{
		f:      f,
		output: output,
	}
}

func (f *FieldParser) Parse(s string) {
	toks := strings.Split(s, ";")
	for _, ss := range toks {
		f.parse(ss)
	}
}

func (f *FieldParser) parse(s string) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return
	}

	toks := strings.Split(s, " ")
	cmd, args := toks[0], toks[1:]
	switch cmd {

	case "info", "dot":
		f.parseInfo()

	case "sleep":
		f.parseSleep(args)

	case "add":
		f.parseAdd(args)

	case "delete", "del", "rm":
		f.parseDelete(args)

	default:
		f.parseArbitrary(cmd, args)
	}
}

func (f *FieldParser) parseInfo() {
	f.output.Printf("\n%s\n", f.f.Dot())
}

func (f *FieldParser) parseSleep(args []string) {
	if len(args) < 1 {
		f.output.Print("usage: sleep <duration>")
		return
	}
	d, err := time.ParseDuration(args[0])
	if err != nil {
		f.output.Printf("sleep: %s: invalid duration", args[0])
		return
	}
	time.Sleep(d)
}

func (f *FieldParser) parseAdd(args []string) {
	if len(args) < 2 {
		f.output.Print("usage: add <kind> <name>")
		return
	}

	kind, name := args[0], args[1]
	if n, _ := f.f.Get(name); n != nil {
		f.output.Printf("%s: already exists", name)
		return
	}

	n, err := createInstanceMap.CreateInstance(kind, name)
	if err != nil {
		f.output.Printf("add %s %s: %s", kind, name, err)
		return
	}

	if err := f.f.Add(n); err != nil {
		f.output.Printf("add %s %s: %s", kind, name, err)
		return
	}

	f.output.Printf("add %s %s: OK", kind, name)
}

func (f *FieldParser) parseDelete(args []string) {
	if len(args) < 1 {
		f.output.Print("usage: delete <name>")
		return
	}
	if err := f.f.Delete(args[0]); err != nil {
		f.output.Printf("%s", err)
		return
	}
}

func (f *FieldParser) parseArbitrary(cmd string, args []string) {
	if _, ok := createInstanceMap[cmd]; ok {
		f.output.Printf("'%s' is an entity type; assuming you meant 'add'", cmd)
		newArgs := []string{cmd}
		newArgs = append(newArgs, args...)
		f.parseAdd(newArgs)
		return
	}

	e, err := f.entity(cmd)
	if err != nil {
		f.output.Printf("%s: %s", cmd, err)
		return
	}

	if len(args) < 1 {
		f.output.Printf("usage: %s <action> [args]", cmd)
		return
	}

	switch x := e.(type) {
	case Node:
		f.parseNodeCmd(x, args[0], args[1:])
	case Event:
		f.parseEventCmd(x, args[0], args[1:])
	default:
		f.output.Printf("%s: I don't know what that is", cmd)
		return
	}
}

func (f *FieldParser) entity(s string) (interface{}, error) {
	// It's possible to name a Node after a builtin, so if we don't parse
	// Nodes before builtins, they can become inaccessible.
	node, err := f.f.Get(s)
	if err == nil {
		return node, nil
	}

	note, err := ParseNote(s)
	if err == nil {
		return KeyDownEvent(note), nil
	}

	switch s {
	case KeyUp, "ø", "Ø", "0":
		return KeyUpEvent(NoteZero()), nil
	}

	// TODO it's not clear to me that I want to allow this.
	// TODO perhaps better to explicitly parse all Event types?
	ev, err := ParseArbitraryEvent(s)
	if err == nil {
		D("parsed arbitrary Event %v", ev)
		return ev, nil
	}

	return nil, fmt.Errorf("unrecognized")
}

func (f *FieldParser) parseEventCmd(ev Event, cmd string, args []string) {
	switch cmd {
	case "->", "=>":
		if len(args) < 1 {
			f.output.Printf("usage: %s -> <target>", ev)
			return
		}
		tgt := args[0]
		node, err := f.f.Get(tgt)
		if err != nil {
			f.output.Printf("%s -> %s: target: %s", ev, tgt, err)
		}
		node.Events() <- ev
		f.output.Printf("[%s %.1f] -> %s: OK", ev.Type, ev.Value, node.Name())

	default:
		f.output.Printf("unknown command '%s'", cmd)
	}
}

func (f *FieldParser) parseNodeCmd(node Node, cmd string, args []string) {
	switch cmd {
	case "=>", "->", "c", "connect":
		if len(args) < 1 {
			f.output.Printf("usage: %s -> <target>", node.Name())
			return
		}
		tgt := args[0]
		if err := f.f.Connect(node.Name(), tgt); err != nil {
			f.output.Printf("%s => %s: %s", node.Name(), tgt, err)
			return
		}
		f.output.Printf("%s => %s: connect OK", node.Name(), tgt)

	case "≠>", "≠", "x", "d", "disconnect":
		if len(args) >= 1 {
			tgt := args[0]
			if err := f.f.Disconnect(node.Name(), tgt); err != nil {
				f.output.Printf("%s ≠> %s: %s", node.Name(), tgt, err)
				return
			}
			f.output.Printf("%s ≠> %s: disconnect OK", node.Name(), tgt)
		} else {
			if err := f.f.DisconnectAll(node.Name()); err != nil {
				f.output.Printf("%s ≠> *: %s", node.Name(), err)
				return
			}
			f.output.Printf("%s ≠> *: disconnect OK", node.Name())
		}
	}
}
