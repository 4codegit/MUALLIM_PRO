package ui

import (
        "fmt"
        "image/color"
        "strconv"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

// ------------------------------------------
// COLOR PALETTE — Modern Dark-Accent Design
// ------------------------------------------

var (
        colorNavBG   = color.NRGBA{R: 15, G: 23, B: 42, A: 255}  // Slate-900
        colorCardBlue = color.NRGBA{R: 37, G: 99, B: 235, A: 255} // Blue-600
        colorAccent  = color.NRGBA{R: 56, G: 189, B: 248, A: 255} // Sky-400
        colorSurface = color.NRGBA{R: 248, G: 250, B: 252, A: 255} // Slate-50
)

// ------------------------------------------
// DASHBOARD
// ------------------------------------------

type Dashboard struct {
        controller  Controller
        container   *fyne.Container
        statusLabel *widget.Label

        // Navigation state
        contentStack *fyne.Container
        currentPage  fyne.CanvasObject

        // Bottom tab bar
        tabBar      *fyne.Container
        activeTab   int // 0=Журнал, 1=Темы, 2=Дневник, 3=Итоговые
        tabButtons  []*widget.Button

        // Filters state
        classSel   *widget.Select
        subjectSel *widget.Select
        quarterSel *widget.Select
        refreshBtn *widget.Button

        selectedGroup   *client.JournalGroup
        selectedSubject *client.Subject
        selectedQuarter *client.Quarter

        journalOpts *client.JournalOptions
        dates       []client.Day
        students    []client.Student

        // Tab content objects
        gradesTable     *widget.Table
        gradesContainer *fyne.Container

        // Journal selection state
        selectedCell *widget.TableCellID
        deleteBtn    *widget.Button

        // Tab objects
        topicsTab      *TopicsTab
        diariesTab     *DiariesTab
        finalGradesTab *FinalGradesTab
}

func NewDashboard(c Controller) *Dashboard {
        d := &Dashboard{
                controller:  c,
                statusLabel: widget.NewLabel(""),
        }
        d.buildUI()
        go d.loadJournalOptions()
        return d
}

func (d *Dashboard) Container() fyne.CanvasObject {
        return d.container
}

// ------------------------------------------
// UI BUILD
// ------------------------------------------

func (d *Dashboard) buildUI() {
        header := d.buildHeader()
        filters := d.buildFilters()

        d.gradesContainer = container.NewStack(
                widget.NewLabelWithStyle("Выберите фильтры для загрузки оценок", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
        )

        d.topicsTab = NewTopicsTab(d.controller)
        d.diariesTab = NewDiariesTab(d.controller)
        d.finalGradesTab = NewFinalGradesTab(d.controller)

        // Start on journal page
        d.currentPage = d.gradesContainer

        // Build bottom tab bar
        d.tabBar = d.buildTabBar()

        topSection := container.NewVBox(header, filters, widget.NewSeparator())

        d.contentStack = container.NewStack(d.currentPage)

        d.container = container.NewBorder(
                topSection,
                container.NewVBox(widget.NewSeparator(), d.tabBar),
                nil,
                nil,
                d.contentStack,
        )
}

// ------------------------------------------
// HEADER — Sleek minimal navbar
// ------------------------------------------

func (d *Dashboard) buildHeader() *fyne.Container {
        apiClient := d.controller.GetClient()

        userText := ""
        if apiClient.UserInfo != nil {
                userText = fmt.Sprintf("%s %s", apiClient.UserInfo.LastName, apiClient.UserInfo.FirstName)
        }

        roleText := apiClient.Role
        schoolName := ""
        for _, sch := range apiClient.Schools {
                if sch.SchoolID == apiClient.SchoolID {
                        schoolName = sch.SchoolName
                        break
                }
        }

        appTitle := canvas.NewText("eDonish Auto", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
        appTitle.TextStyle = fyne.TextStyle{Bold: true}
        appTitle.TextSize = 18

        versionTag := canvas.NewText("v5.3", colorAccent)
        versionTag.TextSize = 11
        versionTag.TextStyle = fyne.TextStyle{Bold: true}

        userLabel := canvas.NewText(userText, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
        userLabel.TextStyle = fyne.TextStyle{Bold: true}
        userLabel.TextSize = 14

        roleLabel := canvas.NewText(fmt.Sprintf("%s — %s", roleText, schoolName), color.NRGBA{R: 160, G: 174, B: 192, A: 255})
        roleLabel.TextSize = 11

        logoutBtn := widget.NewButton("Выйти", d.controller.Logout)
        logoutBtn.Importance = widget.DangerImportance

        leftBox := container.NewHBox(appTitle, versionTag)
        userInfoBox := container.NewVBox(userLabel, roleLabel)
        rightBox := container.NewHBox(userInfoBox, logoutBtn)

        bg := canvas.NewRectangle(colorNavBG)
        bg.SetMinSize(fyne.NewSize(0, 52))

        navbar := container.NewBorder(nil, nil, leftBox, rightBox, bg)
        return container.NewStack(bg, container.NewPadded(navbar))
}

// ------------------------------------------
// FILTERS — Clean row
// ------------------------------------------

func (d *Dashboard) buildFilters() *fyne.Container {
        d.classSel = widget.NewSelect([]string{}, d.onClassSelected)
        d.classSel.PlaceHolder = "Класс..."

        d.subjectSel = widget.NewSelect([]string{}, d.onSubjectSelected)
        d.subjectSel.PlaceHolder = "Предмет..."

        d.quarterSel = widget.NewSelect([]string{}, d.onQuarterSelected)
        d.quarterSel.PlaceHolder = "Четверть..."

        d.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go d.loadData()
        })
        d.refreshBtn.Disable()

        d.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
                d.deleteSelectedCell()
        })
        d.deleteBtn.Importance = widget.DangerImportance
        d.deleteBtn.Disable()

        return container.NewHBox(
                widget.NewLabel("Фильтры:"),
                d.classSel,
                d.subjectSel,
                d.quarterSel,
                d.refreshBtn,
                d.deleteBtn,
        )
}

