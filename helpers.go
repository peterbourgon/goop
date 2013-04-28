package goop

import (
	"log"
)

var Debug bool

func D(format string, args ...interface{}) {
	if Debug {
		log.Printf(format, args...)
	}
}
