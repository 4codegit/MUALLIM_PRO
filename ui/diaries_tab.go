package ui

import (
        "fmt"
        "image/color"
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
// DILIGENCE FILL COMBOS
// ------------------------------------------

// DiligenceCombo defines a named diligence fill preset.
type DiligenceCombo struct {
        Label       string
        Diligences  []string // pool to randomly pick from
        Description string
}

var DiligenceCombos = []DiligenceCombo{
        {
                Label:       "Отлично",
                Diligences:  []string{"Отличный"},
                Description: "Ставит «Отличный» на все пустые даты до сегодняшнего дня",
        },
        {
                Label:       "Отлично && Хорошо",
                Diligences:  []string{"Отличный", "Хорошо"},
                Description: "Рандом: «Отличный» или «Хорошо» на каждую пустую дату",
        },
        {
                Label:       "Хорошо && Удовлетворительный",
                Diligences:  []string{"Хорошо", "Удовлетворительный"},
                Description: "Рандом: «Хорошо» или «Удовлетворительный» на каждую пустую дату",
        },
        {
                Label:       "Неудовлетворительный",
                Diligences:  []string{"Неудовлетворительно"},
                Description: "Ставит «Неудовлетворительно» на все пустые даты до сегодняшнего дня",
        },
}

// ------------------------------------------
// DIARIES TAB — Simplified
// ------------------------------------------

// DiariesTab manages the Diaries (Дневник) tab.
//
// Simplified workflow:
//  1. Select student from the list
//  2. Press one of the diligence buttons:
//     [Отлично] [Отлично && Хорошо] [Хорошо && Удовлетворительный] [Неудовлетворительный]
//  3. System fills random diligence marks + behavior comments on all empty
//     diary dates up to the current date for the selected student.
type DiariesTab struct {
        controller Controller
        container  *fyne.Container

        // Filters
        classSel   *widget.Select
        subjectSel *widget.Select
        quarterSel *widget.Select

        // State
        journalOpts     *client.JournalOptions
        selectedGroup   *client.JournalGroup
        selectedSubject *client.Subject
        selectedQuarter *client.Quarter
        students        []client.Student
        dates           []client.Day

        // UI
        selectedStudentIdx int // -1 = none selected
        studentsList       *widget.List
        statusLabel        *widget.Label
        diligenceButtons   []*widget.Button
}

// NewDiariesTab creates a new DiariesTab.
func NewDiariesTab(c Controller) *DiariesTab {
        dt := &DiariesTab{
                controller:        c,
                statusLabel:       widget.NewLabel("Выберите класс, предмет и четверть"),
                selectedStudentIdx: -1,
        }
        dt.buildUI()
        go dt.loadJournalOptions()
        return dt
}

// Container returns the root container for this tab.
func (dt *DiariesTab) Container() fyne.CanvasObject {
        return dt.container
}

// buildUI creates the full UI layout for the diaries tab.
func (dt *DiariesTab) buildUI() {
        dt.classSel = widget.NewSelect([]string{}, dt.onClassSelected)
        dt.classSel.PlaceHolder = "Класс..."

        dt.subjectSel = widget.NewSelect([]string{}, dt.onSubjectSelected)
        dt.subjectSel.PlaceHolder = "Предмет..."

        dt.quarterSel = widget.NewSelect([]string{}, dt.onQuarterSelected)
        dt.quarterSel.PlaceHolder = "Четверть..."

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go dt.loadData()
        })

        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"),
                dt.classSel,
                dt.subjectSel,
                dt.quarterSel,
                refreshBtn,
        )

        // Diligence action buttons — disabled until a student is selected
        dt.diligenceButtons = make([]*widget.Button, len(DiligenceCombos))
        btnColors := []color.Color{
                color.NRGBA{R: 22, G: 163, B: 74, A: 255},  // green — Отлично
                color.NRGBA{R: 37, G: 99, B: 235, A: 255},   // blue — Отлично && Хорошо
                color.NRGBA{R: 217, G: 119, B: 6, A: 255},    // orange — Хорошо && Удовл.
                color.NRGBA{R: 220, G: 38, B: 38, A: 255},    // red — Неудовлетворительный
        }

        for i, combo := range DiligenceCombos {
                idx := i // capture for closure
                btn := widget.NewButton(combo.Label, func() {
                        dt.onDiligenceButton(idx)
                })
                if i == 0 {
                        btn.Importance = widget.HighImportance
                }
                btn.Disable()
                dt.diligenceButtons[i] = btn
                _ = btnColors[i] // color applied via Importance for now
        }

        actionRow := container.NewHBox()
        for i := range dt.diligenceButtons {
                actionRow.Add(dt.diligenceButtons[i])
        }

        placeholder := widget.NewLabelWithStyle(
                "Выберите класс, предмет и четверть\n\n"+
                        "Затем выберите ученика и нажмите кнопку прилежания:\n"+
                        "«Отлично», «Отлично && Хорошо» и т.д.\n\n"+
                        "Система автоматически заполнит дневник до текущей даты.",
                fyne.TextAlignCenter,
                fyne.TextStyle{Italic: true},
        )

        dt.container = container.NewBorder(
                container.NewVBox(filterRow, actionRow, widget.NewSeparator()),
                dt.statusLabel,
                nil,
                nil,
                placeholder,
        )
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (dt *DiariesTab) loadJournalOptions() {
        dt.statusLabel.SetText("Загрузка списка классов...")
        opts, err := dt.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v", err))
                })
                return
        }

        dt.journalOpts = opts

        classNames := make([]string, len(opts.Groups))
        for i, g := range opts.Groups {
                classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
        }

        fyne.Do(func() {
                dt.classSel.Options = classNames
                dt.classSel.Refresh()
                dt.statusLabel.SetText("Выберите класс, предмет и четверть")
                if len(classNames) > 0 {
                        dt.classSel.SetSelectedIndex(0)
                }
        })
}

