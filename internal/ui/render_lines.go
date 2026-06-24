package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/osleff/git-history-tool/internal/git"
)

// Selection styling rule: a selected row is rendered as plain text wrapped in a
// single uniform highlight (so a background color isn't broken by inner ANSI
// resets); an unselected row keeps its per-token colors.

// renderCommitLine renders one commit row: mark, short hash, refs, subject,
// and a right-aligned relative age. The subject is prioritized for space.
func renderCommitLine(c git.Commit, markA, markB string, selected, focused bool, width int) string {
	markCh := " "
	switch c.Hash {
	case markA:
		markCh = "A"
	case markB:
		markCh = "B"
	}

	prefix := markCh + " " + c.Short + " "
	age := humanizeTime(c.When)
	prefixW := runeLen(prefix)
	ageW := runeLen(age)

	avail := width - prefixW - ageW - 1
	if avail < 1 {
		avail = 1
	}

	refsPlain := plainRefs(c.Refs)
	refsW := runeLen(refsPlain)
	showRefs := refsPlain != "" && refsW+2 <= avail

	subjAvail := avail
	if showRefs {
		subjAvail = avail - refsW - 1
	}
	subj := truncate(c.Subject, subjAvail)

	bodyW := runeLen(subj)
	if showRefs {
		bodyW += refsW + 1
	}
	pad := width - prefixW - bodyW - ageW
	if pad < 1 {
		pad = 1
	}

	if selected {
		body := subj
		if showRefs {
			body = refsPlain + " " + subj
		}
		line := prefix + body + strings.Repeat(" ", pad) + age
		return selLine(line, width, focused)
	}

	// Colored variant.
	mark := markCh
	switch c.Hash {
	case markA:
		mark = lipgloss.NewStyle().Foreground(colMarkA).Bold(true).Render("A")
	case markB:
		mark = lipgloss.NewStyle().Foreground(colMarkB).Bold(true).Render("B")
	}
	body := subj
	if showRefs {
		body = refBadges(c.Refs) + " " + subj
	}
	return mark + " " + hashStyle.Render(c.Short) + " " + body +
		strings.Repeat(" ", pad) + mutedStyle.Render(age)
}

// renderFileLine renders one changed-file row; index 0 is the "Overall" entry.
func renderFileLine(idx int, fc git.FileChange, overall bool, selected, focused bool, width int) string {
	if overall {
		if selected {
			return selLine("▸ Overall (all files)", width, focused)
		}
		return padLine(titleStyle.Render("▸ Overall")+mutedStyle.Render(" (all files)"), width)
	}

	path := fc.Path
	if fc.Status == "R" && fc.OldPath != "" {
		path = fc.OldPath + " → " + fc.Path
	}
	path = truncate(path, width-3)

	if selected {
		return selLine(fc.Status+" "+path, width, focused)
	}
	return padLine(statusBadge(fc.Status)+" "+path, width)
}

// renderBranchLine renders a branch row in the branch overlay.
func renderBranchLine(b git.Branch, selected bool, width int) string {
	marker := "  "
	if b.IsCurrent {
		marker = "* "
	}
	hash := short(b.Hash)

	if selected {
		return selLine(marker+b.Name+"  "+hash, width, true)
	}

	mk := marker
	if b.IsCurrent {
		mk = okStyle.Render("* ")
	}
	var name string
	switch {
	case b.IsRemote:
		name = lipgloss.NewStyle().Foreground(colRefRemote).Render(b.Name)
	case b.IsCurrent:
		name = lipgloss.NewStyle().Foreground(colRefHead).Bold(true).Render(b.Name)
	default:
		name = lipgloss.NewStyle().Foreground(colRefBranch).Render(b.Name)
	}
	return padLine(mk+name+"  "+mutedStyle.Render(hash), width)
}

// renderWorktreeLine renders a worktree row in the worktree overlay.
func renderWorktreeLine(w git.Worktree, selected bool, width int) string {
	marker := "  "
	if w.IsCurrent {
		marker = "* "
	}
	ref := w.Branch
	switch {
	case w.Bare:
		ref = "(bare)"
	case ref == "":
		ref = "detached@" + short(w.Head)
	}
	path := truncate(w.Path, width-runeLen(ref)-6)

	if selected {
		return selLine(marker+path+"  "+ref, width, true)
	}
	mk := marker
	if w.IsCurrent {
		mk = okStyle.Render("* ")
	}
	return padLine(mk+path+"  "+mutedStyle.Render(ref), width)
}

func plainRefs(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	return "(" + strings.Join(refs, ", ") + ")"
}

func runeLen(s string) int { return len([]rune(s)) }

func short(h string) string {
	if len(h) >= 7 {
		return h[:7]
	}
	return h
}
