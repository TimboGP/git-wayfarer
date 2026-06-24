# githist

An interactive terminal UI for walking git history, switching branches and
worktrees, and inspecting diffs between changesets — all without leaving the
keyboard.

```
 githist   repo myproject  HEAD main  delta
╭───────────────────────────────╮╭──────────────────────────────────────────╮
│Commits (248)                  ││Changes (3)                                 │
│  a1b2c3d (HEAD, main) Fix … 2h ││▸ Overall (all files)                       │
│  e4f5g6h Add caching      5h  ││M internal/cache/cache.go                   │
│  …                            ││A internal/cache/cache_test.go              │
│                               │╰──────────────────────────────────────────╯
│                               │╭──────────────────────────────────────────╮
│                               ││Diff · a1b2c3d                              │
│                               ││  (delta-rendered diff scrolls here)        │
╰───────────────────────────────╯╰──────────────────────────────────────────╯
↑/k up • ↓/j down • tab switch pane • v mark A/B • b branches • w worktrees • ?
```

## Features

- **Walk history** commit by commit with `git log`-style decoration (HEAD,
  branches, tags) on each entry.
- **Inspect changes** for the selected commit (vs. its first parent), per file
  or as one overall diff — with rename detection (`R old → new`).
- **Commit detail view** — press `enter` for the full message, author/date,
  parents, refs, and diffstat in a scrollable overlay.
- **Search history** — press `/` to live-filter commits by subject, author, or
  hash.
- **Compare changesets** — mark commit `A`, move, mark commit `B`, and view the
  full `A..B` range diff.
- **Switch branches** non-destructively (re-point the view to inspect another
  branch's history) or destructively (`git checkout`, guarded by a dirty-tree
  confirmation).
- **Switch worktrees** — open another worktree in place, or add/remove
  worktrees from the UI.
- **delta-aware diffs** — renders through [delta](https://github.com/dandavison/delta)
  when it's on your `PATH`, falling back to git's own colored output.

## Install

Build from source (produces a `githist` binary):

```sh
go build -o githist .
```

Or install with `go install` (note: this installs a binary named
`git-history-tool`, matching the module path — rename or alias it to `githist`
if you like):

```sh
go install github.com/osleff/git-history-tool@latest
```

Requires Go 1.26+ and a `git` binary on `PATH`. `delta` is optional but
recommended.

## Usage

```sh
githist                 # browse the repo in the current directory, from HEAD
githist v1.2.0          # start browsing from a specific revision
githist -C ~/code/repo  # browse a repo elsewhere
githist --no-delta      # force git's colored diff instead of delta
githist --version
```

### Keys

| Key | Action |
| --- | --- |
| `↑`/`k`, `↓`/`j` | move within the focused pane |
| `pgup`/`pgdn`, `g`/`G` | page / jump to top / bottom |
| `tab` | cycle focus: commits → changes → diff |
| `enter` | show full commit details (message, diffstat) |
| `/` | search / filter commits by subject, author, or hash |
| `v` | mark commit `A`, then `B`, to compare a range (`esc` clears) |
| `b` | open the branches overlay (`enter` inspect · `c` checkout) |
| `w` | open the worktrees overlay (`enter` open · `a` add · `d` remove) |
| `r` | refresh |
| `?` | full keybinding help |
| `q` / `ctrl+c` | quit |

## How it works

`githist` uses a hybrid git backend:

- **[go-git](https://github.com/go-git/go-git)** drives the structured reads —
  walking the commit graph, listing branches, and computing changed-file lists
  from tree diffs.
- The system **`git`** binary handles mutations (`checkout`, `worktree`),
  worktree enumeration, the dirty-tree guard (`git status --porcelain`), and
  producing the diff text that is piped through `delta`.

> A TUI cannot change its parent shell's working directory, so "switching" a
> worktree re-points the running app at the selected worktree's path rather than
> `cd`-ing your shell.

Built with the [Charm](https://charm.sh) stack — Bubble Tea, Bubbles, and Lip
Gloss.

## Development

```sh
go test ./...     # unit + integration + headless UI tests
go vet ./...
```