// ------------------------------------------
// BOTTOM TAB BAR — instant section switching
// ------------------------------------------

func (d *Dashboard) buildTabBar() *fyne.Container {
        d.tabButtons = make([]*widget.Button, 4)

        d.tabButtons[0] = widget.NewButton("Журнал", func() {
                d.switchTab(0, d.gradesContainer)
        })
        d.tabButtons[1] = widget.NewButton("Темы и ДЗ", func() {
                d.switchTab(1, d.topicsTab.Container())
        })
        d.tabButtons[2] = widget.NewButton("Дневник", func() {
                d.switchTab(2, d.diariesTab.Container())
        })
        d.tabButtons[3] = widget.NewButton("Итоговые", func() {
                d.switchTab(3, d.finalGradesTab.Container())
        })

        // Set initial active style
        d.activeTab = 0
        d.highlightActiveTab()

        tabRow := container.NewGridWithColumns(4,
                d.tabButtons[0], d.tabButtons[1], d.tabButtons[2], d.tabButtons[3])

        // Status label + tab bar
        bottomBox := container.NewVBox(
                d.statusLabel,
                tabRow,
        )

        bg := canvas.NewRectangle(colorNavBG)
        bg.SetMinSize(fyne.NewSize(0, 56))

        return container.NewStack(bg, container.NewPadded(bottomBox))
}

// highlightActiveTab visually marks the currently active tab button.
func (d *Dashboard) highlightActiveTab() {
        for i, btn := range d.tabButtons {
                if i == d.activeTab {
                        btn.Importance = widget.HighImportance
                } else {
                        btn.Importance = widget.MediumImportance
                }
                btn.Refresh()
        }
}

func (d *Dashboard) switchTab(idx int, page fyne.CanvasObject) {
        d.activeTab = idx
        d.currentPage = page
        d.contentStack.Objects = []fyne.CanvasObject{page}
        d.contentStack.Refresh()
        d.highlightActiveTab()
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (d *Dashboard) loadJournalOptions() {
        d.statusLabel.SetText("Загрузка настроек журнала...")
        opts, err := d.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        d.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v", err))
                })
                return
        }

        d.journalOpts = opts

        classNames := make([]string, len(opts.Groups))
        for i, g := range opts.Groups {
                classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
        }

        fyne.Do(func() {
                d.classSel.Options = classNames
                d.classSel.Refresh()
                d.statusLabel.SetText("")
                if len(classNames) > 0 {
                        d.classSel.SetSelectedIndex(0)
                }
        })
}

