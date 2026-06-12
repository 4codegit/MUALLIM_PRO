package ui

import (
        "fmt"
        "image/color"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/layout"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/internal/config"
        "github.com/4codegit/edonish-auto/internal/engine"
)

// AutoGradePage holds the auto-grade page UI components.
type AutoGradePage struct {
        app *App

        // Settings
        classSelect     *widget.Select
        subjectSelect   *widget.Select
        quarterSelect   *widget.Select
        minGradeEntry   *widget.Entry
        maxGradeEntry   *widget.Entry
        fillEmptyChk    *widget.Check

        // What to fill
        fillDailyChk    *widget.Check
        fillQuarterChk  *widget.Check
        fillSemesterChk *widget.Check
        fillYearChk     *widget.Check

        // Actions
        analyzeBtn *widget.Button
        startBtn   *widget.Button
        stopBtn    *widget.Button
        limitsBtn  *widget.Button

        // Progress
        progressBar   *widget.ProgressBar
        progressLabel *canvas.Text
        statsLabel    *canvas.Text

        // Results
        resultsEntry *widget.Entry

        // Header
        headerTitle *canvas.Text
}

// NewAutoGradePage creates a new auto-grade page.
func NewAutoGradePage(app *App) *AutoGradePage {
        return &AutoGradePage{app: app}
}

// makeColoredText creates a canvas.Text with the given color.
func makeColoredText(text string, col color.Color, bold bool) *canvas.Text {
        t := canvas.NewText(text, col)
        t.TextStyle = fyne.TextStyle{Bold: bold}
        return t
}

// makeSmallColoredText creates a small canvas.Text with the given color.
func makeSmallColoredText(text string, col color.Color) *canvas.Text {
        t := canvas.NewText(text, col)
        t.TextSize = 12
        return t
}

// hexColor parses a hex color string like "#2563EB" into a color.NRGBA.
func hexColor(hex string) color.NRGBA {
        var r, g, b uint8
        fmt.Sscanf(hex, "#%02X%02X%02X", &r, &g, &b)
        return color.NRGBA{R: r, G: g, B: b, A: 255}
}

