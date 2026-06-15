package ui

import (
        "fmt"
        "strings"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

// ------------------------------------------
// TOPICS & HW TAB — Text input, sequential fill
// ------------------------------------------

type TopicsTab struct {
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
        dates           []client.Day

        // UI
        statusLabel   *widget.Label
        topicEntry    *widget.Entry
        homeworkEntry *widget.Entry
}

// NewTopicsTab creates a new TopicsTab.
func NewTopicsTab(c Controller) *TopicsTab {
        t := &TopicsTab{
                controller:  c,
                statusLabel: widget.NewLabel("Выберите класс, предмет и четверть"),
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

        t.quarterSel = widget.NewSelect([]string{}, t.onQuarterSelected)
        t.quarterSel.PlaceHolder = "Четверть..."

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go t.loadData()
        })

        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"),
                t.classSel,
                t.subjectSel,
                t.quarterSel,
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

        // Save button
        saveBtn := widget.NewButton("Сохранить темы и ДЗ", t.onSaveTopics)
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
                t.statusLabel.SetText("Выберите класс, предмет и четверть")
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
        t.selectedQuarter = nil

        subjectNames := make([]string, len(t.selectedGroup.Subjects))
        for i, s := range t.selectedGroup.Subjects {
                subjectNames[i] = s.SubjectName
        }

        quarterNames := make([]string, len(t.selectedGroup.Quarters))
        for i, q := range t.selectedGroup.Quarters {
                quarterNames[i] = q.Name
        }

        fyne.Do(func() {
                t.subjectSel.Options = subjectNames
                t.subjectSel.Refresh()
                t.subjectSel.ClearSelected()
                t.quarterSel.Options = quarterNames
                t.quarterSel.Refresh()
                t.quarterSel.ClearSelected()

                if len(subjectNames) > 0 {
                        t.subjectSel.SetSelectedIndex(0)
                }
                for i, q := range t.selectedGroup.Quarters {
                        if q.CurrentQuarter {
                                t.quarterSel.SetSelectedIndex(i)
                                break
                        }
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
        t.tryLoadData()
}

func (t *TopicsTab) onQuarterSelected(selected string) {
        if t.selectedGroup == nil {
                return
        }
        for i, q := range t.selectedGroup.Quarters {
                if q.Name == selected {
                        t.selectedQuarter = &t.selectedGroup.Quarters[i]
                        break
                }
        }
        t.tryLoadData()
}

func (t *TopicsTab) tryLoadData() {
        if t.selectedGroup != nil && t.selectedSubject != nil && t.selectedQuarter != nil {
                go t.loadData()
        }
}

func (t *TopicsTab) loadData() {
        if t.selectedGroup == nil || t.selectedSubject == nil || t.selectedQuarter == nil {
                return
        }

        fyne.Do(func() {
                t.statusLabel.SetText("Загрузка дат...")
        })

        apiClient := t.controller.GetClient()
        dates, err := apiClient.GetJournalDates(
                t.selectedGroup.ID,
                t.selectedSubject.SubjectID,
                t.selectedQuarter.ID,
        )

        fyne.Do(func() {
                if err != nil {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка: %v", err))
                        return
                }

                t.dates = dates
                t.statusLabel.SetText(fmt.Sprintf("Загружено: %d дат", len(dates)))
                t.updatePreview()
        })
}

// updatePreview rebuilds the right panel with current topic/HW info per date.
func (t *TopicsTab) updatePreview() {
        if len(t.dates) == 0 {
                return
        }

        // Collect existing topics and HW from dates
        var existingTopics []string
        var existingHW []string
        var rows []fyne.CanvasObject

        rows = append(rows, widget.NewLabelWithStyle(
                fmt.Sprintf("Текущие темы и ДЗ (%d дат):", len(t.dates)),
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

                dateLabel := widget.NewLabel(fmt.Sprintf("%d. %s %s", i+1, d.WeekdayShortName, d.AssignmentDate[5:]))
                dateLabel.TextStyle = fyne.TextStyle{Bold: true}

                topicLabel := widget.NewLabel(fmt.Sprintf("   Тема: %s", topicText))
                hwLabel := widget.NewLabel(fmt.Sprintf("   ДЗ: %s", hwText))

                rows = append(rows, dateLabel, topicLabel, hwLabel)

                if topicText != "(пусто)" {
                        existingTopics = append(existingTopics, topicText)
                }
                if hwText != "(пусто)" {
                        existingHW = append(existingHW, hwText)
                }
        }

        // Pre-fill entries with existing topics/HW
        if len(existingTopics) > 0 && t.topicEntry.Text == "" {
                t.topicEntry.SetText(strings.Join(existingTopics, "\n"))
        }
        if len(existingHW) > 0 && t.homeworkEntry.Text == "" {
                t.homeworkEntry.SetText(strings.Join(existingHW, "\n"))
        }

        // Update right panel
        rightContent := container.NewVBox(rows...)
        rightScroll := container.NewVScroll(rightContent)

        leftPanel := container.NewVBox(
                widget.NewLabelWithStyle("Темы уроков (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                t.topicEntry,
                widget.NewSeparator(),
                widget.NewLabelWithStyle("Домашние задания (по строкам):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                t.homeworkEntry,
                widget.NewSeparator(),
                widget.NewButton("Сохранить темы и ДЗ", t.onSaveTopics),
        )

        content := container.NewHSplit(leftPanel, rightScroll)
        content.SetOffset(0.6)

        topBar := container.NewVBox(
                container.NewHBox(
                        widget.NewLabel("Фильтры:"),
                        t.classSel, t.subjectSel, t.quarterSel,
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
// SAVE TOPICS & HW
// ------------------------------------------

func (t *TopicsTab) onSaveTopics() {
        if len(t.dates) == 0 {
                dialog.ShowInformation("Внимание", "Сначала загрузите данные", t.controller.GetWindow())
                return
        }

        topicLines := splitLines(t.topicEntry.Text)
        hwLines := splitLines(t.homeworkEntry.Text)

        dateCount := len(t.dates)

        confirmMsg := fmt.Sprintf(
                "Будут сохранены темы и ДЗ для %d дат:\n\n"+
                        "Тем: %d строк (циклично по %d датам)\n"+
                        "ДЗ: %d строк (циклично по %d датам)\n\n"+
                        "Продолжить?",
                dateCount, len(topicLines), dateCount, len(hwLines), dateCount)

        dialog.ShowConfirm("Сохранить темы и ДЗ", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go t.executeSaveTopics(topicLines, hwLines)
        }, t.controller.GetWindow())
}

func (t *TopicsTab) executeSaveTopics(topicLines, hwLines []string) {
        apiClient := t.controller.GetClient()
        quarterID := t.selectedQuarter.ID
        total := len(t.dates)
        successCount := 0

        for i, date := range t.dates {
                // Get topic cyclically
                topic := ""
                if len(topicLines) > 0 {
                        topic = topicLines[i%len(topicLines)]
                }

                // Get homework cyclically
                hw := ""
                if len(hwLines) > 0 {
                        hw = hwLines[i%len(hwLines)]
                }

                // Skip if both empty
                if topic == "" && hw == "" {
                        continue
                }

                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Сохранение %d/%d: %s — %s",
                                i+1, total, date.AssignmentDate[5:], topic))
                })

                err := apiClient.UpdateAssignment(
                        date.AssignmentDateID,
                        topic,
                        hw,
                        quarterID,
                )
                if err == nil {
                        successCount++
                }
        }

        fyne.Do(func() {
                t.statusLabel.SetText(fmt.Sprintf("Сохранено: %d из %d дат", successCount, total))
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
