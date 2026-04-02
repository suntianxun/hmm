package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type filterModel struct {
	colIndex   int
	colName    string
	input      textinput.Model
	allVals    []string        // all unique values for this column, sorted
	visible    []string        // values matching the search query
	selected   map[string]bool // true = value is included in filter
	cursor     int             // cursor position within visible list
	scrollOff  int             // scroll offset for the visible list
	listHeight int             // how many items fit on screen
}

func newFilterModel(colIndex int, colName string, rows [][]string, existing map[string]bool) filterModel {
	// Collect unique values
	seen := make(map[string]struct{})
	for _, row := range rows {
		if colIndex < len(row) {
			seen[row[colIndex]] = struct{}{}
		}
	}

	vals := make([]string, 0, len(seen))
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Strings(vals)

	// Initialize selection: if existing filter, copy it; otherwise all selected
	sel := make(map[string]bool, len(vals))
	if existing != nil {
		for _, v := range vals {
			sel[v] = existing[v]
		}
	} else {
		for _, v := range vals {
			sel[v] = true
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	fm := filterModel{
		colIndex:   colIndex,
		colName:    colName,
		input:      ti,
		allVals:    vals,
		selected:   sel,
		listHeight: 15,
	}
	fm.updateVisible()
	return fm
}

func (f *filterModel) updateVisible() {
	query := strings.ToLower(f.input.Value())
	if query == "" {
		f.visible = f.allVals
	} else {
		f.visible = nil
		for _, v := range f.allVals {
			if fuzzyMatch(strings.ToLower(v), query) {
				f.visible = append(f.visible, v)
			}
		}
		// When searching, auto-select only the matching values
		for _, v := range f.allVals {
			f.selected[v] = false
		}
		for _, v := range f.visible {
			f.selected[v] = true
		}
	}
	if f.cursor >= len(f.visible) {
		f.cursor = max(0, len(f.visible)-1)
	}
	f.clampScroll()
}

func (f *filterModel) clampScroll() {
	if f.cursor < f.scrollOff {
		f.scrollOff = f.cursor
	}
	if f.cursor >= f.scrollOff+f.listHeight {
		f.scrollOff = f.cursor - f.listHeight + 1
	}
	if f.scrollOff < 0 {
		f.scrollOff = 0
	}
}

func (f *filterModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if f.cursor > 0 {
				f.cursor--
				f.clampScroll()
			}
			return nil
		case "down":
			if f.cursor < len(f.visible)-1 {
				f.cursor++
				f.clampScroll()
			}
			return nil
		case " ":
			if f.cursor < len(f.visible) {
				v := f.visible[f.cursor]
				f.selected[v] = !f.selected[v]
			}
			return nil
		case "ctrl+a":
			for _, v := range f.visible {
				f.selected[v] = true
			}
			return nil
		case "ctrl+n":
			for _, v := range f.visible {
				f.selected[v] = false
			}
			return nil
		}
	}

	prevVal := f.input.Value()
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	if f.input.Value() != prevVal {
		f.cursor = 0
		f.scrollOff = 0
		f.updateVisible()
	}
	return cmd
}

func (f *filterModel) view(maxWidth int) string {
	var b strings.Builder

	title := filterTitleStyle.Render("Filter: " + f.colName)
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(f.input.View())
	b.WriteString("\n\n")

	if len(f.visible) == 0 {
		b.WriteString(filterHintStyle.Render("  No matching values"))
		b.WriteString("\n")
	} else {
		end := f.scrollOff + f.listHeight
		if end > len(f.visible) {
			end = len(f.visible)
		}
		for i := f.scrollOff; i < end; i++ {
			v := f.visible[i]
			check := "[ ]"
			if f.selected[v] {
				check = "[x]"
			}

			display := v
			if display == "" {
				display = "(empty)"
			}
			maxValWidth := maxWidth - 10
			if maxValWidth < 10 {
				maxValWidth = 10
			}
			if len(display) > maxValWidth {
				display = display[:maxValWidth-1] + "…"
			}

			line := check + " " + display
			if i == f.cursor {
				if f.selected[v] {
					b.WriteString(filterItemCursorSelectedStyle.Render(line))
				} else {
					b.WriteString(filterItemCursorStyle.Render(line))
				}
			} else {
				if f.selected[v] {
					b.WriteString(filterItemSelectedStyle.Render(line))
				} else {
					b.WriteString(filterItemStyle.Render(line))
				}
			}
			b.WriteString("\n")
		}

		if len(f.visible) > f.listHeight {
			b.WriteString(filterHintStyle.Render(fmt.Sprintf(
				"   ↑↓ scroll | showing %d-%d of %d",
				f.scrollOff+1, end, len(f.visible))))
			b.WriteString("\n")
		}
	}

	selCount := 0
	for _, v := range f.allVals {
		if f.selected[v] {
			selCount++
		}
	}

	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(
		"Space: toggle | Ctrl+A: all | Ctrl+N: none"))
	b.WriteString("\n")
	b.WriteString(filterHintStyle.Render(fmt.Sprintf(
		"Enter: apply | Esc: cancel | %d/%d selected",
		selCount, len(f.allVals))))

	return filterBorderStyle.MaxWidth(maxWidth).Render(b.String())
}

// fuzzyMatch checks if all characters of pattern appear in s in order.
func fuzzyMatch(s, pattern string) bool {
	pi := 0
	for i := 0; i < len(s) && pi < len(pattern); i++ {
		if s[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}
