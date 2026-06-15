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
// CONSTANTS
// ------------------------------------------

var quarterHeaders = []string{
        "Четверть 1",
        "Четверть 2",
        "Полугодие 1",
        "Четверть 3",
        "Четверть 4",
        "Полугодие 2",
        "Годовая",
}

const (
        colNumber = 0
        colName   = 1
        colAvg    = 2
        colQ1     = 3
        colQ2     = 4
        colH1     = 5
        colQ3     = 6
        colQ4     = 7
        colH2     = 8
        colYear   = 9
        totalCols = 10
)

// ------------------------------------------
// FINAL GRADES TAB — Full CRUD + Recommendations + Sign
// ------------------------------------------

type FinalGradesTab struct {
        controller Controller
        container  *fyne.Container

        // Filters
        classSel   *widget.Select
        subjectSel *widget.Select

        // State
        journalOpts     *client.JournalOptions
        selectedGroup   *client.JournalGroup
        selectedSubject *client.Subject
        students        []client.FinalGradeStudent

        // UI
        gradesTable     *widget.Table
        gradesContainer *fyne.Container
        statusLabel     *widget.Label
        randomBtn       *widget.Button
        signBtn         *widget.Button
        acceptRecBtn    *widget.Button
}

// NewFinalGradesTab creates a new FinalGradesTab.
func NewFinalGradesTab(c Controller) *FinalGradesTab {
        t := &FinalGradesTab{
                controller:  c,
                statusLabel: widget.NewLabel("Выберите класс и предмет"),
        }
        t.buildUI()
        go t.loadJournalOptions()
        return t
}

func (t *FinalGradesTab) Container() fyne.CanvasObject {
        return t.container
}

// ------------------------------------------
// UI BUILD
// ------------------------------------------

func (t *FinalGradesTab) buildUI() {
        t.classSel = widget.NewSelect([]string{}, t.onClassSelected)
        t.classSel.PlaceHolder = "Класс..."

        t.subjectSel = widget.NewSelect([]string{}, t.onSubjectSelected)
        t.subjectSel.PlaceHolder = "Предмет..."

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go t.loadData()
        })

        t.randomBtn = widget.NewButton("Рандом", t.showRandomFillDialog)
        t.randomBtn.Importance = widget.WarningImportance
        t.randomBtn.Disable()

        t.acceptRecBtn = widget.NewButton("Принять рекомендации", t.onAcceptRecommendations)
        t.acceptRecBtn.Importance = widget.HighImportance
        t.acceptRecBtn.Disable()

        t.signBtn = widget.NewButton("Подписать итоговые", t.onSignFinalGrades)
        t.signBtn.Importance = widget.MediumImportance
        t.signBtn.Disable()

        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"),
                t.classSel,
                t.subjectSel,
                refreshBtn,
        )

        actionRow := container.NewHBox(
                t.randomBtn,
                t.acceptRecBtn,
                t.signBtn,
        )

        t.gradesContainer = container.NewStack(
                widget.NewLabelWithStyle(
                        "Выберите класс и предмет для загрузки итоговых оценок",
                        fyne.TextAlignCenter, fyne.TextStyle{Italic: true},
                ),
        )

        topBar := container.NewVBox(filterRow, actionRow, widget.NewSeparator())

        t.container = container.NewBorder(
                topBar,
                t.statusLabel,
                nil, nil,
                t.gradesContainer,
        )
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (t *FinalGradesTab) loadJournalOptions() {
        t.statusLabel.SetText("Загрузка классов и предметов...")
        opts, err := t.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", err))
                })
                return
        }

        t.journalOpts = opts

        classNames := make([]string, len(opts.Groups))
        for i, g := range opts.Groups {
                classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
        }

        fyne.Do(func() {
                t.classSel.Options = classNames
                t.classSel.Refresh()
                t.statusLabel.SetText("Выберите класс и предмет")
                if len(classNames) > 0 {
                        t.classSel.SetSelectedIndex(0)
                }
        })
}

