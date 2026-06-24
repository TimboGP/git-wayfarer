package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/osleff/git-history-tool/internal/git"
)

func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	write("a.txt", "hello\n")
	run("add", "a.txt")
	run("commit", "-qm", "first commit")
	write("a.txt", "hello world\n")
	write("b.txt", "new file\n")
	run("add", ".")
	run("commit", "-qm", "second commit")
	return dir
}

// drive runs a model through Init + the given messages, executing every
// returned command synchronously and feeding resulting messages back.
func drive(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	var model tea.Model = m
	run := func(cmd tea.Cmd) {
		for cmd != nil {
			msg := cmd()
			if msg == nil {
				return
			}
			model, cmd = model.Update(msg)
		}
	}
	run(model.(Model).Init())
	for _, msg := range msgs {
		var cmd tea.Cmd
		model, cmd = model.Update(msg)
		run(cmd)
	}
	return model.(Model)
}

func TestModelRendersHistoryAndDiff(t *testing.T) {
	repo, err := git.Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m := drive(t, New(repo, false, ""), tea.WindowSizeMsg{Width: 120, Height: 40})

	content := m.View().Content
	if content == "" {
		t.Fatal("empty view")
	}
	// "first commit" (root, undecorated) always fits; the decorated tip
	// subject can truncate in a narrow pane, so only assert a prefix of it.
	for _, want := range []string{"Commits (2)", "first commit", "second", "Diff", "Changes"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q", want)
		}
	}
	// The default selection (newest commit) added b.txt, so its diff should
	// have loaded and the overall changes should list b.txt.
	if !strings.Contains(content, "b.txt") {
		t.Errorf("view missing b.txt from changes/diff:\n%s", content)
	}
}

func TestCompareModeAcrossCommits(t *testing.T) {
	repo, err := git.Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	// Mark A on newest, move down to root, mark B -> compare A..B.
	m := drive(t, New(repo, false, ""),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		keyPress("v"),
		keyPress("j"),
		keyPress("v"),
	)
	if !m.compareMode() {
		t.Fatalf("expected compare mode after two marks")
	}
	content := m.View().Content
	if !strings.Contains(content, "COMPARE") {
		t.Errorf("header missing COMPARE indicator:\n%s", content)
	}
}

func TestTabCyclesFocus(t *testing.T) {
	repo, err := git.Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m := drive(t, New(repo, false, ""), tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.focus != focusCommits {
		t.Fatalf("initial focus = %v", m.focus)
	}
	m, _ = updateModel(m, keyPress("tab"))
	if m.focus != focusFiles {
		t.Fatalf("after one tab focus = %v, want files", m.focus)
	}
	m, _ = updateModel(m, keyPress("tab"))
	if m.focus != focusDiff {
		t.Fatalf("after two tabs focus = %v, want diff", m.focus)
	}
}

func updateModel(m Model, msg tea.Msg) (Model, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(Model), cmd
}

func keyPress(s string) tea.KeyPressMsg {
	r := []rune(s)
	if len(r) == 1 {
		return tea.KeyPressMsg{Code: r[0], Text: s}
	}
	// named keys (tab, esc, ...) map onto a key code via the rune-less path;
	// String() returns the provided name through Code lookup below.
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	return tea.KeyPressMsg{Text: s}
}
