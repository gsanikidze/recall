package main

import (
	"fmt"
	"os"

	"recall/cmd"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		// No subcommand: launch the interactive TUI.
		runTUI()
		return
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
	case "init":
		if err := cmd.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "recall init: %v\n", err)
			os.Exit(1)
		}
	case "version", "-v", "--version":
		fmt.Printf("recall %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "recall: unknown command %q\n", args[0])
		fmt.Fprintln(os.Stderr, "run \"recall help\" for usage")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`recall - your memory, recalled

Usage:
  recall [command]

Commands:
  init        Initialize a new recall workspace
  help        Show this help
  version     Print version

Run with no command to open the interactive TUI.
`)
}
