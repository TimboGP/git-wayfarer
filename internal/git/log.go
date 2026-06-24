package git

import (
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Commit is a single entry in the history, with refs that point at it decorated.
type Commit struct {
	Hash    string
	Short   string
	Author  string
	Email   string
	When    time.Time
	Subject string
	Body    string
	Parents []string
	Refs    []string // branch/tag/HEAD labels pointing at this commit
}

// Commits returns up to limit commits reachable from rev (HEAD when rev is
// empty), skipping the first skip commits. Ordering follows committer time to
// match `git log`.
func (r *Repo) Commits(rev string, limit, skip int) ([]Commit, error) {
	opts := &gogit.LogOptions{Order: gogit.LogOrderCommitterTime}
	if rev != "" {
		h, err := r.repo.ResolveRevision(plumbing.Revision(rev))
		if err != nil {
			return nil, err
		}
		opts.From = *h
	}

	iter, err := r.repo.Log(opts)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	decor := r.refDecorations()

	var commits []Commit
	i := 0
	err = iter.ForEach(func(c *object.Commit) error {
		defer func() { i++ }()
		if i < skip {
			return nil
		}
		if limit > 0 && len(commits) >= limit {
			return storer.ErrStop
		}
		commits = append(commits, toCommit(c, decor))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func toCommit(c *object.Commit, decor map[string][]string) Commit {
	subject, body := splitMessage(c.Message)
	parents := make([]string, 0, c.NumParents())
	for _, p := range c.ParentHashes {
		parents = append(parents, p.String())
	}
	hash := c.Hash.String()
	return Commit{
		Hash:    hash,
		Short:   hash[:7],
		Author:  c.Author.Name,
		Email:   c.Author.Email,
		When:    c.Author.When,
		Subject: subject,
		Body:    body,
		Parents: parents,
		Refs:    decor[hash],
	}
}

func splitMessage(msg string) (subject, body string) {
	msg = strings.TrimRight(msg, "\n")
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		return strings.TrimSpace(msg[:i]), strings.TrimSpace(msg[i+1:])
	}
	return strings.TrimSpace(msg), ""
}

// refDecorations maps a commit hash to the labels (HEAD, branches, tags,
// remotes) that point at it.
func (r *Repo) refDecorations() map[string][]string {
	m := map[string][]string{}

	if refs, err := r.repo.References(); err == nil {
		_ = refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Type() != plumbing.HashReference {
				return nil
			}
			name := ref.Name()
			var label string
			switch {
			case name.IsBranch():
				label = name.Short()
			case name.IsTag():
				label = "tag: " + name.Short()
			case name.IsRemote():
				label = name.Short()
			default:
				return nil
			}
			h := ref.Hash().String()
			m[h] = append(m[h], label)
			return nil
		})
	}

	if head, err := r.repo.Head(); err == nil {
		h := head.Hash().String()
		m[h] = append([]string{"HEAD"}, m[h]...)
	}

	return m
}
