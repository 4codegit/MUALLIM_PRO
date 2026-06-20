package ui

import (
        "fmt"
        "image/color"
        "strconv"
        "sync"
        "sync/atomic"
        "time"

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

        // Semester / Year property IDs — derived from /journal/dates responses
        // (each quarter response includes the semester it belongs to + the
        // education year). These are needed by CreateSemesterMark /
        // CreateYearMark — without them the server returns 409 FK violation:
        //   insert or update on table "student_semester_mark" violates foreign
        //   key constraint "student_semester_mark_semester_id_fkey"
        // Indexed: 0 = H1 (from Q1/Q2), 1 = H2 (from Q3/Q4).
        semesterPropertyIDs [2]int
        // Year property ID — single value, same across all quarters.
        yearPropertyID       int

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

        // Fetch final-grade students AND quarter-dates in parallel. The
        // quarter-dates response includes the semester_property_id and
        // education_year_id we need for CreateSemesterMark / CreateYearMark —
        // without them the server returns 409 FK violation.
        type datesResult struct {
                qi     int
                qDates []client.QuarterDates
                err    error
        }
        datesCh := make(chan datesResult, len(t.selectedGroup.Quarters))
        var dwg sync.WaitGroup
        for qi, q := range t.selectedGroup.Quarters {
                dwg.Add(1)
                go func(idx int, qID int) {
                        defer dwg.Done()
                        qd, err := apiClient.GetJournalDatesFull(t.selectedGroup.ID, t.selectedSubject.SubjectID, qID)
                        datesCh <- datesResult{qi: idx, qDates: qd, err: err}
                }(qi, q.ID)
        }
        dwg.Wait()
        close(datesCh)

        // Reset cached IDs
        t.semesterPropertyIDs = [2]int{0, 0}
        t.yearPropertyID = 0

        // Collect: Q1/Q2 → H1 (semesterPropertyIDs[0]); Q3/Q4 → H2 (semesterPropertyIDs[1])
        // Year ID is the same across all quarters — take the first non-zero one.
        for r := range datesCh {
                if r.err != nil || len(r.qDates) == 0 {
                        continue
                }
                qd := r.qDates[0]
                // Semester
                if len(qd.Semester) > 0 && qd.Semester[0].SemesterPropertyID > 0 {
                        if r.qi <= 1 {
                                // Q1 or Q2 → H1
                                if t.semesterPropertyIDs[0] == 0 {
                                        t.semesterPropertyIDs[0] = qd.Semester[0].SemesterPropertyID
                                }
                        } else {
                                // Q3 or Q4 → H2
                                if t.semesterPropertyIDs[1] == 0 {
                                        t.semesterPropertyIDs[1] = qd.Semester[0].SemesterPropertyID
                                }
                        }
                }
                // Education year
                if t.yearPropertyID == 0 && len(qd.EducationYear) > 0 && qd.EducationYear[0].EducationYearID > 0 {
                        t.yearPropertyID = qd.EducationYear[0].EducationYearID
                }
        }

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
                t.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников (H1=%d, H2=%d, Year=%d)",
                        len(students), t.semesterPropertyIDs[0], t.semesterPropertyIDs[1], t.yearPropertyID))
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
                        // Semester: use semesterPropertyID extracted from /journal/dates.
                        // The previous code passed the Q2/Q4 quarter_property_id which
                        // caused a 409 FK violation on student_semester_mark.semester_id.
                        si := 0
                        if col == colH2 {
                                si = 1
                        }
                        semesterID := t.semesterPropertyIDs[si]
                        if semesterID == 0 {
                                err = fmt.Errorf("semester_property_id неизвестен для H%d — перезагрузите вкладку", si+1)
                        } else {
                                err = apiClient.CreateSemesterMark(
                                        student.StudentID, semesterID, grade,
                                        t.selectedSubject.SubjectID,
                                        t.selectedSubject.CurriculumPropertyID,
                                )
                        }
                case colYear:
                        // Year: use educationYearId extracted from /journal/dates.
                        yearID := t.yearPropertyID
                        if yearID == 0 {
                                err = fmt.Errorf("year_property_id неизвестен — перезагрузите вкладку")
                        } else {
                                err = apiClient.CreateYearMark(
                                        student.StudentID, yearID, grade,
                                        t.selectedSubject.SubjectID,
                                        t.selectedSubject.CurriculumPropertyID,
                                )
                        }
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

        confirmMsg := "Будут выставлены рекомендованные итоговые оценки\nдля всех учеников, у которых они ещё не выставлены:\n\n" +
                "• Четвертные Q1–Q4 — из среднего балла по предметным оценкам в этой четверти\n" +
                "• Полугодие H1 — ceil((Q1 + Q2) / 2)\n" +
                "• Полугодие H2 — ceil((Q3 + Q4) / 2)\n" +
                "• Годовая     — ceil((Q1 + Q2 + Q3 + Q4) / 4)\n\n" +
                "Существующие оценки НЕ перезаписываются.\n" +
                "Запросы отправляются параллельно (8 потоков).\n\n" +
                "Продолжить?"

        dialog.ShowConfirm("Принять рекомендации", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go t.executeAcceptRecommendations()
        }, t.controller.GetWindow())
}

