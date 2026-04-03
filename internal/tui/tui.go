package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/suntianxun/hmm/internal/reader"
)

type model struct {
	data         *reader.TableData
	filteredRows [][]string

	// Cursor position
	cursorRow int
	cursorCol int

	// Viewport scroll
	scrollRow int
	scrollCol int

	// Column widths (computed)
	colWidths []int

	// Terminal dimensions
	width  int
	height int

	// Per-column filters: colIndex -> set of selected values
	filters map[int]map[string]bool

	// Filter overlay state
	filterActive bool
	filterModel  filterModel

	// Sort overlay state
	sortActive bool
	sortModel  sortModel

	// Active sort state
	sortColumns []int  // column indices to sort by, in order
	sortAsc     bool   // true = ascending, false = descending

	// Copy mode
	copyMode    bool
	copyMessage string // transient message like "Copied!"

	filename string
}

func New(data *reader.TableData, filename string) model {
	m := model{
		data:         data,
		filteredRows: data.Rows,
		filters:      make(map[int]map[string]bool),
		filename:     filename,
		width:        80,
		height:       24,
	}
	m.computeColWidths()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.computeColWidths()
		m.clampCursor()
		return m, nil

	case tea.KeyMsg:
		// Clear transient copy message on any keypress
		m.copyMessage = ""

		if m.filterActive {
			return m.updateFilter(msg)
		}
		if m.sortActive {
			return m.updateSort(msg)
		}
		if m.copyMode {
			return m.updateCopyMode(msg)
		}
		return m.updateTable(msg)
	}

	return m, nil
}

func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Apply filter
		allSelected := true
		for _, v := range m.filterModel.allVals {
			if !m.filterModel.selected[v] {
				allSelected = false
				break
			}
		}
		if allSelected {
			// All selected = remove filter for this column
			delete(m.filters, m.filterModel.colIndex)
		} else {
			// Copy the selection map so it's not shared with the filter model
			sel := make(map[string]bool, len(m.filterModel.selected))
			for k, v := range m.filterModel.selected {
				sel[k] = v
			}
			m.filters[m.filterModel.colIndex] = sel
		}
		m.filterActive = false
		if len(m.sortColumns) > 0 {
			m.applySorting()
		} else {
			m.applyFilters()
		}
		m.clampCursor()
		return m, nil

	case "esc":
		m.filterActive = false
		return m, nil
	}

	cmd := m.filterModel.update(msg)
	return m, cmd
}

func (m model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursorRow > 0 {
			m.cursorRow--
			m.scrollIntoView()
		}
	case "down", "j":
		if m.cursorRow < len(m.filteredRows)-1 {
			m.cursorRow++
			m.scrollIntoView()
		}
	case "left", "h":
		if m.cursorCol > 0 {
			m.cursorCol--
			m.scrollColIntoView()
		}
	case "right", "l":
		if m.cursorCol < len(m.data.Columns)-1 {
			m.cursorCol++
			m.scrollColIntoView()
		}
	case "pgup":
		m.cursorRow -= m.tableHeight()
		if m.cursorRow < 0 {
			m.cursorRow = 0
		}
		m.scrollIntoView()
	case "pgdown":
		m.cursorRow += m.tableHeight()
		if m.cursorRow >= len(m.filteredRows) {
			m.cursorRow = len(m.filteredRows) - 1
		}
		if m.cursorRow < 0 {
			m.cursorRow = 0
		}
		m.scrollIntoView()
	case "g", "home":
		m.cursorRow = 0
		m.scrollIntoView()
	case "G", "end":
		m.cursorRow = max(0, len(m.filteredRows)-1)
		m.scrollIntoView()
	case "f":
		// Open filter for current column
		m.filterActive = true
		existing := m.filters[m.cursorCol]
		m.filterModel = newFilterModel(
			m.cursorCol,
			m.data.Columns[m.cursorCol],
			m.data.Rows, // always filter from all rows to show all unique values
			existing,
		)
	case "F":
		// Clear filter on current column
		delete(m.filters, m.cursorCol)
		m.applyFilters()
		if len(m.sortColumns) > 0 {
			m.applySorting()
		}
		m.clampCursor()
	case "s":
		// Open sort overlay
		m.sortActive = true
		m.sortModel = newSortModel(m.data.Columns)
		// Pre-select currently active sort columns
		if len(m.sortColumns) > 0 {
			m.sortModel.selected = make([]int, len(m.sortColumns))
			copy(m.sortModel.selected, m.sortColumns)
		}
	case "S":
		// Clear sort
		m.sortColumns = nil
		m.applyFilters()
		m.clampCursor()
	case "y":
		m.copyMode = true
	}
	return m, nil
}

