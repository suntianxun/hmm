package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type sortModel struct {
	columns    []string   // all column names
	visible    []int      // indices into columns matching the search
	selected   []int      // column indices chosen so far, in order
	cursor     int        // cursor within visible list
	scrollOff  int        // scroll offset
	listHeight int        // max visible items
	input      textinput.Model
}

func newSortModel(columns []string) sortModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search columns..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	visible := make([]int, len(columns))
	for i := range columns {
		visible[i] = i
	}

	return sortModel{
		columns:    columns,
		visible:    visible,
		listHeight: 15,
		input:      ti,
	}
}

func (s *sortModel) updateVisible() {
	query := strings.ToLower(s.input.Value())
	if query == "" {
		s.visible = make([]int, len(s.columns))
		for i := range s.columns {
			s.visible[i] = i
		}
	} else {
		s.visible = nil
		for i, col := range s.columns {
			if fuzzyMatch(strings.ToLower(col), query) {
				s.visible = append(s.visible, i)
			}
		}
	}
	if s.cursor >= len(s.visible) {
		s.cursor = max(0, len(s.visible)-1)
	}
	s.clampScroll()
}

func (s *sortModel) clampScroll() {
	if s.cursor < s.scrollOff {
		s.scrollOff = s.cursor
	}
	if s.cursor >= s.scrollOff+s.listHeight {
		s.scrollOff = s.cursor - s.listHeight + 1
	}
	if s.scrollOff < 0 {
		s.scrollOff = 0
	}
}

func (s *sortModel) isSelected(colIdx int) int {
	for i, ci := range s.selected {
		if ci == colIdx {
			return i + 1 // 1-based order
		}
	}
	return 0
}

func (s *sortModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if s.cursor > 0 {
				s.cursor--
				s.clampScroll()
			}
			return nil
		case "down":
			if s.cursor < len(s.visible)-1 {
				s.cursor++
				s.clampScroll()
			}
			return nil
		case " ", "enter":
			if s.cursor < len(s.visible) {
				colIdx := s.visible[s.cursor]
				if order := s.isSelected(colIdx); order > 0 {
					// Deselect: remove from selected
					for i, ci := range s.selected {
						if ci == colIdx {
							s.selected = append(s.selected[:i], s.selected[i+1:]...)
							break
						}
					}
				} else {
					s.selected = append(s.selected, colIdx)
				}
			}
			return nil
		}
	}

	prevVal := s.input.Value()
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	if s.input.Value() != prevVal {
		s.cursor = 0
		s.scrollOff = 0
		s.updateVisible()
	}
	return cmd
}

func (s *sortModel) view(maxWidth int) string {
	var b strings.Builder

	title := filterTitleStyle.Render("Sort by columns")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(s.input.View())
	b.WriteString("\n\n")

	if len(s.visible) == 0 {
		b.WriteString(filterHintStyle.Render("  No matching columns"))
		b.WriteString("\n")
	} else {
		end := s.scrollOff + s.listHeight
		if end > len(s.visible) {
			end = len(s.visible)
		}
		for i := s.scrollOff; i < end; i++ {
			colIdx := s.visible[i]
			name := s.columns[colIdx]

			order := s.isSelected(colIdx)
			var prefix string
			if order > 0 {
				prefix = fmt.Sprintf("[%d]", order)
			} else {
				prefix = "[ ]"
			}

			maxValWidth := maxWidth - 10
			if maxValWidth < 10 {
				maxValWidth = 10
			}
			if len(name) > maxValWidth {
				name = name[:maxValWidth-1] + "…"
			}

			line := prefix + " " + name
			if i == s.cursor {
				if order > 0 {
					b.WriteString(filterItemCursorSelectedStyle.Render(line))
				} else {
					b.WriteString(filterItemCursorStyle.Render(line))
				}
			} else {
				if order > 0 {
					b.WriteString(filterItemSelectedStyle.Render(line))
				} else {
					b.WriteString(filterItemStyle.Render(line))
				}
			}
			b.WriteString("\n")
		}

		if len(s.visible) > s.listHeight {
			b.WriteString(filterHintStyle.Render(fmt.Sprintf(
				"   ↑↓ scroll | showing %d-%d of %d",
				s.scrollOff+1, end, len(s.visible))))
			b.WriteString("\n")
		}
	}

	// Show current sort order
	if len(s.selected) > 0 {
		b.WriteString("\n")
		names := make([]string, len(s.selected))
		for i, ci := range s.selected {
			names[i] = s.columns[ci]
		}
		b.WriteString(filterHintStyle.Render("Order: " + strings.Join(names, " → ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(
		"Space/Enter: toggle | a: sort asc | d: sort desc"))
	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(
		"S: clear sort | Esc: cancel"))

	return filterBorderStyle.MaxWidth(maxWidth).Render(b.String())
}
