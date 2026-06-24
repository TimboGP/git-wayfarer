package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// Color palette (256-color indices, tuned for dark terminals).
var (
	colBorder    = lipgloss.Color("240")
	colBorderHi  = lipgloss.Color("39")
	colTitle     = lipgloss.Color("39")
	colMuted     = lipgloss.Color("245")
	colHash      = lipgloss.Color("214")
	colRefHead   = lipgloss.Color("48")
	colRefBranch = lipgloss.Color("76")
	colRefRemote = lipgloss.Color("167")
	colRefTag    = lipgloss.Color("220")
	colSelBg     = lipgloss.Color("238")
	colSelBgBlur = lipgloss.Color("236")
	colSelFg     = lipgloss.Color("231")
	colMarkA     = lipgloss.Color("204")
	colMarkB     = lipgloss.Color("75")
	colAdd       = lipgloss.Color("78")
	colDel       = lipgloss.Color("167")
	colMod       = lipgloss.Color("221")
	colRen       = lipgloss.Color("147")
	colErr       = lipgloss.Color("203")
	colOk        = lipgloss.Color("78")
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colSelFg)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(colTitle)
	mutedStyle  = lipgloss.NewStyle().Foreground(colMuted)
	hashStyle   = lipgloss.NewStyle().Foreground(colHash)
	errStyle    = lipgloss.NewStyle().Foreground(colErr).Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(colOk)
)

// paneBox renders body inside a titled, bordered box of the given outer size.
// The body is expected to already be height-3 lines tall (title takes one
// inner row, the border two). focused brightens the border.
func paneBox(title, body string, width, height int, focused bool) string {
	border := colBorder
	tstyle := mutedStyle
	if focused {
		border = colBorderHi
		tstyle = titleStyle
	}
	inner := lipgloss.JoinVertical(lipgloss.Left, tstyle.Render(title), body)
	// In lipgloss v2, Width/Height are the total box size *including* the
	// border, so we pass the full outer dimensions here; the inner content
	// area is width-2 × height-2 (panes size their content to match).
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Height(height).
		Render(inner)
}

// selLine applies selection styling to a full line within an inner width.
func selLine(s string, width int, focused bool) string {
	bg := colSelBgBlur
	if focused {
		bg = colSelBg
	}
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(colSelFg).
		Width(width).
		Render(s)
}

// padLine pads/truncates a plain line to width without selection styling.
func padLine(s string, width int) string {
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(s)
}

// statusBadge renders a one-letter change-status badge.
func statusBadge(status string) string {
	var c color.Color
	switch status {
	case "A":
		c = colAdd
	case "D":
		c = colDel
	case "R", "C":
		c = colRen
	case "?":
		c = colMuted
	default:
		c = colMod
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(status)
}

// refBadges renders decoration labels (HEAD, branches, tags) for a commit.
func refBadges(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		var c color.Color
		switch {
		case r == "HEAD":
			c = colRefHead
		case strings.HasPrefix(r, "tag: "):
			c = colRefTag
		case strings.Contains(r, "/"):
			c = colRefRemote
		default:
			c = colRefBranch
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(c).Render(r))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// overlayBox centers a bordered overlay of given content on the full screen.
func overlayBox(title, body string, screenW, screenH int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorderHi).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(title), "", body))
	return lipgloss.Place(screenW, screenH, lipgloss.Center, lipgloss.Center, box)
}

// humanizeTime renders a compact relative time (e.g. "3d ago").
func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

// truncate shortens s to max runes, adding an ellipsis when cut.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