// ------------------------------------------
// FILTER HANDLERS
// ------------------------------------------

func (d *Dashboard) onClassSelected(selected string) {
        if d.journalOpts == nil {
                return
        }
        for i, g := range d.journalOpts.Groups {
                if fmt.Sprintf("%d %s", g.Number, g.Name) == selected {
                        d.selectedGroup = &d.journalOpts.Groups[i]
                        break
                }
        }
        if d.selectedGroup == nil {
                return
        }

        // Remember the previously selected subject name
        prevSubjectName := ""
        if d.selectedSubject != nil {
                prevSubjectName = d.selectedSubject.SubjectName
        }
        // Remember previously selected quarter
        prevQuarterName := ""
        if d.selectedQuarter != nil {
                prevQuarterName = d.selectedQuarter.Name
        }

        d.selectedSubject = nil
        d.selectedQuarter = nil

        subjectNames := make([]string, len(d.selectedGroup.Subjects))
        for i, s := range d.selectedGroup.Subjects {
                subjectNames[i] = s.SubjectName
        }

        quarterNames := make([]string, len(d.selectedGroup.Quarters))
        for i, q := range d.selectedGroup.Quarters {
                quarterNames[i] = q.Name
        }

        fyne.Do(func() {
                d.subjectSel.Options = subjectNames
                d.subjectSel.Refresh()
                d.quarterSel.Options = quarterNames
                d.quarterSel.Refresh()
                d.refreshBtn.Disable()

                // Try to keep the same subject if it exists in the new class
                subjectFound := false
                if prevSubjectName != "" {
                        for i, name := range subjectNames {
                                if name == prevSubjectName {
                                        d.subjectSel.SetSelectedIndex(i)
                                        subjectFound = true
                                        break
                                }
                        }
                }
                if !subjectFound && len(subjectNames) > 0 {
                        d.subjectSel.SetSelectedIndex(0)
                }

                // Try to keep the same quarter
                quarterFound := false
                if prevQuarterName != "" {
                        for i, name := range quarterNames {
                                if name == prevQuarterName {
                                        d.quarterSel.SetSelectedIndex(i)
                                        quarterFound = true
                                        break
                                }
                        }
                }
                if !quarterFound {
                        // Default to current quarter
                        for i, q := range d.selectedGroup.Quarters {
                                if q.CurrentQuarter {
                                        d.quarterSel.SetSelectedIndex(i)
                                        break
                                }
                        }
                }
        })
}

func (d *Dashboard) onSubjectSelected(selected string) {
        if d.selectedGroup == nil {
                return
        }
        for i, s := range d.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        d.selectedSubject = &d.selectedGroup.Subjects[i]
                        break
                }
        }
        d.checkFilterCompletion()
}

func (d *Dashboard) onQuarterSelected(selected string) {
        if d.selectedGroup == nil {
                return
        }
        for i, q := range d.selectedGroup.Quarters {
                if q.Name == selected {
                        d.selectedQuarter = &d.selectedGroup.Quarters[i]
                        break
                }
        }
        d.checkFilterCompletion()
}

func (d *Dashboard) checkFilterCompletion() {
        ready := d.selectedGroup != nil && d.selectedSubject != nil && d.selectedQuarter != nil
        if ready {
                d.refreshBtn.Enable()
                go d.loadData()
        } else {
                d.refreshBtn.Disable()
        }
}

func (d *Dashboard) loadData() {
        if d.selectedGroup == nil || d.selectedSubject == nil || d.selectedQuarter == nil {
                return
        }

        fyne.Do(func() {
                d.statusLabel.SetText("Загрузка данных журнала...")
        })

        apiClient := d.controller.GetClient()
        gID := d.selectedGroup.ID
        sID := d.selectedSubject.SubjectID
        qID := d.selectedQuarter.ID

        students, errS := apiClient.GetJournalStudents(gID, sID, qID)
        dates, errD := apiClient.GetJournalDates(gID, sID, qID)

        fyne.Do(func() {
                if errS != nil || errD != nil {
                        msg := ""
                        if errS != nil {
                                msg = fmt.Sprintf("Ошибка учеников: %v", errS)
                        } else {
                                msg = fmt.Sprintf("Ошибка дат: %v", errD)
                        }
                        d.statusLabel.SetText(msg)
                        dialog.ShowError(fmt.Errorf(msg), d.controller.GetWindow())
                        return
                }

                d.students = students
                d.dates = dates

                if len(students) == 0 {
                        d.statusLabel.SetText("Нет данных")
                } else {
                        d.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников, %d дат", len(students), len(dates)))
                }

                d.rebuildGradesTable()
        })
}

