package git

import (
	"sort"

	"github.com/go-git/go-git/v5/plumbing"
)

// Branch is a local or remote branch reference.
type Branch struct {
	Name      string // short name, e.g. "main" or "origin/main"
	Ref       string // full ref name
	Hash      string
	IsRemote  bool
	IsCurrent bool
}

// CurrentRef returns a human label for HEAD: the branch name when on a branch,
// or a short hash when detached.
func (r *Repo) CurrentRef() string {
	head, err := r.repo.Head()
	if err != nil {
		return "(unknown)"
	}
	if head.Name().IsBranch() {
		return head.Name().Short()
	}
	return "detached@" + head.Hash().String()[:7]
}

// Branches returns local branches first, then remote-tracking branches, each
// group sorted by name. The current branch is flagged.
func (r *Repo) Branches() ([]Branch, error) {
	current := ""
	if head, err := r.repo.Head(); err == nil && head.Name().IsBranch() {
		current = head.Name().Short()
	}

	var local, remote []Branch

	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}
	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			return nil
		}
		name := ref.Name()
		switch {
		case name.IsBranch():
			local = append(local, Branch{
				Name:      name.Short(),
				Ref:       name.String(),
				Hash:      ref.Hash().String(),
				IsCurrent: name.Short() == current,
			})
		case name.IsRemote():
			// Skip the symbolic origin/HEAD entry.
			if name.Short() == "origin/HEAD" {
				return nil
			}
			remote = append(remote, Branch{
				Name:     name.Short(),
				Ref:      name.String(),
				Hash:     ref.Hash().String(),
				IsRemote: true,
			})
		}
		return nil
	})

	sort.Slice(local, func(i, j int) bool { return local[i].Name < local[j].Name })
	sort.Slice(remote, func(i, j int) bool { return remote[i].Name < remote[j].Name })

	return append(local, remote...), nil
}

// Checkout switches the working tree to the named branch (mutating).
func (r *Repo) Checkout(branch string) error {
	_, err := r.runGit("checkout", branch)
	return err
}
