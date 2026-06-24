package ui

import "strings"

// scrollList is a lightweight single-column scrollable list. It owns only the
// cursor/scroll state; each owner supplies count and a per-item render callback
// so it can style selection however it likes.
type scrollList struct {
	count   int
	cursor  int
	offset  int
	width   int
	height  int
	focused bool
	render  func(i int, selected, focused bool, width int) string
}

func (s *scrollList) setCount(n int) {
	s.count = n
	if s.cursor >= n {
		s.cursor = n - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
	s.clamp()
}

func (s *scrollList) setSize(w, h int) {
	s.width = w
	s.height = h
	s.clamp()
}

func (s *scrollList) moveUp(n int) {
	s.cursor -= n
	if s.cursor < 0 {
		s.cursor = 0
	}
	s.clamp()
}

func (s *scrollList) moveDown(n int) {
	s.cursor += n
	if s.cursor >= s.count {
		s.cursor = s.count - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
	s.clamp()
}

func (s *scrollList) toTop()    { s.cursor = 0; s.clamp() }
func (s *scrollList) toBottom() { s.cursor = s.count - 1; s.clamp() }

// clamp keeps the cursor within the visible window, scrolling as needed.
func (s *scrollList) clamp() {
	if s.height <= 0 {
		s.offset = 0
		return
	}
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+s.height {
		s.offset = s.cursor - s.height + 1
	}
	maxOffset := s.count - s.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if s.offset > maxOffset {
		s.offset = maxOffset
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

// view renders exactly height rows, padding with blank lines.
func (s *scrollList) view() string {
	if s.height <= 0 {
		return ""
	}
	lines := make([]string, 0, s.height)
	for row := 0; row < s.height; row++ {
		i := s.offset + row
		if i >= s.count || s.render == nil {
			lines = append(lines, padLine("", s.width))
			continue
		}
		lines = append(lines, s.render(i, i == s.cursor, s.focused, s.width))
	}
	return strings.Join(lines, "\n")
}

// textInput is a minimal single-line input (runes, backspace, no cursor render
// beyond a trailing block).
type textInput struct {
	value  string
	prompt string
}

func (t *textInput) insert(r string) { t.value += r }
func (t *textInput) backspace() {
	r := []rune(t.value)
	if len(r) > 0 {
		t.value = string(r[:len(r)-1])
	}
}

func (t *textInput) view() string {
	return mutedStyle.Render(t.prompt) + t.value + "█"
}
