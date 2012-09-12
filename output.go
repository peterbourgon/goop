package main

import (
	"fmt"
)

type Output interface {
	Print(s string)
	Printf(format string, args ...interface{})
}

//
//
//

type StdOutput struct{}

func (o StdOutput) Print(s string) {
	fmt.Printf("%s\n", s)
}

func (o StdOutput) Printf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
