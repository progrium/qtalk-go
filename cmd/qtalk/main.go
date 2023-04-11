package main

import (
	"context"
	"log"
	"os"

	"github.com/progrium/qtalk-go/cmd/qtalk/cli"
)

func main() {
	root := &cli.Command{
		Usage: "qtalk",
		Long:  `qtalk is a utility for working with the qtalk protocol stack`,
	}

	root.AddCommand(callCmd)
	root.AddCommand(interopCmd)
	root.AddCommand(checkCmd)
	root.AddCommand(benchCmd)

	if err := cli.Execute(context.Background(), root, os.Args[1:]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
