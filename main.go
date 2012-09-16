package main

import (
	"flag"
	"os"
)

var (
	cmdfile = flag.String("cmdfile", "default.txt", "command file")
	dotfile = flag.String("dotfile", "", "Field representation will be written here")
)

func init() {
	flag.Parse()
}

func main() {
	o := StdOutput{}
	f := NewField()
	f.Add(NewMixer())
	f.Add(NewClock(f))
	p := NewFieldParser(f, o)

	if fi, err := NewFileInput(*cmdfile); err == nil {
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

func writeDotfile(f Field) {
	if *dotfile == "" {
		return
	}
	file, err := os.Create(*dotfile)
	if err != nil {
		D("couldn't write %s: %s", *dotfile, err)
		return
	}
	defer file.Close()
	_, err = file.Write([]byte(f.Dot()))
	if err != nil {
		D("error writing %s: %s", *dotfile, err)
		return
	}
}
