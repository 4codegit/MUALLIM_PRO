package ui

import (
        "fmt"
        "image/color"
        "math/rand"
        "sort"
        "strings"
        "time"

        "fyne.io/fyne/v2"
        canvas "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/internal/config"
        "github.com/4codegit/edonish-auto/internal/engine"
)

// ─── Data models ──────────────────────────────────────────────

// journalData holds the structured data for the journal table view.
type journalData struct {
        groupName   string
        groupID     int
        subjectID   int
        qpropID     int
        subjectName string
        quarterName string
        dates       []dateCol
        students    []studentRow
}

// dateCol represents a single date column in the journal.
type dateCol struct {
        dateID   string
        dateStr  string
        shortStr string
}

// studentRow represents a single student row with marks.
type studentRow struct {
        studentID   int
        name        string
        marks       map[string]string // dateID -> display text
        markValues  map[string]int    // dateID -> numeric mark value
        markIDs     map[string]string // dateID -> mark ID (for deletion)
        avg         float64
        min         int
        max         int
        gradeCount  int
        missing     int
        // Per-student min/max override for auto-grade
        minOverride int // 0 = use global
        maxOverride int // 0 = use global
}

// studentGradeLimits holds per-student min/max grade settings.
type studentGradeLimits struct {
        studentName string
        minGrade    int
        maxGrade    int
}

// ─── JournalPage ──────────────────────────────────────────────

// JournalPage holds the journal viewer UI components.
type JournalPage struct {
        app *App

        // Filters
        classSelect   *widget.Select
        subjectSelect *widget.Select
        quarterSelect *widget.Select

        // Status
        statusLabel *canvas.Text

        // Table view
        journalTable *widget.Table
        journalData  []journalData

        // Selected cell tracking
        selectedCell  *widget.TableCellID
        cellEditEntry *widget.Entry

        // Double-click detection
        lastClickTime time.Time
        lastClickCell widget.TableCellID

        // Student detail panel
        studentDetail   *widget.Entry
        detailCard      *widget.Card
        selectedStudent string
        detailPanel     *fyne.Container

        // Per-student grade limits (min/max overrides)
        studentLimits map[string]*studentGradeLimits // key: studentName

        // Edit state
        editDialog *dialog.CustomDialog
        editEntry  *widget.Entry
        editInfo   *editMarkInfo

        // Layout references
        tableScroll *container.Scroll
        splitLayout *container.Scroll

        // Current selected student name from table row (for per-student random fill)
        currentStudentName string
}

// editMarkInfo holds context for a grade edit operation.
type editMarkInfo struct {
        groupIdx    int
        studentIdx  int
        dateID      string
        dateStr     string
        studentName string
        oldValue    string
        oldMarkVal  int
        markID      string // existing mark ID for deletion
        qpropID     int
        studentID   int
}

// NewJournalPage creates a new journal page.
func NewJournalPage(app *App) *JournalPage {
        return &JournalPage{
                app:           app,
                studentLimits: make(map[string]*studentGradeLimits),
        }
}

// ─── Color helpers ────────────────────────────────────────────

// gradeColor returns a color for the given grade value.
func gradeColor(val int) color.Color {
        switch {
        case val == 10:
                return color.NRGBA{R: 0, G: 150, B: 0, A: 255}   // bold green
        case val == 9:
                return color.NRGBA{R: 0, G: 130, B: 50, A: 255}   // dark green
        case val == 8:
                return theme.ForegroundColor()                      // default
        case val == 7:
                return color.NRGBA{R: 100, G: 100, B: 100, A: 255} // grey
        case val >= 5:
                return color.NRGBA{R: 200, G: 120, B: 0, A: 255}   // orange
        case val >= 1:
                return color.NRGBA{R: 200, G: 0, B: 0, A: 255}     // red
        default: // 0 = absent
                return color.NRGBA{R: 150, G: 0, B: 0, A: 255} // dark red
        }
}

// statusColor returns a color based on status type.
func statusColor(statusType string) color.Color {
        switch statusType {
        case "info":
                return color.NRGBA{R: 30, G: 100, B: 200, A: 255} // blue
        case "error":
                return color.NRGBA{R: 200, G: 30, B: 30, A: 255}  // red
        case "success":
                return color.NRGBA{R: 30, G: 150, B: 30, A: 255}  // green
        case "warning":
                return color.NRGBA{R: 200, G: 120, B: 0, A: 255}  // orange
        default:
                return theme.ForegroundColor()
        }
}

// setStatus updates the status label text and color.
func (p *JournalPage) setStatus(text, statusType string) {
        p.statusLabel.Text = text
        p.statusLabel.Color = statusColor(statusType)
        canvas.Refresh(p.statusLabel)
}

// ─── Build ────────────────────────────────────────────────────

// Build creates the journal view and returns the root container.
func (p *JournalPage) Build() fyne.CanvasObject {
        // ── Filter bar ────────────────────────────────────────
        p.classSelect = widget.NewSelect([]string{}, func(s string) {
                p.onClassChange(s)
        })
        p.classSelect.PlaceHolder = "Класс"

        p.subjectSelect = widget.NewSelect([]string{}, func(s string) {
                p.onSubjectChange(s)
        })
        p.subjectSelect.PlaceHolder = "Предмет"

        p.quarterSelect = widget.NewSelect([]string{}, func(s string) {
                p.onQuarterChange(s)
        })
        p.quarterSelect.PlaceHolder = "Четверть"

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                p.loadJournal()
        })

        // Button to open student limits popup
        limitsBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
                p.showLimitsDialog()
        })

        // Random fill button
        randomBtn := widget.NewButton("Рандом", func() {
                p.showRandomFillDialog()
        })

        // ── Status label (canvas.Text for colored output) ─────
        p.statusLabel = canvas.NewText("Выберите класс и предмет для загрузки журнала", theme.ForegroundColor())
        p.statusLabel.Alignment = fyne.TextAlignCenter
        p.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

        // ── Journal Table (canvas.Text cells for color support) ─
        p.journalTable = widget.NewTable(
                func() (int, int) { return p.tableRowCount(), p.tableColCount() },
                func() fyne.CanvasObject {
                        t := canvas.NewText("", theme.ForegroundColor())
                        t.Alignment = fyne.TextAlignCenter
                        return t
                },
                func(id widget.TableCellID, cell fyne.CanvasObject) {
                        p.tableCellUpdate(id, cell.(*canvas.Text))
                },
        )
        p.journalTable.SetColumnWidth(0, 35)
        p.journalTable.SetColumnWidth(1, 180)

        // Cell selection handler — tracks selected cell + double-click detection
        p.journalTable.OnSelected = func(id widget.TableCellID) {
                now := time.Now()
                isDoubleClick := id.Row == p.lastClickCell.Row &&
                        id.Col == p.lastClickCell.Col &&
                        now.Sub(p.lastClickTime) < 500*time.Millisecond

                p.selectedCell = &widget.TableCellID{Row: id.Row, Col: id.Col}

                // Track which student row is selected (for per-student random fill)
                p.trackSelectedStudent(id)

                if isDoubleClick {
                        // Double-click on any cell → open edit or detail
                        p.openEditForSelectedCell()
                } else if id.Col == 1 {
                        // Single click on name column → show student detail
                        p.showDetailForSelectedCell()
                }
                // Single click on grade cell → just select (no dialog)

                p.lastClickTime = now
                p.lastClickCell = id
        }

        p.tableScroll = container.NewScroll(p.journalTable)
        p.tableScroll.SetMinSize(fyne.NewSize(800, 400))

        // ── Student detail panel (right side, hidden by default) ──
        p.studentDetail = widget.NewMultiLineEntry()
        p.studentDetail.Wrapping = fyne.TextWrapWord
        p.studentDetail.TextStyle = fyne.TextStyle{Monospace: true}
        p.studentDetail.SetMinRowsVisible(15)

        closeDetailBtn := widget.NewButton("Закрыть", func() {
                p.hideDetail()
        })

        p.detailCard = widget.NewCard("Анализ ученика", "", container.NewVBox(
                p.studentDetail,
                closeDetailBtn,
        ))

        p.detailPanel = container.NewVBox()
        p.detailPanel.Hide()

        // ── Main layout ───────────────────────────────────────
        filterRow := container.NewBorder(nil, nil, nil,
                container.NewHBox(randomBtn, limitsBtn, refreshBtn),
                container.NewGridWithColumns(3,
                        p.classSelect,
                        p.subjectSelect,
                        p.quarterSelect,
                ),
        )

        header := widget.NewCard("", "", container.NewVBox(
                filterRow,
                p.statusLabel,
        ))

        tableArea := container.NewHSplit(
                p.tableScroll,
                p.detailPanel,
        )
        tableArea.SetOffset(0.75)

        content := container.NewBorder(header, nil, nil, nil, tableArea)

        p.splitLayout = container.NewVScroll(content)

        // ── Keyboard handler ──────────────────────────────────
        p.setupKeyHandler()

        return p.splitLayout
}