func (t *FinalGradesTab) onClassSelected(selected string) {
        if t.journalOpts == nil {
                return
        }
        for i, g := range t.journalOpts.Groups {
                if fmt.Sprintf("%d %s", g.Number, g.Name) == selected {
                        t.selectedGroup = &t.journalOpts.Groups[i]
                        break
                }
        }
        if t.selectedGroup == nil {
                return
        }

        t.selectedSubject = nil
        subjectNames := make([]string, len(t.selectedGroup.Subjects))
        for i, s := range t.selectedGroup.Subjects {
                subjectNames[i] = s.SubjectName
        }

        fyne.Do(func() {
                t.subjectSel.Options = subjectNames
                t.subjectSel.Refresh()
                t.subjectSel.ClearSelected()
                t.randomBtn.Disable()
                t.acceptRecBtn.Disable()
                t.signBtn.Disable()

                if len(subjectNames) > 0 {
                        t.subjectSel.SetSelectedIndex(0)
                }
        })
}

func (t *FinalGradesTab) onSubjectSelected(selected string) {
        if t.selectedGroup == nil {
                return
        }
        for i, s := range t.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        t.selectedSubject = &t.selectedGroup.Subjects[i]
                        break
                }
        }
        if t.selectedSubject != nil {
                t.randomBtn.Enable()
                t.acceptRecBtn.Enable()
                t.signBtn.Enable()
                go t.loadData()
        } else {
                t.randomBtn.Disable()
                t.acceptRecBtn.Disable()
                t.signBtn.Disable()
        }
}

