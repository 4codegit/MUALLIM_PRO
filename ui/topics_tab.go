package ui

import (
        "fmt"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

// qualityLevels defines the available quality levels for sequential topic generation.
// These must match keys in the shared TopicTemplates variable (defined in helpers.go).
var qualityLevels = []string{
        "Отличный",
        "Хорошо",
        "Удовлетворительно",
        "Неудовлетворительно",
}

// TopicsTab manages the Topics (Темы) tab with CRUD and sequential topic generation.
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

        // Fill controls
        weightSel *widget.Select
        qualitySel *widget.Select
        fillBtn   *widget.Button

        // UI
        topicsList  *widget.List
        statusLabel *widget.Label
}

// NewTopicsTab creates a new TopicsTab instance.
func NewTopicsTab(c Controller) *TopicsTab {
        t := &TopicsTab{
                controller:  c,
                statusLabel: widget.NewLabel("Готово"),
        }
        t.buildUI()
        go t.loadJournalOptions()
        return t
}

// Container returns the main canvas object for this tab.
func (t *TopicsTab) Container() fyne.CanvasObject {
        return t.container
}

// buildUI creates the filter row, topic list, and action buttons.
func (t *TopicsTab) buildUI() {
        // --- Filters ---
        t.classSel = widget.NewSelect([]string{}, t.onClassSelected)
        t.classSel.PlaceHolder = "Класс..."

        t.subjectSel = widget.NewSelect([]string{}, t.onSubjectSelected)
        t.subjectSel.PlaceHolder = "Предмет..."

        t.quarterSel = widget.NewSelect([]string{}, t.onQuarterSelected)
        t.quarterSel.PlaceHolder = "Четверть..."

        refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
                go t.loadData()
        })
        refreshBtn.Disable()

        filterRow := container.NewHBox(
                widget.NewLabel("Фильтры:"),
                t.classSel,
                t.subjectSel,
                t.quarterSel,
                refreshBtn,
        )

        // --- Fill controls ---
        t.weightSel = widget.NewSelect(WeightPeriods, nil)
        t.weightSel.PlaceHolder = "Диапазон дат..."
        if len(WeightPeriods) > 2 {
                t.weightSel.SetSelectedIndex(2) // "Весь год" by default
        }

        t.qualitySel = widget.NewSelect(qualityLevels, nil)
        t.qualitySel.PlaceHolder = "Уровень тем..."
        if len(qualityLevels) > 1 {
                t.qualitySel.SetSelectedIndex(1) // "Хорошо" by default
        }

        t.fillBtn = widget.NewButton("Заполнить", t.onFill)
        t.fillBtn.Importance = widget.HighImportance
        t.fillBtn.Disable()

        actionRow := container.NewHBox(
                widget.NewLabel("Заполнение:"),
                t.weightSel,
                widget.NewLabel("Уровень:"),
                t.qualitySel,
                t.fillBtn,
        )

        // --- Placeholder for topics list ---
        placeholder := widget.NewLabelWithStyle(
                "Выберите фильтры для загрузки тем",
                fyne.TextAlignCenter,
                fyne.TextStyle{Italic: true},
        )

        // --- Layout ---
        topBar := container.NewVBox(
                filterRow,
                actionRow,
                widget.NewSeparator(),
        )

        t.container = container.NewBorder(
                topBar,
                t.statusLabel,
                nil,
                nil,
                placeholder,
        )
}

// ------------------------------------------
// JOURNAL OPTIONS LOADING
// ------------------------------------------

func (t *TopicsTab) loadJournalOptions() {
        t.statusLabel.SetText("Загрузка классов и предметов...")
        opts, err := t.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки настроек журнала: %v", err))
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
                t.statusLabel.SetText("Выберите класс")
                if len(classNames) > 0 {
                        t.classSel.SetSelectedIndex(0)
                }
        })
}

// ------------------------------------------
// FILTER HANDLERS
// ------------------------------------------

