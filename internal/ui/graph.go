package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/osleff/wayfarer/internal/git"
)

// laneColors cycles per graph lane so parallel branches are distinguishable.
var laneColors = []color.Color{
	lipgloss.Color("39"),  // blue
	lipgloss.Color("213"), // magenta
	lipgloss.Color("82"),  // green
	lipgloss.Color("214"), // orange
	lipgloss.Color("51"),  // cyan
	lipgloss.Color("220"), // yellow
	lipgloss.Color("141"), // purple
}

type graphCell struct {
	r    rune
	lane int // for coloring; -1 = blank
}

// computeGraph renders an ASCII commit-graph prefix for each commit, in one row
// per commit. It maintains a set of lanes (each waiting for a particular commit
// hash) and walks the list newest-first, drawing a node (●, or a merge node ◆)
// in the commit's lane and verticals (│) for every other open lane. Lanes are
// advanced to the commit's first parent; additional parents (merges) open new
// lanes, and lanes that converge on the commit are closed.
//
// The returned strings are colored and padded to a uniform width. Callers
// should only use it for the full, unfiltered history (a filtered subset would
// show disconnected lanes).
func computeGraph(commits []git.Commit) []string {
	var lanes []string // hash each lane is waiting to reach ("" = free)
	rows := make([][]graphCell, len(commits))
	maxLanes := 0

	for idx, c := range commits {
		h := c.Hash

		myLane := indexOf(lanes, h)
		if myLane == -1 {
			myLane = firstFree(lanes)
			if myLane == len(lanes) {
				lanes = append(lanes, h)
			} else {
				lanes[myLane] = h
			}
		}

		node := '●'
		if len(c.Parents) > 1 {
			node = '◆'
		}

		row := make([]graphCell, len(lanes))
		for i, l := range lanes {
			switch {
			case i == myLane:
				row[i] = graphCell{r: node, lane: i}
			case l == "":
				row[i] = graphCell{r: ' ', lane: -1}
			default:
				row[i] = graphCell{r: '│', lane: i}
			}
		}
		rows[idx] = row
		if len(lanes) > maxLanes {
			maxLanes = len(lanes)
		}

		// Close other lanes that were waiting for this same commit (converging).
		for i, l := range lanes {
			if i != myLane && l == h {
				lanes[i] = ""
			}
		}
		// Advance my lane to the first parent.
		if len(c.Parents) >= 1 {
			lanes[myLane] = c.Parents[0]
		} else {
			lanes[myLane] = ""
		}
		// Open lanes for additional (merge) parents.
		for i := 1; i < len(c.Parents); i++ {
			p := c.Parents[i]
			if indexOf(lanes, p) != -1 {
				continue // already tracked
			}
			f := firstFree(lanes)
			if f == len(lanes) {
				lanes = append(lanes, p)
			} else {
				lanes[f] = p
			}
		}
		lanes = trimTrailingFree(lanes)
	}

	out := make([]string, len(commits))
	for idx, row := range rows {
		out[idx] = colorizeGraphRow(row, maxLanes)
	}
	return out
}

// colorizeGraphRow renders a row to a colored string of fixed width
// (2 columns per lane: glyph + separator).
func colorizeGraphRow(row []graphCell, maxLanes int) string {
	var b strings.Builder
	for i := 0; i < maxLanes; i++ {
		if i < len(row) && row[i].r != ' ' {
			st := lipgloss.NewStyle().Foreground(laneColors[row[i].lane%len(laneColors)])
			if row[i].r == '●' || row[i].r == '◆' {
				st = st.Bold(true)
			}
			b.WriteString(st.Render(string(row[i].r)))
		} else {
			b.WriteByte(' ')
		}
		b.WriteByte(' ')
	}
	return b.String()
}

func indexOf(lanes []string, h string) int {
	for i, l := range lanes {
		if l == h {
			return i
		}
	}
	return -1
}

func firstFree(lanes []string) int {
	for i, l := range lanes {
		if l == "" {
			return i
		}
	}
	return len(lanes)
}

func trimTrailingFree(lanes []string) []string {
	n := len(lanes)
	for n > 0 && lanes[n-1] == "" {
		n--
	}
	return lanes[:n]
}