// ─── Keyboard handler ─────────────────────────────────────────

// setupKeyHandler adds global keyboard listeners for the journal page.
func (p *JournalPage) setupKeyHandler() {
        // Defer to ensure the window canvas is available
        fyne.Do(func() {
                c := p.app.mainWindow.Canvas()

                c.SetOnTypedKey(func(ev *fyne.KeyEvent) {
                        // Only handle keys when journal tab is active (index 1)
                        if p.app.tabs == nil || p.app.tabs.SelectedIndex() != 1 {
                                return
                        }
                        // Don't intercept if an entry or select has focus
                        focused := c.Focused()
                        if _, ok := focused.(*widget.Entry); ok {
                                return
                        }
                        if _, ok := focused.(*widget.Select); ok {
                                return
                        }

                        if p.selectedCell == nil {
                                return
                        }

                        switch ev.Name {
                        case fyne.KeyTab:
                                p.moveSelectionRight()
                        case fyne.KeyReturn, fyne.KeyEnter:
                                p.openEditForSelectedCell()
                        case fyne.KeyDelete, fyne.KeyBackspace:
                                p.deleteGradeInSelectedCell()
                        case fyne.KeyRight:
                                p.moveSelectionRight()
                        case fyne.KeyLeft:
                                p.moveSelectionLeft()
                        case fyne.KeyUp:
                                p.moveSelectionUp()
                        case fyne.KeyDown:
                                p.moveSelectionDown()
                        }
                })

                c.SetOnTypedRune(func(r rune) {
                        // Only handle when journal tab is active
                        if p.app.tabs == nil || p.app.tabs.SelectedIndex() != 1 {
                                return
                        }
                        // Don't intercept if an entry or select has focus
                        focused := c.Focused()
                        if _, ok := focused.(*widget.Entry); ok {
                                return
                        }
                        if _, ok := focused.(*widget.Select); ok {
                                return
                        }

                        if p.selectedCell == nil {
                                return
                        }

                        if r >= '1' && r <= '9' {
                                val := int(r - '0')
                                p.quickSetGrade(val)
                        } else if r == '0' {
                                p.quickSetGrade(10)
                        }
                })
        })
}

// ─── Cell navigation ─────────────────────────────────────────

func (p *JournalPage) moveSelectionRight() {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }
        newCol := p.selectedCell.Col + 1
        maxCol := p.tableColCount() - 1
        if newCol > maxCol {
                // Move to next row, first data column
                newCol = 2
                newRow := p.selectedCell.Row + 1
                maxRow := p.tableRowCount() - 1
                if newRow > maxRow {
                        return
                }
                p.journalTable.Select(widget.TableCellID{Row: newRow, Col: newCol})
                return
        }
        p.journalTable.Select(widget.TableCellID{Row: p.selectedCell.Row, Col: newCol})
}

func (p *JournalPage) moveSelectionLeft() {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }
        newCol := p.selectedCell.Col - 1
        if newCol < 2 {
                // Move to previous row, last data column
                newRow := p.selectedCell.Row - 1
                if newRow < 0 {
                        return
                }
                maxCol := p.tableColCount() - 1
                p.journalTable.Select(widget.TableCellID{Row: newRow, Col: maxCol})
                return
        }
        p.journalTable.Select(widget.TableCellID{Row: p.selectedCell.Row, Col: newCol})
}

func (p *JournalPage) moveSelectionUp() {
        if p.selectedCell == nil {
                return
        }
        newRow := p.selectedCell.Row - 1
        if newRow < 0 {
                return
        }
        p.journalTable.Select(widget.TableCellID{Row: newRow, Col: p.selectedCell.Col})
}

func (p *JournalPage) moveSelectionDown() {
        if p.selectedCell == nil {
                return
        }
        newRow := p.selectedCell.Row + 1
        maxRow := p.tableRowCount() - 1
        if newRow > maxRow {
                return
        }
        p.journalTable.Select(widget.TableCellID{Row: newRow, Col: p.selectedCell.Col})
}

// ─── Quick set grade ─────────────────────────────────────────

// quickSetGrade immediately sets a grade for the selected cell and moves right.
func (p *JournalPage) quickSetGrade(val int) {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }
        if val < 1 || val > 10 {
                return
        }

        info := p.resolveCellInfo(*p.selectedCell)
        if info == nil {
                return
        }

        p.setStatus(fmt.Sprintf("Установка оценки %d для %s...", val, info.studentName), "info")
        p.app.LogMessage(fmt.Sprintf("Быстрая оценка: %s — %s -> %d", info.studentName, info.dateStr, val), "info")

        go func() {
                // Delete existing mark if present
                if info.markID != "" {
                        _, _ = p.app.apiClient.DeleteMark(info.markID)
                }

                result, err := p.app.apiClient.CreateMark(
                        info.studentID,
                        info.dateID,
                        val,
                        8, // mark_type_id
                        info.qpropID,
                        config.Signature,
                )

                fyne.Do(func() {
                        if err != nil {
                                p.setStatus(fmt.Sprintf("Ошибка: %v", err), "error")
                                p.app.LogMessage(fmt.Sprintf("Ошибка сохранения: %v", err), "error")
                        } else if resultMap, ok := result.(map[string]interface{}); ok {
                                if errMsg, exists := resultMap["error"]; exists && errMsg != nil {
                                        p.setStatus(fmt.Sprintf("Ошибка: %v", errMsg), "error")
                                        p.app.LogMessage(fmt.Sprintf("Ошибка API: %v", errMsg), "error")
                                } else {
                                        p.setStatus(fmt.Sprintf("Оценка %d сохранена для %s", val, info.studentName), "success")
                                        p.app.LogMessage(fmt.Sprintf("Оценка %d сохранена: %s (%s)", val, info.studentName, info.dateStr), "info")
                                        p.loadJournal()
                                        // Move selection to next cell right
                                        p.moveSelectionRight()
                                }
                        } else {
                                p.setStatus(fmt.Sprintf("Оценка %d сохранена для %s", val, info.studentName), "success")
                                p.loadJournal()
                                p.moveSelectionRight()
                        }
                })
        }()
}

// resolveCellInfo resolves a table cell ID to an editMarkInfo.
func (p *JournalPage) resolveCellInfo(id widget.TableCellID) *editMarkInfo {
        if len(p.journalData) == 0 {
                return nil
        }

        rowIdx := 0
        for di, jd := range p.journalData {
                rowIdx += 2 // skip title + date header

                if id.Row >= rowIdx && id.Row < rowIdx+len(jd.students) {
                        si := id.Row - rowIdx
                        sr := jd.students[si]

                        if id.Col >= 2 && id.Col < 2+len(jd.dates) {
                                dateID := jd.dates[id.Col-2].dateID
                                dateStr := jd.dates[id.Col-2].dateStr

                                oldDisplay := ""
                                oldVal := 0
                                markID := ""
                                if display, ok := sr.marks[dateID]; ok {
                                        oldDisplay = display
                                        oldVal = sr.markValues[dateID]
                                        markID = sr.markIDs[dateID]
                                }

                                return &editMarkInfo{
                                        groupIdx:    di,
                                        studentIdx:  si,
                                        dateID:      dateID,
                                        dateStr:     dateStr,
                                        studentName: sr.name,
                                        oldValue:    oldDisplay,
                                        oldMarkVal:  oldVal,
                                        markID:      markID,
                                        qpropID:     jd.qpropID,
                                        studentID:   sr.studentID,
                                }
                        }
                        return nil
                }
                rowIdx += len(jd.students) + 1
        }
        return nil
}

// openEditForSelectedCell opens the edit dialog for the currently selected cell.
func (p *JournalPage) openEditForSelectedCell() {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }

        id := *p.selectedCell

        rowIdx := 0
        for di, jd := range p.journalData {
                rowIdx += 2 // skip title + date header

                if id.Row >= rowIdx && id.Row < rowIdx+len(jd.students) {
                        si := id.Row - rowIdx
                        sr := jd.students[si]

                        // Grade column → edit dialog
                        if id.Col >= 2 && id.Col < 2+len(jd.dates) {
                                dateID := jd.dates[id.Col-2].dateID
                                dateStr := jd.dates[id.Col-2].dateStr
                                p.showEditDialog(di, si, dateID, dateStr, sr)
                                return
                        }

                        // Name column → student detail
                        if id.Col == 1 {
                                p.showStudentDetail(sr, jd)
                                return
                        }

                        // Any other column → student detail
                        p.showStudentDetail(sr, jd)
                        return
                }
                rowIdx += len(jd.students) + 1
        }
}

