// Package git provides the data layer for githist. It uses go-git for
// structured reads (commit graph, branches, tree diffs) and shells out to the
// system git binary for mutations, worktree enumeration, the dirty-tree guard,
// and producing the rendered diff text fed to delta.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	gogit "github.com/go-git/go-git/v5"
)

// emptyTree is git's well-known empty tree object, used as the base when
// diffing a root commit (which has no parent).
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Repo is a handle on a single git repository (or linked worktree).
type Repo struct {
	repo     *gogit.Repository
	root     string // working-tree root
	useDelta bool   // render diffs through delta when available
}

// Open opens the repository containing dir, walking up to find the .git dir.
// If useDelta is true and the delta binary is on PATH, diffs are rendered
// through it; otherwise git's own colored output is used.
func Open(dir string, useDelta bool) (*Repo, error) {
	r, err := gogit.PlainOpenWithOptions(dir, &gogit.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, fmt.Errorf("not a git repository (or any parent): %s", dir)
	}

	root := dir
	if wt, err := r.Worktree(); err == nil && wt.Filesystem != nil {
		root = wt.Filesystem.Root()
	}

	delta := useDelta && hasDelta()

	return &Repo{repo: r, root: root, useDelta: delta}, nil
}

// Root returns the working-tree root directory.
func (r *Repo) Root() string { return r.root }

// UsingDelta reports whether diffs are rendered through delta.
func (r *Repo) UsingDelta() bool { return r.useDelta }

// runGit runs the system git binary in the repository root and returns stdout.
func (r *Repo) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

func hasDelta() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}
