package ui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/osleff/wayfarer/internal/git"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.layout()
		if len(m.commits) > 0 {
			return m, m.reloadDiff() // re-wrap diff to the new width
		}
		return m, nil

	case commitsLoadedMsg:
		return m.onCommitsLoaded(msg)

	case diffLoadedMsg:
		if msg.token != m.diffToken {
			return m, nil // stale; selection moved on
		}
		m.loadingDiff = false
		if msg.err != nil {
			m.vp.SetContent(errStyle.Render("diff error: " + msg.err.Error()))
		} else if msg.content == "" {
			m.vp.SetContent(mutedStyle.Render("(no changes)"))
		} else {
			m.vp.SetContent(msg.content)
			m.vp.GotoTop()
		}
		return m, nil

	case actionDoneMsg:
		return m.onActionDone(msg)

	case tea.KeyPressMsg:
		return m.onKey(msg)
	}
	return m, nil
}

func (m Model) onCommitsLoaded(msg commitsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}
	m.commits = msg.commits
	m.graph = computeGraph(m.commits)
	m.rev = msg.rev
	m.markA, m.markB = "", ""
	m.searching = false
	m.searchQuery = ""
	m.applyFilter() // sets visible, count, cursor, and rebinds the renderer
	return m, m.onSelectionChanged()
}

func (m Model) onActionDone(msg actionDoneMsg) (tea.Model, tea.Cmd) {
	m.overlay = ovNone
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}
	m.setStatus(msg.ok)
	m.markA, m.markB = "", ""
	return m, loadCommitsCmd(m.repo, "")
}

// ---- key routing ----

func (m Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.searchKey(msg)
	}
	switch m.overlay {
	case ovInput:
		return m.inputKey(msg)
	case ovConfirm:
		return m.confirmKey(msg)
	case ovBranches:
		return m.branchKey(msg)
	case ovWorktrees:
		return m.worktreeKey(msg)
	case ovDetail:
		return m.detailKey(msg)
	case ovHelp:
		if key.Matches(msg, m.keys.Esc, m.keys.Help, m.keys.Quit) {
			m.overlay = ovNone
		}
		return m, nil
	}
	return m.mainKey(msg)
}

func (m Model) mainKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.overlay = ovHelp
		return m, nil
	case key.Matches(msg, m.keys.Branches):
		return m.openBranches()
	case key.Matches(msg, m.keys.Worktrees):
		return m.openWorktrees()
	case key.Matches(msg, m.keys.Tab):
		m.cycleFocus()
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.setStatus("refreshed")
		return m, loadCommitsCmd(m.repo, m.rev)
	case key.Matches(msg, m.keys.Mark):
		return m.toggleMark()
	case key.Matches(msg, m.keys.Graph):
		m.showGraph = !m.showGraph
		m.bindCommitRender()
		if m.showGraph {
			m.setStatus("graph on")
		} else {
			m.setStatus("graph off")
		}
		return m, nil
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.setStatus("")
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		return m.openDetail()
	case key.Matches(msg, m.keys.Esc):
		if m.markA != "" || m.markB != "" {
			m.markA, m.markB = "", ""
			m.bindCommitRender()
			return m, m.onSelectionChanged()
		}
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.applyFilter()
			return m, m.onSelectionChanged()
		}
		return m, nil
	}

	switch m.focus {
	case focusDiff:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	case focusCommits:
		return m.commitNav(msg)
	case focusFiles:
		return m.fileNav(msg)
	}
	return m, nil
}

func (m Model) commitNav(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	moved := navigate(&m.commitList, msg, m.keys)
	if moved && !m.compareMode() {
		return m, m.onSelectionChanged()
	}
	return m, nil
}

func (m Model) fileNav(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if navigate(&m.fileList, msg, m.keys) {
		return m, m.reloadDiff()
	}
	return m, nil
}