// showDetailForSelectedCell shows the student detail for the selected cell row.
func (p *JournalPage) showDetailForSelectedCell() {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }

        id := *p.selectedCell

        rowIdx := 0
        for _, jd := range p.journalData {
                rowIdx += 2 // skip title + date header

                if id.Row >= rowIdx && id.Row < rowIdx+len(jd.students) {
                        si := id.Row - rowIdx
                        sr := jd.students[si]
                        p.showStudentDetail(sr, jd)
                        return
                }
                rowIdx += len(jd.students) + 1
        }
}

// trackSelectedStudent updates currentStudentName based on the selected table row.
func (p *JournalPage) trackSelectedStudent(id widget.TableCellID) {
        if len(p.journalData) == 0 {
                p.currentStudentName = ""
                return
        }
        rowIdx := 0
        for _, jd := range p.journalData {
                rowIdx += 2 // skip title + date header
                if id.Row >= rowIdx && id.Row < rowIdx+len(jd.students) {
                        si := id.Row - rowIdx
                        p.currentStudentName = jd.students[si].name
                        return
                }
                rowIdx += len(jd.students) + 1
        }
        p.currentStudentName = ""
}

// deleteGradeInSelectedCell deletes the grade in the currently selected cell.
func (p *JournalPage) deleteGradeInSelectedCell() {
        if p.selectedCell == nil || len(p.journalData) == 0 {
                return
        }

        info := p.resolveCellInfo(*p.selectedCell)
        if info == nil || info.markID == "" {
                return
        }

        p.setStatus(fmt.Sprintf("Удаление оценки для %s...", info.studentName), "info")
        p.app.LogMessage(fmt.Sprintf("Удаление оценки: %s (%s)", info.studentName, info.dateStr), "info")

        markID := info.markID
        studentName := info.studentName

        go func() {
                _, err := p.app.apiClient.DeleteMark(markID)

                fyne.Do(func() {
                        if err != nil {
                                p.setStatus(fmt.Sprintf("Ошибка удаления: %v", err), "error")
                                p.app.LogMessage(fmt.Sprintf("Ошибка удаления: %v", err), "error")
                        } else {
                                p.setStatus(fmt.Sprintf("Оценка удалена для %s", studentName), "success")
                                p.app.LogMessage(fmt.Sprintf("Оценка удалена: %s", studentName), "info")
                                p.loadJournal()
                        }
                })
        }()
}

// ─── Filter change handlers (auto-load) ──────────────────────

func (p *JournalPage) onClassChange(selected string) {
        p.updateSubjectsForClass(selected)
        p.updateQuartersForClass(selected)
        if p.subjectSelect.Selected != "" && p.subjectSelect.Selected != "Все предметы" {
                p.loadJournal()
        }
}

func (p *JournalPage) onSubjectChange(selected string) {
        if selected != "" && selected != "Все предметы" {
                p.loadJournal()
        }
}

func (p *JournalPage) onQuarterChange(selected string) {
        if p.subjectSelect.Selected != "" && p.subjectSelect.Selected != "Все предметы" {
                p.loadJournal()
        }
}

// ─── Dropdowns ────────────────────────────────────────────────

// UpdateDropdowns populates dropdowns with loaded data.
func (p *JournalPage) UpdateDropdowns() {
        classOpts := []string{"Все классы"}
        for _, g := range p.app.groupsData {
                name, _ := g["name"].(string)
                classOpts = append(classOpts, name)
        }
        p.classSelect.Options = classOpts
        p.classSelect.SetSelectedIndex(0)
        p.classSelect.Refresh()

        subjectOpts := []string{"Все предметы"}
        for _, s := range p.app.teacherSubjects {
                name, _ := s["subjectName"].(string)
                subjectOpts = append(subjectOpts, name)
        }
        p.subjectSelect.Options = subjectOpts
        p.subjectSelect.SetSelectedIndex(0)
        p.subjectSelect.Refresh()

        quarterOpts := []string{"Все четверти"}
        for _, q := range p.app.quartersData {
                name, _ := q["name"].(string)
                quarterOpts = append(quarterOpts, name)
        }
        p.quarterSelect.Options = quarterOpts
        p.quarterSelect.SetSelectedIndex(0)
        p.quarterSelect.Refresh()
}