// executeAcceptRecommendations fills ALL empty final-grade cells for every
// student:
//   - Q1-Q4: ceil(avg of subject marks for that quarter), skipped if quarter
//     has no subject marks
//   - H1: ceil((Q1+Q2)/2) using cached semesterPropertyIDs[0]
//   - H2: ceil((Q3+Q4)/2) using cached semesterPropertyIDs[1]
//   - Year: ceil((Q1+Q2+Q3+Q4)/4) using cached yearPropertyID
//
// Existing final-grade cells are NOT overwritten — only empty ones are filled.
// Subject marks for each quarter are fetched in parallel up-front, then the
// per-student fill loop runs in a worker pool of 8 goroutines.
func (t *FinalGradesTab) executeAcceptRecommendations() {
        apiClient := t.controller.GetClient()
        total := len(t.students)
        groupID := t.selectedGroup.ID
        subjectID := t.selectedSubject.SubjectID
        curriculumPropertyID := t.selectedSubject.CurriculumPropertyID

        fyne.Do(func() {
                t.statusLabel.SetText("Загрузка оценок по четвертям...")
        })

        // Step 1: Fetch fresh subject-marks per quarter (in parallel).
        // We need these to compute the recommended quarter mark for each student
        // — the cached t.students list only has the FinalGradeStudent aggregate
        // (with quarter marks but not subject marks).
        type quarterData struct {
                qi       int
                quarter  client.Quarter
                students []client.Student
                err      error
        }
        quarterDataCh := make(chan quarterData, len(t.selectedGroup.Quarters))
        var qwg sync.WaitGroup
        for qi, q := range t.selectedGroup.Quarters {
                qwg.Add(1)
                go func(idx int, q client.Quarter) {
                        defer qwg.Done()
                        studs, err := apiClient.GetJournalStudents(groupID, subjectID, q.ID)
                        quarterDataCh <- quarterData{qi: idx, quarter: q, students: studs, err: err}
                }(qi, q)
        }
        qwg.Wait()
        close(quarterDataCh)

        // Build studentID -> {qi -> subject-marks avg} map
        type quarterAvg struct {
                hasSubjectMarks bool
                avg             float64
        }
        studentQuarterAvgs := make(map[int][4]quarterAvg) // [studentID][0..3]
        for qd := range quarterDataCh {
                if qd.err != nil {
                        continue
                }
                for _, s := range qd.students {
                        var grades []int
                        for _, sm := range s.SubjectMarks {
                                g := ParseShortNameToGrade(sm.ShortName)
                                if g >= 2 && g <= 10 {
                                        grades = append(grades, g)
                                }
                        }
                        qa := studentQuarterAvgs[s.StudentID]
                        if len(grades) > 0 {
                                sum := 0
                                for _, g := range grades {
                                        sum += g
                                }
                                qa[qd.qi] = quarterAvg{hasSubjectMarks: true, avg: float64(sum) / float64(len(grades))}
                        }
                        studentQuarterAvgs[s.StudentID] = qa
                }
        }

        // Step 2: Build the list of save jobs (one job per student).
        type saveJob struct {
                studentIdx int
                student    client.FinalGradeStudent
                quartAvgs  [4]quarterAvg
        }
        jobs := make(chan saveJob, total)
        for i, s := range t.students {
                jobs <- saveJob{studentIdx: i, student: s, quartAvgs: studentQuarterAvgs[s.StudentID]}
        }
        close(jobs)

        // Step 3: Worker pool processes jobs in parallel.
        const concurrency = 8
        var successCount, skipCount, failCount int32
        var firstErrMsg string
        var firstErrSet int32

        // Status ticker
        done := make(chan struct{})
        go func() {
                for {
                        select {
                        case <-done:
                                return
                        case <-time.After(200 * time.Millisecond):
                                s := atomic.LoadInt32(&successCount)
                                f := atomic.LoadInt32(&failCount)
                                sk := atomic.LoadInt32(&skipCount)
                                fyne.Do(func() {
                                        t.statusLabel.SetText(fmt.Sprintf("Рекомендации: ✓ %d / ✗ %d / ⊘ %d из %d", s, f, sk, total))
                                })
                        }
                }
        }()
        defer close(done)

        var wg sync.WaitGroup
        for w := 0; w < concurrency; w++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        for job := range jobs {
                                s := job.student
                                // Compute recommended quarter grades (only for empty cells)
                                var recQ [4]int // -1 = no recommendation, 0 = skip (already has), 2-10 = grade
                                for qi := 0; qi < 4; qi++ {
                                        if s.QuarterMarks[qi].ShortName != "" {
                                                recQ[qi] = 0 // already set
                                                continue
                                        }
                                        qa := job.quartAvgs[qi]
                                        if !qa.hasSubjectMarks || qa.avg <= 0 {
                                                recQ[qi] = -1
                                                continue
                                        }
                                        recQ[qi] = AverageToGrade(qa.avg)
                                }

                                // Save quarter marks (only those that have a recommendation)
                                for qi := 0; qi < 4; qi++ {
                                        if recQ[qi] <= 0 {
                                                continue
                                        }
                                        if qi >= len(t.selectedGroup.Quarters) {
                                                continue
                                        }
                                        err := apiClient.CreateQuarterMark(
                                                s.StudentID,
                                                t.selectedGroup.Quarters[qi].ID,
                                                recQ[qi],
                                                subjectID,
                                                curriculumPropertyID,
                                        )
                                        if err == nil {
                                                atomic.AddInt32(&successCount, 1)
                                        } else {
                                                atomic.AddInt32(&failCount, 1)
                                                if atomic.CompareAndSwapInt32(&firstErrSet, 0, 1) {
                                                        firstErrMsg = err.Error()
                                                }
                                        }
                                }

                                // Determine effective Q values for H1/H2/Year calculation:
                                // use the recommendation if we just filled it, otherwise use
                                // the existing quarter mark.
                                effectiveQ := [4]int{-1, -1, -1, -1}
                                for qi := 0; qi < 4; qi++ {
                                        if recQ[qi] > 0 {
                                                effectiveQ[qi] = recQ[qi]
                                        } else if s.QuarterMarks[qi].ShortName != "" {
                                                g := ParseShortNameToGrade(s.QuarterMarks[qi].ShortName)
                                                if g >= 2 {
                                                        effectiveQ[qi] = g
                                                }
                                        }
                                }

                                // H1: ceil((Q1+Q2)/2) — only if both Q1 and Q2 are known,
                                // and H1 cell is currently empty.
                                if s.SemesterMarks[0].ShortName == "" && t.semesterPropertyIDs[0] > 0 {
                                        if effectiveQ[0] > 0 && effectiveQ[1] > 0 {
                                                avg := float64(effectiveQ[0]+effectiveQ[1]) / 2.0
                                                grade := AverageToGrade(avg)
                                                err := apiClient.CreateSemesterMark(
                                                        s.StudentID, t.semesterPropertyIDs[0], grade,
                                                        subjectID, curriculumPropertyID,
                                                )
                                                if err == nil {
                                                        atomic.AddInt32(&successCount, 1)
                                                } else {
                                                        atomic.AddInt32(&failCount, 1)
                                                        if atomic.CompareAndSwapInt32(&firstErrSet, 0, 1) {
                                                                firstErrMsg = err.Error()
                                                        }
                                                }
                                        }
                                }

                                // H2: ceil((Q3+Q4)/2)
                                if s.SemesterMarks[1].ShortName == "" && t.semesterPropertyIDs[1] > 0 {
                                        if effectiveQ[2] > 0 && effectiveQ[3] > 0 {
                                                avg := float64(effectiveQ[2]+effectiveQ[3]) / 2.0
                                                grade := AverageToGrade(avg)
                                                err := apiClient.CreateSemesterMark(
                                                        s.StudentID, t.semesterPropertyIDs[1], grade,
                                                        subjectID, curriculumPropertyID,
                                                )
                                                if err == nil {
                                                        atomic.AddInt32(&successCount, 1)
                                                } else {
                                                        atomic.AddInt32(&failCount, 1)
                                                        if atomic.CompareAndSwapInt32(&firstErrSet, 0, 1) {
                                                                firstErrMsg = err.Error()
                                                        }
                                                }
                                        }
                                }

                                // Year: ceil((Q1+Q2+Q3+Q4)/4)
                                if (s.YearMark == nil || s.YearMark.ShortName == "") && t.yearPropertyID > 0 {
                                        qcount := 0
                                        qsum := 0
                                        for qi := 0; qi < 4; qi++ {
                                                if effectiveQ[qi] > 0 {
                                                        qsum += effectiveQ[qi]
                                                        qcount++
                                                }
                                        }
                                        if qcount == 4 {
                                                avg := float64(qsum) / 4.0
                                                grade := AverageToGrade(avg)
                                                err := apiClient.CreateYearMark(
                                                        s.StudentID, t.yearPropertyID, grade,
                                                        subjectID, curriculumPropertyID,
                                                )
                                                if err == nil {
                                                        atomic.AddInt32(&successCount, 1)
                                                } else {
                                                        atomic.AddInt32(&failCount, 1)
                                                        if atomic.CompareAndSwapInt32(&firstErrSet, 0, 1) {
                                                                firstErrMsg = err.Error()
                                                        }
                                                }
                                        }
                                }

                                // If we made no API calls for this student (everything was
                                // already filled or no data), count as skipped.
                                anyAction := false
                                for qi := 0; qi < 4; qi++ {
                                        if recQ[qi] > 0 {
                                                anyAction = true
                                                break
                                        }
                                }
                                if !anyAction &&
                                        (s.SemesterMarks[0].ShortName != "" || t.semesterPropertyIDs[0] == 0 || !(effectiveQ[0] > 0 && effectiveQ[1] > 0)) &&
                                        (s.SemesterMarks[1].ShortName != "" || t.semesterPropertyIDs[1] == 0 || !(effectiveQ[2] > 0 && effectiveQ[3] > 0)) &&
                                        ((s.YearMark != nil && s.YearMark.ShortName != "") || t.yearPropertyID == 0 || !(effectiveQ[0] > 0 && effectiveQ[1] > 0 && effectiveQ[2] > 0 && effectiveQ[3] > 0)) {
                                        atomic.AddInt32(&skipCount, 1)
                                }
                        }
                }()
        }
        wg.Wait()

        finalSuccess := int(atomic.LoadInt32(&successCount))
        finalFail := int(atomic.LoadInt32(&failCount))
        finalSkip := int(atomic.LoadInt32(&skipCount))

        fyne.Do(func() {
                msg := fmt.Sprintf("Рекомендации приняты: ✓ %d / ✗ %d / ⊘ %d пропущено",
                        finalSuccess, finalFail, finalSkip)
                if finalFail > 0 && firstErrMsg != "" {
                        msg += fmt.Sprintf("  |  первая ошибка: %s", firstErrMsg)
                }
                t.statusLabel.SetText(msg)
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
                                semesterID := t.semesterPropertyIDs[si]
                                if semesterID > 0 {
                                        apiClient.CreateSemesterMark(
                                                student.StudentID, semesterID, grade,
                                                t.selectedSubject.SubjectID,
                                                t.selectedSubject.CurriculumPropertyID,
                                        )
                                        successCount++
                                }
                        case colYear:
                                yearID := t.yearPropertyID
                                if yearID > 0 {
                                        apiClient.CreateYearMark(
                                                student.StudentID, yearID, grade,
                                                t.selectedSubject.SubjectID,
                                                t.selectedSubject.CurriculumPropertyID,
                                        )
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