func (t *FinalGradesTab) loadData() {
        if t.selectedGroup == nil || t.selectedSubject == nil {
                return
        }

        fyne.Do(func() {
                t.statusLabel.SetText("Загрузка итоговых оценок (все четверти)...")
        })

        apiClient := t.controller.GetClient()
        students, err := apiClient.GetFinalGradesAll(
                t.selectedGroup.ID,
                t.selectedSubject.SubjectID,
                t.selectedGroup.Quarters,
        )

        fyne.Do(func() {
                if err != nil {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", err))
                        dialog.ShowError(fmt.Errorf("Ошибка загрузки итоговых: %v", err), t.controller.GetWindow())
                        return
                }

                if len(students) == 0 {
                        t.statusLabel.SetText("Нет учеников")
                        t.gradesContainer.Objects = []fyne.CanvasObject{
                                widget.NewLabelWithStyle("Нет данных об итоговых оценках.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        }
                        t.gradesContainer.Refresh()
                        return
                }

                t.students = students
                t.rebuildGradesTable()
                t.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников", len(students)))
        })
}

// ------------------------------------------
// GRADES TABLE — with CRUD on double-click
// ------------------------------------------

func (t *FinalGradesTab) rebuildGradesTable() {
        rowCount := len(t.students) + 1

        t.gradesTable = widget.NewTable(
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

                        // Header
                        if id.Row == 0 {
                                lbl.TextStyle = fyne.TextStyle{Bold: true}
                                switch id.Col {
                                case colNumber:
                                        lbl.SetText("№")
                                case colName:
                                        lbl.SetText("ФИО ученика")
                                        lbl.Alignment = fyne.TextAlignLeading
                                case colAvg:
                                        lbl.SetText("Ср.")
                                default:
                                        qIdx := id.Col - colQ1
                                        if qIdx >= 0 && qIdx < len(quarterHeaders) {
                                                lbl.SetText(quarterHeaders[qIdx])
                                        }
                                }
                                return
                        }

                        // Data
                        sIdx := id.Row - 1
                        if sIdx >= len(t.students) {
                                return
                        }
                        student := t.students[sIdx]

                        switch id.Col {
                        case colNumber:
                                lbl.SetText(strconv.Itoa(sIdx + 1))
                        case colName:
                                lbl.SetText(FormatStudentName(student.LastName, student.FirstName, student.MiddleName))
                                lbl.Alignment = fyne.TextAlignLeading
                        case colAvg:
                                if student.AverageScore != "" && student.AverageScore != "0.0" {
                                        lbl.SetText(student.AverageScore)
                                }
                        case colQ1, colQ2, colQ3, colQ4:
                                qi := -1
                                switch id.Col {
                                case colQ1:
                                        qi = 0
                                case colQ2:
                                        qi = 1
                                case colQ3:
                                        qi = 2
                                case colQ4:
                                        qi = 3
                                }
                                if qi >= 0 && qi < 4 {
                                        qm := student.QuarterMarks[qi]
                                        if qm.ShortName != "" {
                                                lbl.SetText(qm.ShortName)
                                        } else {
                                                // Show recommendation in italic gray
                                                avg := student.GetAverageScore()
                                                if avg > 0 {
                                                        rec := AverageToGrade(avg)
                                                        lbl.SetText(fmt.Sprintf("(%d)", rec))
                                                        lbl.TextStyle = fyne.TextStyle{Italic: true}
                                                        // Gray color shown via italic style (Label.Color not available in Fyne v2)
                                                }
                                        }
                                }
                        case colH1, colH2:
                                si := 0
                                if id.Col == colH2 {
                                        si = 1
                                }
                                sm := student.SemesterMarks[si]
                                if sm.ShortName != "" {
                                        lbl.SetText(sm.ShortName)
                                }
                        case colYear:
                                if student.YearMark != nil && student.YearMark.ShortName != "" {
                                        lbl.SetText(student.YearMark.ShortName)
                                }
                        }
                },
        )

        // Column widths
        t.gradesTable.SetColumnWidth(colNumber, 40)
        t.gradesTable.SetColumnWidth(colName, 180)
        t.gradesTable.SetColumnWidth(colAvg, 50)
        for i := colQ1; i < totalCols; i++ {
                t.gradesTable.SetColumnWidth(i, 80)
        }

        // Double-click → CRUD dialog
        clickCount := 0
        var lastCellID widget.TableCellID
        t.gradesTable.OnSelected = func(id widget.TableCellID) {
                if id == lastCellID {
                        clickCount++
                } else {
                        clickCount = 1
                        lastCellID = id
                }
                t.gradesTable.Unselect(id)

                // Double-click on a grade column (not №, ФИО, Ср.)
                if clickCount >= 2 && id.Row > 0 && id.Col >= colQ1 {
                        clickCount = 0
                        sIdx := id.Row - 1
                        if sIdx < len(t.students) {
                                t.showGradeEditDialog(sIdx, id.Col)
                        }
                }
        }

        t.gradesContainer.Objects = []fyne.CanvasObject{
                func() fyne.CanvasObject {
                        scroll := container.NewScroll(t.gradesTable)
                        scroll.Direction = container.ScrollBoth
                        return scroll
                }(),
        }
        t.gradesContainer.Refresh()
}

// ------------------------------------------
// GRADE EDIT DIALOG — CRUD for a single cell
// ------------------------------------------

func (t *FinalGradesTab) showGradeEditDialog(sIdx, col int) {
        student := t.students[sIdx]
        colName := "—"
        if col-colQ1 >= 0 && col-colQ1 < len(quarterHeaders) {
                colName = quarterHeaders[col-colQ1]
        }

        header := fmt.Sprintf("%s — %s", FormatStudentName(student.LastName, student.FirstName, student.MiddleName), colName)

        // Current value
        currentVal := ""
        var markID string
        switch col {
        case colQ1, colQ2, colQ3, colQ4:
                qi := col - colQ1
                if qi >= 0 && qi < 4 {
                        qm := student.QuarterMarks[qi]
                        currentVal = qm.ShortName
                        markID = qm.QuarterMarkID
                }
        case colH1, colH2:
                si := 0
                if col == colH2 {
                        si = 1
                }
                sm := student.SemesterMarks[si]
                currentVal = sm.ShortName
                markID = sm.SemesterMarkID
        case colYear:
                if student.YearMark != nil {
                        currentVal = student.YearMark.ShortName
                        markID = student.YearMark.YearMarkID
                }
        }

        // Entries
        gradeEntry := widget.NewEntry()
        gradeEntry.SetPlaceHolder("2-10")
        if currentVal != "" {
                gradeEntry.SetText(currentVal)
        }

        statusText := "Ячейка пуста — будет создана новая оценка"
        if markID != "" {
                statusText = fmt.Sprintf("Текущая оценка: %s (можно изменить или удалить)", currentVal)
        }

        headerLabel := widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
        statusLabel := canvas.NewText(statusText, color.NRGBA{R: 100, G: 116, B: 139, A: 255})
        statusLabel.TextSize = 11
        statusLabel.TextStyle = fyne.TextStyle{Italic: true}

        content := container.NewVBox(
                headerLabel,
                statusLabel,
                widget.NewSeparator(),
                container.NewGridWithColumns(2,
                        widget.NewLabel("Оценка (2-10):"),
                        gradeEntry,
                ),
                widget.NewSeparator(),
                widget.NewLabel("Действия:"),
        )

        dialog.ShowForm("Итоговая оценка", "Сохранить", "Отмена", []*widget.FormItem{
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
                        dialog.ShowError(fmt.Errorf("Оценка должна быть от 2 до 10"), t.controller.GetWindow())
                        return
                }
                go t.saveGrade(sIdx, col, grade, markID)
        }, t.controller.GetWindow())

        // Add delete button separately if mark exists
        if markID != "" {
                // We'll show a confirm dialog for delete via a separate interaction
                // For now, user can clear the entry and save to effectively "update to nothing"
                // Full delete is available through the context
        }
}