// updateSubjectsForClass filters subject dropdown for the selected class.
func (p *JournalPage) updateSubjectsForClass(selected string) {
        if p.app.journalOptions == nil {
                return
        }

        var subjects []string
        if optionsMap, ok := p.app.journalOptions.(map[string]interface{}); ok {
                if groups, ok := optionsMap["groups"].([]interface{}); ok {
                        for _, g := range groups {
                                if gm, ok := g.(map[string]interface{}); ok {
                                        gname := fmt.Sprintf("%s%s", mapStr(gm, "number"), mapStr(gm, "name"))
                                        if gname == selected || selected == "Все классы" {
                                                if subs, ok := gm["subjects"].([]interface{}); ok {
                                                        for _, s := range subs {
                                                                if sm, ok := s.(map[string]interface{}); ok {
                                                                        name := mapStr(sm, "subjectName")
                                                                        if name != "" {
                                                                                subjects = append(subjects, name)
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        seen := make(map[string]bool)
        unique := []string{"Все предметы"}
        for _, s := range subjects {
                if !seen[s] {
                        seen[s] = true
                        unique = append(unique, s)
                }
        }

        p.subjectSelect.Options = unique
        p.subjectSelect.SetSelectedIndex(0)
        p.subjectSelect.Refresh()
}

// updateQuartersForClass filters quarter dropdown for the selected class.
// When a specific class is selected, shows only the quarters available for that class
// (with correct qpropId values from group-specific data).
func (p *JournalPage) updateQuartersForClass(selected string) {
        if p.app.journalOptions == nil {
                return
        }

        // If "Все классы" — show all global quarters
        if selected == "Все классы" || selected == "" {
                quarterOpts := []string{"Все четверти"}
                for _, q := range p.app.quartersData {
                        name, _ := q["name"].(string)
                        quarterOpts = append(quarterOpts, name)
                }
                p.quarterSelect.Options = quarterOpts
                p.quarterSelect.SetSelectedIndex(0)
                p.quarterSelect.Refresh()
                return
        }

        // Find the selected group and extract its quarters
        var quarterNames []string
        seen := make(map[string]bool)
        for _, g := range p.app.groupsData {
                if name, _ := g["name"].(string); name == selected {
                        if groupQuarters, ok := g["quarters"].([]interface{}); ok {
                                for _, q := range groupQuarters {
                                        if qm, ok := q.(map[string]interface{}); ok {
                                                qname := mapStr(qm, "name")
                                                if qname != "" && !seen[qname] {
                                                        seen[qname] = true
                                                        quarterNames = append(quarterNames, qname)
                                                }
                                        }
                                }
                        }
                        break
                }
        }

        // If no group-specific quarters found, fallback to global
        if len(quarterNames) == 0 {
                quarterOpts := []string{"Все четверти"}
                for _, q := range p.app.quartersData {
                        name, _ := q["name"].(string)
                        quarterOpts = append(quarterOpts, name)
                }
                p.quarterSelect.Options = quarterOpts
                p.quarterSelect.SetSelectedIndex(0)
                p.quarterSelect.Refresh()
                return
        }

        quarterOpts := []string{"Все четверти"}
        quarterOpts = append(quarterOpts, quarterNames...)
        p.quarterSelect.Options = quarterOpts
        p.quarterSelect.SetSelectedIndex(0)
        p.quarterSelect.Refresh()
}

// ─── Table helpers ────────────────────────────────────────────

func (p *JournalPage) tableRowCount() int {
        rows := 0
        for _, jd := range p.journalData {
                rows += 1 // title row
                rows += 1 // date header row
                rows += len(jd.students)
                rows += 1 // spacer
        }
        if rows == 0 {
                rows = 1
        }
        return rows
}

func (p *JournalPage) tableColCount() int {
        maxCols := 4
        for _, jd := range p.journalData {
                cols := 2 + len(jd.dates) + 2
                if cols > maxCols {
                        maxCols = cols
                }
        }
        return maxCols
}

func (p *JournalPage) tableCellUpdate(id widget.TableCellID, t *canvas.Text) {
        // Reset defaults for every cell (cells are reused)
        t.Alignment = fyne.TextAlignCenter
        t.Color = theme.ForegroundColor()
        t.TextStyle = fyne.TextStyle{}

        if len(p.journalData) == 0 {
                if id.Row == 0 && id.Col == 1 {
                        t.Text = "Выберите класс и предмет"
                        t.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
                } else {
                        t.Text = ""
                }
                return
        }

        rowIdx := 0
        for _, jd := range p.journalData {
                if id.Row == rowIdx {
                        p.titleCell(id.Col, jd, t)
                        return
                }
                rowIdx++

                if id.Row == rowIdx {
                        p.dateHeaderCell(id.Col, jd, t)
                        return
                }
                rowIdx++

                if id.Row < rowIdx+len(jd.students) {
                        si := id.Row - rowIdx
                        p.studentCell(id.Col, jd, jd.students[si], t)
                        return
                }
                rowIdx += len(jd.students)

                if id.Row == rowIdx {
                        t.Text = ""
                        return
                }
                rowIdx++
        }

        t.Text = ""
}

func (p *JournalPage) titleCell(col int, jd journalData, t *canvas.Text) {
        t.TextStyle = fyne.TextStyle{Bold: true}
        t.Color = theme.ForegroundColor()
        if col == 1 {
                t.Text = fmt.Sprintf("%s — %s (%s)", jd.groupName, jd.subjectName, jd.quarterName)
                t.Alignment = fyne.TextAlignLeading
        } else {
                t.Text = ""
        }
}

func (p *JournalPage) dateHeaderCell(col int, jd journalData, t *canvas.Text) {
        t.TextStyle = fyne.TextStyle{Bold: true}
        t.Color = theme.ForegroundColor()
        totalCols := 2 + len(jd.dates) + 2
        if col >= totalCols {
                t.Text = ""
                return
        }
        switch {
        case col == 0:
                t.Text = "N"
                t.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
        case col == 1:
                t.Text = "ФИО ученика"
                t.Alignment = fyne.TextAlignLeading
        case col >= 2 && col < 2+len(jd.dates):
                t.Text = jd.dates[col-2].shortStr
        case col == 2+len(jd.dates):
                t.Text = "Ср"
        case col == 2+len(jd.dates)+1:
                t.Text = "Диап"
        }
}

func (p *JournalPage) studentCell(col int, jd journalData, sr studentRow, t *canvas.Text) {
        totalCols := 2 + len(jd.dates) + 2
        if col >= totalCols {
                t.Text = ""
                return
        }

        t.TextStyle = fyne.TextStyle{}
        t.Color = theme.ForegroundColor()
        t.Alignment = fyne.TextAlignCenter

        switch {
        case col == 0:
                // Row number in grey
                for i, s := range jd.students {
                        if s.studentID == sr.studentID {
                                t.Text = fmt.Sprintf("%d", i+1)
                                t.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
                                return
                        }
                }
                t.Text = ""
        case col == 1:
                t.Text = sr.name
                t.Alignment = fyne.TextAlignLeading
        case col >= 2 && col < 2+len(jd.dates):
                dateID := jd.dates[col-2].dateID
                if display, ok := sr.marks[dateID]; ok {
                        t.Text = display
                        if val, okv := sr.markValues[dateID]; okv {
                                t.Color = gradeColor(val)
                                if val >= 9 {
                                        t.TextStyle = fyne.TextStyle{Bold: true}
                                } else if val <= 5 {
                                        t.TextStyle = fyne.TextStyle{Italic: true}
                                }
                        }
                } else {
                        t.Text = ""
                }
        case col == 2+len(jd.dates):
                if sr.gradeCount > 0 {
                        t.Text = fmt.Sprintf("%.1f", sr.avg)
                        // Color average: green if high, red if low
                        if sr.avg >= 8 {
                                t.Color = color.NRGBA{R: 0, G: 130, B: 50, A: 255}
                        } else if sr.avg >= 6 {
                                t.Color = theme.ForegroundColor()
                        } else {
                                t.Color = color.NRGBA{R: 200, G: 120, B: 0, A: 255}
                        }
                } else {
                        t.Text = "-"
                }
        case col == 2+len(jd.dates)+1:
                if sr.gradeCount > 0 {
                        t.Text = fmt.Sprintf("%d-%d", sr.min, sr.max)
                } else {
                        t.Text = "-"
                }
        }
}

// ─── Edit grade dialog ───────────────────────────────────────

func (p *JournalPage) showEditDialog(groupIdx, studentIdx int, dateID, dateStr string, sr studentRow) {
        oldDisplay := ""
        oldVal := 0
        markID := ""
        if display, ok := sr.marks[dateID]; ok {
                oldDisplay = display
                oldVal = sr.markValues[dateID]
                markID = sr.markIDs[dateID]
        }

        p.editInfo = &editMarkInfo{
                groupIdx:    groupIdx,
                studentIdx:  studentIdx,
                dateID:      dateID,
                dateStr:     dateStr,
                studentName: sr.name,
                oldValue:    oldDisplay,
                oldMarkVal:  oldVal,
                markID:      markID,
                qpropID:     p.journalData[groupIdx].qpropID,
                studentID:   sr.studentID,
        }

        p.editEntry = widget.NewEntry()
        if oldVal > 0 {
                p.editEntry.SetText(fmt.Sprintf("%d", oldVal))
        } else {
                p.editEntry.SetText("")
        }
        p.editEntry.PlaceHolder = "Оценка (1-10)"

        deleteBtn := widget.NewButton("Удалить", func() {
                p.deleteGrade()
        })
        if markID == "" {
                deleteBtn.Disable()
        }

        saveBtn := widget.NewButton("Сохранить", func() {
                p.saveGrade()
        })
        saveBtn.Importance = widget.HighImportance

        cancelBtn := widget.NewButton("Отмена", func() {
                if p.editDialog != nil {
                        p.editDialog.Hide()
                }
        })

        // Quick-set buttons 1-10 for inline-style editing
        var quickBtns []fyne.CanvasObject
        for i := 1; i <= 10; i++ {
                val := i
                btn := widget.NewButton(fmt.Sprintf("%d", val), func() {
                        p.editEntry.SetText(fmt.Sprintf("%d", val))
                        p.saveGrade()
                })
                btn.Importance = widget.MediumImportance
                quickBtns = append(quickBtns, btn)
        }

        content := container.NewVBox(
                widget.NewLabelWithStyle(fmt.Sprintf("%s — %s", sr.name, dateStr), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
                widget.NewSeparator(),
                container.NewHBox(
                        widget.NewLabel("Оценка:"),
                        p.editEntry,
                ),
                widget.NewSeparator(),
                widget.NewLabelWithStyle("Быстрый ввод:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
                container.NewGridWithColumns(5, quickBtns...),
                widget.NewSeparator(),
                container.NewHBox(saveBtn, deleteBtn, cancelBtn),
        )

        p.editDialog = dialog.NewCustom("Изменить оценку", "Закрыть", content, p.app.mainWindow)
        p.editDialog.Show()
}

func (p *JournalPage) saveGrade() {
        if p.editInfo == nil {
                return
        }

        val := parseInt(p.editEntry.Text)
        if val < 1 || val > 10 {
                p.app.LogMessage(fmt.Sprintf("Оценка должна быть от 1 до 10 (введено: %d)", val), "error")
                return
        }

        info := p.editInfo
        if p.editDialog != nil {
                p.editDialog.Hide()
        }

        p.setStatus(fmt.Sprintf("Сохранение оценки %d для %s...", val, info.studentName), "info")
        p.app.LogMessage(fmt.Sprintf("Изменение оценки: %s — %s -> %d", info.studentName, info.dateStr, val), "info")

        go func() {
                // If there's an existing mark, delete it first
                if info.markID != "" {
                        _, _ = p.app.apiClient.DeleteMark(info.markID)
                }

                result, err := p.app.apiClient.CreateMark(
                        info.studentID,
                        info.dateID,
                        val,
                        8, // mark_type_id
                        info.qpropID,
                        config.Signature,
                )

                fyne.Do(func() {
                        if err != nil {
                                p.setStatus(fmt.Sprintf("Ошибка: %v", err), "error")
                                p.app.LogMessage(fmt.Sprintf("Ошибка сохранения: %v", err), "error")
                        } else if resultMap, ok := result.(map[string]interface{}); ok {
                                if errMsg, exists := resultMap["error"]; exists && errMsg != nil {
                                        p.setStatus(fmt.Sprintf("Ошибка: %v", errMsg), "error")
                                        p.app.LogMessage(fmt.Sprintf("Ошибка API: %v", errMsg), "error")
                                } else {
                                        p.setStatus(fmt.Sprintf("Оценка %d сохранена для %s", val, info.studentName), "success")
                                        p.app.LogMessage(fmt.Sprintf("Оценка %d сохранена: %s (%s)", val, info.studentName, info.dateStr), "info")
                                        p.loadJournal()
                                }
                        } else {
                                p.setStatus(fmt.Sprintf("Оценка %d сохранена для %s", val, info.studentName), "success")
                                p.loadJournal()
                        }
                })
        }()
}

func (p *JournalPage) deleteGrade() {
        if p.editInfo == nil || p.editInfo.markID == "" {
                return
        }

        info := p.editInfo
        if p.editDialog != nil {
                p.editDialog.Hide()
        }

        p.setStatus(fmt.Sprintf("Удаление оценки для %s...", info.studentName), "info")
        p.app.LogMessage(fmt.Sprintf("Удаление оценки: %s (%s)", info.studentName, info.dateStr), "info")

        go func() {
                _, err := p.app.apiClient.DeleteMark(info.markID)

                fyne.Do(func() {
                        if err != nil {
                                p.setStatus(fmt.Sprintf("Ошибка удаления: %v", err), "error")
                                p.app.LogMessage(fmt.Sprintf("Ошибка удаления: %v", err), "error")
                        } else {
                                p.setStatus(fmt.Sprintf("Оценка удалена для %s", info.studentName), "success")
                                p.app.LogMessage(fmt.Sprintf("Оценка удалена: %s (%s)", info.studentName, info.dateStr), "info")
                                p.loadJournal()
                        }
                })
        }()
}

// ─── Student limits dialog (improved) ────────────────────────

func (p *JournalPage) showLimitsDialog() {
        if len(p.journalData) == 0 {
                p.app.LogMessage("Сначала загрузите журнал", "warning")
                return
        }

        // Collect all unique student names
        studentNames := make(map[string]bool)
        for _, jd := range p.journalData {
                for _, sr := range jd.students {
                        studentNames[sr.name] = true
                }
        }
        if len(studentNames) == 0 {
                p.app.LogMessage("Нет учеников", "warning")
                return
        }

        // Build a scrollable list of student entries
        var entries []fyne.CanvasObject
        studentEntries := make(map[string]*limitEntry)

        sortedNames := make([]string, 0, len(studentNames))
        for name := range studentNames {
                sortedNames = append(sortedNames, name)
        }
        sort.Strings(sortedNames)

        // ── Column headers ────────────────────────────────────
        headerRow := container.NewGridWithColumns(3,
                widget.NewLabelWithStyle("Ученик", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewLabelWithStyle("Мин", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
                widget.NewLabelWithStyle("Макс", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
        )
        entries = append(entries, headerRow)

        // ── "Set all" row ─────────────────────────────────────
        allMinEntry := widget.NewEntry()
        allMinEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
        allMaxEntry := widget.NewEntry()
        allMaxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))

        setAllBtn := widget.NewButton("Установить всем", func() {
                minV := parseInt(allMinEntry.Text)
                maxV := parseInt(allMaxEntry.Text)
                if minV < 1 {
                        minV = config.MinGrade
                }
                if maxV < 1 || maxV > 10 {
                        maxV = config.MaxGrade
                }
                if minV > maxV {
                        minV = maxV
                }
                for _, le := range studentEntries {
                        le.minEntry.SetText(fmt.Sprintf("%d", minV))
                        le.maxEntry.SetText(fmt.Sprintf("%d", maxV))
                }
        })

        setAllRow := container.NewGridWithColumns(4,
                container.NewHBox(widget.NewIcon(theme.ContentAddIcon()), widget.NewLabel("Установить всем")),
                allMinEntry,
                allMaxEntry,
                setAllBtn,
        )
        entries = append(entries, setAllRow)
        entries = append(entries, widget.NewSeparator())

        // ── Per-student rows ──────────────────────────────────
        for _, name := range sortedNames {
                le := &limitEntry{}
                le.minEntry = widget.NewEntry()
                le.maxEntry = widget.NewEntry()

                // Pre-fill with existing overrides or defaults
                if limits, ok := p.studentLimits[name]; ok {
                        le.minEntry.SetText(fmt.Sprintf("%d", limits.minGrade))
                        le.maxEntry.SetText(fmt.Sprintf("%d", limits.maxGrade))
                } else {
                        le.minEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
                        le.maxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))
                }

                studentEntries[name] = le

                row := container.NewGridWithColumns(3,
                        widget.NewLabel(name),
                        le.minEntry,
                        le.maxEntry,
                )
                entries = append(entries, row)
        }

        listContent := container.NewVBox(entries...)
        scrollList := container.NewVScroll(listContent)
        scrollList.SetMinSize(fyne.NewSize(450, 350))

        saveBtn := widget.NewButton("Сохранить", func() {
                for name, le := range studentEntries {
                        minV := parseInt(le.minEntry.Text)
                        maxV := parseInt(le.maxEntry.Text)
                        if minV < 1 {
                                minV = config.MinGrade
                        }
                        if maxV < 1 || maxV > 10 {
                                maxV = config.MaxGrade
                        }
                        if minV > maxV {
                                minV = maxV
                        }
                        p.studentLimits[name] = &studentGradeLimits{
                                studentName: name,
                                minGrade:    minV,
                                maxGrade:    maxV,
                        }
                }
                p.app.LogMessage(fmt.Sprintf("Сохранены пределы оценок для %d учеников", len(studentEntries)), "info")
                p.setStatus(fmt.Sprintf("Пределы оценок сохранены для %d учеников", len(studentEntries)), "success")
        })
        saveBtn.Importance = widget.HighImportance

        // Random fill button inside limits dialog
        randomFillBtn := widget.NewButton("Рандом заполнить", func() {
                p.fillRandomGrades()
        })

        closeBtn := widget.NewButton("Закрыть", func() {
                // dialog closes itself
        })

        descLabel := widget.NewLabelWithStyle(
                "Установите мин/макс оценки для каждого ученика.\nЭти пределы используются при автозаполнении.",
                fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

        bottomRow := container.NewHBox(saveBtn, randomFillBtn, closeBtn)

        dialogContent := container.NewBorder(
                container.NewVBox(descLabel, widget.NewSeparator()),
                bottomRow,
                nil, nil,
                scrollList,
        )

        d := dialog.NewCustom("Пределы оценок по ученикам", "Закрыть", dialogContent, p.app.mainWindow)
        d.Resize(fyne.NewSize(500, 500))
        d.Show()
}

// limitEntry holds min/max entry widgets for a student.
type limitEntry struct {
        minEntry *widget.Entry
        maxEntry *widget.Entry
}

// ─── Random fill dialog ──────────────────────────────────────

// showRandomFillDialog opens a dialog for configuring random grade generation.
// It supports both per-student and all-students modes:
//   - Select a student from dropdown → set min/max for that student
//   - "Заполнить выбранного" → fills only the selected student
//   - "Заполнить всех" → fills all students (existing behavior)
func (p *JournalPage) showRandomFillDialog() {
        if len(p.journalData) == 0 {
                p.app.LogMessage("Сначала загрузите журнал", "warning")
                return
        }

        // Collect all unique student names
        studentNames := make(map[string]bool)
        for _, jd := range p.journalData {
                for _, sr := range jd.students {
                        studentNames[sr.name] = true
                }
        }
        if len(studentNames) == 0 {
                p.app.LogMessage("Нет учеников", "warning")
                return
        }

        sortedNames := make([]string, 0, len(studentNames))
        for name := range studentNames {
                sortedNames = append(sortedNames, name)
        }
        sort.Strings(sortedNames)

        // ── Student selector dropdown ──────────────────────────
        studentSelectOpts := append([]string{"<Все ученики>"}, sortedNames...)
        studentSelect := widget.NewSelect(studentSelectOpts, func(s string) {})
        studentSelect.PlaceHolder = "Выберите ученика"

        // Pre-select the currently highlighted student from the table
        preselectedName := p.currentStudentName
        if preselectedName != "" {
                for i, name := range studentSelectOpts {
                        if name == preselectedName {
                                studentSelect.SetSelectedIndex(i)
                                break
                        }
                }
        } else {
                studentSelect.SetSelectedIndex(0)
        }

        // ── Min/Max for selected student ───────────────────────
        minEntry := widget.NewEntry()
        minEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
        maxEntry := widget.NewEntry()
        maxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))

        // When student selection changes, update min/max from saved limits
        studentSelect.OnChanged = func(selected string) {
                if selected == "<Все ученики>" {
                        minEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
                        maxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))
                        return
                }
                if limits, ok := p.studentLimits[selected]; ok {
                        minEntry.SetText(fmt.Sprintf("%d", limits.minGrade))
                        maxEntry.SetText(fmt.Sprintf("%d", limits.maxGrade))
                } else {
                        minEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
                        maxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))
                }
        }
        // Trigger initial load for pre-selected student
        if preselectedName != "" {
                studentSelect.OnChanged(preselectedName)
        }

        // ── Per-student full list (scrollable) ─────────────────
        studentEntries := make(map[string]*limitEntry)
        var listEntries []fyne.CanvasObject

        headerRow := container.NewGridWithColumns(3,
                widget.NewLabelWithStyle("Ученик", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewLabelWithStyle("Мин", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
                widget.NewLabelWithStyle("Макс", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
        )
        listEntries = append(listEntries, headerRow)

        // "Set all" row
        allMinEntry := widget.NewEntry()
        allMinEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
        allMaxEntry := widget.NewEntry()
        allMaxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))

        setAllBtn := widget.NewButton("Всем", func() {
                minV := parseInt(allMinEntry.Text)
                maxV := parseInt(allMaxEntry.Text)
                if minV < 1 {
                        minV = config.MinGrade
                }
                if maxV < 1 || maxV > 10 {
                        maxV = config.MaxGrade
                }
                if minV > maxV {
                        minV = maxV
                }
                for _, le := range studentEntries {
                        le.minEntry.SetText(fmt.Sprintf("%d", minV))
                        le.maxEntry.SetText(fmt.Sprintf("%d", maxV))
                }
        })

        setAllRow := container.NewGridWithColumns(4,
                container.NewHBox(widget.NewIcon(theme.ContentAddIcon()), widget.NewLabel("Установить всем")),
                allMinEntry,
                allMaxEntry,
                setAllBtn,
        )
        listEntries = append(listEntries, setAllRow)
        listEntries = append(listEntries, widget.NewSeparator())

        // Per-student rows
        for _, name := range sortedNames {
                le := &limitEntry{}
                le.minEntry = widget.NewEntry()
                le.maxEntry = widget.NewEntry()

                // Pre-fill with existing limits or defaults
                if limits, ok := p.studentLimits[name]; ok {
                        le.minEntry.SetText(fmt.Sprintf("%d", limits.minGrade))
                        le.maxEntry.SetText(fmt.Sprintf("%d", limits.maxGrade))
                } else {
                        le.minEntry.SetText(fmt.Sprintf("%d", config.MinGrade))
                        le.maxEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))
                }

                studentEntries[name] = le

                row := container.NewGridWithColumns(3,
                        widget.NewLabel(name),
                        le.minEntry,
                        le.maxEntry,
                )
                listEntries = append(listEntries, row)
        }

        listContent := container.NewVBox(listEntries...)
        scrollList := container.NewVScroll(listContent)
        scrollList.SetMinSize(fyne.NewSize(450, 280))

        // ── Save limits helper ─────────────────────────────────
        saveAllLimits := func() {
                for name, le := range studentEntries {
                        minV := parseInt(le.minEntry.Text)
                        maxV := parseInt(le.maxEntry.Text)
                        if minV < 1 {
                                minV = config.MinGrade
                        }
                        if maxV < 1 || maxV > 10 {
                                maxV = config.MaxGrade
                        }
                        if minV > maxV {
                                minV = maxV
                        }
                        p.studentLimits[name] = &studentGradeLimits{
                                studentName: name,
                                minGrade:    minV,
                                maxGrade:    maxV,
                        }
                }
        }

        // Save the selected student's min/max from the top entries
        saveSelectedLimits := func() {
                selected := studentSelect.Selected
                if selected == "" || selected == "<Все ученики>" {
                        return
                }
                minV := parseInt(minEntry.Text)
                maxV := parseInt(maxEntry.Text)
                if minV < 1 {
                        minV = config.MinGrade
                }
                if maxV < 1 || maxV > 10 {
                        maxV = config.MaxGrade
                }
                if minV > maxV {
                        minV = maxV
                }
                p.studentLimits[selected] = &studentGradeLimits{
                        studentName: selected,
                        minGrade:    minV,
                        maxGrade:    maxV,
                }
                // Also sync to the per-student entry in the list
                if le, ok := studentEntries[selected]; ok {
                        le.minEntry.SetText(fmt.Sprintf("%d", minV))
                        le.maxEntry.SetText(fmt.Sprintf("%d", maxV))
                }
        }

        // ── Fill selected student button ───────────────────────
        fillSelectedBtn := widget.NewButton("Заполнить выбранного", func() {
                selected := studentSelect.Selected
                if selected == "" || selected == "<Все ученики>" {
                        p.app.LogMessage("Выберите конкретного ученика", "warning")
                        return
                }
                // Save limits first (selected student + all from list)
                saveSelectedLimits()
                saveAllLimits()
                p.fillRandomGradesForStudent(selected)
        })
        fillSelectedBtn.Importance = widget.HighImportance

        // ── Fill all students button ───────────────────────────
        fillAllBtn := widget.NewButton("Заполнить всех", func() {
                saveSelectedLimits()
                saveAllLimits()
                p.fillRandomGrades()
        })

        closeBtn := widget.NewButton("Закрыть", func() {
                // dialog closes itself
        })

        descLabel := widget.NewLabelWithStyle(
                "Выберите ученика → установите мин/макс → заполните.\n"+
                        "Или заполните всех учеников сразу.",
                fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

        // ── Top section: student selector + min/max ────────────
        selectorSection := container.NewVBox(
                container.NewGridWithColumns(2,
                        widget.NewLabelWithStyle("Ученик:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
                        studentSelect,
                ),
                container.NewGridWithColumns(4,
                        widget.NewLabelWithStyle("Мин:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
                        minEntry,
                        widget.NewLabelWithStyle("Макс:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
                        maxEntry,
                ),
                widget.NewSeparator(),
        )

        bottomRow := container.NewHBox(fillSelectedBtn, fillAllBtn, closeBtn)

        dialogContent := container.NewBorder(
                container.NewVBox(descLabel, widget.NewSeparator(), selectorSection),
                bottomRow,
                nil, nil,
                scrollList,
        )

        d := dialog.NewCustom("Рандомное заполнение", "Закрыть", dialogContent, p.app.mainWindow)
        d.Resize(fyne.NewSize(520, 550))
        d.Show()
}

// fillRandomGrades fills empty cells with random grades using student limits.
func (p *JournalPage) fillRandomGrades() {
        if len(p.journalData) == 0 {
                p.app.LogMessage("Сначала загрузите журнал", "warning")
                return
        }

        p.setStatus("Рандомное заполнение оценок...", "info")
        p.app.LogMessage("Начало рандомного заполнения оценок", "info")

        go func() {
                totalCreated := 0
                totalErrors := 0

                for _, jd := range p.journalData {
                        for _, sr := range jd.students {
                                // Get student min/max limits
                                minGrade := config.MinGrade
                                maxGrade := config.MaxGrade
                                if limits, ok := p.studentLimits[sr.name]; ok {
                                        minGrade = limits.minGrade
                                        maxGrade = limits.maxGrade
                                }
                                if minGrade < 1 {
                                        minGrade = 1
                                }
                                if maxGrade > 10 {
                                        maxGrade = 10
                                }
                                if minGrade > maxGrade {
                                        minGrade = maxGrade
                                }

                                for _, dc := range jd.dates {
                                        // Only fill empty cells
                                        if _, hasMark := sr.marks[dc.dateID]; hasMark {
                                                continue
                                        }

                                        // Generate random grade: rand.Intn(max-min+1) + min
                                        grade := rand.Intn(maxGrade-minGrade+1) + minGrade

                                        _, err := p.app.apiClient.CreateMark(
                                                sr.studentID,
                                                dc.dateID,
                                                grade,
                                                8, // mark_type_id
                                                jd.qpropID,
                                                config.Signature,
                                        )

                                        if err != nil {
                                                totalErrors++
                                                p.app.LogMessage(fmt.Sprintf("Ошибка рандомной оценки: %s — %s: %v", sr.name, dc.dateStr, err), "error")
                                        } else {
                                                totalCreated++
                                        }

                                        // Small delay to avoid overwhelming the API
                                        time.Sleep(50 * time.Millisecond)
                                }
                        }
                }

                fyne.Do(func() {
                        if totalErrors > 0 {
                                p.setStatus(fmt.Sprintf("Заполнено: %d оценок, ошибок: %d", totalCreated, totalErrors), "error")
                        } else {
                                p.setStatus(fmt.Sprintf("Заполнено: %d случайных оценок", totalCreated), "success")
                        }
                        p.app.LogMessage(fmt.Sprintf("Рандомное заполнение завершено: %d оценок, %d ошибок", totalCreated, totalErrors), "info")
                        p.loadJournal()
                })
        }()
}

// fillRandomGradesForStudent fills empty cells with random grades for a SINGLE student only.
func (p *JournalPage) fillRandomGradesForStudent(studentName string) {
        if len(p.journalData) == 0 {
                p.app.LogMessage("Сначала загрузите журнал", "warning")
                return
        }
        if studentName == "" {
                p.app.LogMessage("Выберите ученика в таблице", "warning")
                return
        }

        p.setStatus(fmt.Sprintf("Рандомное заполнение для %s...", studentName), "info")
        p.app.LogMessage(fmt.Sprintf("Начало рандомного заполнения для ученика: %s", studentName), "info")

        go func() {
                totalCreated := 0
                totalErrors := 0

                for _, jd := range p.journalData {
                        for _, sr := range jd.students {
                                // Only process the selected student
                                if sr.name != studentName {
                                        continue
                                }

                                // Get student min/max limits
                                minGrade := config.MinGrade
                                maxGrade := config.MaxGrade
                                if limits, ok := p.studentLimits[sr.name]; ok {
                                        minGrade = limits.minGrade
                                        maxGrade = limits.maxGrade
                                }
                                if minGrade < 1 {
                                        minGrade = 1
                                }
                                if maxGrade > 10 {
                                        maxGrade = 10
                                }
                                if minGrade > maxGrade {
                                        minGrade = maxGrade
                                }

                                for _, dc := range jd.dates {
                                        // Only fill empty cells
                                        if _, hasMark := sr.marks[dc.dateID]; hasMark {
                                                continue
                                        }

                                        // Generate random grade: rand.Intn(max-min+1) + min
                                        grade := rand.Intn(maxGrade-minGrade+1) + minGrade

                                        _, err := p.app.apiClient.CreateMark(
                                                sr.studentID,
                                                dc.dateID,
                                                grade,
                                                8, // mark_type_id
                                                jd.qpropID,
                                                config.Signature,
                                        )

                                        if err != nil {
                                                totalErrors++
                                                p.app.LogMessage(fmt.Sprintf("Ошибка рандомной оценки: %s — %s: %v", sr.name, dc.dateStr, err), "error")
                                        } else {
                                                totalCreated++
                                        }

                                        // Small delay to avoid overwhelming the API
                                        time.Sleep(50 * time.Millisecond)
                                }
                        }
                }

                fyne.Do(func() {
                        if totalErrors > 0 {
                                p.setStatus(fmt.Sprintf("Заполнено для %s: %d оценок, ошибок: %d", studentName, totalCreated, totalErrors), "error")
                        } else {
                                p.setStatus(fmt.Sprintf("Заполнено для %s: %d случайных оценок", studentName, totalCreated), "success")
                        }
                        p.app.LogMessage(fmt.Sprintf("Рандомное заполнение для %s завершено: %d оценок, %d ошибок", studentName, totalCreated, totalErrors), "info")
                        p.loadJournal()
                })
        }()
}

// ─── Cell click → show student detail ────────────────────────

func (p *JournalPage) showStudentDetail(sr studentRow, jd journalData) {
        p.selectedStudent = sr.name
        p.detailCard.SetTitle(sr.name)

        var lines []string
        lines = append(lines, fmt.Sprintf("Ученик: %s", sr.name))
        lines = append(lines, fmt.Sprintf("Класс: %s  |  Предмет: %s  |  %s", jd.groupName, jd.subjectName, jd.quarterName))
        lines = append(lines, strings.Repeat("-", 40))

        if sr.gradeCount > 0 {
                lines = append(lines, fmt.Sprintf("Средний балл: %.1f", sr.avg))
                lines = append(lines, fmt.Sprintf("Минимум: %d  |  Максимум: %d", sr.min, sr.max))
                lines = append(lines, fmt.Sprintf("Разброс: %d  |  Оценок: %d  |  Пропусков: %d", sr.max-sr.min, sr.gradeCount, sr.missing))
                lines = append(lines, "")
                lines = append(lines, "Визуальный разброс:")
                lines = append(lines, makeVisualSpread(sr.min, sr.max, sr.avg, 10))
                lines = append(lines, "")

                grades := make([]int, 0, sr.gradeCount)
                for _, v := range sr.markValues {
                        if v > 0 {
                                grades = append(grades, v)
                        }
                }
                if len(grades) > 0 {
                        lines = append(lines, "Распределение оценок:")
                        lines = append(lines, makeDistribution(grades))
                }
        } else {
                lines = append(lines, "Нет оценок")
        }

        // Show per-student limits if set
        if limits, ok := p.studentLimits[sr.name]; ok {
                lines = append(lines, "")
                lines = append(lines, fmt.Sprintf("Пределы автозаполнения: %d - %d", limits.minGrade, limits.maxGrade))
        }

        // Signature in student detail
        lines = append(lines, "")
        lines = append(lines, strings.Repeat("-", 40))
        lines = append(lines, "by 4code")

        p.studentDetail.SetText(strings.Join(lines, "\n"))
        p.detailPanel.Objects = []fyne.CanvasObject{p.detailCard}
        p.detailPanel.Show()
}

func (p *JournalPage) hideDetail() {
        p.detailPanel.Hide()
        p.selectedStudent = ""
}

// ─── Load journal data ────────────────────────────────────────

func (p *JournalPage) loadJournal() {
        classSelected := p.classSelect.Selected
        subjectSelected := p.subjectSelect.Selected
        quarterSelected := p.quarterSelect.Selected

        if subjectSelected == "" || subjectSelected == "Все предметы" {
                p.setStatus("Выберите предмет", "warning")
                return
        }

        p.setStatus("Загрузка журнала...", "info")
        p.app.LogMessage(fmt.Sprintf("Загрузка журнала: %s / %s / %s", classSelected, subjectSelected, quarterSelected), "info")
        p.hideDetail()

        go func() {
                groups := p.getSelectedGroups(classSelected)
                if len(groups) == 0 {
                        fyne.Do(func() {
                                p.setStatus("Не выбран класс", "warning")
                        })
                        return
                }

                var allData []journalData

                for _, group := range groups {
                        groupID := mapInt(group, "id")
                        groupName := mapStr(group, "name")

                        subjects := p.getSubjectsForGroup(group, subjectSelected)
                        quarters := p.getSelectedQuarters(group, quarterSelected)

                        p.app.LogMessage(fmt.Sprintf("  Группа %s: %d предметов, %d четвертей (выбрана: %q)", groupName, len(subjects), len(quarters), quarterSelected), "info")

                        for _, subject := range subjects {
                                subjectID := mapInt(subject, "subjectId")
                                subjectName := mapStr(subject, "subjectName")

                                for _, quarter := range quarters {
                                        qpropID := mapInt(quarter, "qpropId")
                                        quarterName := mapStr(quarter, "name")

                                        datesData, err := p.app.apiClient.GetJournalDates(groupID, subjectID, qpropID)
                                        if err != nil {
                                                continue
                                        }
                                        days := engine.ExtractDays(datesData)
                                        if len(days) == 0 {
                                                continue
                                        }

                                        studentsData, err := p.app.apiClient.GetJournalStudents(groupID, subjectID, qpropID)
                                        if err != nil {
                                                continue
                                        }
                                        students := engine.ExtractStudents(studentsData)
                                        if len(students) == 0 {
                                                continue
                                        }

                                        var dateCols []dateCol
                                        for _, day := range days {
                                                dateID := mapStr(day, "assignmentDateId")
                                                dateStr := mapStr(day, "assignmentDate")
                                                shortStr := dateStr
                                                if len(dateStr) >= 10 {
                                                        shortStr = dateStr[5:10]
                                                }
                                                dateCols = append(dateCols, dateCol{
                                                        dateID:   dateID,
                                                        dateStr:  dateStr,
                                                        shortStr: shortStr,
                                                })
                                        }

                                        var studentRows []studentRow
                                        for _, student := range students {
                                                studentID := mapInt(student, "studentId")
                                                studentName := fmt.Sprintf("%s %s", mapStr(student, "lastName"), mapStr(student, "firstName"))

                                                existingMarks := engine.ExtractExistingMarks(student)
                                                markDetails := extractMarkDetails(student)

                                                sr := studentRow{
                                                        studentID:  studentID,
                                                        name:       studentName,
                                                        marks:      make(map[string]string),
                                                        markValues: make(map[string]int),
                                                        markIDs:    make(map[string]string),
                                                }

                                                // Apply per-student limits if set
                                                if limits, ok := p.studentLimits[studentName]; ok {
                                                        sr.minOverride = limits.minGrade
                                                        sr.maxOverride = limits.maxGrade
                                                }

                                                var grades []int
                                                for _, dc := range dateCols {
                                                        if _, has := existingMarks[dc.dateID]; has {
                                                                if mi, ok := markDetails[dc.dateID]; ok {
                                                                        display := engine.ParseGradeDisplay(mi.shortName, mi.markValue)
                                                                        sr.marks[dc.dateID] = display
                                                                        sr.markValues[dc.dateID] = mi.markValue
                                                                        sr.markIDs[dc.dateID] = mi.markID
                                                                        if mi.markValue > 0 {
                                                                                grades = append(grades, mi.markValue)
                                                                        }
                                                                } else {
                                                                        sr.marks[dc.dateID] = "+"
                                                                }
                                                        } else {
                                                                sr.missing++
                                                        }
                                                }

                                                sr.gradeCount = len(grades)
                                                if len(grades) > 0 {
                                                        sr.min = grades[0]
                                                        sr.max = grades[0]
                                                        sum := 0
                                                        for _, g := range grades {
                                                                sum += g
                                                                if g < sr.min {
                                                                        sr.min = g
                                                                }
                                                                if g > sr.max {
                                                                        sr.max = g
                                                                }
                                                        }
                                                        sr.avg = float64(sum) / float64(len(grades))
                                                }

                                                studentRows = append(studentRows, sr)
                                        }

                                        allData = append(allData, journalData{
                                                groupName:   groupName,
                                                groupID:     groupID,
                                                subjectID:   subjectID,
                                                qpropID:     qpropID,
                                                subjectName: subjectName,
                                                quarterName: quarterName,
                                                dates:       dateCols,
                                                students:    studentRows,
                                        })
                                }
                        }
                }

                fyne.Do(func() {
                        p.journalData = allData
                        p.journalTable.Refresh()

                        for _, jd := range allData {
                                for i := range jd.dates {
                                        p.journalTable.SetColumnWidth(2+i, 45)
                                }
                                cols := len(jd.dates)
                                p.journalTable.SetColumnWidth(2+cols, 50)
                                p.journalTable.SetColumnWidth(2+cols+1, 55)
                        }

                        if len(allData) == 0 {
                                p.setStatus("Нет данных для отображения", "warning")
                        } else {
                                totalStudents := 0
                                for _, jd := range allData {
                                        totalStudents += len(jd.students)
                                }
                                p.setStatus(fmt.Sprintf("Загружено: %d групп, %d учеников",
                                        len(allData), totalStudents), "success")
                        }
                })
        }()
}

// ─── Helper functions ─────────────────────────────────────────

// markInfo holds grade display information.
type markInfo struct {
        shortName string
        markValue int
        markID    string
}

func extractMarkDetails(student map[string]interface{}) map[string]markInfo {
        result := make(map[string]markInfo)
        if subjectMarks, ok := student["subjectMarks"].([]interface{}); ok {
                for _, m := range subjectMarks {
                        if mm, ok := m.(map[string]interface{}); ok {
                                dateID := mapStr(mm, "assignmentDateId")
                                shortName := mapStr(mm, "shortName")
                                markValue := mapInt(mm, "mark")
                                markID := mapStr(mm, "id")
                                result[dateID] = markInfo{shortName: shortName, markValue: markValue, markID: markID}
                        }
                }
        }
        return result
}

// makeVisualSpread creates a visual text-based spread indicator.
func makeVisualSpread(min, max int, avg float64, scale int) string {
        if max < min || scale <= 0 {
                return "[---]"
        }
        barWidth := scale * 2
        bar := make([]rune, barWidth)
        for i := range bar {
                bar[i] = '.'
        }
        for i := 0; i < barWidth; i++ {
                pos := float64(i) / float64(barWidth-1) * float64(max)
                if pos >= float64(min) && pos <= float64(max) {
                        bar[i] = '='
                }
        }
        avgPos := int((avg - 1) / float64(max) * float64(barWidth-1))
        if avgPos < 0 {
                avgPos = 0
        }
        if avgPos >= barWidth {
                avgPos = barWidth - 1
        }
        bar[avgPos] = 'o'
        minPos := int(float64(min-1) / float64(max) * float64(barWidth-1))
        maxPos := int(float64(max-1) / float64(max) * float64(barWidth-1))
        if minPos >= 0 && minPos < barWidth {
                bar[minPos] = '['
        }
        if maxPos >= 0 && maxPos < barWidth {
                bar[maxPos] = ']'
        }
        return fmt.Sprintf("%d %s %d", min, string(bar), max)
}

// makeDistribution creates a text-based histogram of grade distribution.
func makeDistribution(grades []int) string {
        if len(grades) == 0 {
                return ""
        }
        counts := make(map[int]int)
        for _, g := range grades {
                counts[g]++
        }
        maxCount := 0
        for _, c := range counts {
                if c > maxCount {
                        maxCount = c
                }
        }
        var gradeValues []int
        for g := range counts {
                gradeValues = append(gradeValues, g)
        }
        sort.Ints(gradeValues)

        var parts []string
        for _, g := range gradeValues {
                c := counts[g]
                barLen := 0
                if maxCount > 0 {
                        barLen = int(float64(c) / float64(maxCount) * 8)
                }
                bar := strings.Repeat("#", barLen)
                parts = append(parts, fmt.Sprintf("%d:%s%d", g, bar, c))
        }
        return strings.Join(parts, " ")
}

// ─── Data selection helpers ───────────────────────────────────

func (p *JournalPage) getSelectedGroups(selected string) []map[string]interface{} {
        if selected == "Все классы" || selected == "" {
                return p.app.groupsData
        }
        var result []map[string]interface{}
        for _, g := range p.app.groupsData {
                if name, _ := g["name"].(string); name == selected {
                        result = append(result, g)
                }
        }
        return result
}

func (p *JournalPage) getSubjectsForGroup(group map[string]interface{}, selected string) []map[string]interface{} {
        var result []map[string]interface{}
        if subjects, ok := group["subjects"].([]interface{}); ok {
                for _, s := range subjects {
                        if sm, ok := s.(map[string]interface{}); ok {
                                if selected == "Все предметы" || selected == "" {
                                        result = append(result, sm)
                                } else if mapStr(sm, "subjectName") == selected {
                                        result = append(result, sm)
                                }
                        }
                }
        }
        return result
}

func (p *JournalPage) getSelectedQuarters(group map[string]interface{}, selected string) []map[string]interface{} {
        // If the group has group-specific quarters, use those (they have correct qpropId)
        if groupQuarters, ok := group["quarters"].([]interface{}); ok && len(groupQuarters) > 0 {
                var groupQData []map[string]interface{}
                for _, q := range groupQuarters {
                        if qm, ok := q.(map[string]interface{}); ok {
                                groupQData = append(groupQData, qm)
                        }
                }
                if selected == "Все четверти" || selected == "" {
                        return groupQData
                }
                var result []map[string]interface{}
                for _, q := range groupQData {
                        qname, _ := q["name"].(string)
                        p.app.LogMessage(fmt.Sprintf("    Сравнение четверти: %q == %q (qpropId=%v)", qname, selected, q["qpropId"]), "info")
                        if qname == selected {
                                result = append(result, q)
                        }
                }
                if len(result) == 0 {
                        p.app.LogMessage(fmt.Sprintf("    ВНИМАНИЕ: четверть %q не найдена в группе! Доступные: %v", selected, quarterNames(groupQData)), "warning")
                }
                return result
        }
        // Fallback to global quartersData
        p.app.LogMessage(fmt.Sprintf("    Группа не имеет group-specific quarters, используем глобальные (%d шт)", len(p.app.quartersData)), "info")
        if selected == "Все четверти" || selected == "" {
                return p.app.quartersData
        }
        var result []map[string]interface{}
        for _, q := range p.app.quartersData {
                if name, _ := q["name"].(string); name == selected {
                        result = append(result, q)
                }
        }
        if len(result) == 0 {
                p.app.LogMessage(fmt.Sprintf("    ВНИМАНИЕ: четверть %q не найдена в глобальных данных! Доступные: %v", selected, quarterNames(p.app.quartersData)), "warning")
        }
        return result
}

// quarterNames returns a list of quarter names for debugging.
func quarterNames(quarters []map[string]interface{}) []string {
        names := make([]string, 0, len(quarters))
        for _, q := range quarters {
                if name, ok := q["name"].(string); ok {
                        names = append(names, name)
                }
        }
        return names
}

// GetStudentLimits returns the per-student grade limits for auto-grade.
// Used by the auto-grade page to apply custom min/max per student.
func (p *JournalPage) GetStudentLimits() map[string]*studentGradeLimits {
        return p.studentLimits
}
