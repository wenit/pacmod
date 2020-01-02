package main

import (
	"github.com/wenit/pacmod/internal/commands"
	"os"
)

func main() {
	if err := commands.NewDefaultCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