func (m model) updateCopyMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.copyMode = false
	switch msg.String() {
	case "y":
		// Copy current cell value
		val := ""
		if m.cursorRow < len(m.filteredRows) && m.cursorCol < len(m.filteredRows[m.cursorRow]) {
			val = m.filteredRows[m.cursorRow][m.cursorCol]
		}
		if err := clipboard.WriteAll(val); err != nil {
			m.copyMessage = "Copy failed"
		} else {
			m.copyMessage = "Copied cell value"
		}
	case "d":
		// Copy distinct values in current column
		seen := make(map[string]struct{})
		var distinct []string
		for _, row := range m.filteredRows {
			if m.cursorCol < len(row) {
				v := row[m.cursorCol]
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					distinct = append(distinct, v)
				}
			}
		}
		text := strings.Join(distinct, "\n")
		if err := clipboard.WriteAll(text); err != nil {
			m.copyMessage = "Copy failed"
		} else {
			m.copyMessage = fmt.Sprintf("Copied %d distinct values", len(distinct))
		}
	case "r":
		// Copy current row with column names
		if m.cursorRow < len(m.filteredRows) {
			row := m.filteredRows[m.cursorRow]
			var parts []string
			for i, col := range m.data.Columns {
				val := ""
				if i < len(row) {
					val = row[i]
				}
				parts = append(parts, fmt.Sprintf("%s: %s", col, val))
			}
			text := strings.Join(parts, "\n")
			if err := clipboard.WriteAll(text); err != nil {
				m.copyMessage = "Copy failed"
			} else {
				m.copyMessage = "Copied row"
			}
		}
	case "esc":
		// Just exit copy mode
	default:
		// Unknown key, just exit copy mode
	}
	return m, nil
}

func (m model) updateSort(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		// Apply ascending sort
		if len(m.sortModel.selected) > 0 {
			m.sortColumns = make([]int, len(m.sortModel.selected))
			copy(m.sortColumns, m.sortModel.selected)
			m.sortAsc = true
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "d":
		// Apply descending sort
		if len(m.sortModel.selected) > 0 {
			m.sortColumns = make([]int, len(m.sortModel.selected))
			copy(m.sortColumns, m.sortModel.selected)
			m.sortAsc = false
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "S":
		// Clear sort
		m.sortColumns = nil
		m.applyFilters() // reapply filters without sort
		m.clampCursor()
		m.sortActive = false
		return m, nil
	case "esc":
		m.sortActive = false
		return m, nil
	}

	cmd := m.sortModel.update(msg)
	return m, cmd
}

func (m *model) applySorting() {
	m.applyFilters() // start from filtered rows
	if len(m.sortColumns) == 0 {
		return
	}

	slices.SortStableFunc(m.filteredRows, func(a, b []string) int {
		for _, ci := range m.sortColumns {
			va, vb := "", ""
			if ci < len(a) {
				va = a[ci]
			}
			if ci < len(b) {
				vb = b[ci]
			}
			if c := cmp.Compare(va, vb); c != 0 {
				if m.sortAsc {
					return c
				}
				return -c
			}
		}
		return 0
	})
}

func (m model) View() string {
	if m.filterActive {
		return m.viewWithOverlay(m.filterModel.view(min(m.width-4, 50)))
	}
	if m.sortActive {
		return m.viewWithOverlay(m.sortModel.view(min(m.width-4, 50)))
	}
	return m.viewTable()
}

func (m model) viewTable() string {
	var b strings.Builder
	th := m.tableHeight()
	visibleCols := m.visibleColumns()

	// Header row
	m.renderHeaderRow(&b, visibleCols)
	b.WriteString("\n")

	// Separator
	m.renderSeparator(&b, visibleCols)
	b.WriteString("\n")

	// Data rows
	endRow := m.scrollRow + th
	if endRow > len(m.filteredRows) {
		endRow = len(m.filteredRows)
	}
	for ri := m.scrollRow; ri < endRow; ri++ {
		m.renderDataRow(&b, ri, visibleCols)
		b.WriteString("\n")
	}

	// Pad remaining lines
	rendered := endRow - m.scrollRow
	for i := rendered; i < th; i++ {
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.statusBar())

	return b.String()
}

func (m model) viewWithOverlay(overlayView string) string {
	// Draw the table dimmed behind the overlay
	tableView := m.viewTable()
	lines := strings.Split(tableView, "\n")

	overlayLines := strings.Split(overlayView, "\n")

	// Overlay on top of the table, centered
	startRow := 2
	startCol := max(0, (m.width-50)/2)

	for i, ol := range overlayLines {
		row := startRow + i
		if row < len(lines) {
			// Replace part of the line with overlay content
			line := lines[row]
			// Pad line to full width
			lineRunes := []rune(line)
			for len(lineRunes) < m.width {
				lineRunes = append(lineRunes, ' ')
			}
			lines[row] = string(lineRunes[:max(0, startCol)]) + ol
		} else {
			lines = append(lines, strings.Repeat(" ", startCol)+ol)
		}
	}

	return strings.Join(lines, "\n")
}

func (m model) renderHeaderRow(b *strings.Builder, visibleCols []int) {
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(separatorStyle.Render(" │ "))
		}
		name := m.data.Columns[ci]
		w := m.colWidths[ci]
		cell := truncOrPad(name, w)

		if ci == m.cursorCol {
			b.WriteString(headerActiveStyle.Render(cell))
		} else if _, hasFilter := m.filters[ci]; hasFilter {
			b.WriteString(headerFilteredStyle.Render(cell))
		} else {
			b.WriteString(headerStyle.Render(cell))
		}
	}
}