// navigate applies a navigation keypress to a scrollList, returning whether the
// cursor moved.
func navigate(l *scrollList, msg tea.KeyPressMsg, k keyMap) bool {
	switch {
	case key.Matches(msg, k.Up):
		l.moveUp(1)
	case key.Matches(msg, k.Down):
		l.moveDown(1)
	case key.Matches(msg, k.PageUp):
		l.moveUp(l.height)
	case key.Matches(msg, k.PageDown):
		l.moveDown(l.height)
	case key.Matches(msg, k.Top):
		l.toTop()
	case key.Matches(msg, k.Bottom):
		l.toBottom()
	default:
		return false
	}
	return true
}

func (m Model) toggleMark() (tea.Model, tea.Cmd) {
	c := m.currentCommit()
	if c == nil {
		return m, nil
	}
	switch {
	case m.markA == "":
		m.markA = c.Hash
		m.setStatus("marked A — move and press v to set B")
	case m.markB == "" && c.Hash != m.markA:
		m.markB = c.Hash
		m.setStatus("comparing A..B")
	default:
		m.markA = c.Hash
		m.markB = ""
		m.setStatus("marked A — move and press v to set B")
	}
	m.bindCommitRender()
	return m, m.onSelectionChanged()
}

// ---- branch overlay ----

func (m Model) openBranches() (tea.Model, tea.Cmd) {
	bs, err := m.repo.Branches()
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.branches = bs
	m.branchList = scrollList{focused: true}
	m.branchList.setCount(len(bs))
	for i, b := range bs {
		if b.IsCurrent {
			m.branchList.cursor = i
			break
		}
	}
	m.bindBranchRender()
	m.branchList.setSize(overlayListW(m.width), overlayListH(m.height, len(bs)))
	m.overlay = ovBranches
	return m, nil
}

func (m Model) branchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Esc, m.keys.Quit, m.keys.Branches):
		m.overlay = ovNone
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if b := m.selectedBranch(); b != nil {
			m.overlay = ovNone
			m.setStatus("inspecting " + b.Name)
			return m, loadCommitsCmd(m.repo, b.Ref)
		}
		return m, nil
	case key.Matches(msg, m.keys.Checkout):
		return m.checkoutSelected()
	}
	navigate(&m.branchList, msg, m.keys)
	return m, nil
}

func (m Model) selectedBranch() *git.Branch {
	i := m.branchList.cursor
	if i >= 0 && i < len(m.branches) {
		return &m.branches[i]
	}
	return nil
}

func (m Model) checkoutSelected() (tea.Model, tea.Cmd) {
	b := m.selectedBranch()
	if b == nil {
		return m, nil
	}
	dirty, err := m.repo.IsDirty()
	if err != nil {
		m.setError(err)
		m.overlay = ovNone
		return m, nil
	}
	if dirty {
		m.confirmMsg = fmt.Sprintf("Working tree has uncommitted changes.\nCheck out %q anyway? (changes are kept, not discarded)", b.Name)
		m.confirmCmd = checkoutCmd(m.repo, b.Name)
		m.overlay = ovConfirm
		return m, nil
	}
	m.overlay = ovNone
	return m, checkoutCmd(m.repo, b.Name)
}

// ---- worktree overlay ----

func (m Model) openWorktrees() (tea.Model, tea.Cmd) {
	wts, err := m.repo.Worktrees()
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.worktrees = wts
	m.worktreeList = scrollList{focused: true}
	m.worktreeList.setCount(len(wts))
	for i, w := range wts {
		if w.IsCurrent {
			m.worktreeList.cursor = i
			break
		}
	}
	m.bindWorktreeRender()
	m.worktreeList.setSize(overlayListW(m.width), overlayListH(m.height, len(wts)))
	m.overlay = ovWorktrees
	return m, nil
}

func (m Model) worktreeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Esc, m.keys.Quit, m.keys.Worktrees):
		m.overlay = ovNone
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		return m.openSelectedWorktree()
	case key.Matches(msg, m.keys.Add):
		m.input = textInput{prompt: "New worktree path: "}
		m.inputCmd = func(v string) tea.Cmd { return addWorktreeCmd(m.repo, v) }
		m.overlay = ovInput
		return m, nil
	case key.Matches(msg, m.keys.Delete):
		if w := m.selectedWorktree(); w != nil && !w.IsCurrent {
			m.confirmMsg = fmt.Sprintf("Remove worktree?\n%s", w.Path)
			m.confirmCmd = removeWorktreeCmd(m.repo, w.Path)
			m.overlay = ovConfirm
		}
		return m, nil
	}
	navigate(&m.worktreeList, msg, m.keys)
	return m, nil
}

