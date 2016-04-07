package main

import (
	"fmt"
	"github.com/kohkimakimoto/essh/color"
	"github.com/kohkimakimoto/essh/essh"
	"os"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(color.StderrWriter, color.FgRB("[essh error] %v\n", err))
			os.Exit(1)
		}
	}()

	if err := essh.Start(); err != nil {
		fmt.Fprintf(color.StderrWriter, color.FgRB("[essh error] %v\n", err))
		os.Exit(1)
	}

	os.Exit(0)
}