func (t *TopicsTab) onClassSelected(selected string) {
        if t.journalOpts == nil {
                return
        }

        var group *client.JournalGroup
        for i, g := range t.journalOpts.Groups {
                gName := fmt.Sprintf("%d %s", g.Number, g.Name)
                if gName == selected {
                        group = &t.journalOpts.Groups[i]
                        break
                }
        }

        if group == nil {
                return
        }

        t.selectedGroup = group
        t.selectedSubject = nil
        t.selectedQuarter = nil

        subjectNames := make([]string, len(group.Subjects))
        for i, s := range group.Subjects {
                subjectNames[i] = s.SubjectName
        }

        quarterNames := make([]string, len(group.Quarters))
        for i, q := range group.Quarters {
                quarterNames[i] = q.Name
        }

        fyne.Do(func() {
                t.subjectSel.Options = subjectNames
                t.subjectSel.Refresh()
                t.subjectSel.ClearSelected()

                t.quarterSel.Options = quarterNames
                t.quarterSel.Refresh()
                t.quarterSel.ClearSelected()

                t.fillBtn.Disable()

                // Auto select first subject
                if len(subjectNames) > 0 {
                        t.subjectSel.SetSelectedIndex(0)
                }
                // Auto select current quarter if exists
                for i, q := range group.Quarters {
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

        var subject *client.Subject
        for i, s := range t.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        subject = &t.selectedGroup.Subjects[i]
                        break
                }
        }

        t.selectedSubject = subject
        t.checkFilterCompletion()
}

func (t *TopicsTab) onQuarterSelected(selected string) {
        if t.selectedGroup == nil {
                return
        }

        var quarter *client.Quarter
        for i, q := range t.selectedGroup.Quarters {
                if q.Name == selected {
                        quarter = &t.selectedGroup.Quarters[i]
                        break
                }
        }

        t.selectedQuarter = quarter
        t.checkFilterCompletion()
}

func (t *TopicsTab) checkFilterCompletion() {
        if t.selectedGroup != nil && t.selectedSubject != nil && t.selectedQuarter != nil {
                t.fillBtn.Enable()
                go t.loadData()
        } else {
                t.fillBtn.Disable()
        }
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (t *TopicsTab) loadData() {
        if t.selectedGroup == nil || t.selectedSubject == nil || t.selectedQuarter == nil {
                return
        }

        fyne.Do(func() {
                t.statusLabel.SetText("Загрузка данных...")
        })

        apiClient := t.controller.GetClient()
        gID := t.selectedGroup.ID
        sID := t.selectedSubject.SubjectID
        qID := t.selectedQuarter.ID

        dates, err := apiClient.GetJournalDates(gID, sID, qID)

        fyne.Do(func() {
                if err != nil {
                        t.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки дат: %v", err))
                        return
                }

                t.dates = dates
                t.rebuildTopicsList()
                t.statusLabel.SetText(fmt.Sprintf("Загружено: %d дат", len(dates)))
        })
}

// ------------------------------------------
// TOPICS LIST
// ------------------------------------------

func (t *TopicsTab) rebuildTopicsList() {
        dates := t.dates

        if len(dates) == 0 {
                t.container.Objects = []fyne.CanvasObject{
                        container.NewBorder(
                                container.NewVBox(
                                        container.NewHBox(widget.NewLabel("Фильтры:"), t.classSel, t.subjectSel, t.quarterSel),
                                        container.NewHBox(
                                                widget.NewLabel("Заполнение:"), t.weightSel,
                                                widget.NewLabel("Уровень:"), t.qualitySel,
                                                t.fillBtn,
                                        ),
                                        widget.NewSeparator(),
                                ),
                                t.statusLabel,
                                nil,
                                nil,
                                widget.NewLabelWithStyle("Нет дат для выбранной четверти", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        ),
                }
                t.container.Refresh()
                return
        }

        t.topicsList = widget.NewList(
                func() int {
                        return len(dates)
                },
                func() fyne.CanvasObject {
                        dateLabel := widget.NewLabel("")
                        dateLabel.TextStyle = fyne.TextStyle{Bold: true}

                        dayLabel := widget.NewLabel("")
                        dayLabel.TextStyle = fyne.TextStyle{Italic: true}

                        topicLabel := widget.NewLabel("")
                        topicLabel.Wrapping = fyne.TextWrapWord

                        hwLabel := widget.NewLabel("")
                        hwLabel.Wrapping = fyne.TextWrapWord
                        hwLabel.TextStyle = fyne.TextStyle{Italic: true}

                        topRow := container.NewHBox(dateLabel, dayLabel)
                        bottomRow := container.NewHBox(topicLabel, hwLabel)
                        return container.NewVBox(topRow, bottomRow)
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        if id < 0 || id >= len(dates) {
                                return
                        }
                        day := dates[id]

                        vbox := cell.(*fyne.Container)
                        topRow := vbox.Objects[0].(*fyne.Container)
                        bottomRow := vbox.Objects[1].(*fyne.Container)

                        dateLabel := topRow.Objects[0].(*widget.Label)
                        dayLabel := topRow.Objects[1].(*widget.Label)
                        topicLabel := bottomRow.Objects[0].(*widget.Label)
                        hwLabel := bottomRow.Objects[1].(*widget.Label)

                        dateLabel.SetText(day.AssignmentDate)
                        dayLabel.SetText(day.WeekdayName)

                        if day.Topic != "" {
                                topicLabel.SetText(day.Topic)
                                topicLabel.TextStyle = fyne.TextStyle{}
                        } else {
                                topicLabel.SetText("Нажмите для ввода темы...")
                                topicLabel.TextStyle = fyne.TextStyle{Italic: true}
                        }

                        if day.HomeWork != "" {
                                hwLabel.SetText("ДЗ: " + day.HomeWork)
                                hwLabel.TextStyle = fyne.TextStyle{}
                        } else {
                                hwLabel.SetText("")
                        }

                        dateLabel.Refresh()
                        dayLabel.Refresh()
                        topicLabel.Refresh()
                        hwLabel.Refresh()
                },
        )

        t.topicsList.OnSelected = func(id widget.ListItemID) {
                t.topicsList.Unselect(id)
                t.showEditTopicDialog(id)
        }

        // Rebuild the entire container with the new list as center
        topBar := container.NewVBox(
                container.NewHBox(
                        widget.NewLabel("Фильтры:"),
                        t.classSel,
                        t.subjectSel,
                        t.quarterSel,
                ),
                container.NewHBox(
                        widget.NewLabel("Заполнение:"),
                        t.weightSel,
                        widget.NewLabel("Уровень:"),
                        t.qualitySel,
                        t.fillBtn,
                ),
                widget.NewSeparator(),
        )

        t.container.Objects = []fyne.CanvasObject{
                container.NewBorder(
                        topBar,
                        t.statusLabel,
                        nil,
                        nil,
                        t.topicsList,
                ),
        }
        t.container.Refresh()
}

// ------------------------------------------
// EDIT TOPIC DIALOG
// ------------------------------------------

func (t *TopicsTab) showEditTopicDialog(idx int) {
        if idx < 0 || idx >= len(t.dates) {
                return
        }
        day := t.dates[idx]

        topicEntry := widget.NewMultiLineEntry()
        topicEntry.SetText(day.Topic)
        topicEntry.SetPlaceHolder("Введите тему урока...")

        hwEntry := widget.NewMultiLineEntry()
        hwEntry.SetText(day.HomeWork)
        hwEntry.SetPlaceHolder("Введите домашнее задание...")

        form := widget.NewForm(
                &widget.FormItem{Text: "Тема", Widget: topicEntry},
                &widget.FormItem{Text: "ДЗ", Widget: hwEntry},
        )

        dialogTitle := fmt.Sprintf("Редактирование: %s (%s)", day.AssignmentDate, day.WeekdayName)

        var dlg dialog.Dialog
        saveBtn := widget.NewButton("Сохранить", func() {
                dlg.Hide()
                newTopic := topicEntry.Text
                newHW := hwEntry.Text
                if newTopic != day.Topic || newHW != day.HomeWork {
                        go t.updateAssignment(day.AssignmentDateID, newTopic, newHW)
                }
        })
        saveBtn.Importance = widget.HighImportance

        cancelBtn := widget.NewButton("Отмена", func() {
                dlg.Hide()
        })

        content := container.NewVBox(
                widget.NewLabel(dialogTitle),
                widget.NewSeparator(),
                form,
                container.NewHBox(saveBtn, cancelBtn),
        )

        dlg = dialog.NewCustomWithoutButtons("Тема урока", content, t.controller.GetWindow())
        dlg.Show()
}

// updateAssignment saves a topic and homework for a single date via the API.
func (t *TopicsTab) updateAssignment(dateID, topic, homework string) {
        fyne.Do(func() {
                t.statusLabel.SetText("Сохранение темы...")
        })

        err := t.controller.GetClient().UpdateAssignment(dateID, topic, homework)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка сохранения темы: %v", err), t.controller.GetWindow())
                        t.statusLabel.SetText("Ошибка сохранения темы")
                } else {
                        t.statusLabel.SetText("Тема успешно сохранена")
                        go t.loadData()
                }
        })
}