func (m model) renderSeparator(b *strings.Builder, visibleCols []int) {
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(separatorStyle.Render("─┼─"))
		}
		b.WriteString(separatorStyle.Render(strings.Repeat("─", m.colWidths[ci])))
	}
}

func (m model) renderDataRow(b *strings.Builder, rowIdx int, visibleCols []int) {
	row := m.filteredRows[rowIdx]
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(separatorStyle.Render(" │ "))
		}
		val := ""
		if ci < len(row) {
			val = row[ci]
		}
		cell := truncOrPad(val, m.colWidths[ci])

		if rowIdx == m.cursorRow && ci == m.cursorCol {
			b.WriteString(cursorCellStyle.Render(cell))
		} else if rowIdx == m.cursorRow {
			b.WriteString(cursorRowStyle.Render(cell))
		} else {
			b.WriteString(cellStyle.Render(cell))
		}
	}
}

func (m model) statusBar() string {
	rowInfo := fmt.Sprintf("Row %d/%d", m.cursorRow+1, len(m.filteredRows))
	colInfo := fmt.Sprintf("Col %d/%d", m.cursorCol+1, len(m.data.Columns))

	filterInfo := ""
	if len(m.filters) > 0 {
		names := make([]string, 0, len(m.filters))
		for ci := range m.filters {
			names = append(names, m.data.Columns[ci])
		}
		filterInfo = fmt.Sprintf(" | Filtered: %s", strings.Join(names, ", "))
	}

	sortInfo := ""
	if len(m.sortColumns) > 0 {
		names := make([]string, 0, len(m.sortColumns))
		for _, ci := range m.sortColumns {
			names = append(names, m.data.Columns[ci])
		}
		dir := "↑"
		if !m.sortAsc {
			dir = "↓"
		}
		sortInfo = fmt.Sprintf(" | Sort %s: %s", dir, strings.Join(names, " → "))
	}

	total := ""
	if len(m.filteredRows) != len(m.data.Rows) {
		total = fmt.Sprintf(" (of %d total)", len(m.data.Rows))
	}

	if m.copyMode {
		return statusStyle.Render(fmt.Sprintf(
			" %s | %s | %s%s%s%s | COPY: y=cell d=distinct r=row esc=cancel",
			m.filename, rowInfo, colInfo, total, filterInfo, sortInfo))
	}

	copyInfo := ""
	if m.copyMessage != "" {
		copyInfo = " | " + m.copyMessage
	}

	statusLine := statusStyle.Render(fmt.Sprintf(
		" %s | %s | %s%s%s%s%s | f: filter | s: sort | y: copy | q: quit",
		m.filename, rowInfo, colInfo, total, filterInfo, sortInfo, copyInfo))

	// Cell preview line: show full value of cell under cursor, right-aligned
	cellVal := ""
	if m.cursorRow >= 0 && m.cursorRow < len(m.filteredRows) && m.cursorCol < len(m.filteredRows[m.cursorRow]) {
		cellVal = m.filteredRows[m.cursorRow][m.cursorCol]
	}
	colName := ""
	if m.cursorCol >= 0 && m.cursorCol < len(m.data.Columns) {
		colName = m.data.Columns[m.cursorCol]
	}
	preview := fmt.Sprintf("%s: %s", colName, cellVal)
	if len(preview) > m.width {
		preview = preview[:m.width]
	}
	padding := ""
	if pad := m.width - len(preview); pad > 0 {
		padding = strings.Repeat(" ", pad)
	}
	cellLine := statusStyle.Render(padding + preview)

	return statusLine + "\n" + cellLine
}

