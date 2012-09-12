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
		if len(args) < 1 {
			f.output.Print("usage: sleep <duration>")
			return
		}
		f.parseSleep(args[0])

	case "add":
		if len(args) < 2 {
			f.output.Print("usage: add <kind> <name>")
			return
		}
		kind, name := args[0], args[1]
		f.parseAdd(kind, name)

	case "delete", "del", "rm":
		if len(args) < 1 {
			f.output.Print("usage: delete <name>")
			return
		}
		f.parseDelete(args[0])

	default:
		e, err := f.Entity(cmd)
		if err != nil {
			f.output.Printf("%s: %s", cmd, err)
			return
		}
		if len(args) < 1 {
			f.output.Printf("usage: %s <action> [args]", cmd)
			return
		}
		switch x := e.(type) {
		case Note:
			f.parseKeyDown(x, args[0])
		case Node:
			f.parseNodeCmd(x, args[0], args[1:])
		default:
			f.output.Printf("%s: I don't know what that is", cmd)
			return
		}
	}
}

func (f *FieldParser) Entity(s string) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (f *FieldParser) parseInfo() {
	f.output.Printf("%v", f.f)
}

func (f *FieldParser) parseSleep(s string) {
	d, err := time.ParseDuration(s)
	if err != nil {
		f.output.Printf("sleep: %s: invalid duration", s)
		return
	}
	time.Sleep(d)
}

func (f *FieldParser) parseAdd(kind, name string) {
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

func (f *FieldParser) parseDelete(name string) {
	if err := f.f.Delete(name); err != nil {
		f.output.Printf("%s", err)
		return
	}
}

func (f *FieldParser) parseKeyDown(n Note, name string) {
	f.output.Printf("keyDown %s %s", n, name)
}

func (f *FieldParser) parseNodeCmd(n Node, cmd string, args []string) {
	switch cmd {
	case "->":
		if len(args) < 1 {
			f.output.Printf("usage: %s -> <target>", n.Name())
			return
		}
		if err := f.f.Connect(n.Name(), args[0]); err != nil {
			f.output.Printf("%s -> %s: %s", n.Name(), args[0], err)
			return
		}
		f.output.Printf("%s -> %s: connect OK", n.Name(), args[0])
	}
}
