package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestRepo builds a temp repository with two commits (root adds a.txt;
// second modifies a.txt and adds b.txt) plus a "feature" branch.
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
	run("commit", "-qm", "second commit\n\nthis is the body")

	run("branch", "feature")
	return dir
}

func TestCommitsAndDecoration(t *testing.T) {
	r, err := Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}

	commits, err := r.Commits("", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	if commits[0].Subject != "second commit" {
		t.Errorf("newest subject = %q", commits[0].Subject)
	}
	if commits[0].Body != "this is the body" {
		t.Errorf("newest body = %q", commits[0].Body)
	}
	if len(commits[0].Parents) != 1 {
		t.Errorf("newest should have one parent, got %v", commits[0].Parents)
	}
	if len(commits[1].Parents) != 0 {
		t.Errorf("root should have no parents, got %v", commits[1].Parents)
	}

	// HEAD and both branch labels should decorate the tip.
	refs := strings.Join(commits[0].Refs, ",")
	for _, want := range []string{"HEAD", "feature"} {
		if !strings.Contains(refs, want) {
			t.Errorf("tip refs %q missing %q", refs, want)
		}
	}
}

func TestChangedFiles(t *testing.T) {
	r, err := Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	commits, err := r.Commits("", 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Second commit: a.txt modified, b.txt added.
	files, err := r.ChangedFiles(commits[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	got := statusMap(files)
	if got["a.txt"] != "M" {
		t.Errorf("a.txt status = %q, want M", got["a.txt"])
	}
	if got["b.txt"] != "A" {
		t.Errorf("b.txt status = %q, want A", got["b.txt"])
	}

	// Root commit: a.txt added against the empty tree.
	rootFiles, err := r.ChangedFiles(commits[1].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if m := statusMap(rootFiles); m["a.txt"] != "A" {
		t.Errorf("root a.txt status = %q, want A", m["a.txt"])
	}
}

func TestRangeAndRenderDiff(t *testing.T) {
	r, err := Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}
	commits, _ := r.Commits("", 10, 0)
	root, tip := commits[1].Hash, commits[0].Hash

	files, err := r.ChangedFilesRange(root, tip)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("range root..tip changed %d files, want 2", len(files))
	}

	// RenderDiff (git colored path) should mention the new file.
	diff, err := r.RenderDiff(root, tip, "", 80)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "b.txt") {
		t.Errorf("range diff missing b.txt:\n%s", diff)
	}

	// Scoped to a single file.
	scoped, err := r.RenderDiff(commits[0].Parents[0], tip, "b.txt", 80)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scoped, "b.txt") || strings.Contains(scoped, "a.txt") {
		t.Errorf("file-scoped diff wrong:\n%s", scoped)
	}
}

func TestBranchesStatusWorktrees(t *testing.T) {
	r, err := Open(newTestRepo(t), false)
	if err != nil {
		t.Fatal(err)
	}

	branches, err := r.Branches()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	current := 0
	for _, b := range branches {
		names = append(names, b.Name)
		if b.IsCurrent {
			current++
		}
	}
	if !contains(names, "feature") {
		t.Errorf("branches %v missing feature", names)
	}
	if current != 1 {
		t.Errorf("expected exactly one current branch, got %d", current)
	}

	// Clean then dirty.
	if dirty, _ := r.IsDirty(); dirty {
		t.Error("fresh repo reported dirty")
	}
	if err := os.WriteFile(filepath.Join(r.Root(), "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirty, _ := r.IsDirty(); !dirty {
		t.Error("modified repo reported clean")
	}

	wts, err := r.Worktrees()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 || !wts[0].IsCurrent {
		t.Errorf("expected one current worktree, got %+v", wts)
	}
}

func statusMap(files []FileChange) map[string]string {
	m := make(map[string]string, len(files))
	for _, f := range files {
		m[f.Path] = f.Status
	}
	return m
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
