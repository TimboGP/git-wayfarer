package git

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// FileChange describes a single changed path between two trees.
type FileChange struct {
	Path    string // current path (or old path for deletions)
	OldPath string // populated for renames/copies
	Status  string // A, M, D, R
}

// ChangedFiles returns the files changed by a commit relative to its first
// parent (or the empty tree for a root commit).
func (r *Repo) ChangedFiles(hash string) ([]FileChange, error) {
	c, err := r.repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, err
	}
	to, err := c.Tree()
	if err != nil {
		return nil, err
	}

	var from *object.Tree
	if c.NumParents() > 0 {
		p, err := c.Parent(0)
		if err != nil {
			return nil, err
		}
		if from, err = p.Tree(); err != nil {
			return nil, err
		}
	}

	return treeChanges(from, to)
}

// ChangedFilesRange returns the files that differ between commits a and b
// (the diff a..b).
func (r *Repo) ChangedFilesRange(a, b string) ([]FileChange, error) {
	from, err := r.commitTree(a)
	if err != nil {
		return nil, err
	}
	to, err := r.commitTree(b)
	if err != nil {
		return nil, err
	}
	return treeChanges(from, to)
}

func (r *Repo) commitTree(rev string) (*object.Tree, error) {
	h, err := r.repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return nil, err
	}
	c, err := r.repo.CommitObject(*h)
	if err != nil {
		return nil, err
	}
	return c.Tree()
}

func treeChanges(from, to *object.Tree) ([]FileChange, error) {
	if from == nil {
		from = &object.Tree{}
	}
	changes, err := from.Diff(to)
	if err != nil {
		return nil, err
	}

	out := make([]FileChange, 0, len(changes))
	for _, ch := range changes {
		action, err := ch.Action()
		if err != nil {
			return nil, err
		}
		fc := FileChange{}
		switch action {
		case merkletrie.Insert:
			fc.Status = "A"
			fc.Path = ch.To.Name
		case merkletrie.Delete:
			fc.Status = "D"
			fc.Path = ch.From.Name
		case merkletrie.Modify:
			fc.Status = "M"
			fc.Path = ch.To.Name
			if ch.From.Name != ch.To.Name {
				fc.Status = "R"
				fc.OldPath = ch.From.Name
			}
		}
		out = append(out, fc)
	}
	return out, nil
}
