package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/osleff/wayfarer/internal/git"
)

var graphAnsi = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripGraph(s string) string { return graphAnsi.ReplaceAllString(s, "") }

func TestComputeGraphLinear(t *testing.T) {
	commits := []git.Commit{
		{Hash: "c", Parents: []string{"b"}},
		{Hash: "b", Parents: []string{"a"}},
		{Hash: "a"},
	}
	rows := computeGraph(commits)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for i, r := range rows {
		s := stripGraph(r)
		if !strings.HasPrefix(s, "●") {
			t.Errorf("row %d = %q, want a node first", i, s)
		}
		if strings.Contains(s, "│") {
			t.Errorf("linear row %d should have no extra lane: %q", i, s)
		}
	}
}

func TestComputeGraphMerge(t *testing.T) {
	// Diamond (newest first): e -> m(parents d,c); d -> b; c -> b; b -> a; a.
	commits := []git.Commit{
		{Hash: "e", Parents: []string{"m"}},
		{Hash: "m", Parents: []string{"d", "c"}},
		{Hash: "d", Parents: []string{"b"}},
		{Hash: "c", Parents: []string{"b"}},
		{Hash: "b", Parents: []string{"a"}},
		{Hash: "a"},
	}
	rows := computeGraph(commits)
	var joined strings.Builder
	for _, r := range rows {
		joined.WriteString(stripGraph(r) + "\n")
	}
	out := joined.String()

	if !strings.Contains(out, "◆") {
		t.Errorf("merge commit should render a ◆ node:\n%s", out)
	}
	if !strings.Contains(out, "│") {
		t.Errorf("diamond should draw a second lane:\n%s", out)
	}
}