// ------------------------------------------
// GRADES TABLE — selection, arrows, Enter, Delete
// ------------------------------------------

func (d *Dashboard) rebuildGradesTable() {
        if len(d.students) == 0 || len(d.dates) == 0 {
                d.gradesContainer.Objects = []fyne.CanvasObject{
                        widget.NewLabelWithStyle("Нет данных для отображения", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                }
                d.gradesContainer.Refresh()
                return
        }

        // Columns: № | ФИО | date1 | date2 | ... | dateN | Ср.балл
        numDateCols := len(d.dates)
        totalCols := 2 + numDateCols + 1 // №, ФИО, dates, avg
        rowCount := len(d.students) + 1   // +1 header

        d.gradesTable = widget.NewTable(
                func() (int, int) { return rowCount, totalCols },
                func() fyne.CanvasObject {
                        lbl := widget.NewLabel("")
                        lbl.Alignment = fyne.TextAlignCenter
                        lbl.Wrapping = fyne.TextWrapOff
                        return container.NewMax(lbl)
                },
                func(id widget.TableCellID, cell fyne.CanvasObject) {
                        c := cell.(*fyne.Container)
                        lbl := c.Objects[0].(*widget.Label)
                        lbl.SetText("—")
                        lbl.Alignment = fyne.TextAlignCenter
                        lbl.TextStyle = fyne.TextStyle{}

                        // Header row
                        if id.Row == 0 {
                                lbl.TextStyle = fyne.TextStyle{Bold: true}
                                switch id.Col {
                                case 0:
                                        lbl.SetText("№")
                                case 1:
                                        lbl.SetText("ФИО ученика")
                                        lbl.Alignment = fyne.TextAlignLeading
                                case totalCols - 1:
                                        lbl.SetText("Ср.")
                                default:
                                        dateIdx := id.Col - 2
                                        if dateIdx >= 0 && dateIdx < len(d.dates) {
                                                day := d.dates[dateIdx]
                                                lbl.SetText(day.WeekdayShortName + "\n" + day.AssignmentDate[5:])
                                        }
                                }
                                return
                        }

                        // Data rows
                        studentIdx := id.Row - 1
                        if studentIdx >= len(d.students) {
                                return
                        }
                        student := d.students[studentIdx]

                        switch id.Col {
                        case 0:
                                lbl.SetText(strconv.Itoa(studentIdx + 1))
                        case 1:
                                lbl.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
                                lbl.Alignment = fyne.TextAlignLeading
                        case totalCols - 1:
                                if student.AverageScore != "" && student.AverageScore != "0.0" {
                                        lbl.SetText(student.AverageScore)
                                }
                        default:
                                dateIdx := id.Col - 2
                                if dateIdx >= 0 && dateIdx < len(d.dates) {
                                        dateID := d.dates[dateIdx].AssignmentDateID
                                        for _, sm := range student.SubjectMarks {
                                                if sm.AssignmentDateID == dateID {
                                                        lbl.SetText(sm.ShortName)
                                                        break
                                                }
                                        }
                                }
                        }
                },
        )

        // Set column widths
        d.gradesTable.SetColumnWidth(0, 40)  // №
        d.gradesTable.SetColumnWidth(1, 180) // ФИО
        for i := 0; i < numDateCols; i++ {
                d.gradesTable.SetColumnWidth(2+i, 50) // date columns
        }
        d.gradesTable.SetColumnWidth(totalCols-1, 50) // avg

        // Track selection state
        clickCount := 0
        var lastCellID widget.TableCellID

        d.gradesTable.OnSelected = func(id widget.TableCellID) {
                // Track selected cell
                d.selectedCell = &id

                // Enable delete button if a grade cell is selected
                if id.Row > 0 && id.Col >= 2 && id.Col < totalCols-1 {
                        d.deleteBtn.Enable()
                } else {
                        d.deleteBtn.Disable()
                }

                // Detect double-click
                if id == lastCellID {
                        clickCount++
                } else {
                        clickCount = 1
                        lastCellID = id
                }

                // Double-click on student name column (col 1) → random fill all dates
                if clickCount >= 2 && id.Row > 0 && id.Col == 1 {
                        clickCount = 0
                        studentIdx := id.Row - 1
                        if studentIdx < len(d.students) {
                                d.showRandomFillForStudent(studentIdx)
                        }
                        return
                }

                // Double-click on a grade cell (date column) → edit single grade
                if clickCount >= 2 && id.Row > 0 && id.Col >= 2 && id.Col < totalCols-1 {
                        clickCount = 0
                        studentIdx := id.Row - 1
                        dateIdx := id.Col - 2
                        if studentIdx < len(d.students) && dateIdx < len(d.dates) {
                                d.showEditGradePopup(studentIdx, dateIdx)
                        }
                        return
                }

                // Don't unselect on single click — keep it selected for arrow keys
        }

        // Wrap table in scroll for cross-platform responsiveness (horizontal + vertical)
        scrollWrap := container.NewScroll(d.gradesTable)
        scrollWrap.Direction = container.ScrollBoth

        d.gradesContainer.Objects = []fyne.CanvasObject{scrollWrap}
        d.gradesContainer.Refresh()

        // Install keyboard handler for arrow navigation, Delete key
        d.installKeyboardHandler()
}

// ------------------------------------------
// KEYBOARD HANDLER — Arrow keys + Delete key
// ------------------------------------------

// installKeyboardHandler sets up arrow key navigation and Delete key on the
// window's canvas so that the user can move between journal cells and delete
// grades without reaching for the mouse.
func (d *Dashboard) installKeyboardHandler() {
        w := d.controller.GetWindow()
        if w == nil {
                return
        }
        c := w.Canvas()
        if c == nil {
                return
        }

        c.SetOnTypedKey(func(ev *fyne.KeyEvent) {
                // Only handle when the journal table is visible
                if d.gradesTable == nil || d.currentPage != d.gradesContainer {
                        return
                }

                switch ev.Name {
                case fyne.KeyUp:
                        d.navigateCell(0, -1)
                case fyne.KeyDown:
                        d.navigateCell(0, 1)
                case fyne.KeyLeft:
                        d.navigateCell(-1, 0)
                case fyne.KeyRight:
                        d.navigateCell(1, 0)
                case fyne.KeyDelete:
                        d.deleteSelectedCell()
                }
        })
}

// navigateCell moves the selected cell by (dCol, dRow) and selects it.
// It clamps to valid cell boundaries within the journal table.
func (d *Dashboard) navigateCell(dCol, dRow int) {
        if d.gradesTable == nil {
                return
        }

        rowCount := len(d.students) + 1 // +1 header
        numDateCols := len(d.dates)
        totalCols := 2 + numDateCols + 1

        // If no cell selected yet, start at first data cell
        if d.selectedCell == nil {
                start := widget.TableCellID{Row: 1, Col: 2}
                d.selectedCell = &start
                d.gradesTable.Select(start)
                d.deleteBtn.Enable()
                return
        }

        newCol := d.selectedCell.Col + dCol
        newRow := d.selectedCell.Row + dRow

        // Clamp to table bounds
        if newCol < 0 {
                newCol = 0
        }
        if newCol >= totalCols {
                newCol = totalCols - 1
        }
        if newRow < 0 {
                newRow = 0
        }
        if newRow >= rowCount {
                newRow = rowCount - 1
        }

        newID := widget.TableCellID{Row: newRow, Col: newCol}
        d.selectedCell = &newID
        d.gradesTable.Select(newID)

        // Enable/disable delete button based on whether it's a grade cell
        if newRow > 0 && newCol >= 2 && newCol < totalCols-1 {
                d.deleteBtn.Enable()
        } else {
                d.deleteBtn.Disable()
        }
}

// ------------------------------------------
// DELETE SELECTED CELL
// ------------------------------------------

func (d *Dashboard) deleteSelectedCell() {
        if d.selectedCell == nil || d.students == nil || d.dates == nil {
                return
        }

        id := *d.selectedCell
        numDateCols := len(d.dates)
        totalCols := 2 + numDateCols + 1

        // Only grade cells (date columns)
        if id.Row <= 0 || id.Col < 2 || id.Col >= totalCols-1 {
                return
        }

        studentIdx := id.Row - 1
        dateIdx := id.Col - 2

        if studentIdx >= len(d.students) || dateIdx >= len(d.dates) {
                return
        }

        student := d.students[studentIdx]
        date := d.dates[dateIdx]

        // Find the mark ID for this cell
        var markID string
        for _, sm := range student.SubjectMarks {
                if sm.AssignmentDateID == date.AssignmentDateID {
                        if sm.AssignmentMarkID != "" {
                                markID = sm.AssignmentMarkID
                        }
                        break
                }
        }

        if markID == "" {
                d.statusLabel.SetText("Ячейка пуста — нечего удалять")
                return
        }

        confirmMsg := fmt.Sprintf("Удалить оценку %s для %s %s — %s?",
                student.SubjectMarks[0].ShortName, student.LastName, student.FirstName,
                date.AssignmentDate[5:])

        dialog.ShowConfirm("Удалить оценку", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go func() {
                        err := d.controller.GetClient().DeleteMark(markID)
                        fyne.Do(func() {
                                if err != nil {
                                        dialog.ShowError(fmt.Errorf("Ошибка удаления: %v", err), d.controller.GetWindow())
                                } else {
                                        d.statusLabel.SetText(fmt.Sprintf("Оценка удалена: %s %s — %s",
                                                student.LastName, student.FirstName, date.AssignmentDate[5:]))
                                        go d.loadData()
                                }
                        })
                }()
        }, d.controller.GetWindow())
}