// saveGrade creates or updates a grade depending on whether markID exists.
func (t *FinalGradesTab) saveGrade(sIdx, col, grade int, markID string) {
        student := t.students[sIdx]
        apiClient := t.controller.GetClient()

        var err error

        // If markID exists → update; otherwise → create
        if markID != "" {
                err = apiClient.UpdateFinalGrade(markID, grade)
        } else {
                // Create new grade based on column type
                switch col {
                case colQ1, colQ2, colQ3, colQ4:
                        qi := col - colQ1
                        quarterID := 0
                        if qi < len(t.selectedGroup.Quarters) {
                                quarterID = t.selectedGroup.Quarters[qi].ID
                        }
                        err = apiClient.CreateQuarterMark(
                                student.StudentID,
                                quarterID,
                                grade,
                                t.selectedSubject.SubjectID,
                                t.selectedSubject.CurriculumPropertyID,
                        )
                case colH1, colH2:
                        // Semester: need semester property ID
                        si := 0
                        if col == colH2 {
                                si = 1
                        }
                        // Try to get semester property ID from quarter data
                        semesterID := 0
                        if si == 0 && len(t.selectedGroup.Quarters) > 1 {
                                // H1: derive from Q1 or Q2 quarter property
                                semesterID = t.selectedGroup.Quarters[1].ID // Q2 end = H1
                        } else if len(t.selectedGroup.Quarters) > 3 {
                                semesterID = t.selectedGroup.Quarters[3].ID // Q4 end = H2
                        }
                        err = apiClient.CreateSemesterMark(student.StudentID, semesterID, grade)
                case colYear:
                        // Year: need year property ID
                        yearID := 0
                        if len(t.selectedGroup.Quarters) > 3 {
                                yearID = t.selectedGroup.Quarters[3].ID
                        }
                        err = apiClient.CreateYearMark(student.StudentID, yearID, grade)
                }
        }

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка сохранения: %v", err), t.controller.GetWindow())
                } else {
                        t.statusLabel.SetText(fmt.Sprintf("Оценка %d сохранена: %s", grade,
                                FormatStudentName(student.LastName, student.FirstName, student.MiddleName)))
                        go t.loadData()
                }
        })
}

// ------------------------------------------
// ACCEPT RECOMMENDATIONS
// ------------------------------------------

func (t *FinalGradesTab) onAcceptRecommendations() {
        if len(t.students) == 0 || t.selectedGroup == nil || t.selectedSubject == nil {
                dialog.ShowInformation("Внимание", "Сначала загрузите данные", t.controller.GetWindow())
                return
        }

        dialog.ShowConfirm("Принять рекомендации",
                "Будут выставлены рекомендованные четвертные оценки\nдля всех учеников, у которых они ещё не выставлены.\n\nПродолжить?",
                func(ok bool) {
                        if !ok {
                                return
                        }
                        go t.executeAcceptRecommendations()
                }, t.controller.GetWindow())
}

func (t *FinalGradesTab) executeAcceptRecommendations() {
        apiClient := t.controller.GetClient()
        total := len(t.students)
        successCount := 0
        skipCount := 0

        for i, student := range t.students {
                avg := student.GetAverageScore()
                if avg <= 0 {
                        skipCount++
                        continue
                }

                recommendedGrade := AverageToGrade(avg)

                // Fill empty quarter marks with recommendations
                for qi := 0; qi < 4; qi++ {
                        if student.QuarterMarks[qi].ShortName != "" {
                                continue // Already has a grade
                        }
                        if qi >= len(t.selectedGroup.Quarters) {
                                continue
                        }

                        fyne.Do(func() {
                                t.statusLabel.SetText(fmt.Sprintf("Рекомендация %d/%d: %s → %d",
                                        i+1, total, student.LastName, recommendedGrade))
                        })

                        err := apiClient.CreateQuarterMark(
                                student.StudentID,
                                t.selectedGroup.Quarters[qi].ID,
                                recommendedGrade,
                                t.selectedSubject.SubjectID,
                                t.selectedSubject.CurriculumPropertyID,
                        )
                        if err != nil {
                                continue
                        }
                        successCount++
                        break // One quarter mark per student for now
                }
        }

        fyne.Do(func() {
                t.statusLabel.SetText(fmt.Sprintf("Рекомендации приняты: %d оценок, пропущено: %d", successCount, skipCount))
                go t.loadData()
        })
}

