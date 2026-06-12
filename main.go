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
		err = cmd.Init(rest)
	case "add":
		err = cmd.Add(rest)
	case "search":
		err = cmd.Search(rest)
	case "embed":
		err = cmd.Embed(rest)
	case "get":
		err = cmd.Get(rest)
	case "delete", "rm":
		err = cmd.Delete(rest)
	case "domain":
		err = cmd.Domain(rest)
	case "doctor":
		err = cmd.Doctor(rest)
	case "reindex":
		err = cmd.Reindex(rest)
	case "mcp":
		err = cmd.MCP(rest, version)
	case "ui":
		err = cmd.UI(rest)
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
  init        Initialize a new recall workspace (--path DIR --force)
  add         Add a memory (--title --domain --body, or pipe body on stdin)
  search      Search memories (default keyword; --mode keyword|semantic|hybrid or --semantic/--hybrid for vectors)
  embed       Embed indexed memories (--provider ollama|fake --model MODEL)
  get         Print a memory by id
  delete      Delete a memory by id
  domain      Manage domains: domain list | domain add <name> --desc "..."
  doctor      Check config, vault, SQLite index, and domains
  reindex     Rebuild the SQLite index from the vault
  mcp         Run the MCP server (stdio) for LLM agents
  ui          Start the web UI at localhost:8888 (--port N --no-browser)
  help        Show this help
  version     Print version

Vector search quick start:
  ollama pull nomic-embed-text
  recall embed --provider ollama --model nomic-embed-text
  recall search "phone sync" --hybrid --provider ollama --model nomic-embed-text

Run with no command to open the interactive TUI.
`)
}