// ------------------------------------------
// FILTER HANDLERS
// ------------------------------------------

func (dt *DiariesTab) onClassSelected(selected string) {
        if dt.journalOpts == nil {
                return
        }

        for i, g := range dt.journalOpts.Groups {
                if fmt.Sprintf("%d %s", g.Number, g.Name) == selected {
                        dt.selectedGroup = &dt.journalOpts.Groups[i]
                        break
                }
        }
        if dt.selectedGroup == nil {
                return
        }

        dt.selectedSubject = nil
        dt.selectedQuarter = nil
        dt.students = nil
        dt.dates = nil
        dt.selectedStudentIdx = -1
        dt.setDiligenceButtonsEnabled(false)

        subjectNames := make([]string, len(dt.selectedGroup.Subjects))
        for i, s := range dt.selectedGroup.Subjects {
                subjectNames[i] = s.SubjectName
        }

        quarterNames := make([]string, len(dt.selectedGroup.Quarters))
        for i, q := range dt.selectedGroup.Quarters {
                quarterNames[i] = q.Name
        }

        fyne.Do(func() {
                dt.subjectSel.Options = subjectNames
                dt.subjectSel.Refresh()
                dt.quarterSel.Options = quarterNames
                dt.quarterSel.Refresh()

                if len(subjectNames) > 0 {
                        dt.subjectSel.SetSelectedIndex(0)
                }
                for i, q := range dt.selectedGroup.Quarters {
                        if q.CurrentQuarter {
                                dt.quarterSel.SetSelectedIndex(i)
                                break
                        }
                }
        })
}

func (dt *DiariesTab) onSubjectSelected(selected string) {
        if dt.selectedGroup == nil {
                return
        }
        for i, s := range dt.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        dt.selectedSubject = &dt.selectedGroup.Subjects[i]
                        break
                }
        }
        dt.tryLoadData()
}

func (dt *DiariesTab) onQuarterSelected(selected string) {
        if dt.selectedGroup == nil {
                return
        }
        for i, q := range dt.selectedGroup.Quarters {
                if q.Name == selected {
                        dt.selectedQuarter = &dt.selectedGroup.Quarters[i]
                        break
                }
        }
        dt.tryLoadData()
}

func (dt *DiariesTab) tryLoadData() {
        if dt.selectedGroup != nil && dt.selectedSubject != nil && dt.selectedQuarter != nil {
                go dt.loadData()
        }
}

func (dt *DiariesTab) loadData() {
        if dt.selectedGroup == nil || dt.selectedSubject == nil || dt.selectedQuarter == nil {
                return
        }

        fyne.Do(func() {
                dt.statusLabel.SetText("Загрузка данных дневника...")
        })

        apiClient := dt.controller.GetClient()
        gID := dt.selectedGroup.ID
        sID := dt.selectedSubject.SubjectID
        qID := dt.selectedQuarter.ID

        students, errS := apiClient.GetJournalStudents(gID, sID, qID)
        dates, errD := apiClient.GetJournalDates(gID, sID, qID)

        fyne.Do(func() {
                if errS != nil {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", errS))
                        dialog.ShowError(fmt.Errorf("Ошибка загрузки: %v", errS), dt.controller.GetWindow())
                        return
                }
                if errD != nil {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", errD))
                        dialog.ShowError(fmt.Errorf("Ошибка дат: %v", errD), dt.controller.GetWindow())
                        return
                }

                dt.students = students
                dt.dates = dates
                dt.selectedStudentIdx = -1
                dt.setDiligenceButtonsEnabled(false)

                if len(students) == 0 {
                        dt.statusLabel.SetText("Нет учеников")
                } else {
                        dt.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников, %d дат. Выберите ученика.", len(students), len(dates)))
                }

                dt.rebuildStudentsList()
        })
}