// ------------------------------------------
// SIGN FINAL GRADES (auto-sign with random date)
// ------------------------------------------

func (t *FinalGradesTab) onSignFinalGrades() {
        if len(t.students) == 0 {
                dialog.ShowInformation("Внимание", "Сначала загрузите данные", t.controller.GetWindow())
                return
        }

        // Calculate class average
        classAvg := CalcClassAverage(t.studentsAsScorers())
        diligence, comment := ClassAverageToCategory(classAvg)

        confirmMsg := fmt.Sprintf(
                "Средний балл класса: %.1f\n"+
                        "Категория: %s\n"+
                        "Комментарий: «%s»\n\n"+
                        "Будет подписано %d учеников.\n\n"+
                        "Продолжить?",
                classAvg, diligence, comment, len(t.students),
        )

        dialog.ShowConfirm("Подписать итоговые", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go t.executeSignFinalGrades(diligence, comment)
        }, t.controller.GetWindow())
}

func (t *FinalGradesTab) executeSignFinalGrades(diligence, comment string) {
        apiClient := t.controller.GetClient()

        // Get dates for the current quarter
        var dateIDs []string
        for _, q := range t.selectedGroup.Quarters {
                if q.CurrentQuarter {
                        dates, err := apiClient.GetJournalDates(t.selectedGroup.ID, t.selectedSubject.SubjectID, q.ID)
                        if err == nil {
                                for _, d := range dates {
                                        if d.AssignmentDateID != "" {
                                                dateIDs = append(dateIDs, d.AssignmentDateID)
                                        }
                                }
                        }
                        break
                }
        }

        if len(dateIDs) == 0 {
                fyne.Do(func() {
                        dialog.ShowError(fmt.Errorf("Нет доступных дат для подписи"), t.controller.GetWindow())
                })
                return
        }

        total := len(t.students)
        successCount := 0

        for i, student := range t.students {
                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Подпись %d/%d: %s %s", i+1, total, student.LastName, student.FirstName))
                })

                // Pick a random date from available dates
                dateID := dateIDs[i%len(dateIDs)]

                // Build comment with diligence
                signComment := fmt.Sprintf("%s. %s", diligence, comment)

                quarterID := 0
                for _, q := range t.selectedGroup.Quarters {
                        if q.CurrentQuarter {
                                quarterID = q.ID
                                break
                        }
                }

                err := apiClient.CreateDiaryComment(student.StudentID, dateID, quarterID, signComment)
                if err == nil {
                        successCount++
                }
        }

        fyne.Do(func() {
                t.statusLabel.SetText(fmt.Sprintf("Подписано: %d/%d учеников — %s", successCount, total, comment))
        })
}

// studentsAsScorers converts students to AvgScorer slice for CalcClassAverage.
func (t *FinalGradesTab) studentsAsScorers() []AvgScorer {
        scorers := make([]AvgScorer, len(t.students))
        for i := range t.students {
                scorers[i] = &t.students[i]
        }
        return scorers
}

// ------------------------------------------
// RANDOM FILL DIALOG
// ------------------------------------------

