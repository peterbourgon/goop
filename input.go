package main

import (
	"bufio"
	"fmt"
	"os"
)

type Input interface {
	ReadOne() (string, error)
}

//
//
//

type FileInput struct{ bufio.Reader }

func NewFileInput(filename string) (*FileInput, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	return &FileInput{*r}, nil
}

func (i *FileInput) ReadOne() (string, error) {
	line, isPrefix, err := i.ReadLine()
	if err != nil {
		return "", err
	}
	if isPrefix {
		return "", fmt.Errorf("truncated")
	}
	return string(line), nil
}

//
//
//

type InteractiveInput struct{}

func (i *InteractiveInput) ReadOne() (string, error) {
	fmt.Printf("> ")
	buf, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
	return string(buf), err
}
