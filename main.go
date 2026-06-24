// Command wayfarer is an interactive terminal UI for walking git history,
// switching branches/worktrees, and inspecting diffs between changesets.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/osleff/wayfarer/internal/git"
	"github.com/osleff/wayfarer/internal/ui"
)

// Build-time metadata, injected by GoReleaser via -ldflags (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		dir     string
		noDelta bool
		showVer bool
	)
	flag.StringVar(&dir, "C", "", "run as if wayfarer was started in `dir`")
	flag.BoolVar(&noDelta, "no-delta", false, "disable delta rendering (use git's colored diff)")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.Usage = usage
	flag.Parse()

	if showVer {
		fmt.Printf("wayfarer %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			fatal(err)
		}
		dir = wd
	}

	useDelta := !noDelta
	repo, err := git.Open(dir, useDelta)
	if err != nil {
		fatal(err)
	}

	rev := flag.Arg(0) // optional starting revision
	model := ui.New(repo, useDelta, rev)

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `wayfarer %s — interactive git history TUI

Usage:
  wayfarer [flags] [revision]

Arguments:
  revision    optional revision to start browsing from (default: HEAD)

Flags:
  -C dir       run as if started in dir
  --no-delta   use git's colored diff instead of delta
  --version    print version and exit

Keys (press ? inside the app for the full list):
  ↑/k ↓/j  navigate      tab  switch pane     v  mark A/B to compare
  b  branches            w  worktrees         r  refresh        q  quit
`, version)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "wayfarer:", err)
	os.Exit(1)
}