// ------------------------------------------
// STUDENTS LIST
// ------------------------------------------

func (dt *DiariesTab) rebuildStudentsList() {
        if len(dt.students) == 0 {
                dt.container.Objects = []fyne.CanvasObject{
                        container.NewBorder(
                                dt.buildTopBar(),
                                dt.statusLabel,
                                nil, nil,
                                widget.NewLabelWithStyle("Нет учеников", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        ),
                }
                dt.container.Refresh()
                return
        }

        dt.studentsList = widget.NewList(
                func() int { return len(dt.students) },
                func() fyne.CanvasObject {
                        numLabel := widget.NewLabel("")
                        numLabel.TextStyle = fyne.TextStyle{Bold: true}
                        nameLabel := widget.NewLabel("")
                        nameLabel.TextStyle = fyne.TextStyle{Bold: true}
                        avgLabel := widget.NewLabel("")
                        marksLabel := widget.NewLabel("")
                        statusIcon := canvas.NewText("", color.NRGBA{R: 22, G: 163, B: 74, A: 255})
                        statusIcon.TextSize = 14
                        statusIcon.TextStyle = fyne.TextStyle{Bold: true}

                        return container.NewBorder(nil, nil,
                                container.NewHBox(numLabel, nameLabel),
                                statusIcon,
                                container.NewVBox(avgLabel, marksLabel),
                        )
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        if id < 0 || id >= len(dt.students) {
                                return
                        }
                        student := dt.students[id]

                        border := cell.(*fyne.Container)
                        leftBox := border.Objects[0].(*fyne.Container)
                        rightBox := border.Objects[1].(*fyne.Container)
                        statusIcon := border.Objects[2].(*canvas.Text)

                        numLabel := leftBox.Objects[0].(*widget.Label)
                        nameLabel := leftBox.Objects[1].(*widget.Label)
                        avgLabel := rightBox.Objects[0].(*widget.Label)
                        marksLabel := rightBox.Objects[1].(*widget.Label)

                        numLabel.SetText(fmt.Sprintf("%d.", id+1))
                        nameLabel.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))

                        if student.AverageScore != "" && student.AverageScore != "0.0" {
                                avgLabel.SetText(fmt.Sprintf("Ср: %s", student.AverageScore))
                        } else {
                                avgLabel.SetText("")
                        }

                        markCount := 0
                        for _, sm := range student.SubjectMarks {
                                if sm.ShortName != "" && sm.ShortName != "—" {
                                        markCount++
                                }
                        }
                        marksLabel.SetText(fmt.Sprintf("Оценок: %d/%d", markCount, len(dt.dates)))

                        // Show selected indicator
                        if id == dt.selectedStudentIdx {
                                statusIcon.Text = "▶"
                                statusIcon.Color = color.NRGBA{R: 56, G: 189, B: 248, A: 255}
                        } else {
                                statusIcon.Text = ""
                        }
                        statusIcon.Refresh()
                },
        )

        dt.studentsList.OnSelected = func(id widget.ListItemID) {
                dt.studentsList.Unselect(id)
                dt.selectedStudentIdx = id
                dt.setDiligenceButtonsEnabled(true)
                dt.studentsList.Refresh()
                student := dt.students[id]
                dt.statusLabel.SetText(fmt.Sprintf("Выбран: %s %s — нажмите кнопку прилежания", student.LastName, student.FirstName))
        }

        dt.container.Objects = []fyne.CanvasObject{
                container.NewBorder(
                        dt.buildTopBar(),
                        dt.statusLabel,
                        nil, nil,
                        dt.studentsList,
                ),
        }
        dt.container.Refresh()
}

func (dt *DiariesTab) buildTopBar() *fyne.Container {
        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"), dt.classSel, dt.subjectSel, dt.quarterSel,
                widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { go dt.loadData() }),
        )
        actionRow := container.NewHBox()
        for _, btn := range dt.diligenceButtons {
                actionRow.Add(btn)
        }
        return container.NewVBox(filterRow, actionRow, widget.NewSeparator())
}