// ------------------------------------------
// SEQUENTIAL FILL
// ------------------------------------------

// onFill handles the "Заполнить" button click.
// It collects all dates matching the selected weight, fills empty topics
// sequentially from the selected quality pool (in order, no randomness),
// and saves each via API.
func (t *TopicsTab) onFill() {
        if t.selectedGroup == nil || t.selectedSubject == nil {
                dialog.ShowError(fmt.Errorf("Выберите класс и предмет"), t.controller.GetWindow())
                return
        }

        weight := t.weightSel.Selected
        quality := t.qualitySel.Selected

        if weight == "" || quality == "" {
                dialog.ShowError(fmt.Errorf("Выберите диапазон дат и уровень тем"), t.controller.GetWindow())
                return
        }

        // Verify the quality level has templates (using shared TopicTemplates)
        if _, ok := TopicTemplates[quality]; !ok {
                dialog.ShowError(fmt.Errorf("Неизвестный уровень тем: %s", quality), t.controller.GetWindow())
                return
        }

        // Determine which quarter IDs to load dates for
        quarterIDs := t.getQuarterIDsForWeight(weight)
        if len(quarterIDs) == 0 {
                dialog.ShowError(fmt.Errorf("Не удалось определить четверти для диапазона: %s", weight), t.controller.GetWindow())
                return
        }

        t.fillBtn.Disable()
        t.fillBtn.SetText("Заполнение...")

        go t.doFill(quarterIDs, quality, weight)
}

