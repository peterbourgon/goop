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

	fi, err := NewFileInput(*filename)
	if err != nil {
		panic(err)
	}
	REPL(fi, p, o)

	ii := &InteractiveInput{}
	REPL(ii, p, o)
}

func REPL(r Input, e Parser, p Output) {
	for {
		input, err := r.ReadOne() // R
		if err != nil {
			return
		}
		e.Parse(input) // E+P
	} // L
}
