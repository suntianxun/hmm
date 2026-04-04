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

	cursorRow int
	cursorCol int

	scrollRow int
	scrollCol int

	colWidths []int

	width  int
	height int

	filters map[int]map[string]bool

	filterActive bool
	filterModel  filterModel

	sortActive bool
	sortModel  sortModel

	sortColumns []int
	sortAsc     bool

	copyMode    bool
	copyMessage string

	exportActive bool
	exportModel  exportModel

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
		m.copyMessage = ""

		if m.filterActive {
			return m.updateFilter(msg)
		}
		if m.sortActive {
			return m.updateSort(msg)
		}
		if m.exportActive {
			return m.updateExport(msg)
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
		allSelected := true
		for _, v := range m.filterModel.allVals {
			if !m.filterModel.selected[v] {
				allSelected = false
				break
			}
		}
		if allSelected {
			delete(m.filters, m.filterModel.colIndex)
		} else {
			sel := make(map[string]bool, len(m.filterModel.selected))
			for k, v := range m.filterModel.selected {
				sel[k] = v
			}
			m.filters[m.filterModel.colIndex] = sel
		}
		m.filterActive = false
		m.applyFilters()
		if len(m.sortColumns) > 0 {
			m.applySorting()
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
		m.filterActive = true
		existing := m.filters[m.cursorCol]
		m.filterModel = newFilterModel(
			m.cursorCol,
			m.data.Columns[m.cursorCol],
			m.data.Rows,
			existing,
		)
	case "F":
		delete(m.filters, m.cursorCol)
		m.applyFilters()
		if len(m.sortColumns) > 0 {
			m.applySorting()
		}
		m.clampCursor()
	case "s":
		m.sortActive = true
		m.sortModel = newSortModel(m.data.Columns)
		if len(m.sortColumns) > 0 {
			m.sortModel.selected = make([]int, len(m.sortColumns))
			copy(m.sortModel.selected, m.sortColumns)
		}
	case "S":
		m.sortColumns = nil
		m.applyFilters()
		m.clampCursor()
	case "y":
		m.copyMode = true
	case "e":
		m.exportActive = true
		m.exportModel = newExportModel(m.data, m.filteredRows)
	}
	return m, nil
}

func (m model) updateCopyMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.copyMode = false
	switch msg.String() {
	case "y":
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
		if err := clipboard.WriteAll(strings.Join(distinct, "\n")); err != nil {
			m.copyMessage = "Copy failed"
		} else {
			m.copyMessage = fmt.Sprintf("Copied %d distinct values", len(distinct))
		}
	case "r":
		if m.cursorRow < len(m.filteredRows) {
			row := m.filteredRows[m.cursorRow]
			parts := make([]string, len(m.data.Columns))
			for i, col := range m.data.Columns {
				val := ""
				if i < len(row) {
					val = row[i]
				}
				parts[i] = col + ": " + val
			}
			if err := clipboard.WriteAll(strings.Join(parts, "\n")); err != nil {
				m.copyMessage = "Copy failed"
			} else {
				m.copyMessage = "Copied row"
			}
		}
	case "esc":
	default:
	}
	return m, nil
}

func (m model) updateExport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.exportActive = false
		return m, nil
	}

	cmd := m.exportModel.update(msg)
	if m.exportModel.done {
		m.exportActive = false
		m.copyMessage = m.exportModel.success
	}
	return m, cmd
}

