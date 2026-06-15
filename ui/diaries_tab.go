package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/client"
)

// DiariesTab manages the Diaries (Дневник) tab.
//
// In edonish.tj, diary data comes from /journal/students — each student has
// marks and comments per date. Teachers write comments using
// POST /journal/comment with the student's group_subgroup_student_id.
//
// This tab works like a real school diary:
//   - Teacher writes a comment about the student (praise, complaint, behavior note)
//   - The comment is stored via /journal/comment API
//   - Parent signs to acknowledge
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
	studentsList *widget.List
	statusLabel  *widget.Label
}

// NewDiariesTab creates a new DiariesTab.
func NewDiariesTab(c Controller) *DiariesTab {
	dt := &DiariesTab{
		controller:  c,
		statusLabel: widget.NewLabel("Выберите класс, предмет и четверть"),
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

	filterRow := container.NewHBox(
		widget.NewLabel("Фильтры:"),
		dt.classSel,
		dt.subjectSel,
		dt.quarterSel,
	)

	// Batch actions
	batchPraiseBtn := widget.NewButton("Подписать: Похвала", func() {
		dt.onBatchSignWithCategory(BehaviorPraise)
	})
	batchPraiseBtn.Importance = widget.HighImportance

	batchMixedBtn := widget.NewButton("Подписать: Смешанный", func() {
		dt.onBatchSignWithCategory(BehaviorMixed)
	})

	batchComplaintBtn := widget.NewButton("Подписать: Жалоба", func() {
		dt.onBatchSignWithCategory(BehaviorComplaint)
	})

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		go dt.loadData()
	})

	actionRow := container.NewHBox(
		batchPraiseBtn,
		batchMixedBtn,
		batchComplaintBtn,
		refreshBtn,
	)

	placeholder := widget.NewLabelWithStyle(
		"Выберите класс, предмет и четверть\n\n"+
			"В дневнике учитель пишет комментарий о поведении ученика,\n"+
			"а родитель подписывает и может ответить.",
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

// loadJournalOptions loads class list from API.
func (dt *DiariesTab) loadJournalOptions() {
	dt.statusLabel.SetText("Загрузка списка классов...")
	opts, err := dt.controller.GetClient().GetJournalOptions()
	if err != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки настроек журнала: %v", err))
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

// onClassSelected handles class filter selection.
func (dt *DiariesTab) onClassSelected(selected string) {
	if dt.journalOpts == nil {
		return
	}

	var group *client.JournalGroup
	for i, g := range dt.journalOpts.Groups {
		gName := fmt.Sprintf("%d %s", g.Number, g.Name)
		if gName == selected {
			group = &dt.journalOpts.Groups[i]
			break
		}
	}
	if group == nil {
		return
	}

	dt.selectedGroup = group
	dt.selectedSubject = nil
	dt.selectedQuarter = nil
	dt.students = nil
	dt.dates = nil

	subjectNames := make([]string, len(group.Subjects))
	for i, s := range group.Subjects {
		subjectNames[i] = s.SubjectName
	}

	quarterNames := make([]string, len(group.Quarters))
	for i, q := range group.Quarters {
		quarterNames[i] = q.Name
	}

	fyne.Do(func() {
		dt.subjectSel.Options = subjectNames
		dt.subjectSel.Refresh()
		dt.subjectSel.ClearSelected()

		dt.quarterSel.Options = quarterNames
		dt.quarterSel.Refresh()
		dt.quarterSel.ClearSelected()

		// Auto select first subject
		if len(subjectNames) > 0 {
			dt.subjectSel.SetSelectedIndex(0)
		}
		// Auto select current quarter
		for i, q := range group.Quarters {
			if q.CurrentQuarter {
				dt.quarterSel.SetSelectedIndex(i)
				break
			}
		}
	})
}

// onSubjectSelected handles subject filter selection.
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

// onQuarterSelected handles quarter filter selection.
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

// tryLoadData loads data if all filters are selected.
func (dt *DiariesTab) tryLoadData() {
	if dt.selectedGroup != nil && dt.selectedSubject != nil && dt.selectedQuarter != nil {
		go dt.loadData()
	}
}

// loadData loads students and dates from the journal API.
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
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки учеников: %v", errS))
			dialog.ShowError(fmt.Errorf("Ошибка загрузки учеников: %v", errS), dt.controller.GetWindow())
			return
		}
		if errD != nil {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки дат: %v", errD))
			dialog.ShowError(fmt.Errorf("Ошибка загрузки дат: %v", errD), dt.controller.GetWindow())
			return
		}

		dt.students = students
		dt.dates = dates

		if len(students) == 0 {
			dt.statusLabel.SetText("Нет учеников для выбранных фильтров")
		} else {
			dt.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников, %d дат", len(students), len(dates)))
		}

		dt.rebuildStudentsList()
	})
}

