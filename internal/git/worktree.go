package git

import (
	"path/filepath"
	"strings"
)

// Worktree is one entry from `git worktree list`.
type Worktree struct {
	Path      string
	Head      string
	Branch    string // short branch name, empty when detached
	Detached  bool
	Bare      bool
	Locked    bool
	IsCurrent bool
}

// Worktrees lists the repository's worktrees (the main one plus any linked
// worktrees), flagging the one this Repo is currently pointed at.
func (r *Repo) Worktrees() ([]Worktree, error) {
	out, err := r.runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(out, r.root), nil
}

// parseWorktrees parses the output of `git worktree list --porcelain`.
// Records are separated by blank lines; each record starts with a "worktree"
// line followed by attribute lines (HEAD, branch, detached, bare, locked).
func parseWorktrees(out, current string) []Worktree {
	var res []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil {
			cur.IsCurrent = sameDir(cur.Path, current)
			res = append(res, *cur)
			cur = nil
		}
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: val}
		case "HEAD":
			if cur != nil {
				cur.Head = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(val, "refs/heads/")
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
			}
		}
	}
	flush()
	return res
}

func sameDir(a, b string) bool {
	ap, err1 := filepath.Abs(a)
	bp, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return filepath.Clean(ap) == filepath.Clean(bp)
}

// AddWorktree creates a new worktree at path. When branch is non-empty it
// checks out that branch; otherwise git uses its default behavior.
func (r *Repo) AddWorktree(path, branch string) error {
	args := []string{"worktree", "add", path}
	if branch != "" {
		args = append(args, branch)
	}
	_, err := r.runGit(args...)
	return err
}

// RemoveWorktree removes the worktree at path.
func (r *Repo) RemoveWorktree(path string) error {
	_, err := r.runGit("worktree", "remove", path)
	return err
}