func (m model) updateSort(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		if len(m.sortModel.selected) > 0 {
			m.sortColumns = make([]int, len(m.sortModel.selected))
			copy(m.sortColumns, m.sortModel.selected)
			m.sortAsc = true
			m.applyFilters()
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "d":
		if len(m.sortModel.selected) > 0 {
			m.sortColumns = make([]int, len(m.sortModel.selected))
			copy(m.sortColumns, m.sortModel.selected)
			m.sortAsc = false
			m.applyFilters()
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "S":
		m.sortColumns = nil
		m.applyFilters()
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
	if m.exportActive {
		return m.viewWithOverlay(m.exportModel.view(min(m.width-4, 60)))
	}
	return m.viewTable()
}

func (m model) viewTable() string {
	var b strings.Builder
	b.Grow(m.width * m.height * 2) // pre-allocate

	th := m.tableHeight()
	visibleCols := m.visibleColumns()

	// Header row (full-width background)
	m.renderHeaderRow(&b, visibleCols)
	b.WriteByte('\n')

	// Separator
	m.renderSeparator(&b, visibleCols)
	b.WriteByte('\n')

	// Data rows
	endRow := min(m.scrollRow+th, len(m.filteredRows))
	for ri := m.scrollRow; ri < endRow; ri++ {
		m.renderDataRow(&b, ri, visibleCols)
		b.WriteByte('\n')
	}

	// Pad remaining lines
	for i := endRow - m.scrollRow; i < th; i++ {
		b.WriteByte('\n')
	}

	// Status bar
	b.WriteString(m.statusBar())

	return b.String()
}

func (m model) viewWithOverlay(overlayView string) string {
	tableView := m.viewTable()
	lines := strings.Split(tableView, "\n")
	overlayLines := strings.Split(overlayView, "\n")

	startRow := 2
	startCol := max(0, (m.width-50)/2)

	for i, ol := range overlayLines {
		row := startRow + i
		if row < len(lines) {
			lineRunes := []rune(lines[row])
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
			b.WriteString(headerSepStyle.Render(" │ "))
		}
		cell := truncOrPad(m.data.Columns[ci], m.colWidths[ci])

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

	var extra strings.Builder

	if len(m.filteredRows) != len(m.data.Rows) {
		fmt.Fprintf(&extra, " (of %d total)", len(m.data.Rows))
	}

	if len(m.filters) > 0 {
		names := make([]string, 0, len(m.filters))
		for ci := range m.filters {
			names = append(names, m.data.Columns[ci])
		}
		fmt.Fprintf(&extra, " | Filtered: %s", strings.Join(names, ", "))
	}

	if len(m.sortColumns) > 0 {
		names := make([]string, 0, len(m.sortColumns))
		for _, ci := range m.sortColumns {
			names = append(names, m.data.Columns[ci])
		}
		dir := "↑"
		if !m.sortAsc {
			dir = "↓"
		}
		fmt.Fprintf(&extra, " | Sort %s: %s", dir, strings.Join(names, " → "))
	}

	var statusText string
	if m.copyMode {
		statusText = fmt.Sprintf(" %s | %s | %s%s | COPY: y=cell d=distinct r=row esc=cancel",
			m.filename, rowInfo, colInfo, extra.String())
	} else {
		copyInfo := ""
		if m.copyMessage != "" {
			copyInfo = " | " + m.copyMessage
		}
		statusText = fmt.Sprintf(" %s | %s | %s%s%s | f: filter | s: sort | y: copy | e: export | q: quit",
			m.filename, rowInfo, colInfo, extra.String(), copyInfo)
	}

	statusLine := statusStyle.Render(statusText)

	// Cell preview line
	cellVal := ""
	if m.cursorRow >= 0 && m.cursorRow < len(m.filteredRows) && m.cursorCol < len(m.filteredRows[m.cursorRow]) {
		cellVal = m.filteredRows[m.cursorRow][m.cursorCol]
	}
	colName := ""
	if m.cursorCol >= 0 && m.cursorCol < len(m.data.Columns) {
		colName = m.data.Columns[m.cursorCol]
	}
	preview := colName + ": " + cellVal
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

	sampleSize := min(len(m.data.Rows), 200)
	for i := 0; i < sampleSize; i++ {
		for j, cell := range m.data.Rows[i] {
			if j < len(widths) && len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	for i := range widths {
		widths[i] = max(widths[i], 4)
		widths[i] = min(widths[i], 50)
	}

	m.colWidths = widths
}

func (m model) visibleColumns() []int {
	if len(m.colWidths) == 0 {
		return nil
	}

	cols := make([]int, 0, len(m.colWidths))
	usedWidth := 0
	const sepWidth = 3 // " │ "

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
	return max(1, m.height-5) // header + separator + status + cell preview + padding
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

	if m.cursorCol < visible[0] {
		m.scrollCol = m.cursorCol
		return
	}

	if m.cursorCol > visible[len(visible)-1] {
		// Binary-search-style: find the minimum scrollCol that makes cursorCol visible
		m.scrollCol = m.cursorCol
		for m.scrollCol > 0 {
			m.scrollCol--
			vc := m.visibleColumns()
			if len(vc) == 0 || vc[len(vc)-1] < m.cursorCol {
				m.scrollCol++
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

	m.filteredRows = make([][]string, 0, len(m.data.Rows)/2)
	for _, row := range m.data.Rows {
		keep := true
		for colIdx, selected := range m.filters {
			if colIdx < len(row) && !selected[row[colIdx]] {
				keep = false
				break
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
	if len(s) == w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
