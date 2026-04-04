package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/suntianxun/hmm/internal/reader"
)

// LoadFunc reads a file and returns sheets. Passed to NewLoading so the
// TUI can kick off the read asynchronously.
type LoadFunc func() ([]reader.SheetData, error)

// Messages for async file loading.
type fileLoadedMsg struct {
	sheets []reader.SheetData
}

type fileErrorMsg struct {
	err error
}

type sheetState struct {
	data         *reader.TableData
	filteredRows [][]string

	cursorRow int
	cursorCol int

	scrollRow int
	scrollCol int

	colWidths []int

	filters map[int]map[string]bool

	sortColumns []int
	sortAsc     bool
}

type model struct {
	// Loading state
	loading  bool
	loadFn   LoadFunc
	loadErr  string
	spinner  spinner.Model

	sheets   []reader.SheetData
	states   []sheetState
	activeTab int

	filterActive bool
	filterModel  filterModel

	sortActive bool
	sortModel  sortModel

	copyMode    bool
	copyMessage string

	exportActive bool
	exportModel  exportModel

	width  int
	height int

	filename string
}

func New(data *reader.TableData, filename string) model {
	return NewMulti([]reader.SheetData{{Name: filename, Data: data}}, filename)
}

func NewMulti(sheets []reader.SheetData, filename string) model {
	m := model{
		filename: filename,
		width:    80,
		height:   24,
	}
	m.initSheets(sheets)
	return m
}

// NewLoading creates a model that shows a spinner while loadFn runs.
func NewLoading(filename string, loadFn LoadFunc) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return model{
		loading:  true,
		loadFn:   loadFn,
		spinner:  s,
		filename: filename,
		width:    80,
		height:   24,
	}
}

func (m *model) initSheets(sheets []reader.SheetData) {
	m.sheets = sheets
	states := make([]sheetState, len(sheets))
	for i, s := range sheets {
		states[i] = sheetState{
			data:         s.Data,
			filteredRows: s.Data.Rows,
			filters:      make(map[int]map[string]bool),
		}
	}
	m.states = states
	for i := range m.states {
		m.computeColWidthsFor(i)
	}
}

func (m *model) active() *sheetState {
	return &m.states[m.activeTab]
}

func (m model) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(m.spinner.Tick, m.doLoad())
	}
	return nil
}

func (m model) doLoad() tea.Cmd {
	loadFn := m.loadFn
	return func() tea.Msg {
		sheets, err := loadFn()
		if err != nil {
			return fileErrorMsg{err: err}
		}
		return fileLoadedMsg{sheets: sheets}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading && len(m.states) > 0 {
			for i := range m.states {
				m.computeColWidthsFor(i)
			}
			m.clampCursor()
		}
		return m, nil

	case fileLoadedMsg:
		m.loading = false
		m.loadFn = nil
		if len(msg.sheets) == 0 {
			m.loadErr = "File has no data"
			return m, nil
		}
		m.initSheets(msg.sheets)
		return m, nil

	case fileErrorMsg:
		m.loading = false
		m.loadFn = nil
		m.loadErr = msg.err.Error()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.loadErr != "" {
			return m, tea.Quit
		}

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
	s := m.active()
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
			delete(s.filters, m.filterModel.colIndex)
		} else {
			sel := make(map[string]bool, len(m.filterModel.selected))
			for k, v := range m.filterModel.selected {
				sel[k] = v
			}
			s.filters[m.filterModel.colIndex] = sel
		}
		m.filterActive = false
		m.applyFilters()
		if len(s.sortColumns) > 0 {
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
	s := m.active()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, tea.Quit
	case "tab":
		if len(m.sheets) > 1 {
			m.activeTab = (m.activeTab + 1) % len(m.sheets)
		}
	case "shift+tab":
		if len(m.sheets) > 1 {
			m.activeTab = (m.activeTab - 1 + len(m.sheets)) % len(m.sheets)
		}
	case "up", "k":
		if s.cursorRow > 0 {
			s.cursorRow--
			m.scrollIntoView()
		}
	case "down", "j":
		if s.cursorRow < len(s.filteredRows)-1 {
			s.cursorRow++
			m.scrollIntoView()
		}
	case "left", "h":
		if s.cursorCol > 0 {
			s.cursorCol--
			m.scrollColIntoView()
		}
	case "right", "l":
		if s.cursorCol < len(s.data.Columns)-1 {
			s.cursorCol++
			m.scrollColIntoView()
		}
	case "pgup":
		s.cursorRow -= m.tableHeight()
		if s.cursorRow < 0 {
			s.cursorRow = 0
		}
		m.scrollIntoView()
	case "pgdown":
		s.cursorRow += m.tableHeight()
		if s.cursorRow >= len(s.filteredRows) {
			s.cursorRow = len(s.filteredRows) - 1
		}
		if s.cursorRow < 0 {
			s.cursorRow = 0
		}
		m.scrollIntoView()
	case "g", "home":
		s.cursorRow = 0
		m.scrollIntoView()
	case "G", "end":
		s.cursorRow = max(0, len(s.filteredRows)-1)
		m.scrollIntoView()
	case "f":
		if len(s.data.Columns) > 0 {
			m.filterActive = true
			existing := s.filters[s.cursorCol]
			m.filterModel = newFilterModel(
				s.cursorCol,
				s.data.Columns[s.cursorCol],
				s.data.Rows,
				existing,
			)
		}
	case "F":
		delete(s.filters, s.cursorCol)
		m.applyFilters()
		if len(s.sortColumns) > 0 {
			m.applySorting()
		}
		m.clampCursor()
	case "s":
		if len(s.data.Columns) > 0 {
			m.sortActive = true
			m.sortModel = newSortModel(s.data.Columns)
			if len(s.sortColumns) > 0 {
				m.sortModel.selected = make([]int, len(s.sortColumns))
				copy(m.sortModel.selected, s.sortColumns)
			}
		}
	case "S":
		s.sortColumns = nil
		m.applyFilters()
		m.clampCursor()
	case "y":
		m.copyMode = true
	case "e":
		if len(s.data.Columns) > 0 {
			m.exportActive = true
			m.exportModel = newExportModel(s.data, s.filteredRows)
		}
	}
	return m, nil
}

