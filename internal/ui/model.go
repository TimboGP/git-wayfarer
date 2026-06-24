package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/osleff/wayfarer/internal/git"
)

type focusArea int

const (
	focusCommits focusArea = iota
	focusFiles
	focusDiff
)

type overlayKind int

const (
	ovNone overlayKind = iota
	ovBranches
	ovWorktrees
	ovHelp
	ovConfirm
	ovInput
	ovDetail
)

type statusKind int

const (
	stWorking statusKind = iota // unstaged changes (working tree vs index)
	stStaged                    // staged changes (index vs HEAD)
)

// statusRow is a synthetic top-of-list entry for uncommitted changes.
type statusRow struct {
	kind  statusKind
	count int
}

// Model is the root Bubble Tea model.
type Model struct {
	repo     *git.Repo
	useDelta bool

	width, height int
	ready         bool

	// layout (computed in layout())
	leftW, rightW int
	contentH      int
	fileH, diffH  int
	diffWidth     int

	// history view
	rev        string       // current view revision ("" = HEAD)
	commits    []git.Commit // all loaded commits
	visible    []git.Commit // commits passing the active search filter
	graph      []string     // colored graph prefix per commit (indexed like commits)
	showGraph  bool         // render the commit-graph topology column
	statusRows []statusRow  // synthetic working/staged entries at the top of the list
	commitList scrollList

	// search/filter over commits
	searching   bool
	searchQuery string

	// changed files for current selection (Overall is index 0)
	files    []git.FileChange
	fileList scrollList

	// diff pane
	vp          viewport.Model
	diffToken   int
	loadingDiff bool

	// commit detail overlay
	detailVp viewport.Model

	focus focusArea

	// compare marks (commit hashes; both set => compare mode)
	markA, markB string

	// overlays
	overlay      overlayKind
	branches     []git.Branch
	branchList   scrollList
	worktrees    []git.Worktree
	worktreeList scrollList

	confirmMsg string
	confirmCmd tea.Cmd

	input    textInput
	inputCmd func(string) tea.Cmd

	help help.Model
	keys keyMap

	status    string
	statusErr bool
}