// rebuildStudentsList builds the list of students with their diary status.
func (dt *DiariesTab) rebuildStudentsList() {
	// Build top bar components
	filterRow := container.NewHBox(
		widget.NewLabel("Фильтры:"), dt.classSel, dt.subjectSel, dt.quarterSel,
	)
	actionRow := container.NewHBox(
		widget.NewButton("Подписать: Похвала", func() { dt.onBatchSignWithCategory(BehaviorPraise) }),
		widget.NewButton("Подписать: Смешанный", func() { dt.onBatchSignWithCategory(BehaviorMixed) }),
		widget.NewButton("Подписать: Жалоба", func() { dt.onBatchSignWithCategory(BehaviorComplaint) }),
		widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() { go dt.loadData() }),
	)
	topBar := container.NewVBox(filterRow, actionRow, widget.NewSeparator())

	if len(dt.students) == 0 {
		dt.container.Objects = []fyne.CanvasObject{
			container.NewBorder(
				topBar,
				dt.statusLabel,
				nil, nil,
				widget.NewLabelWithStyle("Нет учеников для выбранных фильтров", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
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

			return container.NewBorder(nil, nil,
				container.NewHBox(numLabel, nameLabel),
				nil,
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

			numLabel := leftBox.Objects[0].(*widget.Label)
			nameLabel := leftBox.Objects[1].(*widget.Label)

			avgLabel := rightBox.Objects[0].(*widget.Label)
			marksLabel := rightBox.Objects[1].(*widget.Label)

			numLabel.SetText(fmt.Sprintf("%d.", id+1))
			nameLabel.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))

			if student.AverageScore != "" && student.AverageScore != "0.0" {
				avgLabel.SetText(fmt.Sprintf("Ср. балл: %s", student.AverageScore))
			} else {
				avgLabel.SetText("")
			}

			// Count marks
			markCount := 0
			for _, sm := range student.SubjectMarks {
				if sm.ShortName != "" && sm.ShortName != "—" {
					markCount++
				}
			}
			marksLabel.SetText(fmt.Sprintf("Оценок: %d / Дат: %d", markCount, len(dt.dates)))

			numLabel.Refresh()
			nameLabel.Refresh()
			avgLabel.Refresh()
			marksLabel.Refresh()
		},
	)

	dt.studentsList.OnSelected = func(id widget.ListItemID) {
		dt.studentsList.Unselect(id)
		dt.showStudentDiaryDialog(id)
	}

	dt.container.Objects = []fyne.CanvasObject{
		container.NewBorder(
			topBar,
			dt.statusLabel,
			nil, nil,
			dt.studentsList,
		),
	}
	dt.container.Refresh()
}

// showStudentDiaryDialog shows a dialog for a single student with diary comment functionality.
func (dt *DiariesTab) showStudentDiaryDialog(idx int) {
	if idx < 0 || idx >= len(dt.students) {
		return
	}
	student := dt.students[idx]

	var dlg dialog.Dialog

	headerText := fmt.Sprintf("%s %s — %s (%s)",
		student.LastName, student.FirstName,
		dt.selectedSubject.SubjectName, dt.selectedQuarter.Name)
	headerLabel := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Average score
	avgText := "Средний балл: "
	if student.AverageScore != "" && student.AverageScore != "0.0" {
		avgText += student.AverageScore
	} else {
		avgText += "—"
	}

	// --- CONVERSATION AREA ---
	teacherBubbleTitle := canvas.NewText("Учитель:", color.NRGBA{R: 37, G: 99, B: 235, A: 255})
	teacherBubbleTitle.TextStyle = fyne.TextStyle{Bold: true}
	teacherBubbleTitle.TextSize = 13

	// Count existing marks
	existingMarks := 0
	for _, sm := range student.SubjectMarks {
		if sm.ShortName != "" && sm.ShortName != "—" {
			existingMarks++
		}
	}
	teacherCommentText := fmt.Sprintf("Оценок: %d из %d дат", existingMarks, len(dt.dates))
	teacherCommentLabel := widget.NewLabel(teacherCommentText)

	teacherBubble := container.NewVBox(teacherBubbleTitle, teacherCommentLabel)

	parentBubbleTitle := canvas.NewText("Родитель:", color.NRGBA{R: 22, G: 163, B: 74, A: 255})
	parentBubbleTitle.TextStyle = fyne.TextStyle{Bold: true}
	parentBubbleTitle.TextSize = 13
	parentCommentLabel := widget.NewLabel("(подпись через комментарий)")
	parentBubble := container.NewVBox(parentBubbleTitle, parentCommentLabel)

	// --- ACTION AREA ---
	// Behavior category selector
	behaviorSel := widget.NewSelect(BehaviorCategories, nil)
	behaviorSel.PlaceHolder = "Категория комментария..."

	// Quick comment templates
	quickCommentSel := widget.NewSelect([]string{}, nil)
	quickCommentSel.PlaceHolder = "Выберите шаблон комментария..."

	behaviorSel.OnChanged = func(cat string) {
		templates := BehaviorTemplates[BehaviorCategory(cat)]
		opts := make([]string, len(templates))
		copy(opts, templates)
		quickCommentSel.Options = opts
		quickCommentSel.Refresh()
		if len(opts) > 0 {
			quickCommentSel.SetSelectedIndex(0)
		}
	}

	// Custom comment entry
	commentEntry := widget.NewMultiLineEntry()
	commentEntry.SetPlaceHolder("Введите комментарий о поведении ученика...")
	commentEntry.Wrapping = fyne.TextWrapWord

	quickCommentSel.OnChanged = func(selected string) {
		if selected != "" {
			commentEntry.SetText(selected)
		}
	}

	// Write comment button — sends via /journal/comment API
	writeCommentBtn := widget.NewButton("Написать комментарий", func() {
		comment := commentEntry.Text
		if comment == "" {
			dialog.ShowInformation("Внимание", "Напишите комментарий", dt.controller.GetWindow())
			return
		}
		dlg.Hide()
		go dt.writeCommentForStudent(student.StudentID, comment, idx)
	})
	writeCommentBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		headerLabel,
		widget.NewLabel(avgText),
		widget.NewSeparator(),
		teacherBubble,
		parentBubble,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Написать комментарий:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(widget.NewLabel("Категория:"), behaviorSel),
		container.NewHBox(widget.NewLabel("Шаблон:"), quickCommentSel),
		commentEntry,
		writeCommentBtn,
	)

	dlg = dialog.NewCustom("Дневник ученика", "Закрыть", content, dt.controller.GetWindow())
	dlg.Show()
}