// getQuarterIDsForWeight returns the quarter IDs matching the selected weight scope.
func (t *TopicsTab) getQuarterIDsForWeight(weight string) []int {
        if t.selectedGroup == nil {
                return nil
        }

        quarters := t.selectedGroup.Quarters
        var ids []int

        switch weight {
        case "Полугодие 1":
                // Quarters 1 and 2 (first semester)
                for _, q := range quarters {
                        qNum := quarterNumberFromName(q.Name)
                        if qNum == 1 || qNum == 2 {
                                ids = append(ids, q.ID)
                        }
                }
        case "Полугодие 2":
                // Quarters 3 and 4 (second semester)
                for _, q := range quarters {
                        qNum := quarterNumberFromName(q.Name)
                        if qNum == 3 || qNum == 4 {
                                ids = append(ids, q.ID)
                        }
                }
        case "Весь год":
                for _, q := range quarters {
                        ids = append(ids, q.ID)
                }
        case "До текущей даты":
                // Load all quarters, filter by date later
                for _, q := range quarters {
                        ids = append(ids, q.ID)
                }
        }

        return ids
}

// quarterNumberFromName extracts the quarter number from a quarter name.
// Handles names like "1 четверть", "Четверть 1", "I четверть", etc.
func quarterNumberFromName(name string) int {
        // Map of Roman numerals to numbers
        romanMap := map[string]int{
                "I": 1, "II": 2, "III": 3, "IV": 4,
                "1": 1, "2": 2, "3": 3, "4": 4,
        }

        // Try to find a number in the name
        numStr := ""
        for i := 0; i < len(name); i++ {
                ch := name[i]
                if ch >= '0' && ch <= '9' {
                        numStr += string(ch)
                } else if numStr != "" {
                        break
                }
        }

        if numStr != "" {
                n := 0
                for _, ch := range numStr {
                        n = n*10 + int(ch-'0')
                }
                return n
        }

        // Try Roman numerals
        for roman, num := range romanMap {
                if len(roman) > len(numStr) {
                        numStr = roman
                }
                // Simple substring check
                for i := 0; i <= len(name)-len(roman); i++ {
                        if name[i:i+len(roman)] == roman {
                                return num
                        }
                }
        }

        return 0
}

