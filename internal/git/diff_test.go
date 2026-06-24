package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNameStatus(t *testing.T) {
	out := "M\tmain.go\n" +
		"A\tnew.go\n" +
		"D\tgone.go\n" +
		"R100\told/name.go\tnew/name.go\n" +
		"C075\tsrc.go\tcopy.go\n" +
		"T\tlink\n"

	files := parseNameStatus(out)
	if len(files) != 6 {
		t.Fatalf("got %d entries, want 6", len(files))
	}
	want := []FileChange{
		{Status: "M", Path: "main.go"},
		{Status: "A", Path: "new.go"},
		{Status: "D", Path: "gone.go"},
		{Status: "R", OldPath: "old/name.go", Path: "new/name.go"},
		{Status: "C", OldPath: "src.go", Path: "copy.go"},
		{Status: "M", Path: "link"}, // type change normalized to M
	}
	for i, w := range want {
		if files[i] != w {
			t.Errorf("entry %d = %+v, want %+v", i, files[i], w)
		}
	}
}

func TestRenameDetectionIntegration(t *testing.T) {
	dir := newTestRepo(t)
	gitRun(t, dir, "mv", "a.txt", "renamed.txt")
	gitRun(t, dir, "commit", "-qm", "rename a.txt to renamed.txt")

	r, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	commits, err := r.Commits("", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	files, err := r.ChangedFiles(commits[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("rename commit changed %d files, want 1: %+v", len(files), files)
	}
	got := files[0]
	if got.Status != "R" || got.OldPath != "a.txt" || got.Path != "renamed.txt" {
		t.Errorf("rename = %+v, want {R a.txt renamed.txt}", got)
	}

	// CommitStat should mention the rename.
	stat, err := r.CommitStat(commits[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stat, "renamed.txt") {
		t.Errorf("stat missing renamed.txt:\n%s", stat)
	}
}

func TestWorkingAndStagedFiles(t *testing.T) {
	dir := newTestRepo(t)
	write := func(n, c string) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.txt", "modified but unstaged\n") // tracked, unstaged
	write("new.txt", "brand new file\n")      // untracked
	write("b.txt", "modified and staged\n")
	gitRun(t, dir, "add", "b.txt") // staged

	r, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	working, err := r.WorkingFiles()
	if err != nil {
		t.Fatal(err)
	}
	wm := statusMap(working)
	if wm["a.txt"] != "M" {
		t.Errorf("working a.txt = %q, want M", wm["a.txt"])
	}
	if wm["new.txt"] != "?" {
		t.Errorf("working new.txt = %q, want ? (untracked)", wm["new.txt"])
	}

	staged, err := r.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if sm := statusMap(staged); sm["b.txt"] != "M" {
		t.Errorf("staged b.txt = %q, want M", sm["b.txt"])
	}

	if d, _ := r.RenderStaged("", 80); !strings.Contains(d, "b.txt") {
		t.Errorf("staged diff missing b.txt:\n%s", d)
	}
	if d, _ := r.RenderWorking("", 80); !strings.Contains(d, "a.txt") {
		t.Errorf("working diff missing a.txt:\n%s", d)
	}
	d, err := r.RenderUntracked("new.txt", 80)
	if err != nil {
		t.Fatalf("RenderUntracked error: %v", err)
	}
	if !strings.Contains(d, "brand new file") {
		t.Errorf("untracked diff missing file contents:\n%s", d)
	}
}

// gitRun runs a git command in dir, failing the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
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
