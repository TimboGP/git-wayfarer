package git

import (
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

// FileChange describes a single changed path between two trees.
type FileChange struct {
	Path    string // current path (or old path for deletions)
	OldPath string // populated for renames/copies
	Status  string // A, M, D, R, C, T
}

// base resolves the diff base for a commit: its first parent, or the empty tree
// for a root commit.
func (r *Repo) base(hash string) (string, error) {
	c, err := r.repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return "", err
	}
	if c.NumParents() > 0 {
		return c.ParentHashes[0].String(), nil
	}
	return emptyTree, nil
}

// ChangedFiles returns the files changed by a commit relative to its first
// parent (or the empty tree for a root commit), with rename detection.
func (r *Repo) ChangedFiles(hash string) ([]FileChange, error) {
	base, err := r.base(hash)
	if err != nil {
		return nil, err
	}
	return r.changedFilesDiff(base, hash)
}

// ChangedFilesRange returns the files that differ between commits a and b
// (the diff a..b), with rename detection.
func (r *Repo) ChangedFilesRange(a, b string) ([]FileChange, error) {
	return r.changedFilesDiff(a, b)
}

// changedFilesDiff shells out to git for an accurate name-status listing
// (go-git's tree diff cannot detect renames).
func (r *Repo) changedFilesDiff(base, target string) ([]FileChange, error) {
	out, err := r.runGit("diff", "--name-status", "--find-renames", base, target)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

// parseNameStatus parses `git diff --name-status -M` output. Lines are
// tab-separated: "M\tpath", "A\tpath", "D\tpath", or "R100\told\tnew".
func parseNameStatus(out string) []FileChange {
	var files []FileChange
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[0] == "" {
			continue
		}
		fc := FileChange{}
		switch fields[0][0] {
		case 'A':
			fc.Status, fc.Path = "A", fields[1]
		case 'D':
			fc.Status, fc.Path = "D", fields[1]
		case 'R':
			if len(fields) >= 3 {
				fc.Status, fc.OldPath, fc.Path = "R", fields[1], fields[2]
			} else {
				fc.Status, fc.Path = "R", fields[1]
			}
		case 'C':
			if len(fields) >= 3 {
				fc.Status, fc.OldPath, fc.Path = "C", fields[1], fields[2]
			} else {
				fc.Status, fc.Path = "C", fields[1]
			}
		case 'T': // type change (e.g. file <-> symlink)
			fc.Status, fc.Path = "M", fields[1]
		default: // M and anything else
			fc.Status, fc.Path = "M", fields[1]
		}
		files = append(files, fc)
	}
	return files
}

// StagedFiles lists files staged in the index (index vs HEAD).
func (r *Repo) StagedFiles() ([]FileChange, error) {
	out, err := r.runGit("diff", "--cached", "--name-status", "--find-renames")
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

// WorkingFiles lists unstaged changes (working tree vs index), including
// untracked files (marked with status "?").
func (r *Repo) WorkingFiles() ([]FileChange, error) {
	out, err := r.runGit("diff", "--name-status", "--find-renames")
	if err != nil {
		return nil, err
	}
	files := parseNameStatus(out)

	untracked, err := r.runGit("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(strings.TrimRight(untracked, "\n"), "\n") {
		if p == "" {
			continue
		}
		files = append(files, FileChange{Status: "?", Path: p})
	}
	return files, nil
}

// CommitStat returns the `git diff --stat` summary for a commit relative to its
// first parent, used in the commit detail view.
func (r *Repo) CommitStat(hash string) (string, error) {
	base, err := r.base(hash)
	if err != nil {
		return "", err
	}
	return r.runGit("diff", "--stat", "--find-renames", base, hash)
}