func (m Model) selectedWorktree() *git.Worktree {
	i := m.worktreeList.cursor
	if i >= 0 && i < len(m.worktrees) {
		return &m.worktrees[i]
	}
	return nil
}

// openSelectedWorktree re-points the whole app at another worktree's path. A
// TUI cannot change the parent shell's cwd, so this is the in-process
// equivalent of switching worktrees.
func (m Model) openSelectedWorktree() (tea.Model, tea.Cmd) {
	w := m.selectedWorktree()
	if w == nil {
		return m, nil
	}
	repo, err := git.Open(w.Path, m.useDelta)
	if err != nil {
		m.setError(err)
		m.overlay = ovNone
		return m, nil
	}
	m.repo = repo
	m.overlay = ovNone
	m.markA, m.markB = "", ""
	m.setStatus("opened " + w.Path)
	return m, loadCommitsCmd(m.repo, "")
}

// ---- confirm + input overlays ----

func (m Model) confirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		cmd := m.confirmCmd
		m.confirmCmd = nil
		m.overlay = ovNone
		return m, cmd
	default:
		m.confirmCmd = nil
		m.overlay = ovNone
		return m, nil
	}
}

func (m Model) inputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Esc):
		m.overlay = ovNone
		m.inputCmd = nil
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		val := m.input.value
		cmd := m.inputCmd
		m.overlay = ovNone
		m.inputCmd = nil
		if val == "" || cmd == nil {
			return m, nil
		}
		return m, cmd(val)
	case msg.String() == "backspace":
		m.input.backspace()
		return m, nil
	default:
		if t := msg.Key().Text; t != "" {
			m.input.insert(t)
		}
		return m, nil
	}
}

// ---- search ----

func (m Model) searchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		m.searching = false
		if m.searchQuery == "" {
			return m, nil
		}
		if len(m.visible) == 0 {
			m.setStatus("no matches for /" + m.searchQuery)
		} else {
			m.setStatus(fmt.Sprintf("%d match(es) — esc to clear", len(m.visible)))
		}
		return m, nil
	case key.Matches(msg, m.keys.Esc):
		m.searching = false
		had := m.searchQuery != ""
		m.searchQuery = ""
		m.applyFilter()
		if had {
			return m, m.onSelectionChanged()
		}
		return m, nil
	case msg.String() == "backspace":
		if m.searchQuery == "" {
			return m, nil
		}
		r := []rune(m.searchQuery)
		m.searchQuery = string(r[:len(r)-1])
		m.applyFilter()
		return m, m.onSelectionChanged()
	default:
		if t := msg.Key().Text; t != "" {
			m.searchQuery += t
			m.applyFilter()
			return m, m.onSelectionChanged()
		}
		return m, nil
	}
}

// ---- commit detail overlay ----

func (m Model) openDetail() (tea.Model, tea.Cmd) {
	c := m.currentCommit()
	if c == nil {
		return m, nil
	}
	w, h := detailSize(m.width, m.height)
	m.detailVp.SetWidth(w)
	m.detailVp.SetHeight(h)
	m.detailVp.SetContent(m.buildDetail(c))
	m.detailVp.GotoTop()
	m.overlay = ovDetail
	return m, nil
}

func (m Model) detailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Esc, m.keys.Quit, m.keys.Enter) {
		m.overlay = ovNone
		return m, nil
	}
	var cmd tea.Cmd
	m.detailVp, cmd = m.detailVp.Update(msg)
	return m, cmd
}

func overlayListW(screenW int) int {
	w := screenW - 10
	if w > 80 {
		w = 80
	}
	if w < 10 {
		w = 10
	}
	return w
}

func overlayListH(screenH, count int) int {
	h := count
	if maxH := screenH - 10; h > maxH {
		h = maxH
	}
	if h < 1 {
		h = 1
	}
	return h
}