// New builds the initial model for the repository at repo. rev is an optional
// starting revision ("" starts at HEAD).
func New(repo *git.Repo, useDelta bool, rev string) Model {
	return Model{
		repo:       repo,
		useDelta:   useDelta,
		rev:        rev,
		vp:         viewport.New(),
		detailVp:   viewport.New(),
		help:       help.New(),
		keys:       defaultKeys(),
		commitList: scrollList{focused: true},
		focus:      focusCommits,
		showGraph:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return loadCommitsCmd(m.repo, m.rev)
}

// ---- helpers ----

func (m Model) compareMode() bool { return m.markA != "" && m.markB != "" }

// statusCount is the number of synthetic working/staged rows currently shown.
// They are hidden while a search filter is active.
func (m Model) statusCount() int {
	if m.searchQuery != "" {
		return 0
	}
	return len(m.statusRows)
}

// selectedStatus returns the synthetic status row under the cursor, if any.
func (m Model) selectedStatus() (statusRow, bool) {
	i := m.commitList.cursor
	if i >= 0 && i < m.statusCount() {
		return m.statusRows[i], true
	}
	return statusRow{}, false
}

func (m Model) currentCommit() *git.Commit {
	i := m.commitList.cursor - m.statusCount()
	if i >= 0 && i < len(m.visible) {
		return &m.visible[i]
	}
	return nil
}

// selectedFile returns the changed-file path under the file cursor, or "" for
// the Overall entry (index 0).
func (m Model) selectedFile() string {
	if m.fileList.cursor > 0 && m.fileList.cursor-1 < len(m.files) {
		return m.files[m.fileList.cursor-1].Path
	}
	return ""
}

func (m Model) selectedFileChange() *git.FileChange {
	if m.fileList.cursor > 0 && m.fileList.cursor-1 < len(m.files) {
		return &m.files[m.fileList.cursor-1]
	}
	return nil
}

// refreshStatus recomputes the synthetic working/staged rows from the repo.
func (m *Model) refreshStatus() {
	m.statusRows = nil
	if working, err := m.repo.WorkingFiles(); err == nil && len(working) > 0 {
		m.statusRows = append(m.statusRows, statusRow{kind: stWorking, count: len(working)})
	}
	if staged, err := m.repo.StagedFiles(); err == nil && len(staged) > 0 {
		m.statusRows = append(m.statusRows, statusRow{kind: stStaged, count: len(staged)})
	}
}

// applyFilter recomputes the visible commit slice from the active search query
// (matching subject, author, or hash, case-insensitively) and resets the
// cursor to the first match.
func (m *Model) applyFilter() {
	if m.searchQuery == "" {
		m.visible = m.commits
	} else {
		q := strings.ToLower(m.searchQuery)
		out := make([]git.Commit, 0, len(m.commits))
		for _, c := range m.commits {
			if strings.Contains(strings.ToLower(c.Subject), q) ||
				strings.Contains(strings.ToLower(c.Author), q) ||
				strings.Contains(strings.ToLower(c.Hash), q) {
				out = append(out, c)
			}
		}
		m.visible = out
	}
	m.commitList.setCount(m.statusCount() + len(m.visible))
	m.commitList.cursor = 0
	m.commitList.offset = 0
	m.bindCommitRender()
}

// diffParams computes the diff endpoints for the current commit selection.
func (m Model) diffParams() (base, target, file string, ok bool) {
	file = m.selectedFile()
	if m.compareMode() {
		return m.markA, m.markB, file, true
	}
	c := m.currentCommit()
	if c == nil {
		return "", "", "", false
	}
	if len(c.Parents) > 0 {
		base = c.Parents[0]
	}
	return base, c.Hash, file, true
}

func (m *Model) setError(err error) { m.status = err.Error(); m.statusErr = true }
func (m *Model) setStatus(s string) { m.status = s; m.statusErr = false }

// bind* (re)attach render closures that capture current data/marks.
func (m *Model) bindCommitRender() {
	commits, markA, markB := m.visible, m.markA, m.markB
	graphs := m.graph
	status := m.statusRows
	statusCount := m.statusCount()
	// The graph is computed over the full history; only show it when unfiltered
	// (then visible == commits and indices align).
	showGraph := m.showGraph && m.searchQuery == ""
	graphPad := ""
	if showGraph && len(graphs) > 0 {
		graphPad = strings.Repeat(" ", lipgloss.Width(graphs[0]))
	}
	m.commitList.render = func(i int, selected, focused bool, width int) string {
		if i < statusCount {
			return renderStatusLine(status[i], graphPad, selected, focused, width)
		}
		ci := i - statusCount
		g := ""
		if showGraph && ci < len(graphs) {
			g = graphs[ci]
		}
		return renderCommitLine(commits[ci], g, markA, markB, selected, focused, width)
	}
}

func (m *Model) bindFileRender() {
	files := m.files
	m.fileList.render = func(i int, selected, focused bool, width int) string {
		if i == 0 {
			return renderFileLine(0, git.FileChange{}, true, selected, focused, width)
		}
		return renderFileLine(i, files[i-1], false, selected, focused, width)
	}
}

func (m *Model) bindBranchRender() {
	branches := m.branches
	m.branchList.render = func(i int, selected, focused bool, width int) string {
		return renderBranchLine(branches[i], selected, width)
	}
}

func (m *Model) bindWorktreeRender() {
	wts := m.worktrees
	m.worktreeList.render = func(i int, selected, focused bool, width int) string {
		return renderWorktreeLine(wts[i], selected, width)
	}
}

// refreshFiles recomputes the changed-file list for the current selection.
func (m *Model) refreshFiles() {
	var (
		files []git.FileChange
		err   error
	)
	if st, ok := m.selectedStatus(); ok {
		if st.kind == stStaged {
			files, err = m.repo.StagedFiles()
		} else {
			files, err = m.repo.WorkingFiles()
		}
	} else if m.compareMode() {
		files, err = m.repo.ChangedFilesRange(m.markA, m.markB)
	} else if c := m.currentCommit(); c != nil {
		files, err = m.repo.ChangedFiles(c.Hash)
	}
	if err != nil {
		m.setError(err)
		files = nil
	}
	m.files = files
	m.fileList.setCount(len(files) + 1) // +1 for the Overall entry
	m.bindFileRender()
}

// onSelectionChanged refreshes the file list (resetting to Overall) and queues
// a diff reload.
func (m *Model) onSelectionChanged() tea.Cmd {
	m.refreshFiles()
	m.fileList.cursor = 0
	m.fileList.offset = 0
	m.layout()
	return m.reloadDiff()
}

// reloadDiff queues an async render of the current diff selection.
func (m *Model) reloadDiff() tea.Cmd {
	if st, ok := m.selectedStatus(); ok {
		m.diffToken++
		m.loadingDiff = true
		untracked := false
		if fc := m.selectedFileChange(); fc != nil && fc.Status == "?" {
			untracked = true
		}
		return loadStatusDiffCmd(m.repo, st.kind, m.selectedFile(), untracked, m.diffWidth, m.diffToken)
	}
	base, target, file, ok := m.diffParams()
	if !ok {
		m.vp.SetContent(mutedStyle.Render("no commit selected"))
		return nil
	}
	m.diffToken++
	m.loadingDiff = true
	return loadDiffCmd(m.repo, base, target, file, m.diffWidth, m.diffToken)
}

func (m *Model) cycleFocus() {
	m.focus = (m.focus + 1) % 3
	m.commitList.focused = m.focus == focusCommits
	m.fileList.focused = m.focus == focusFiles
}

// layout recomputes pane sizes from the current terminal dimensions.
func (m *Model) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	const headerH, helpH = 1, 1
	m.contentH = m.height - headerH - helpH
	if m.contentH < 5 {
		m.contentH = 5
	}

	m.leftW = m.width * 9 / 20
	if m.leftW < 28 {
		m.leftW = 28
	}
	if m.leftW > m.width-24 {
		m.leftW = m.width - 24
	}
	if m.leftW < 1 {
		m.leftW = m.width / 2
	}
	m.rightW = m.width - m.leftW

	// file box height grows with the file count but is capped.
	m.fileH = len(m.files) + 1 + 3 // entries + Overall + box overhead
	if max := m.contentH / 2; m.fileH > max {
		m.fileH = max
	}
	if m.fileH < 6 {
		m.fileH = 6
	}
	if m.fileH > m.contentH-5 {
		m.fileH = m.contentH - 5
	}
	m.diffH = m.contentH - m.fileH
	m.diffWidth = m.rightW - 2

	m.commitList.setSize(m.leftW-2, m.contentH-3)
	m.fileList.setSize(m.rightW-2, m.fileH-3)
	m.vp.SetWidth(m.rightW - 2)
	m.vp.SetHeight(m.diffH - 3)
	m.help.SetWidth(m.width)
}

