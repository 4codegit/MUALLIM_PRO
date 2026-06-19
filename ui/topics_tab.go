package ui

import (
        "fmt"
        "strings"
        "sync"
        "sync/atomic"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

// ------------------------------------------
// TOPICS & HW TAB — Year-wide bulk fill, parallel
// ------------------------------------------

// saveConcurrency controls how many /journal/assignment/update requests run
// in parallel. 8 is a safe value that won't trip the server's rate-limiter
// but turns a 40-date year from ~40 sequential round-trips into ~5 batches.
const saveConcurrency = 8

type TopicsTab struct {
        controller Controller
        container  *fyne.Container

        // Filters — only class + subject (quarter dropped: we fill the whole year)
        classSel   *widget.Select
        subjectSel *widget.Select

        // State
        journalOpts     *client.JournalOptions
        selectedGroup   *client.JournalGroup
        selectedSubject *client.Subject
        // dates collected across ALL quarters of the year (kept in calendar order)
        dates           []client.Day
        // which quarter each date belongs to (parallel to `dates`); needed because
        // UpdateAssignment takes a quarter_property_id parameter.
        dateQuarterIDs  []int
        // remember all quarters so we can iterate them when loading
        allQuarters     []client.Quarter

        // UI
        statusLabel   *widget.Label
        topicEntry    *widget.Entry
        homeworkEntry *widget.Entry
}

// NewTopicsTab creates a new TopicsTab.
func NewTopicsTab(c Controller) *TopicsTab {
        t := &TopicsTab{
                controller:  c,
                statusLabel: widget.NewLabel("Выберите класс и предмет"),
        }
        t.buildUI()
        go t.loadJournalOptions()
        return t
}

func (t *TopicsTab) Container() fyne.CanvasObject {
        return t.container
}

// ------------------------------------------
// UI BUILD
// ------------------------------------------

func (t *TopicsTab) buildUI() {
        t.classSel = widget.NewSelect([]string{}, t.onClassSelected)
        t.classSel.PlaceHolder = "Класс..."

        t.subjectSel = widget.NewSelect([]string{}, t.onSubjectSelected)
        t.subjectSel.PlaceHolder = "Предмет..."

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go t.loadData()
        })

        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"),
                t.classSel,
                t.subjectSel,
                refreshBtn,
        )

        // Topic text area
        t.topicEntry = widget.NewMultiLineEntry()
        t.topicEntry.SetPlaceHolder("Введите темы по строкам:\nТема 1\nТема 2\nТема 3\n...")
        t.topicEntry.Wrapping = fyne.TextWrapWord
        t.topicEntry.SetText("")

        topicLabel := widget.NewLabelWithStyle("Темы уроков (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

        // Homework text area
        t.homeworkEntry = widget.NewMultiLineEntry()
        t.homeworkEntry.SetPlaceHolder("Введите ДЗ по строкам:\nДЗ 1\nДЗ 2\nДЗ 3\n...")
        t.homeworkEntry.Wrapping = fyne.TextWrapWord
        t.homeworkEntry.SetText("")

        homeworkLabel := widget.NewLabelWithStyle("Домашние задания (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

        // Save button — overwrites existing topics & ДЗ across ALL quarters of the year
        saveBtn := widget.NewButton("Сохранить темы и ДЗ за год", t.onSaveTopics)
        saveBtn.Importance = widget.HighImportance

        // Layout: left side = inputs, right side = current dates preview
        leftPanel := container.NewVBox(
                topicLabel,
                t.topicEntry,
                widget.NewSeparator(),
                homeworkLabel,
                t.homeworkEntry,
                widget.NewSeparator(),
                saveBtn,
        )

        rightPanel := container.NewVBox(
                widget.NewLabelWithStyle("Текущие темы и ДЗ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                widget.NewLabel("Загрузите данные для просмотра"),
        )

        content := container.NewHSplit(leftPanel, rightPanel)
        content.SetOffset(0.6)

        topBar := container.NewVBox(filterRow, widget.NewSeparator())

        t.container = container.NewBorder(
                topBar,
                t.statusLabel,
                nil, nil,
                content,
        )
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (t *TopicsTab) loadJournalOptions() {
        t.statusLabel.SetText("Загрузка классов...")
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

func (t *TopicsTab) onClassSelected(selected string) {
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
        t.allQuarters = t.selectedGroup.Quarters

        subjectNames := make([]string, len(t.selectedGroup.Subjects))
        for i, s := range t.selectedGroup.Subjects {
                subjectNames[i] = s.SubjectName
        }

        fyne.Do(func() {
                t.subjectSel.Options = subjectNames
                t.subjectSel.Refresh()
                t.subjectSel.ClearSelected()

                if len(subjectNames) > 0 {
                        t.subjectSel.SetSelectedIndex(0)
                }
        })
}

func (t *TopicsTab) onSubjectSelected(selected string) {
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
                go t.loadData()
        }
}

// loadData fetches dates from ALL quarters of the year in parallel and merges
// them into a single chronological list. Each date remembers which quarter it
// belongs to (dateQuarterIDs) so UpdateAssignment can pass the right quarterPropertyID.
func (t *TopicsTab) loadData() {
        if t.selectedGroup == nil || t.selectedSubject == nil {
                return
        }

        fyne.Do(func() {
                t.statusLabel.SetText(fmt.Sprintf("Загрузка дат за год (%d четверти)...", len(t.allQuarters)))
        })

        apiClient := t.controller.GetClient()
        groupID := t.selectedGroup.ID
        subjectID := t.selectedSubject.SubjectID

        type quarterResult struct {
                idx     int
                quarter client.Quarter
                days    []client.Day
                err     error
        }

        results := make([]quarterResult, len(t.allQuarters))
        var wg sync.WaitGroup
        // Load all quarters in parallel — 4 sequential GET /journal/dates was the
        // main reason the previous version felt slow when switching tabs.
        for i, q := range t.allQuarters {
                wg.Add(1)
                go func(idx int, quarter client.Quarter) {
                        defer wg.Done()
                        days, err := apiClient.GetJournalDates(groupID, subjectID, quarter.ID)
                        results[idx] = quarterResult{idx: idx, quarter: quarter, days: days, err: err}
                }(i, q)
        }
        wg.Wait()

        // Merge results preserving quarter order
        var mergedDates []client.Day
        var mergedQuarterIDs []int
        var firstErr error
        failedCount := 0
        for _, r := range results {
                if r.err != nil {
                        if firstErr == nil {
                                firstErr = r.err
                        }
                        failedCount++
                        continue
                }
                for _, d := range r.days {
                        mergedDates = append(mergedDates, d)
                        mergedQuarterIDs = append(mergedQuarterIDs, r.quarter.ID)
                }
        }

        fyne.Do(func() {
                if firstErr != nil && len(mergedDates) == 0 {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", firstErr))
                        return
                }
                t.dates = mergedDates
                t.dateQuarterIDs = mergedQuarterIDs
                msg := fmt.Sprintf("Загружено: %d дат за год", len(mergedDates))
                if failedCount > 0 {
                        msg += fmt.Sprintf(" (%d четвертей с ошибкой)", failedCount)
                }
                t.statusLabel.SetText(msg)
                t.updatePreview()
        })
}

// updatePreview rebuilds the right panel with current topic/HW info per date.
func (t *TopicsTab) updatePreview() {
        if len(t.dates) == 0 {
                return
        }

        var rows []fyne.CanvasObject
        rows = append(rows, widget.NewLabelWithStyle(
                fmt.Sprintf("Текущие темы и ДЗ (%d дат за год):", len(t.dates)),
                fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

        for i, d := range t.dates {
                topicText := d.Topic
                if topicText == "" {
                        topicText = "(пусто)"
                }
                hwText := d.HomeWork
                if hwText == "" {
                        hwText = "(пусто)"
                }

                // Find quarter name for this date
                quarterName := ""
                if i < len(t.dateQuarterIDs) {
                        for _, q := range t.allQuarters {
                                if q.ID == t.dateQuarterIDs[i] {
                                        quarterName = q.Name
                                        break
                                }
                        }
                }

                dateLabel := widget.NewLabel(fmt.Sprintf("%d. %s %s — %s", i+1, d.WeekdayShortName, d.AssignmentDate[5:], quarterName))
                dateLabel.TextStyle = fyne.TextStyle{Bold: true}

                topicLabel := widget.NewLabel(fmt.Sprintf("   Тема: %s", topicText))
                hwLabel := widget.NewLabel(fmt.Sprintf("   ДЗ: %s", hwText))

                rows = append(rows, dateLabel, topicLabel, hwLabel)
        }

        // Right panel — keep input fields untouched (we want to overwrite, not pre-fill)
        rightContent := container.NewVBox(rows...)
        rightScroll := container.NewVScroll(rightContent)

        leftPanel := container.NewVBox(
                widget.NewLabelWithStyle("Темы уроков (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                t.topicEntry,
                widget.NewSeparator(),
                widget.NewLabelWithStyle("Домашние задания (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                t.homeworkEntry,
                widget.NewSeparator(),
                widget.NewButton("Сохранить темы и ДЗ за год", t.onSaveTopics),
        )

        content := container.NewHSplit(leftPanel, rightScroll)
        content.SetOffset(0.6)

        topBar := container.NewVBox(
                container.NewHBox(
                        widget.NewLabel("Фильтры:"),
                        t.classSel, t.subjectSel,
                        widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { go t.loadData() }),
                ),
                widget.NewSeparator(),
        )

        t.container.Objects = []fyne.CanvasObject{
                container.NewBorder(topBar, t.statusLabel, nil, nil, content),
        }
        t.container.Refresh()
}

// ------------------------------------------
// SAVE TOPICS & HW — overwrite, parallel
// ------------------------------------------

func (t *TopicsTab) onSaveTopics() {
        if len(t.dates) == 0 {
                dialog.ShowInformation("Внимание", "Сначала загрузите данные", t.controller.GetWindow())
                return
        }

        topicLines := splitLines(t.topicEntry.Text)
        hwLines := splitLines(t.homeworkEntry.Text)

        if len(topicLines) == 0 && len(hwLines) == 0 {
                dialog.ShowInformation("Внимание", "Введите хотя бы одну тему или ДЗ", t.controller.GetWindow())
                return
        }

        dateCount := len(t.dates)

        confirmMsg := fmt.Sprintf(
                "Будут сохранены темы и ДЗ за ГОД (%d дат, все четверти):\n\n"+
                        "Тем: %d строк (циклично по %d датам)\n"+
                        "ДЗ: %d строк (циклично по %d датам)\n\n"+
                        "⚠️ Существующие темы и ДЗ будут ПЕРЕЗАПИСАНЫ.\n"+
                        "Параллельных запросов: %d\n\n"+
                        "Продолжить?",
                dateCount, len(topicLines), dateCount, len(hwLines), dateCount, saveConcurrency)

        dialog.ShowConfirm("Сохранить темы и ДЗ за год", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go t.executeSaveTopics(topicLines, hwLines)
        }, t.controller.GetWindow())
}

// executeSaveTopics sends UpdateAssignment for every date in parallel using a
// worker pool. Previously this was a sequential loop — each request waited for
// the previous to complete, so 40 dates × ~500ms RTT = ~20 seconds. With 8
// workers it drops to ~3 seconds.
//
// Existing topics and ДЗ ARE overwritten — the server treats
// /journal/assignment/update as a PUT-style upsert.
func (t *TopicsTab) executeSaveTopics(topicLines, hwLines []string) {
        apiClient := t.controller.GetClient()
        total := len(t.dates)

        var successCount int32
        var failCount int32
        var firstErr int32 // 0 = no err, 1 = err captured
        var firstErrMsg string

        type job struct {
                date     client.Day
                quarterID int
                topic    string
                homework string
        }

        // Build the job queue — one job per date with its cyclic topic/hw
        jobs := make(chan job, total)
        for i, date := range t.dates {
                topic := ""
                if len(topicLines) > 0 {
                        topic = topicLines[i%len(topicLines)]
                }
                hw := ""
                if len(hwLines) > 0 {
                        hw = hwLines[i%len(hwLines)]
                }
                // Skip if both empty (would be a no-op)
                if topic == "" && hw == "" {
                        continue
                }
                quarterID := 0
                if i < len(t.dateQuarterIDs) {
                        quarterID = t.dateQuarterIDs[i]
                }
                jobs <- job{date: date, quarterID: quarterID, topic: topic, homework: hw}
        }
        close(jobs)

        // Status ticker — updates the status label every 200ms while save runs
        done := make(chan struct{})
        go func() {
                for {
                        select {
                        case <-done:
                                return
                        case <-time.After(200 * time.Millisecond):
                                s := atomic.LoadInt32(&successCount)
                                f := atomic.LoadInt32(&failCount)
                                fyne.Do(func() {
                                        t.statusLabel.SetText(fmt.Sprintf("Сохранение: ✓ %d / ✗ %d из %d", s, f, total))
                                })
                        }
                }
        }()
        defer close(done)

        // Worker pool
        var wg sync.WaitGroup
        for w := 0; w < saveConcurrency; w++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        for j := range jobs {
                                err := apiClient.UpdateAssignment(
                                        j.date.AssignmentDateID,
                                        j.topic,
                                        j.homework,
                                        j.quarterID,
                                )
                                if err == nil {
                                        atomic.AddInt32(&successCount, 1)
                                } else {
                                        atomic.AddInt32(&failCount, 1)
                                        if atomic.CompareAndSwapInt32(&firstErr, 0, 1) {
                                                firstErrMsg = err.Error()
                                        }
                                }
                        }
                }()
        }
        wg.Wait()

        finalSuccess := int(atomic.LoadInt32(&successCount))
        finalFail := int(atomic.LoadInt32(&failCount))

        fyne.Do(func() {
                msg := fmt.Sprintf("Готово: ✓ %d / ✗ %d из %d дат", finalSuccess, finalFail, total)
                if finalFail > 0 && firstErrMsg != "" {
                        msg += fmt.Sprintf("  |  первая ошибка: %s", firstErrMsg)
                }
                t.statusLabel.SetText(msg)
                // Reload to show the new state in the right panel
                go t.loadData()
        })
}

// splitLines splits text into non-empty lines.
func splitLines(text string) []string {
        var lines []string
        for _, line := range strings.Split(text, "\n") {
                trimmed := strings.TrimSpace(line)
                if trimmed != "" {
                        lines = append(lines, trimmed)
                }
        }
        return lines
}
