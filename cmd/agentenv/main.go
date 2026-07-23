package main

import (
	"os"

	"github.com/flobilosaurus/agent-env/internal/cli"
)

func main() { os.Exit(cli.App{}.Run(os.Args[1:])) }
