package main

import (
	"os"

	"github.com/etherinus/cfgstack/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args))
}