func (m model) updateCopyMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.active()
	m.copyMode = false
	switch msg.String() {
	case "y":
		val := ""
		if s.cursorRow < len(s.filteredRows) && s.cursorCol < len(s.filteredRows[s.cursorRow]) {
			val = s.filteredRows[s.cursorRow][s.cursorCol]
		}
		if err := clipboard.WriteAll(val); err != nil {
			m.copyMessage = "Copy failed"
		} else {
			m.copyMessage = "Copied cell value"
		}
	case "d":
		seen := make(map[string]struct{})
		var distinct []string
		for _, row := range s.filteredRows {
			if s.cursorCol < len(row) {
				v := row[s.cursorCol]
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
		if s.cursorRow < len(s.filteredRows) {
			row := s.filteredRows[s.cursorRow]
			parts := make([]string, len(s.data.Columns))
			for i, col := range s.data.Columns {
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
	s := m.active()
	switch msg.String() {
	case "a":
		if len(m.sortModel.selected) > 0 {
			s.sortColumns = make([]int, len(m.sortModel.selected))
			copy(s.sortColumns, m.sortModel.selected)
			s.sortAsc = true
			m.applyFilters()
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "d":
		if len(m.sortModel.selected) > 0 {
			s.sortColumns = make([]int, len(m.sortModel.selected))
			copy(s.sortColumns, m.sortModel.selected)
			s.sortAsc = false
			m.applyFilters()
			m.applySorting()
			m.clampCursor()
		}
		m.sortActive = false
		return m, nil
	case "S":
		s.sortColumns = nil
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
	s := m.active()
	if len(s.sortColumns) == 0 {
		return
	}
	slices.SortStableFunc(s.filteredRows, func(a, b []string) int {
		for _, ci := range s.sortColumns {
			va, vb := "", ""
			if ci < len(a) {
				va = a[ci]
			}
			if ci < len(b) {
				vb = b[ci]
			}
			if c := cmp.Compare(va, vb); c != 0 {
				if s.sortAsc {
					return c
				}
				return -c
			}
		}
		return 0
	})
}

func (m model) View() string {
	if m.loading {
		return m.viewLoading()
	}
	if m.loadErr != "" {
		return m.viewError()
	}
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

func (m model) viewLoading() string {
	// Center the spinner vertically and horizontally.
	line := m.spinner.View() + " Opening " + m.filename + "..."

	padTop := max(0, m.height/2-1)
	padLeft := max(0, (m.width-len(m.filename)-15)/2)

	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat(" ", padLeft))
	b.WriteString(loadingTextStyle.Render(line))
	return b.String()
}

func (m model) viewError() string {
	line := "Error: " + m.loadErr
	hint := "Press any key to exit."

	padTop := max(0, m.height/2-1)
	padLeft := max(0, (m.width-len(line))/2)
	hintPad := max(0, (m.width-len(hint))/2)

	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat(" ", padLeft))
	b.WriteString(exportErrorStyle.Render(line))
	b.WriteString("\n\n")
	b.WriteString(strings.Repeat(" ", hintPad))
	b.WriteString(filterHintStyle.Render(hint))
	return b.String()
}

func (m model) viewTable() string {
	s := m.active()
	var b strings.Builder
	b.Grow(m.width * m.height * 2)

	// Tab bar (only for multi-sheet files)
	hasTabBar := len(m.sheets) > 1
	if hasTabBar {
		m.renderTabBar(&b)
		b.WriteByte('\n')
	}

	th := m.tableHeight()
	visibleCols := m.visibleColumns()

	// Header row
	m.renderHeaderRow(&b, visibleCols)
	b.WriteByte('\n')

	// Separator
	m.renderSeparator(&b, visibleCols)
	b.WriteByte('\n')

	// Data rows
	endRow := min(s.scrollRow+th, len(s.filteredRows))
	for ri := s.scrollRow; ri < endRow; ri++ {
		m.renderDataRow(&b, ri, visibleCols)
		b.WriteByte('\n')
	}

	// Pad remaining lines
	for i := endRow - s.scrollRow; i < th; i++ {
		b.WriteByte('\n')
	}

	// Status bar
	b.WriteString(m.statusBar())

	return b.String()
}

func (m model) renderTabBar(b *strings.Builder) {
	for i, sheet := range m.sheets {
		if i > 0 {
			b.WriteString(tabSepStyle.Render(" "))
		}
		label := " " + sheet.Name + " "
		if i == m.activeTab {
			b.WriteString(tabActiveStyle.Render(label))
		} else {
			b.WriteString(tabInactiveStyle.Render(label))
		}
	}
}

func (m model) viewWithOverlay(overlayView string) string {
	tableView := m.viewTable()
	lines := strings.Split(tableView, "\n")
	overlayLines := strings.Split(overlayView, "\n")

	startRow := 2
	if len(m.sheets) > 1 {
		startRow = 3
	}
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
	s := m.active()
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(headerSepStyle.Render(" │ "))
		}
		cell := truncOrPad(s.data.Columns[ci], s.colWidths[ci])

		if ci == s.cursorCol {
			b.WriteString(headerActiveStyle.Render(cell))
		} else if _, hasFilter := s.filters[ci]; hasFilter {
			b.WriteString(headerFilteredStyle.Render(cell))
		} else {
			b.WriteString(headerStyle.Render(cell))
		}
	}
}

func (m model) renderSeparator(b *strings.Builder, visibleCols []int) {
	s := m.active()
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(separatorStyle.Render("─┼─"))
		}
		b.WriteString(separatorStyle.Render(strings.Repeat("─", s.colWidths[ci])))
	}
}

func (m model) renderDataRow(b *strings.Builder, rowIdx int, visibleCols []int) {
	s := m.active()
	row := s.filteredRows[rowIdx]
	for i, ci := range visibleCols {
		if i > 0 {
			b.WriteString(separatorStyle.Render(" │ "))
		}
		val := ""
		if ci < len(row) {
			val = row[ci]
		}
		cell := truncOrPad(val, s.colWidths[ci])

		if rowIdx == s.cursorRow && ci == s.cursorCol {
			b.WriteString(cursorCellStyle.Render(cell))
		} else if rowIdx == s.cursorRow {
			b.WriteString(cursorRowStyle.Render(cell))
		} else {
			b.WriteString(cellStyle.Render(cell))
		}
	}
}

func (m model) statusBar() string {
	s := m.active()
	rowInfo := fmt.Sprintf("Row %d/%d", s.cursorRow+1, len(s.filteredRows))
	colInfo := fmt.Sprintf("Col %d/%d", s.cursorCol+1, len(s.data.Columns))

	var extra strings.Builder

	if len(s.filteredRows) != len(s.data.Rows) {
		fmt.Fprintf(&extra, " (of %d total)", len(s.data.Rows))
	}

	if len(s.filters) > 0 {
		names := make([]string, 0, len(s.filters))
		for ci := range s.filters {
			names = append(names, s.data.Columns[ci])
		}
		fmt.Fprintf(&extra, " | Filtered: %s", strings.Join(names, ", "))
	}

	if len(s.sortColumns) > 0 {
		names := make([]string, 0, len(s.sortColumns))
		for _, ci := range s.sortColumns {
			names = append(names, s.data.Columns[ci])
		}
		dir := "↑"
		if !s.sortAsc {
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
		tabHint := ""
		if len(m.sheets) > 1 {
			tabHint = " | tab/shift+tab: sheets"
		}
		statusText = fmt.Sprintf(" %s | %s | %s%s%s | f: filter | s: sort | y: copy | e: export%s | q: quit",
			m.filename, rowInfo, colInfo, extra.String(), copyInfo, tabHint)
	}

	statusLine := statusStyle.Render(statusText)

	// Cell preview line
	cellVal := ""
	if s.cursorRow >= 0 && s.cursorRow < len(s.filteredRows) && s.cursorCol < len(s.filteredRows[s.cursorRow]) {
		cellVal = s.filteredRows[s.cursorRow][s.cursorCol]
	}
	colName := ""
	if s.cursorCol >= 0 && s.cursorCol < len(s.data.Columns) {
		colName = s.data.Columns[s.cursorCol]
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


func (m *model) computeColWidthsFor(idx int) {
	s := &m.states[idx]
	if len(s.data.Columns) == 0 {
		return
	}

	widths := make([]int, len(s.data.Columns))
	for i, col := range s.data.Columns {
		widths[i] = len(col)
	}

	sampleSize := min(len(s.data.Rows), 200)
	for i := 0; i < sampleSize; i++ {
		for j, cell := range s.data.Rows[i] {
			if j < len(widths) && len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	for i := range widths {
		widths[i] = max(widths[i], 4)
		widths[i] = min(widths[i], 50)
	}

	s.colWidths = widths
}

func (m model) visibleColumns() []int {
	s := m.active()
	if len(s.colWidths) == 0 {
		return nil
	}

	cols := make([]int, 0, len(s.colWidths))
	usedWidth := 0
	const sepWidth = 3 // " │ "

	for i := s.scrollCol; i < len(s.colWidths); i++ {
		needed := s.colWidths[i]
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
	if len(m.sheets) > 1 {
		h-- // tab bar
	}
	return max(1, h)
}

func (m *model) scrollIntoView() {
	s := m.active()
	th := m.tableHeight()
	if s.cursorRow < s.scrollRow {
		s.scrollRow = s.cursorRow
	}
	if s.cursorRow >= s.scrollRow+th {
		s.scrollRow = s.cursorRow - th + 1
	}
}

func (m *model) scrollColIntoView() {
	s := m.active()
	visible := m.visibleColumns()
	if len(visible) == 0 {
		return
	}

	if s.cursorCol < visible[0] {
		s.scrollCol = s.cursorCol
		return
	}

	if s.cursorCol > visible[len(visible)-1] {
		s.scrollCol = s.cursorCol
		for s.scrollCol > 0 {
			s.scrollCol--
			vc := m.visibleColumns()
			if len(vc) == 0 || vc[len(vc)-1] < s.cursorCol {
				s.scrollCol++
				break
			}
		}
	}
}

func (m *model) clampCursor() {
	s := m.active()
	if s.cursorRow >= len(s.filteredRows) {
		s.cursorRow = max(0, len(s.filteredRows)-1)
	}
	if s.cursorCol >= len(s.data.Columns) {
		s.cursorCol = max(0, len(s.data.Columns)-1)
	}
	m.scrollIntoView()
}

func (m *model) applyFilters() {
	s := m.active()
	if len(s.filters) == 0 {
		s.filteredRows = s.data.Rows
		return
	}

	s.filteredRows = make([][]string, 0, len(s.data.Rows)/2)
	for _, row := range s.data.Rows {
		keep := true
		for colIdx, selected := range s.filters {
			if colIdx < len(row) && !selected[row[colIdx]] {
				keep = false
				break
			}
		}
		if keep {
			s.filteredRows = append(s.filteredRows, row)
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