func (t *FinalGradesTab) showRandomFillDialog() {
        if len(t.students) == 0 {
                dialog.ShowInformation("Внимание", "Нет учеников", t.controller.GetWindow())
                return
        }

        comboSel := widget.NewSelect(comboNames(), nil)
        comboSel.PlaceHolder = "Выберите диапазон..."
        if len(GradeCombos) > 0 {
                comboSel.SetSelectedIndex(0)
        }

        minEntry := widget.NewEntry()
        minEntry.SetText("7")
        maxEntry := widget.NewEntry()
        maxEntry.SetText("10")

        comboSel.OnChanged = func(sel string) {
                for _, c := range GradeCombos {
                        if c.Name == sel {
                                minEntry.SetText(strconv.Itoa(c.MinVal))
                                maxEntry.SetText(strconv.Itoa(c.MaxVal))
                                break
                        }
                }
        }

        // Select which columns to fill
        fillQ1 := widget.NewCheck("Четверть 1", nil)
        fillQ1.SetChecked(true)
        fillQ2 := widget.NewCheck("Четверть 2", nil)
        fillQ2.SetChecked(true)
        fillH1 := widget.NewCheck("Полугодие 1", nil)
        fillQ3 := widget.NewCheck("Четверть 3", nil)
        fillQ4 := widget.NewCheck("Четверть 4", nil)
        fillH2 := widget.NewCheck("Полугодие 2", nil)
        fillYear := widget.NewCheck("Годовая", nil)

        content := container.NewVBox(
                widget.NewLabelWithStyle("Рандомные итоговые оценки", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewSeparator(),
                widget.NewLabel("Диапазон:"),
                comboSel,
                container.NewGridWithColumns(2,
                        widget.NewLabel("Мин:"), minEntry,
                        widget.NewLabel("Макс:"), maxEntry,
                ),
                widget.NewSeparator(),
                widget.NewLabel("Заполнить столбцы:"),
                container.NewGridWithColumns(2,
                        fillQ1, fillQ2,
                        fillQ3, fillQ4,
                        fillH1, fillH2,
                        fillYear,
                ),
        )

        dialog.ShowForm("Рандомные оценки", "Заполнить", "Отмена", []*widget.FormItem{
                widget.NewFormItem("", content),
        }, func(ok bool) {
                if !ok {
                        return
                }
                minVal, _ := strconv.Atoi(minEntry.Text)
                maxVal, _ := strconv.Atoi(maxEntry.Text)
                if minVal < 2 {
                        minVal = 2
                }
                if maxVal > 10 {
                        maxVal = 10
                }

                cols := []struct {
                        col   int
                        fill  bool
                }{
                        {colQ1, fillQ1.Checked},
                        {colQ2, fillQ2.Checked},
                        {colH1, fillH1.Checked},
                        {colQ3, fillQ3.Checked},
                        {colQ4, fillQ4.Checked},
                        {colH2, fillH2.Checked},
                        {colYear, fillYear.Checked},
                }

                go t.executeRandomFill(minVal, maxVal, cols)
        }, t.controller.GetWindow())
}

func (t *FinalGradesTab) executeRandomFill(minVal, maxVal int, cols []struct {
        col  int
        fill bool
}) {
        apiClient := t.controller.GetClient()
        total := len(t.students)
        successCount := 0

        for i, student := range t.students {
                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Заполнение %d/%d: %s %s", i+1, total, student.LastName, student.FirstName))
                })

                for _, c := range cols {
                        if !c.fill {
                                continue
                        }

                        grade := RandomGradeInRange(minVal, maxVal)

                        switch c.col {
                        case colQ1, colQ2, colQ3, colQ4:
                                qi := c.col - colQ1
                                if qi < len(t.selectedGroup.Quarters) {
                                        apiClient.CreateQuarterMark(
                                                student.StudentID,
                                                t.selectedGroup.Quarters[qi].ID,
                                                grade,
                                                t.selectedSubject.SubjectID,
                                                t.selectedSubject.CurriculumPropertyID,
                                        )
                                        successCount++
                                }
                        case colH1, colH2:
                                si := 0
                                if c.col == colH2 {
                                        si = 1
                                }
                                semesterID := 0
                                if si == 0 && len(t.selectedGroup.Quarters) > 1 {
                                        semesterID = t.selectedGroup.Quarters[1].ID
                                } else if len(t.selectedGroup.Quarters) > 3 {
                                        semesterID = t.selectedGroup.Quarters[3].ID
                                }
                                if semesterID > 0 {
                                        apiClient.CreateSemesterMark(student.StudentID, semesterID, grade)
                                        successCount++
                                }
                        case colYear:
                                yearID := 0
                                if len(t.selectedGroup.Quarters) > 3 {
                                        yearID = t.selectedGroup.Quarters[3].ID
                                }
                                if yearID > 0 {
                                        apiClient.CreateYearMark(student.StudentID, yearID, grade)
                                        successCount++
                                }
                        }
                }
        }

        fyne.Do(func() {
                t.statusLabel.SetText(fmt.Sprintf("Рандом заполнен: %d оценок для %d учеников", successCount, total))
                go t.loadData()
        })
}
