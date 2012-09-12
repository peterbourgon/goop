package main

import (
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

	case "sleep", "sl":
		if len(args) < 1 {
			f.output.Print("usage: sleep <duration>")
			return
		}
		f.parseSleep(args[0])

	case "add", "ad", "a":
		if len(args) < 2 {
			f.output.Print("usage: add <kind> <name>")
			return
		}
		kind, name := args[0], args[1]
		f.parseAdd(kind, name)

	case "delete", "del", "d", "rm":
		if len(args) < 1 {
			f.output.Print("usage: delete <name>")
			return
		}
		f.parseDelete(args[0])

	default:
		n, err := f.f.Get(cmd)
		if err != nil {
			f.output.Printf("%s: %s", cmd, err)
			return
		}
		f.parseNodeAction(n, args)
	}
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
	switch kind {
	// TODO
	default:
		f.output.Printf("add: %s: unknown kind", kind)
		return
	}

	if n, _ := f.f.Get(name); n != nil {
		f.output.Printf("add: %s: already exists", name)
		return
	}

	// TODO
}

func (f *FieldParser) parseDelete(name string) {
	if err := f.f.Delete(name); err != nil {
		f.output.Printf("%s", err)
		return
	}
}

func (f *FieldParser) parseNodeAction(n Node, args []string) {
	// TODO
	f.output.Printf("do something with node %s: %v", n.Name(), args)
}