// ------------------------------------------
// EDIT GRADE POPUP — double-click on a grade cell
// ------------------------------------------

func (d *Dashboard) showEditGradePopup(studentIdx, dateIdx int) {
        student := d.students[studentIdx]
        date := d.dates[dateIdx]

        // Find current mark
        var currentMark string
        for _, sm := range student.SubjectMarks {
                if sm.AssignmentDateID == date.AssignmentDateID {
                        currentMark = sm.ShortName
                        break
                }
        }

        gradeEntry := widget.NewEntry()
        gradeEntry.SetPlaceHolder("2-10")
        if currentMark != "" && currentMark != "—" {
                gradeEntry.SetText(currentMark)
        }

        header := fmt.Sprintf("%s %s — %s (%s)",
                student.LastName, student.FirstName,
                date.WeekdayShortName, date.AssignmentDate[5:])

        content := container.NewVBox(
                widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewSeparator(),
                container.NewGridWithColumns(2,
                        widget.NewLabel("Оценка (2-10):"),
                        gradeEntry,
                ),
        )

        dialog.ShowForm("Оценка", "Сохранить", "Отмена", []*widget.FormItem{
                widget.NewFormItem("", content),
        }, func(ok bool) {
                if !ok {
                        return
                }
                gradeStr := gradeEntry.Text
                if gradeStr == "" {
                        return
                }
                grade, err := strconv.Atoi(gradeStr)
                if err != nil || grade < 2 || grade > 10 {
                        dialog.ShowError(fmt.Errorf("Оценка от 2 до 10"), d.controller.GetWindow())
                        return
                }

                go func() {
                        err := d.controller.GetClient().CreateMark(
                                student.StudentID,
                                date.AssignmentDateID,
                                d.selectedQuarter.ID,
                                grade,
                        )
                        fyne.Do(func() {
                                if err != nil {
                                        dialog.ShowError(fmt.Errorf("Ошибка: %v", err), d.controller.GetWindow())
                                } else {
                                        d.statusLabel.SetText(fmt.Sprintf("Оценка %d: %s %s — %s",
                                                grade, student.LastName, student.FirstName, date.AssignmentDate[5:]))
                                        go d.loadData()
                                }
                        })
                }()
        }, d.controller.GetWindow())
}