func shortRev(rev string) string {
	rev = strings.TrimPrefix(rev, "refs/heads/")
	rev = strings.TrimPrefix(rev, "refs/remotes/")
	rev = strings.TrimPrefix(rev, "refs/tags/")
	return rev
}

// ---- view ----

func (m Model) View() tea.View {
	if !m.ready || m.width == 0 || m.height == 0 {
		return tea.NewView("loading…")
	}

	screen := lipgloss.JoinVertical(lipgloss.Left, m.headerView(), m.mainView(), m.footerView())

	switch m.overlay {
	case ovBranches:
		screen = overlayBox("Branches  ·  enter: inspect   c: checkout   esc: close",
			m.branchList.view(), m.width, m.height)
	case ovWorktrees:
		screen = overlayBox("Worktrees  ·  enter: open   a: add   d: remove   esc: close",
			m.worktreeList.view(), m.width, m.height)
	case ovHelp:
		h := help.New()
		h.ShowAll = true
		h.SetWidth(m.width - 8)
		screen = overlayBox("Keybindings  ·  esc: close", h.View(m.keys), m.width, m.height)
	case ovConfirm:
		body := m.confirmMsg + "\n\n" +
			okStyle.Render("[y] yes") + "    " + errStyle.Render("[n] no")
		screen = overlayBox("Confirm", body, m.width, m.height)
	case ovInput:
		body := m.input.view() + "\n\n" + mutedStyle.Render("[enter] ok    [esc] cancel")
		screen = overlayBox("Input", body, m.width, m.height)
	case ovDetail:
		screen = overlayBox("Commit  ·  ↑/↓ scroll · esc close", m.detailVp.View(), m.width, m.height)
	}

	v := tea.NewView(screen)
	v.AltScreen = true
	return v
}

