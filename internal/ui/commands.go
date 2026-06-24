package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/osleff/wayfarer/internal/git"
)

// Message types flowing through the Bubble Tea update loop.
type (
	commitsLoadedMsg struct {
		rev     string
		commits []git.Commit
		err     error
	}
	diffLoadedMsg struct {
		token   int
		content string
		err     error
	}
	actionDoneMsg struct {
		ok  string // success status text
		err error
	}
)

const commitPageSize = 500

// loadCommitsCmd loads history reachable from rev (empty = HEAD).
func loadCommitsCmd(repo *git.Repo, rev string) tea.Cmd {
	return func() tea.Msg {
		commits, err := repo.Commits(rev, commitPageSize, 0)
		return commitsLoadedMsg{rev: rev, commits: commits, err: err}
	}
}

// loadDiffCmd renders the diff base..target (file optional) at the given width.
// token lets the model discard stale results when the selection has moved on.
func loadDiffCmd(repo *git.Repo, base, target, file string, width, token int) tea.Cmd {
	return func() tea.Msg {
		content, err := repo.RenderDiff(base, target, file, width)
		return diffLoadedMsg{token: token, content: content, err: err}
	}
}

// checkoutCmd performs a mutating branch checkout.
func checkoutCmd(repo *git.Repo, branch string) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Checkout(branch); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{ok: "checked out " + branch}
	}
}

// addWorktreeCmd creates a new worktree at path.
func addWorktreeCmd(repo *git.Repo, path string) tea.Cmd {
	return func() tea.Msg {
		if err := repo.AddWorktree(path, ""); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{ok: "added worktree " + path}
	}
}

// removeWorktreeCmd removes the worktree at path.
func removeWorktreeCmd(repo *git.Repo, path string) tea.Cmd {
	return func() tea.Msg {
		if err := repo.RemoveWorktree(path); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{ok: "removed worktree " + path}
	}
}