// doFill loads dates for all relevant quarters, fills empty topics sequentially
// (in order from the quality pool, cycling through if needed),
// and saves each change via the API.
func (t *TopicsTab) doFill(quarterIDs []int, quality string, weight string) {
        apiClient := t.controller.GetClient()
        gID := t.selectedGroup.ID
        sID := t.selectedSubject.SubjectID

        now := time.Now()
        todayStr := now.Format("2006-01-02")

        // Collect dates from all relevant quarters, along with their quarter name
        type dateWithQuarter struct {
                day        client.Day
                quarterNum int
        }

        var allDates []dateWithQuarter
        var updateItems []struct {
                dateID   string
                topic    string
                homework string
        }

        for _, qID := range quarterIDs {
                // Find the quarter name for this ID
                qName := ""
                qNum := 0
                for _, q := range t.selectedGroup.Quarters {
                        if q.ID == qID {
                                qName = q.Name
                                qNum = quarterNumberFromName(q.Name)
                                break
                        }
                }

                dates, err := apiClient.GetJournalDates(gID, sID, qID)
                if err != nil {
                        fyne.Do(func() {
                                dialog.ShowError(fmt.Errorf("Ошибка загрузки дат (четверть %s): %v", qName, err), t.controller.GetWindow())
                                t.fillBtn.Enable()
                                t.fillBtn.SetText("Заполнить")
                        })
                        return
                }

                for _, d := range dates {
                        allDates = append(allDates, dateWithQuarter{day: d, quarterNum: qNum})
                }
        }

        // Determine which dates need filling
        fillIdx := 0 // sequential index for cycling through topic templates
        for _, dqw := range allDates {
                day := dqw.day

                if day.Topic != "" {
                        continue // skip non-empty topics
                }

                // Apply date range filter using ShouldFillDate from random_engine.go
                if !ShouldFillDate(weight, fmt.Sprintf("Четверть %d", dqw.quarterNum), todayStr, day.AssignmentDate) {
                        continue
                }

                // Pick topic sequentially from the quality pool (in order, cycling)
                topicText := SequentialTopicForDiligence(quality, fillIdx)
                fillIdx++

                updateItems = append(updateItems, struct {
                        dateID   string
                        topic    string
                        homework string
                }{
                        dateID:   day.AssignmentDateID,
                        topic:    topicText,
                        homework: day.HomeWork, // preserve existing homework
                })
        }

        if len(updateItems) == 0 {
                fyne.Do(func() {
                        dialog.ShowInformation("Заполнение", "Нет пустых тем для заполнения в выбранном диапазоне.", t.controller.GetWindow())
                        t.fillBtn.Enable()
                        t.fillBtn.SetText("Заполнить")
                })
                return
        }

        // Confirm with the user
        confirmText := fmt.Sprintf("Будет заполнено %d пустых тем уровня «%s» по порядку.\nПродолжить?", len(updateItems), quality)
        confirmed := make(chan bool, 1)

        fyne.Do(func() {
                dialog.ShowConfirm("Заполнить", confirmText, func(ok bool) {
                        confirmed <- ok
                }, t.controller.GetWindow())
        })

        if !<-confirmed {
                fyne.Do(func() {
                        t.fillBtn.Enable()
                        t.fillBtn.SetText("Заполнить")
                })
                return
        }

        // Send updates
        successCount := 0
        errorCount := 0

        for i, item := range updateItems {
                fyne.Do(func() {
                        t.statusLabel.SetText(fmt.Sprintf("Заполнение %d/%d...", i+1, len(updateItems)))
                })

                err := apiClient.UpdateAssignment(item.dateID, item.topic, item.homework)
                if err != nil {
                        errorCount++
                } else {
                        successCount++
                }
        }

        fyne.Do(func() {
                t.fillBtn.Enable()
                t.fillBtn.SetText("Заполнить")

                if errorCount > 0 {
                        t.statusLabel.SetText(fmt.Sprintf("Заполнено %d тем, ошибок: %d", successCount, errorCount))
                        dialog.ShowError(fmt.Errorf("Заполнено %d из %d тем. Ошибок: %d", successCount, len(updateItems), errorCount), t.controller.GetWindow())
                } else {
                        t.statusLabel.SetText(fmt.Sprintf("Успешно заполнено %d тем", successCount))
                        dialog.ShowInformation("Готово", fmt.Sprintf("Успешно заполнено %d тем уровня «%s».", successCount, quality), t.controller.GetWindow())
                }

                // Reload current quarter data to reflect changes
                if t.selectedQuarter != nil {
                        go t.loadData()
                }
        })
}

// Refresh updates the tab with new data from the dashboard context.
// It receives the dates, group, and subject from the dashboard and triggers a reload
// if the tab's current filters match or if data is stale.
func (t *TopicsTab) Refresh(dates []client.Day, group *client.JournalGroup, subject *client.Subject) {
        // Update internal state with dashboard-provided data
        if group != nil {
                t.selectedGroup = group
        }
        if subject != nil {
                t.selectedSubject = subject
        }
        if len(dates) > 0 {
                t.dates = dates
                t.rebuildTopicsList()
        }
}