func (m Model) headerView() string {
	seg := []string{
		headerStyle.Background(colBorderHi).Render(" wayfarer "),
		mutedStyle.Render("repo ") + filepath.Base(m.repo.Root()),
		mutedStyle.Render("HEAD ") + lipgloss.NewStyle().Foreground(colRefHead).Render(m.repo.CurrentRef()),
	}
	if m.rev != "" {
		seg = append(seg, lipgloss.NewStyle().Foreground(colRefRemote).Render("viewing "+shortRev(m.rev)))
	}
	if m.compareMode() {
		seg = append(seg, lipgloss.NewStyle().Foreground(colMarkA).Bold(true).
			Render(fmt.Sprintf("COMPARE %s..%s", short(m.markA), short(m.markB))))
	}
	if m.searchQuery != "" {
		seg = append(seg, lipgloss.NewStyle().Foreground(colTitle).
			Render(fmt.Sprintf("/%s (%d)", m.searchQuery, len(m.visible))))
	}
	if m.useDelta {
		seg = append(seg, mutedStyle.Render("delta"))
	}
	left := strings.Join(seg, "  ")

	right := ""
	if m.status != "" {
		if m.statusErr {
			right = errStyle.Render(m.status)
		} else {
			right = okStyle.Render(m.status)
		}
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) mainView() string {
	commitTitle := fmt.Sprintf("Commits (%d)", len(m.visible))
	if m.searchQuery != "" {
		commitTitle = fmt.Sprintf("Commits (%d/%d)", len(m.visible), len(m.commits))
	}
	commitBox := paneBox(commitTitle,
		m.commitList.view(), m.leftW, m.contentH, m.focus == focusCommits)

	fileTitle := "Changes"
	diffTitle := "Diff"
	if st, ok := m.selectedStatus(); ok {
		name := "Working tree"
		if st.kind == stStaged {
			name = "Staged"
		}
		fileTitle = name
		diffTitle = "Diff · " + name
		if f := m.selectedFile(); f != "" {
			diffTitle = "Diff · " + truncate(filepath.Base(f), 40)
		}
	} else if m.compareMode() {
		fileTitle = "Changes A..B"
		if base, target, _, ok := m.diffParams(); ok {
			diffTitle = fmt.Sprintf("Diff · %s..%s", short(base), short(target))
		}
	} else if _, target, file, ok := m.diffParams(); ok {
		if file != "" {
			diffTitle = "Diff · " + truncate(filepath.Base(file), 40)
		} else {
			diffTitle = "Diff · " + short(target)
		}
	}
	fileBox := paneBox(fmt.Sprintf("%s (%d)", fileTitle, len(m.files)),
		m.fileList.view(), m.rightW, m.fileH, m.focus == focusFiles)

	if m.loadingDiff {
		diffTitle += mutedStyle.Render(" (loading…)")
	}
	diffBox := paneBox(diffTitle, m.vp.View(), m.rightW, m.diffH, m.focus == focusDiff)

	right := lipgloss.JoinVertical(lipgloss.Left, fileBox, diffBox)
	return lipgloss.JoinHorizontal(lipgloss.Top, commitBox, right)
}

func (m Model) footerView() string {
	if m.searching {
		bar := lipgloss.NewStyle().Foreground(colTitle).Render("/") + m.searchQuery + "█"
		return bar + mutedStyle.Render("    enter: keep · esc: clear")
	}
	return m.help.View(m.keys)
}

// buildDetail formats the full metadata, message, and diffstat for a commit.
func (m Model) buildDetail(c *git.Commit) string {
	var b strings.Builder
	b.WriteString(hashStyle.Render("commit " + c.Hash))
	b.WriteByte('\n')
	if len(c.Parents) > 0 {
		parents := make([]string, len(c.Parents))
		for i, p := range c.Parents {
			parents[i] = short(p)
		}
		b.WriteString(mutedStyle.Render("Parent:  ") + strings.Join(parents, " ") + "\n")
	}
	if refs := plainRefs(c.Refs); refs != "" {
		b.WriteString(mutedStyle.Render("Refs:    ") + refs + "\n")
	}
	b.WriteString(mutedStyle.Render("Author:  ") + c.Author + " <" + c.Email + ">\n")
	b.WriteString(mutedStyle.Render("Date:    ") + c.When.Format("Mon Jan 2 15:04:05 2006 -0700") + "\n\n")
	b.WriteString(titleStyle.Render(c.Subject) + "\n")
	if c.Body != "" {
		b.WriteString("\n" + c.Body + "\n")
	}
	if stat, err := m.repo.CommitStat(c.Hash); err == nil && strings.TrimSpace(stat) != "" {
		b.WriteString("\n" + mutedStyle.Render(strings.TrimRight(stat, "\n")))
	}
	return b.String()
}

// detailSize returns the inner content size for the commit detail viewport.
func detailSize(screenW, screenH int) (w, h int) {
	w = screenW - 16
	if w > 90 {
		w = 90
	}
	if w < 20 {
		w = 20
	}
	h = screenH - 8
	if h > 30 {
		h = 30
	}
	if h < 5 {
		h = 5
	}
	return w, h
}
