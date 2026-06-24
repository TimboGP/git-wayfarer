package git

import "strings"

// IsDirty reports whether the working tree has uncommitted changes (tracked or
// untracked). It is the guard run before a mutating checkout.
func (r *Repo) IsDirty() (bool, error) {
	out, err := r.runGit("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// StatusLines returns the porcelain status entries (empty when clean).
func (r *Repo) StatusLines() ([]string, error) {
	out, err := r.runGit("status", "--porcelain")
	if err != nil {
		return nil, err
	}
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
