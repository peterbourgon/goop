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
	f.output.Printf("%v", f.f)
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
		f.output.Printf("add: %s: already exists", name)
		return
	}

	switch kind {
	case "sine-generator", "sine":
		f.f.Add(NewSineGenerator(name))
		f.output.Printf("add: %s: %s: OK", kind, name)
	default:
		f.output.Printf("add: %s: unknown kind", kind)
		return
	}
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
	if len(args) < 1 {
		f.output.Printf("usage: %s <action> [args]", cmd)
		return
	}

	e, err := f.entity(cmd)
	if err != nil {
		f.output.Printf("%s: %s", cmd, err)
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
	note, err := ParseNote(s)
	if err == nil {
		return KeyDownEvent(note), nil
	}

	if s == KeyUp {
		// generic "keyup" yields event with 0 values
		return KeyUpEvent(NoteZero()), nil
	}

	node, err := f.f.Get(s)
	if err == nil {
		return node, nil
	}

	return nil, fmt.Errorf("%s unrecognized", s)
}

func (f *FieldParser) parseEventCmd(ev Event, cmd string, args []string) {
	switch cmd {
	case "->": // send to
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
		f.output.Printf("%s -> %s: OK", ev, node.Name())

	default:
		f.output.Printf("unknown command '%s'", cmd)
	}
}

func (f *FieldParser) parseNodeCmd(node Node, cmd string, args []string) {
	switch cmd {
	case "->":
		if len(args) < 1 {
			f.output.Printf("usage: %s -> <target>", node.Name())
			return
		}
		if err := f.f.Connect(node.Name(), args[0]); err != nil {
			f.output.Printf("%s -> %s: %s", node.Name(), args[0], err)
			return
		}
		f.output.Printf("%s -> %s: connect OK", node.Name(), args[0])
	}
}