// Build creates the auto-grade view and returns the root container.
func (p *AutoGradePage) Build() fyne.CanvasObject {
        // ── Colorful header with app icon ─────────────────────────
        p.headerTitle = canvas.NewText("Авто-оценки", hexColor(config.ColorPrimary))
        p.headerTitle.TextStyle = fyne.TextStyle{Bold: true}
        p.headerTitle.TextSize = 20

        headerIcon := widget.NewIcon(theme.MediaPlayIcon())
        headerSubtitle := canvas.NewText(fmt.Sprintf("v%s • Умный режим оценки", config.AppVersion), hexColor(config.ColorMuted))
        headerSubtitle.TextSize = 12

        headerBar := widget.NewCard("", "", container.NewVBox(
                container.NewHBox(
                        headerIcon,
                        p.headerTitle,
                        headerSubtitle,
                        layout.NewSpacer(),
                ),
        ))

        // ── Settings ────────────────────────────────────────────────
        p.classSelect = widget.NewSelect([]string{"Все классы"}, func(s string) {})
        p.classSelect.PlaceHolder = "Выберите класс"

        p.subjectSelect = widget.NewSelect([]string{"Все предметы"}, func(s string) {})
        p.subjectSelect.PlaceHolder = "Выберите предмет"

        p.quarterSelect = widget.NewSelect([]string{"Все четверти"}, func(s string) {})
        p.quarterSelect.PlaceHolder = "Выберите четверть"

        p.minGradeEntry = widget.NewEntry()
        p.minGradeEntry.SetText(fmt.Sprintf("%d", config.MinGrade))

        p.maxGradeEntry = widget.NewEntry()
        p.maxGradeEntry.SetText(fmt.Sprintf("%d", config.MaxGrade))

        p.fillEmptyChk = widget.NewCheck("Только пустые ячейки", nil)
        p.fillEmptyChk.SetChecked(true)

        p.fillDailyChk = widget.NewCheck("Дневные оценки", nil)
        p.fillDailyChk.SetChecked(true)

        p.fillQuarterChk = widget.NewCheck("Четвертные оценки", nil)
        p.fillQuarterChk.SetChecked(true)

        p.fillSemesterChk = widget.NewCheck("Семестровые оценки", nil)
        p.fillSemesterChk.SetChecked(true)

        p.fillYearChk = widget.NewCheck("Годовые оценки", nil)
        p.fillYearChk.SetChecked(true)

        classLabel := canvas.NewText("Класс", hexColor(config.ColorPrimary))
        classLabel.TextStyle = fyne.TextStyle{Bold: true}
        subjectLabel := canvas.NewText("Предмет", hexColor(config.ColorPrimary))
        subjectLabel.TextStyle = fyne.TextStyle{Bold: true}
        quarterLabel := canvas.NewText("Четверть", hexColor(config.ColorPrimary))
        quarterLabel.TextStyle = fyne.TextStyle{Bold: true}
        rangeLabel := canvas.NewText("Диапазон оценок", hexColor(config.ColorPrimary))
        rangeLabel.TextStyle = fyne.TextStyle{Bold: true}

        gradeRange := container.NewHBox(
                makeSmallColoredText("от", hexColor(config.ColorMuted)),
                p.minGradeEntry,
                makeSmallColoredText("до", hexColor(config.ColorMuted)),
                p.maxGradeEntry,
        )

        fillLabel := canvas.NewText("Что заполнять:", hexColor(config.ColorPrimary))
        fillLabel.TextStyle = fyne.TextStyle{Bold: true}

        settingsCard := widget.NewCard("Настройки оценки", "", container.NewVBox(
                container.NewGridWithColumns(2,
                        container.NewVBox(
                                classLabel,
                                p.classSelect,
                                quarterLabel,
                                p.quarterSelect,
                        ),
                        container.NewVBox(
                                subjectLabel,
                                p.subjectSelect,
                                rangeLabel,
                                gradeRange,
                        ),
                ),
                widget.NewSeparator(),
                p.fillEmptyChk,
                widget.NewSeparator(),
                fillLabel,
                container.NewGridWithColumns(2,
                        p.fillDailyChk,
                        p.fillQuarterChk,
                        p.fillSemesterChk,
                        p.fillYearChk,
                ),
        ))

        // ── Action buttons ──────────────────────────────────────────
        p.analyzeBtn = widget.NewButtonWithIcon("Анализировать", theme.SearchIcon(), func() {
                p.doAnalyze()
        })

        p.startBtn = widget.NewButtonWithIcon("Запустить", theme.MediaPlayIcon(), func() {
                p.doStart()
        })
        p.startBtn.Importance = widget.HighImportance
        p.startBtn.Disable()

        p.stopBtn = widget.NewButtonWithIcon("Стоп", theme.MediaStopIcon(), func() {
                p.doStop()
        })
        p.stopBtn.Disable()

        // Limits button — opens journal limits dialog
        p.limitsBtn = widget.NewButtonWithIcon("Пределы учеников", theme.ContentAddIcon(), func() {
                p.app.journalPg.showLimitsDialog()
        })

        actionCard := widget.NewCard("", "", container.NewHBox(
                p.startBtn,
                p.stopBtn,
                p.analyzeBtn,
                p.limitsBtn,
                layout.NewSpacer(),
        ))

        // ── Progress ────────────────────────────────────────────────
        p.progressLabel = canvas.NewText("Готов к работе", hexColor(config.ColorPrimary))
        p.progressLabel.TextStyle = fyne.TextStyle{Bold: true}
        p.progressBar = widget.NewProgressBar()
        p.progressBar.Min = 0
        p.progressBar.Max = 1
        p.statsLabel = canvas.NewText("", hexColor(config.ColorMuted))
        p.statsLabel.TextSize = 12

        progressCard := widget.NewCard("Прогресс", "", container.NewVBox(
                p.progressLabel,
                p.progressBar,
                p.statsLabel,
        ))

        // ── Results ─────────────────────────────────────────────────
        p.resultsEntry = widget.NewMultiLineEntry()
        p.resultsEntry.SetPlaceHolder("Результаты анализа появятся здесь...\n\nНажмите «Анализировать» для начала")
        p.resultsEntry.Wrapping = fyne.TextWrapWord
        p.resultsEntry.TextStyle = fyne.TextStyle{Monospace: true}
        p.resultsEntry.SetMinRowsVisible(12)

        resultsCard := widget.NewCard("Результаты", "", p.resultsEntry)

        // ── Main layout ─────────────────────────────────────────────
        content := container.NewVBox(
                headerBar,
                settingsCard,
                actionCard,
                progressCard,
                resultsCard,
        )

        scroll := container.NewVScroll(content)
        scroll.SetMinSize(fyne.NewSize(900, 600))

        return scroll
}

