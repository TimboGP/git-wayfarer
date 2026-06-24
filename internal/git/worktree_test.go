package git

import "testing"

func TestParseWorktrees(t *testing.T) {
	out := "worktree /repo/main\n" +
		"HEAD abc123def\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/feature\n" +
		"HEAD def456abc\n" +
		"branch refs/heads/feature\n" +
		"\n" +
		"worktree /repo/detached\n" +
		"HEAD 999aaa111\n" +
		"detached\n"

	wts := parseWorktrees(out, "/repo/feature")
	if len(wts) != 3 {
		t.Fatalf("got %d worktrees, want 3", len(wts))
	}

	if wts[0].Path != "/repo/main" || wts[0].Branch != "main" {
		t.Errorf("wt0 = %+v", wts[0])
	}
	if wts[0].IsCurrent {
		t.Errorf("wt0 should not be current")
	}
	if !wts[1].IsCurrent {
		t.Errorf("wt1 (feature) should be current")
	}
	if wts[1].Branch != "feature" {
		t.Errorf("wt1 branch = %q, want feature", wts[1].Branch)
	}
	if !wts[2].Detached || wts[2].Branch != "" {
		t.Errorf("wt2 should be detached with no branch: %+v", wts[2])
	}
	if wts[2].Head != "999aaa111" {
		t.Errorf("wt2 head = %q", wts[2].Head)
	}
}

func TestParseWorktreesBare(t *testing.T) {
	out := "worktree /repo/bare\nbare\n"
	wts := parseWorktrees(out, "/elsewhere")
	if len(wts) != 1 || !wts[0].Bare {
		t.Fatalf("expected one bare worktree, got %+v", wts)
	}
}

func TestSplitMessage(t *testing.T) {
	cases := []struct {
		in, subj, body string
	}{
		{"single line", "single line", ""},
		{"subject\n\nbody here", "subject", "body here"},
		{"subject\nbody immediately", "subject", "body immediately"},
		{"trailing newline\n", "trailing newline", ""},
	}
	for _, c := range cases {
		s, b := splitMessage(c.in)
		if s != c.subj || b != c.body {
			t.Errorf("splitMessage(%q) = (%q, %q), want (%q, %q)", c.in, s, b, c.subj, c.body)
		}
	}
}