// writeCommentForStudent writes a diary comment for a student using /journal/comment API.
// It writes the same comment on the first available date in the selected quarter.
func (dt *DiariesTab) writeCommentForStudent(studentID int, comment string, idx int) {
	if dt.selectedQuarter == nil || len(dt.dates) == 0 {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("Нет дат для записи комментария"), dt.controller.GetWindow())
		})
		return
	}

	apiClient := dt.controller.GetClient()
	qID := dt.selectedQuarter.ID

	fyne.Do(func() {
		dt.statusLabel.SetText(fmt.Sprintf("Запись комментария для ученика %d...", studentID))
	})

	// Write comment on the first available date
	successCount := 0
	for _, d := range dt.dates {
		if d.AssignmentDateID == "" {
			continue
		}
		err := apiClient.CreateDiaryComment(studentID, d.AssignmentDateID, qID, comment)
		if err != nil {
			fyne.Do(func() {
				dt.statusLabel.SetText(fmt.Sprintf("Ошибка записи комментария: %v", err))
			})
			continue
		}
		successCount++
		break // One comment per student is enough
	}

	fyne.Do(func() {
		if successCount > 0 {
			dt.statusLabel.SetText(fmt.Sprintf("Комментарий записан для ученика %d", studentID))
		}
	})
}

// onBatchSignWithCategory writes behavior comments for all students in the selected quarter.
func (dt *DiariesTab) onBatchSignWithCategory(category BehaviorCategory) {
	if len(dt.students) == 0 {
		dialog.ShowInformation("Внимание", "Нет учеников для подписания.\nВыберите класс, предмет и четверть.", dt.controller.GetWindow())
		return
	}
	if dt.selectedQuarter == nil || len(dt.dates) == 0 {
		dialog.ShowInformation("Внимание", "Выберите четверть и загрузите данные", dt.controller.GetWindow())
		return
	}

	diligence := BehaviorToDiligence[category]
	templates := BehaviorTemplates[category]
	exampleComment := ""
	if len(templates) > 0 {
		exampleComment = templates[0]
	}

	confirmMsg := fmt.Sprintf(
		"Будет установлено:\n"+
			"  Прилежание: «%s»\n"+
			"  Категория комментария: «%s»\n"+
			"  Пример комментария: «%s»\n\n"+
			"Для %d учеников будут записаны комментарии по порядку.\n\n"+
			"Продолжить?",
		diligence, string(category), exampleComment, len(dt.students),
	)

	dialog.ShowConfirm("Подписать все", confirmMsg, func(ok bool) {
		if !ok {
			return
		}
		go dt.executeBatchSignWithComments(category)
	}, dt.controller.GetWindow())
}

// executeBatchSignWithComments performs batch signing with behavior comments.
func (dt *DiariesTab) executeBatchSignWithComments(category BehaviorCategory) {
	total := len(dt.students)
	apiClient := dt.controller.GetClient()
	qID := dt.selectedQuarter.ID

	// Find first available date for comments
	var firstDateID string
	for _, d := range dt.dates {
		if d.AssignmentDateID != "" {
			firstDateID = d.AssignmentDateID
			break
		}
	}
	if firstDateID == "" {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("Нет доступных дат для записи комментариев"), dt.controller.GetWindow())
		})
		return
	}

	errorCount := 0
	successCount := 0

	for i, student := range dt.students {
		progress := fmt.Sprintf("Обработка %d из %d: %s %s", i+1, total, student.LastName, student.FirstName)
		fyne.Do(func() {
			dt.statusLabel.SetText(progress)
		})

		comment := SequentialBehaviorComment(category, i)

		err := apiClient.CreateDiaryComment(student.StudentID, firstDateID, qID, comment)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	fyne.Do(func() {
		dt.statusLabel.SetText(fmt.Sprintf("Готово! Обработано %d учеников (ошибок: %d)", successCount, errorCount))
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