// ------------------------------------------
// RANDOM FILL FOR STUDENT — double-click on name
// Fills ALL empty dates in the current quarter
// ------------------------------------------

func (d *Dashboard) showRandomFillForStudent(studentIdx int) {
        student := d.students[studentIdx]

        minEntry := widget.NewEntry()
        minEntry.SetPlaceHolder("2")
        minEntry.SetText("7")

        maxEntry := widget.NewEntry()
        maxEntry.SetPlaceHolder("10")
        maxEntry.SetText("10")

        comboSel := widget.NewSelect(comboNames(), func(sel string) {
                for _, c := range GradeCombos {
                        if c.Name == sel {
                                minEntry.SetText(strconv.Itoa(c.MinVal))
                                maxEntry.SetText(strconv.Itoa(c.MaxVal))
                                break
                        }
                }
        })
        comboSel.PlaceHolder = "Быстрый выбор..."

        // Count empty dates for this student
        emptyCount := 0
        for _, date := range d.dates {
                hasMark := false
                for _, sm := range student.SubjectMarks {
                        if sm.AssignmentDateID == date.AssignmentDateID && sm.ShortName != "" && sm.ShortName != "—" {
                                hasMark = true
                                break
                        }
                }
                if !hasMark {
                        emptyCount++
                }
        }

        quarterName := ""
        if d.selectedQuarter != nil {
                quarterName = d.selectedQuarter.Name
        }

        header := fmt.Sprintf("%s %s — %s",
                student.LastName, student.FirstName, quarterName)

        infoText := fmt.Sprintf("Пустых дат: %d из %d", emptyCount, len(d.dates))

        content := container.NewVBox(
                widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewLabel(infoText),
                widget.NewSeparator(),
                widget.NewLabel("Быстрый диапазон:"),
                comboSel,
                widget.NewSeparator(),
                container.NewGridWithColumns(2,
                        widget.NewLabel("Мин. оценка:"),
                        minEntry,
                        widget.NewLabel("Макс. оценка:"),
                        maxEntry,
                ),
                widget.NewSeparator(),
                widget.NewLabelWithStyle(fmt.Sprintf("Будут выставлены случайные оценки (%d шт) на все пустые даты", emptyCount),
                        fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
        )

        dialog.ShowForm("Рандомные оценки — все даты", "Заполнить", "Отмена", []*widget.FormItem{
                widget.NewFormItem("", content),
        }, func(ok bool) {
                if !ok {
                        return
                }
                minVal, err1 := strconv.Atoi(minEntry.Text)
                maxVal, err2 := strconv.Atoi(maxEntry.Text)
                if err1 != nil || err2 != nil {
                        dialog.ShowError(fmt.Errorf("Введите корректные числа"), d.controller.GetWindow())
                        return
                }

                go d.executeRandomFillForStudent(studentIdx, minVal, maxVal)
        }, d.controller.GetWindow())
}

func (d *Dashboard) executeRandomFillForStudent(studentIdx, minVal, maxVal int) {
        student := d.students[studentIdx]
        apiClient := d.controller.GetClient()
        quarterID := d.selectedQuarter.ID

        successCount := 0
        skipCount := 0

        for _, date := range d.dates {
                // Skip dates that already have a mark
                hasMark := false
                for _, sm := range student.SubjectMarks {
                        if sm.AssignmentDateID == date.AssignmentDateID && sm.ShortName != "" && sm.ShortName != "—" {
                                hasMark = true
                                break
                        }
                }
                if hasMark {
                        skipCount++
                        continue
                }

                grade := RandomGradeInRange(minVal, maxVal)

                fyne.Do(func() {
                        d.statusLabel.SetText(fmt.Sprintf("Ставлю %d: %s %s — %s (%d/%d)",
                                grade, student.LastName, student.FirstName, date.AssignmentDate[5:],
                                successCount+1, len(d.dates)-skipCount))
                })

                err := apiClient.CreateMark(
                        student.StudentID,
                        date.AssignmentDateID,
                        quarterID,
                        grade,
                )
                if err == nil {
                        successCount++
                }
        }

        fyne.Do(func() {
                d.statusLabel.SetText(fmt.Sprintf("Готово: %d оценок для %s %s (пропущено: %d)",
                        successCount, student.LastName, student.FirstName, skipCount))
                go d.loadData()
        })
}

// comboNames returns list of grade combo names for UI selector.
func comboNames() []string {
        names := make([]string, len(GradeCombos))
        for i, c := range GradeCombos {
                names[i] = c.Name
        }
        return names
}
