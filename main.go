package main

import (
	"flag"
)

var (
	filename = flag.String("filename", "default.txt", "command file")
)

func init() {
	flag.Parse()
}

func main() {
	o := StdOutput{}
	f := NewField()
	f.Add(NewMixer())
	p := NewFieldParser(f, o)

	if fi, err := NewFileInput(*filename); err == nil {
		REPL(fi, p)
	}

	ii := &InteractiveInput{}
	REPL(ii, p)
}

func REPL(r Input, e Parser) {
	for {
		input, err := r.ReadOne() // R
		if err != nil {
			return
		}
		e.Parse(input) // E+P
	} // L
}