func (dt *DiariesTab) setDiligenceButtonsEnabled(enabled bool) {
        for _, btn := range dt.diligenceButtons {
                if enabled {
                        btn.Enable()
                } else {
                        btn.Disable()
                }
        }
}

// ------------------------------------------
// DILIGENCE BUTTON HANDLER
// ------------------------------------------

func (dt *DiariesTab) onDiligenceButton(comboIdx int) {
        if dt.selectedStudentIdx < 0 || dt.selectedStudentIdx >= len(dt.students) {
                dialog.ShowInformation("Внимание", "Сначала выберите ученика из списка", dt.controller.GetWindow())
                return
        }
        if dt.selectedQuarter == nil || len(dt.dates) == 0 {
                dialog.ShowInformation("Внимание", "Нет дат для заполнения", dt.controller.GetWindow())
                return
        }

        combo := DiligenceCombos[comboIdx]
        student := dt.students[dt.selectedStudentIdx]

        // Count empty dates up to today
        today := time.Now().Format("2006-01-02")
        emptyCount := 0
        for _, date := range dt.dates {
                if date.AssignmentDate > today {
                        continue // skip future dates
                }
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

        if emptyCount == 0 {
                dialog.ShowInformation("Готово", fmt.Sprintf("У %s %s нет пустых дат до сегодняшнего дня", student.LastName, student.FirstName), dt.controller.GetWindow())
                return
        }

        confirmMsg := fmt.Sprintf(
                "Ученик: %s %s\n"+
                        "Действие: %s\n"+
                        "Описание: %s\n\n"+
                        "Пустых дат до сегодняшнего дня: %d\n\n"+
                        "Будут выставлены оценки прилежания и комментарии.\n"+
                        "Продолжить?",
                student.LastName, student.FirstName,
                combo.Label, combo.Description, emptyCount,
        )

        dialog.ShowConfirm("Заполнить дневник", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go dt.executeDiligenceFill(dt.selectedStudentIdx, comboIdx)
        }, dt.controller.GetWindow())
}

// executeDiligenceFill fills diary diligence marks + behavior comments
// for the selected student on all empty dates up to today.
func (dt *DiariesTab) executeDiligenceFill(studentIdx, comboIdx int) {
        if studentIdx < 0 || studentIdx >= len(dt.students) {
                return
        }

        student := dt.students[studentIdx]
        combo := DiligenceCombos[comboIdx]
        apiClient := dt.controller.GetClient()
        qID := dt.selectedQuarter.ID
        today := time.Now().Format("2006-01-02")

        successCount := 0
        skipCount := 0

        for _, date := range dt.dates {
                // Only fill dates up to today
                if date.AssignmentDate > today {
                        continue
                }

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

                // Pick random diligence from the combo pool
                diligence := combo.Diligences[RandomGradeInRange(0, len(combo.Diligences)-1)]

                // Generate a matching behavior comment
                var comment string
                switch diligence {
                case "Отличный":
                        comment = SequentialBehaviorComment(BehaviorPraise, successCount)
                case "Хорошо":
                        comment = SequentialBehaviorComment(BehaviorNeutral, successCount)
                case "Удовлетворительный":
                        comment = SequentialBehaviorComment(BehaviorMixed, successCount)
                case "Неудовлетворительно":
                        comment = SequentialBehaviorComment(BehaviorComplaint, successCount)
                default:
                        comment = "Комментарий учителя"
                }

                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ставлю «%s»: %s %s — %s (%d/%d)",
                                diligence, student.LastName, student.FirstName, date.AssignmentDate[5:],
                                successCount+1, len(dt.dates)-skipCount))
                })

                err := apiClient.CreateDiaryComment(student.StudentID, date.AssignmentDateID, qID, comment)
                if err == nil {
                        successCount++
                }
        }

        fyne.Do(func() {
                dt.statusLabel.SetText(fmt.Sprintf("Готово: %d записей для %s %s (пропущено: %d)",
                        successCount, student.LastName, student.FirstName, skipCount))
        })
}

// Refresh updates the tab with new data from the dashboard context.
func (dt *DiariesTab) Refresh(students []client.Student, group *client.JournalGroup, subject *client.Subject, quarter *client.Quarter) {
        if group != nil && (dt.selectedGroup == nil || dt.selectedGroup.ID != group.ID) {
                dt.selectedGroup = group
        }
        if subject != nil {
                dt.selectedSubject = subject
        }
        if quarter != nil {
                dt.selectedQuarter = quarter
        }
        if dt.selectedGroup != nil && dt.selectedSubject != nil && dt.selectedQuarter != nil {
                go dt.loadData()
        }
}
