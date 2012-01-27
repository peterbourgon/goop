package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
)

var infile *string = flag.String("f", "", "file to read initial commands from")

func readInitialCommands() []string {
	if *infile != "" {
		if buf, err := ioutil.ReadFile(*infile); err == nil {
			lines := strings.Split(string(buf), "\n")
			return lines
		} else {
			fmt.Printf("error reading initial commands file: %s\n", err)
		}
	}
	return make([]string, 0)
}

func main() {
	flag.Parse()
	initialCommands := readInitialCommands()
	for _, line := range initialCommands {
		uiParse(line)
	}
	uiLoop()
}