// UpdateDropdowns populates dropdowns with loaded data.
func (p *AutoGradePage) UpdateDropdowns() {
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

// UpdateProgress updates the progress display from the engine.
func (p *AutoGradePage) UpdateProgress(plan *engine.GradePlan) {
        progress := plan.Progress()
        p.progressBar.SetValue(progress)

        completed := int(plan.Completed)
        failed := int(plan.Failed)
        skipped := int(plan.Skipped)
        total := plan.TotalTasks

        p.progressLabel.Text = fmt.Sprintf("Выполнение: %d/%d", completed+failed, total-int(skipped))
        p.progressLabel.Color = hexColor(config.ColorPrimary)
        canvas.Refresh(p.progressLabel)

        // Color the stats based on results
        statsText := fmt.Sprintf("Успешно: %d | Ошибки: %d | Пропущено: %d", completed, failed, skipped)
        p.statsLabel.Text = statsText
        if failed > 0 {
                p.statsLabel.Color = hexColor(config.ColorError)
        } else if completed > 0 {
                p.statsLabel.Color = hexColor(config.ColorSuccess)
        } else {
                p.statsLabel.Color = hexColor(config.ColorMuted)
        }
        canvas.Refresh(p.statsLabel)
}

// getSelectedGroups returns the groups matching the dropdown selection.
func (p *AutoGradePage) getSelectedGroups() []map[string]interface{} {
        selected := p.classSelect.Selected
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

// getSelectedSubjects returns the subjects matching the dropdown selection.
func (p *AutoGradePage) getSelectedSubjects() []map[string]interface{} {
        selected := p.subjectSelect.Selected
        if selected == "Все предметы" || selected == "" {
                return p.app.teacherSubjects
        }
        var result []map[string]interface{}
        for _, s := range p.app.teacherSubjects {
                if name, _ := s["subjectName"].(string); name == selected {
                        result = append(result, s)
                }
        }
        return result
}

// getSelectedQuarters returns the quarters matching the dropdown selection.
// When a specific class is selected and that group has group-specific quarters,
// use those (they have the correct qpropId for that group).
func (p *AutoGradePage) getSelectedQuarters() []map[string]interface{} {
        selected := p.quarterSelect.Selected
        classSelected := p.classSelect.Selected

        // If a specific class is selected, try to use its group-specific quarters
        if classSelected != "Все классы" && classSelected != "" {
                for _, g := range p.app.groupsData {
                        if name, _ := g["name"].(string); name == classSelected {
                                if groupQuarters, ok := g["quarters"].([]interface{}); ok && len(groupQuarters) > 0 {
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
                                                if qname, _ := q["name"].(string); qname == selected {
                                                        result = append(result, q)
                                                }
                                        }
                                        return result
                                }
                                break
                        }
                }
        }

        // Fallback to global quartersData
        if selected == "Все четверти" || selected == "" {
                return p.app.quartersData
        }
        var result []map[string]interface{}
        for _, q := range p.app.quartersData {
                if name, _ := q["name"].(string); name == selected {
                        result = append(result, q)
                }
        }
        return result
}

// getStudentLimitsFromJournal builds per-student limits map from the journal page.
func (p *AutoGradePage) getStudentLimitsFromJournal() map[string][2]int {
        limits := make(map[string][2]int)
        journalLimits := p.app.journalPg.GetStudentLimits()
        for name, sl := range journalLimits {
                limits[name] = [2]int{sl.minGrade, sl.maxGrade}
        }
        return limits
}

// doAnalyze starts the grade analysis.
func (p *AutoGradePage) doAnalyze() {
        p.analyzeBtn.Disable()
        p.analyzeBtn.SetText("Анализ...")
        p.progressBar.SetValue(0)
        p.progressLabel.Text = "Анализ..."
        p.progressLabel.Color = hexColor(config.ColorWarning)
        canvas.Refresh(p.progressLabel)
        p.resultsEntry.SetText("Анализ журнала (умный режим)...")

        go func() {
                groups := p.getSelectedGroups()
                subjects := p.getSelectedSubjects()
                quarters := p.getSelectedQuarters()

                minGrade := config.MinGrade
                maxGrade := config.MaxGrade
                if v := parseInt(p.minGradeEntry.Text); v > 0 {
                        minGrade = v
                }
                if v := parseInt(p.maxGradeEntry.Text); v > 0 {
                        maxGrade = v
                }

                fillEmptyOnly := p.fillEmptyChk.Checked
                includeDaily := p.fillDailyChk.Checked
                includeQuarter := p.fillQuarterChk.Checked
                includeSemester := p.fillSemesterChk.Checked
                includeYear := p.fillYearChk.Checked

                // Pass per-student limits from the journal page
                studentLimits := p.getStudentLimitsFromJournal()
                p.app.engine.SetStudentLimits(studentLimits)

                // Use complete plan builder if any non-daily options are checked
                var plan *engine.GradePlan
                if includeQuarter || includeSemester || includeYear {
                        plan = p.app.engine.BuildCompletePlan(
                                groups, subjects, quarters,
                                minGrade, maxGrade,
                                fillEmptyOnly,
                                includeDaily,
                                includeQuarter,
                                includeSemester,
                                includeYear,
                        )
                } else {
                        plan = p.app.engine.BuildGradePlan(
                                groups, subjects, quarters,
                                minGrade, maxGrade,
                                fillEmptyOnly,
                        )
                }

                p.app.currentPlan = plan
                fyne.Do(func() {
                        p.onAnalyzeComplete(plan)
                })
        }()
}

// onAnalyzeComplete handles the analysis completion.
func (p *AutoGradePage) onAnalyzeComplete(plan *engine.GradePlan) {
        p.analyzeBtn.Enable()
        p.analyzeBtn.SetText("Анализировать")
        p.startBtn.Enable()

        toExecute := plan.PendingCount()

        // Count by task type
        dailyCount := 0
        quarterCount := 0
        semesterCount := 0
        yearCount := 0
        for _, t := range plan.Tasks {
                if t.Status == engine.StatusPending {
                        switch t.TaskType {
                        case engine.TaskDaily:
                                dailyCount++
                        case engine.TaskQuarter:
                                quarterCount++
                        case engine.TaskSemester:
                                semesterCount++
                        case engine.TaskYear:
                                yearCount++
                        }
                }
        }

        // Count students with custom limits
        customLimitsCount := 0
        if p.app.engine.StudentLimits != nil {
                customLimitsCount = len(p.app.engine.StudentLimits)
        }

        lines := "══════════════════════════════════════════════════\n"
        lines += "  ПЛАН ОЦЕНОК (Умный режим)\n"
        lines += "══════════════════════════════════════════════════\n\n"
        lines += fmt.Sprintf("  Всего задач:      %d\n", plan.TotalTasks)
        lines += fmt.Sprintf("  Будет выполнено:  %d\n", toExecute)
        lines += fmt.Sprintf("  Пропущено:        %d\n\n", int(plan.Skipped))

        if customLimitsCount > 0 {
                lines += fmt.Sprintf("  Пользовательские пределы: %d учеников\n\n", customLimitsCount)
        }

        if dailyCount > 0 {
                lines += fmt.Sprintf("  Дневные оценки:   %d\n", dailyCount)
        }
        if quarterCount > 0 {
                lines += fmt.Sprintf("  Четвертные:       %d\n", quarterCount)
        }
        if semesterCount > 0 {
                lines += fmt.Sprintf("  Семестровые:      %d\n", semesterCount)
        }
        if yearCount > 0 {
                lines += fmt.Sprintf("  Годовые:          %d\n", yearCount)
        }
        lines += "\n"

        // Group by class/subject
        type groupKey struct{ group, subject string }
        groupMap := make(map[groupKey][]*engine.GradeTask)
        for _, t := range plan.Tasks {
                if t.Status == engine.StatusPending {
                        key := groupKey{t.GroupName, t.SubjectName}
                        groupMap[key] = append(groupMap[key], t)
                }
        }

        for key, tasks := range groupMap {
                lines += fmt.Sprintf("  %s | %s\n", key.group, key.subject)
                lines += fmt.Sprintf("    Оценок: %d\n", len(tasks))
                for i, t := range tasks {
                        if i >= 5 {
                                lines += fmt.Sprintf("    ... и ещё %d\n", len(tasks)-5)
                                break
                        }
                        typeLabel := taskTypeLabel(t.TaskType)
                        lines += fmt.Sprintf("    - %s -> %d (%s, %s)\n", t.StudentName, t.Mark, t.DateStr, typeLabel)
                }
                lines += "\n"
        }

        // Show students with custom limits
        if customLimitsCount > 0 {
                lines += "── Пользовательские пределы ──────────────────────\n"
                for name, limits := range p.app.engine.StudentLimits {
                        lines += fmt.Sprintf("  %s: [%d - %d]\n", name, limits[0], limits[1])
                }
                lines += "\n"
        }

        p.resultsEntry.SetText(lines)
        p.progressLabel.Text = fmt.Sprintf("Анализ завершён: %d оценок будет добавлено", toExecute)
        p.progressLabel.Color = hexColor(config.ColorSuccess)
        canvas.Refresh(p.progressLabel)
}

func taskTypeLabel(t engine.TaskType) string {
        switch t {
        case engine.TaskDaily:
                return "дневная"
        case engine.TaskQuarter:
                return "четвертная"
        case engine.TaskSemester:
                return "семестровая"
        case engine.TaskYear:
                return "годовая"
        default:
                return "оценка"
        }
}

// doStart starts the grade execution.
func (p *AutoGradePage) doStart() {
        if p.app.currentPlan == nil {
                p.app.LogMessage("Сначала выполните анализ!", "warning")
                return
        }

        if p.app.currentPlan.PendingCount() == 0 {
                p.app.LogMessage("Нет оценок для добавления!", "warning")
                return
        }

        p.startBtn.Disable()
        p.stopBtn.Enable()
        p.analyzeBtn.Disable()
        p.progressLabel.Text = "Заполнение..."
        p.progressLabel.Color = hexColor(config.ColorWarning)
        canvas.Refresh(p.progressLabel)

        go func() {
                p.app.engine.ExecutePlan(p.app.currentPlan, config.DefaultWorkers, 150*time.Millisecond)

                fyne.Do(func() {
                        p.onExecutionComplete()
                })
        }()
}

// doStop stops the engine.
func (p *AutoGradePage) doStop() {
        p.app.engine.Stop()
        p.stopBtn.Disable()
        p.progressLabel.Text = "Остановка..."
        p.progressLabel.Color = hexColor(config.ColorError)
        canvas.Refresh(p.progressLabel)
        p.app.LogMessage("Остановка...", "warning")
}

// onExecutionComplete handles execution completion.
func (p *AutoGradePage) onExecutionComplete() {
        p.startBtn.Enable()
        p.stopBtn.Disable()
        p.analyzeBtn.Enable()

        if p.app.currentPlan != nil {
                plan := p.app.currentPlan
                done := int(plan.Completed) + int(plan.Failed)
                total := plan.TotalTasks - int(plan.Skipped)
                p.progressLabel.Text = fmt.Sprintf("Завершено: %d/%d", done, total)

                if plan.Failed > 0 {
                        p.progressLabel.Color = hexColor(config.ColorError)
                } else {
                        p.progressLabel.Color = hexColor(config.ColorSuccess)
                }
                canvas.Refresh(p.progressLabel)

                statsText := fmt.Sprintf("Успешно: %d | Ошибки: %d | Пропущено: %d",
                        int(plan.Completed), int(plan.Failed), int(plan.Skipped))
                p.statsLabel.Text = statsText
                if plan.Failed > 0 {
                        p.statsLabel.Color = hexColor(config.ColorError)
                } else {
                        p.statsLabel.Color = hexColor(config.ColorSuccess)
                }
                canvas.Refresh(p.statsLabel)
        }
}

// parseInt safely parses an integer from a string.
func parseInt(s string) int {
        var v int
        fmt.Sscanf(s, "%d", &v)
        return v
}
