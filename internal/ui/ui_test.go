package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/osleff/wayfarer/internal/git"
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

func TestSearchFiltersCommits(t *testing.T) {
	repo, err := git.Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m := drive(t, New(repo, false, ""), tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = updateModel(m, keyPress("/"))
	if !m.searching {
		t.Fatal("expected search mode after /")
	}
	for _, ch := range []string{"s", "e", "c", "o", "n", "d"} {
		m, _ = updateModel(m, keyPress(ch))
	}
	if m.searchQuery != "second" {
		t.Fatalf("query = %q, want second", m.searchQuery)
	}
	if len(m.visible) != 1 || m.visible[0].Subject != "second commit" {
		t.Fatalf("visible = %d (%+v), want 1 (second commit)", len(m.visible), m.visible)
	}

	m, _ = updateModel(m, keyPress("enter"))
	if m.searching {
		t.Error("still in search mode after enter")
	}
	if !strings.Contains(m.View().Content, "Commits (1/2)") {
		t.Errorf("view missing filtered count:\n%s", m.View().Content)
	}

	m, _ = updateModel(m, keyPress("esc"))
	if m.searchQuery != "" || len(m.visible) != 2 {
		t.Errorf("esc did not clear filter: query=%q visible=%d", m.searchQuery, len(m.visible))
	}
}

func TestCommitDetailOverlay(t *testing.T) {
	repo, err := git.Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m := drive(t, New(repo, false, ""), tea.WindowSizeMsg{Width: 120, Height: 40})

	m, _ = updateModel(m, keyPress("enter"))
	if m.overlay != ovDetail {
		t.Fatalf("overlay = %v, want ovDetail", m.overlay)
	}
	content := m.View().Content
	for _, want := range []string{"commit ", "Author:", "test@example.com", "second commit"} {
		if !strings.Contains(content, want) {
			t.Errorf("detail view missing %q", want)
		}
	}

	m, _ = updateModel(m, keyPress("esc"))
	if m.overlay != ovNone {
		t.Error("detail overlay did not close on esc")
	}
}

func TestWorkingAndStagedRows(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("unstaged edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("staged edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "b.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %s", out)
	}

	repo, err := git.Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	m := drive(t, New(repo, false, ""), tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.statusCount() != 2 {
		t.Fatalf("statusCount = %d, want 2 (working + staged)", m.statusCount())
	}
	content := m.View().Content
	for _, want := range []string{"Working tree", "Staged"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q row:\n%s", want, content)
		}
	}

	// Default selection is HEAD (first commit), not a status row.
	if _, ok := m.selectedStatus(); ok {
		t.Error("default selection should be a commit, not a status row")
	}

	// Move up onto the staged row, then the working row, and confirm the file
	// list reflects the staged/unstaged change.
	m, _ = updateModel(m, keyPress("k")) // staged (index 1)
	if st, ok := m.selectedStatus(); !ok || st.kind != stStaged {
		t.Fatalf("expected staged row selected, got ok=%v", ok)
	}
	if sf := fileNames(m.files); !contains(sf, "b.txt") {
		t.Errorf("staged files = %v, want b.txt", sf)
	}

	m, _ = updateModel(m, keyPress("k")) // working (index 0)
	if st, ok := m.selectedStatus(); !ok || st.kind != stWorking {
		t.Fatalf("expected working row selected, got ok=%v", ok)
	}
	if wf := fileNames(m.files); !contains(wf, "a.txt") {
		t.Errorf("working files = %v, want a.txt", wf)
	}
}

func fileNames(files []git.FileChange) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
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
