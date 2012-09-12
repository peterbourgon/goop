package main

import (
	"fmt"
)

func D(format string, args ...interface{}) {
	fmt.Printf("DEBUG "+format+"\n", args...)
}
