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

	rest := args[1:]
	var err error
	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
	case "init":
		err = cmd.Init()
	case "add":
		err = cmd.Add(rest)
	case "search":
		err = cmd.Search(rest)
	case "get":
		err = cmd.Get(rest)
	case "domain":
		err = cmd.Domain(rest)
	case "reindex":
		err = cmd.Reindex(rest)
	case "mcp":
		err = cmd.MCP(rest, version)
	case "version", "-v", "--version":
		fmt.Printf("recall %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "recall: unknown command %q\n", args[0])
		fmt.Fprintln(os.Stderr, "run \"recall help\" for usage")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "recall %s: %v\n", args[0], err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`recall - your memory, recalled

Usage:
  recall [command]

Commands:
  init        Initialize a new recall workspace
  add         Add a memory (--title --domain --body, or pipe body on stdin)
  search      Search memories (query plus --domain --tag --project filters)
  get         Print a memory by id
  domain      Manage domains: domain list | domain add <name> --desc "..."
  reindex     Rebuild the SQLite index from the vault
  mcp         Run the MCP server (stdio) for LLM agents
  help        Show this help
  version     Print version

Run with no command to open the interactive TUI.
`)
}