func (m *model) computeColWidths() {
	if len(m.data.Columns) == 0 {
		return
	}

	widths := make([]int, len(m.data.Columns))
	for i, col := range m.data.Columns {
		widths[i] = len(col)
	}

	sampleSize := len(m.data.Rows)
	if sampleSize > 200 {
		sampleSize = 200
	}
	for i := 0; i < sampleSize; i++ {
		for j, cell := range m.data.Rows[i] {
			if j < len(widths) && len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	for i := range widths {
		if widths[i] < 4 {
			widths[i] = 4
		}
		if widths[i] > 50 {
			widths[i] = 50
		}
	}

	m.colWidths = widths
}

// visibleColumns returns column indices that fit in the current viewport.
func (m model) visibleColumns() []int {
	if len(m.colWidths) == 0 {
		return nil
	}

	var cols []int
	usedWidth := 0
	sepWidth := 3 // " │ "

	for i := m.scrollCol; i < len(m.colWidths); i++ {
		needed := m.colWidths[i]
		if len(cols) > 0 {
			needed += sepWidth
		}
		if usedWidth+needed > m.width && len(cols) > 0 {
			break
		}
		cols = append(cols, i)
		usedWidth += needed
	}
	return cols
}

func (m model) tableHeight() int {
	h := m.height - 5 // header + separator + status + cell preview + padding
	if h < 1 {
		h = 1
	}
	return h
}

func (m *model) scrollIntoView() {
	th := m.tableHeight()
	if m.cursorRow < m.scrollRow {
		m.scrollRow = m.cursorRow
	}
	if m.cursorRow >= m.scrollRow+th {
		m.scrollRow = m.cursorRow - th + 1
	}
}

func (m *model) scrollColIntoView() {
	visible := m.visibleColumns()
	if len(visible) == 0 {
		return
	}

	// If cursor column is before visible range, scroll left
	if m.cursorCol < visible[0] {
		m.scrollCol = m.cursorCol
		return
	}

	// If cursor column is after visible range, scroll right
	if m.cursorCol > visible[len(visible)-1] {
		// Find the minimum scrollCol that makes cursorCol visible
		m.scrollCol = m.cursorCol
		for m.scrollCol > 0 {
			test := m.scrollCol - 1
			m.scrollCol = test
			vc := m.visibleColumns()
			found := false
			for _, c := range vc {
				if c == m.cursorCol {
					found = true
					break
				}
			}
			if !found {
				m.scrollCol = test + 1
				break
			}
		}
	}
}

func (m *model) clampCursor() {
	if m.cursorRow >= len(m.filteredRows) {
		m.cursorRow = max(0, len(m.filteredRows)-1)
	}
	if m.cursorCol >= len(m.data.Columns) {
		m.cursorCol = max(0, len(m.data.Columns)-1)
	}
	m.scrollIntoView()
}

func (m *model) applyFilters() {
	if len(m.filters) == 0 {
		m.filteredRows = m.data.Rows
		return
	}

	m.filteredRows = nil
	for _, row := range m.data.Rows {
		keep := true
		for colIdx, selected := range m.filters {
			if colIdx < len(row) {
				if !selected[row[colIdx]] {
					keep = false
					break
				}
			}
		}
		if keep {
			m.filteredRows = append(m.filteredRows, row)
		}
	}
}

func truncOrPad(s string, w int) string {
	if len(s) > w {
		if w > 1 {
			return s[:w-1] + "…"
		}
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}
