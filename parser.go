package goop

import (
	"fmt"
	"github.com/peterbourgon/field"
)

type Parser struct {
	Field *field.Field
}

func NewParser() *Parser {
	f := field.New()

	f.AddNode(NewMixer(f))

	return &Parser{
		Field: f,
	}
}

func (p *Parser) Parse(s string) error {
	return fmt.Errorf("not yet imeplemented")
}
