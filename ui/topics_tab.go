package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/client"
)

// Simple topic templates — just a pool of lesson topics that cycle sequentially.
var simpleTopicPool = []string{
	"Изучение нового материала",
	"Закрепление пройденного",
	"Решение задач и упражнений",
	"Самостоятельная работа",
	"Объяснение новой темы",
	"Практическая работа",
	"Повторение материала",
	"Контрольная работа",
	"Работа с учебником",
	"Устный опрос",
	"Комбинированный урок",
	"Обобщение и систематизация знаний",
	"Беседа по теме",
	"Работа над ошибками",
	"Проверка знаний",
}

// Simple homework templates — pool that cycles sequentially.
var simpleHWPool = []string{
	"Параграф прочитать, вопросы ответить",
	"Упражнения выполнить",
	"Задачи решить",
	"Повторить пройденный материал",
	"Подготовиться к контрольной",
	"Конспект составить",
	"Тест выполнить",
	"Доклад подготовить",
	"Реферат написать",
	"Практическое задание выполнить",
}

// TopicsTab manages the Topics (Темы) tab with simple sequential fill.
// The user's request: just load topics by rows — first row = first topic by date,
// second row = second date, etc. Same for homework.
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
	topicsList  *widget.List
	statusLabel *widget.Label
	fillTopicsBtn *widget.Button
	fillHWBtn     *widget.Button
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

	// --- Fill buttons ---
	t.fillTopicsBtn = widget.NewButton("Заполнить темы", t.onFillTopics)
	t.fillTopicsBtn.Importance = widget.HighImportance
	t.fillTopicsBtn.Disable()

	t.fillHWBtn = widget.NewButton("Заполнить ДЗ", t.onFillHW)
	t.fillHWBtn.Importance = widget.WarningImportance
	t.fillHWBtn.Disable()

	actionRow := container.NewHBox(
		t.fillTopicsBtn,
		t.fillHWBtn,
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

		t.fillTopicsBtn.Disable()
		t.fillHWBtn.Disable()

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
		t.fillTopicsBtn.Enable()
		t.fillHWBtn.Enable()
		go t.loadData()
	} else {
		t.fillTopicsBtn.Disable()
		t.fillHWBtn.Disable()
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
// TOPICS LIST — simple rows showing date + topic + homework
// ------------------------------------------

func (t *TopicsTab) rebuildTopicsList() {
	dates := t.dates

	if len(dates) == 0 {
		placeholder := widget.NewLabelWithStyle("Нет дат для выбранной четверти", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		t.container.Objects = []fyne.CanvasObject{
			container.NewBorder(
				container.NewVBox(
					container.NewHBox(widget.NewLabel("Фильтры:"), t.classSel, t.subjectSel, t.quarterSel),
					container.NewHBox(t.fillTopicsBtn, t.fillHWBtn),
					widget.NewSeparator(),
				),
				t.statusLabel,
				nil,
				nil,
				placeholder,
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
			// Row number
			numLabel := widget.NewLabel("")
			numLabel.TextStyle = fyne.TextStyle{Bold: true}

			// Date + weekday
			dateLabel := widget.NewLabel("")
			dayLabel := widget.NewLabel("")
			dayLabel.TextStyle = fyne.TextStyle{Italic: true}

			// Topic
			topicLabel := widget.NewLabel("")
			topicLabel.Wrapping = fyne.TextWrapWord

			// Homework
			hwLabel := widget.NewLabel("")
			hwLabel.Wrapping = fyne.TextWrapWord
			hwLabel.TextStyle = fyne.TextStyle{Italic: true}

			leftCol := container.NewVBox(numLabel, dateLabel, dayLabel)
			rightCol := container.NewVBox(topicLabel, hwLabel)

			return container.NewBorder(nil, nil, leftCol, nil, rightCol)
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			if id < 0 || id >= len(dates) {
				return
			}
			day := dates[id]

			border := cell.(*fyne.Container)
			leftCol := border.Objects[0].(*fyne.Container)
			rightCol := border.Objects[1].(*fyne.Container)

			numLabel := leftCol.Objects[0].(*widget.Label)
			dateLabel := leftCol.Objects[1].(*widget.Label)
			dayLabel := leftCol.Objects[2].(*widget.Label)

			topicLabel := rightCol.Objects[0].(*widget.Label)
			hwLabel := rightCol.Objects[1].(*widget.Label)

			numLabel.SetText(fmt.Sprintf("%d", id+1))
			dateLabel.SetText(day.AssignmentDate)
			dayLabel.SetText(day.WeekdayName)

			if day.Topic != "" {
				topicLabel.SetText(day.Topic)
				topicLabel.TextStyle = fyne.TextStyle{}
			} else {
				topicLabel.SetText("(пусто — нажмите для ввода)")
				topicLabel.TextStyle = fyne.TextStyle{Italic: true}
			}

			if day.HomeWork != "" {
				hwLabel.SetText("ДЗ: " + day.HomeWork)
				hwLabel.TextStyle = fyne.TextStyle{Italic: true}
			} else {
				hwLabel.SetText("")
			}

			numLabel.Refresh()
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

	// Rebuild the container with the list as center
	topBar := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Фильтры:"),
			t.classSel,
			t.subjectSel,
			t.quarterSel,
		),
		container.NewHBox(
			t.fillTopicsBtn,
			t.fillHWBtn,
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

	dialogTitle := fmt.Sprintf("Строка %d: %s (%s)", idx+1, day.AssignmentDate, day.WeekdayName)

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
		widget.NewLabel("Тема:"),
		topicEntry,
		widget.NewLabel("ДЗ:"),
		hwEntry,
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
// SEQUENTIAL FILL — row 1 = template 1, row 2 = template 2, etc.
// ------------------------------------------

// onFillTopics fills empty topic fields sequentially from the pool.
// Row 1 gets template 0, row 2 gets template 1, etc. Cycles through the pool.
func (t *TopicsTab) onFillTopics() {
	if t.selectedGroup == nil || t.selectedSubject == nil || t.selectedQuarter == nil {
		dialog.ShowError(fmt.Errorf("Выберите класс, предмет и четверть"), t.controller.GetWindow())
		return
	}
	if len(t.dates) == 0 {
		dialog.ShowInformation("Внимание", "Нет дат для заполнения", t.controller.GetWindow())
		return
	}

	// Count empty topics
	emptyCount := 0
	for _, d := range t.dates {
		if d.Topic == "" {
			emptyCount++
		}
	}
	if emptyCount == 0 {
		dialog.ShowInformation("Заполнение", "Все темы уже заполнены", t.controller.GetWindow())
		return
	}

	dialog.ShowConfirm("Заполнить темы",
		fmt.Sprintf("Будет заполнено %d пустых тем по порядку.\nСтрока 1 = тема 1, строка 2 = тема 2 и т.д.\nПродолжить?", emptyCount),
		func(ok bool) {
			if !ok {
				return
			}
			go t.doFillTopics()
		}, t.controller.GetWindow())
}

// doFillTopics fills empty topic fields sequentially.
func (t *TopicsTab) doFillTopics() {
	apiClient := t.controller.GetClient()
	fillIdx := 0 // Sequential counter for cycling through template pool
	successCount := 0
	errorCount := 0

	for i, day := range t.dates {
		if day.Topic != "" {
			continue // skip non-empty
		}

		topicText := simpleTopicPool[fillIdx%len(simpleTopicPool)]
		fillIdx++

		fyne.Do(func() {
			t.statusLabel.SetText(fmt.Sprintf("Заполнение %d/%d...", i+1, len(t.dates)))
		})

		err := apiClient.UpdateAssignment(day.AssignmentDateID, topicText, day.HomeWork)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	fyne.Do(func() {
		if errorCount > 0 {
			t.statusLabel.SetText(fmt.Sprintf("Заполнено %d тем, ошибок: %d", successCount, errorCount))
		} else {
			t.statusLabel.SetText(fmt.Sprintf("Успешно заполнено %d тем", successCount))
		}
		go t.loadData()
	})
}

// onFillHW fills empty homework fields sequentially from the pool.
// Row 1 gets HW template 0, row 2 gets HW template 1, etc.
func (t *TopicsTab) onFillHW() {
	if t.selectedGroup == nil || t.selectedSubject == nil || t.selectedQuarter == nil {
		dialog.ShowError(fmt.Errorf("Выберите класс, предмет и четверть"), t.controller.GetWindow())
		return
	}
	if len(t.dates) == 0 {
		dialog.ShowInformation("Внимание", "Нет дат для заполнения", t.controller.GetWindow())
		return
	}

	// Count empty homework
	emptyCount := 0
	for _, d := range t.dates {
		if d.HomeWork == "" {
			emptyCount++
		}
	}
	if emptyCount == 0 {
		dialog.ShowInformation("Заполнение", "Все ДЗ уже заполнены", t.controller.GetWindow())
		return
	}

	dialog.ShowConfirm("Заполнить ДЗ",
		fmt.Sprintf("Будет заполнено %d пустых ДЗ по порядку.\nСтрока 1 = ДЗ 1, строка 2 = ДЗ 2 и т.д.\nПродолжить?", emptyCount),
		func(ok bool) {
			if !ok {
				return
			}
			go t.doFillHW()
		}, t.controller.GetWindow())
}

// doFillHW fills empty homework fields sequentially.
func (t *TopicsTab) doFillHW() {
	apiClient := t.controller.GetClient()
	fillIdx := 0
	successCount := 0
	errorCount := 0

	for i, day := range t.dates {
		if day.HomeWork != "" {
			continue // skip non-empty
		}

		hwText := simpleHWPool[fillIdx%len(simpleHWPool)]
		fillIdx++

		fyne.Do(func() {
			t.statusLabel.SetText(fmt.Sprintf("Заполнение ДЗ %d/%d...", i+1, len(t.dates)))
		})

		err := apiClient.UpdateAssignment(day.AssignmentDateID, day.Topic, hwText)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	fyne.Do(func() {
		if errorCount > 0 {
			t.statusLabel.SetText(fmt.Sprintf("Заполнено %d ДЗ, ошибок: %d", successCount, errorCount))
		} else {
			t.statusLabel.SetText(fmt.Sprintf("Успешно заполнено %d ДЗ", successCount))
		}
		go t.loadData()
	})
}

// Refresh updates the tab with new data from the dashboard context.
func (t *TopicsTab) Refresh(dates []client.Day, group *client.JournalGroup, subject *client.Subject) {
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
