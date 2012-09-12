package main

import (
	"fmt"
	"math"
	"strings"
)

type Note interface {
	String() string
	Hz() float32
}

type note struct {
	str string
	hz  float32
}

func (n note) String() string { return n.str }
func (n note) Hz() float32    { return n.hz }

func NoteZero() Note { return note{"Ø", 0.0} }

func ParseNote(s string) (Note, error) {
	ss := strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 {
		return nil, fmt.Errorf("too short")
	}

	str := ""
	offset := 0
	switch ss[0] {
	case 'c':
		offset = 0
		str += "C"
	case 'd':
		offset = 2
		str += "D"
	case 'e':
		offset = 4
		str += "E"
	case 'f':
		offset = 5
		str += "F"
	case 'g':
		offset = 7
		str += "G"
	case 'a':
		offset = 9
		str += "A"
	case 'b':
		offset = 11
		str += "B"
	default:
		return nil, fmt.Errorf("unrecognized note")
	}

	switch ss[1] {
	case '#':
		offset++
		str += "#"
	case 'b':
		offset--
		str += "b"
	}

	octaveChar := ss[1]
	if len(s) == 3 {
		octaveChar = ss[2]
	}
	octave := 0
	switch octaveChar {
	case '0':
		octave = 0
	case '1':
		octave = 1
	case '2':
		octave = 2
	case '3':
		octave = 3
	case '4':
		octave = 4
	case '5':
		octave = 5
	case '6':
		octave = 6
	case '7':
		octave = 7
	case '8':
		octave = 8
	case '9':
		octave = 9
	default:
		return nil, fmt.Errorf("unrecognized octave")
	}
	str += string(octaveChar)

	// http://en.wikipedia.org/wiki/Note#Note_frequency_.28hertz.29
	p := (12 * octave) + offset
	hz := math.Pow(2, (float64(p)-69.0)/12.0) * 440.0
	n := note{str, float32(hz)}
	return n, nil
}
