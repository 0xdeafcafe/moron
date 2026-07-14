package main

import (
	"fmt"
	"os"

	"github.com/0xdeafcafe/moron/tui"
)

const usage = `moron — a git client you'd be a moron for using

Usage:
  moron [dir]

Opens the TUI for the repository containing dir (default: the current
directory). Worktrees listed in the branches panel can be opened with
Enter to inspect them without touching the filesystem.
`

func main() {
	dir := "."
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Print(usage)
			return
		default:
			dir = os.Args[1]
		}
	}

	if err := tui.Run(dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
