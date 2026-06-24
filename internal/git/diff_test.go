package git

import (
	"os"
	"os/exec"
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
