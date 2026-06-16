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
//  1. Groups, students, and behavior options are loaded automatically
//  2. User selects a student from the list
//  3. User presses one of the diligence buttons:
//     [Отлично] [Отлично && Хорошо] [Хорошо && Удовлетворительный] [Неудовлетворительный]
//  4. System signs diary entries with the appropriate behavior for all weeks
//     up to the current date.
//
// No manual class/subject/quarter selection — everything is auto-detected.
// Only the class teacher (классный руководитель) has the right to sign.
type DiariesTab struct {
	controller Controller
	container  *fyne.Container

	// State
	groups          []client.MyClassGroup
	students        []client.MyClassStudent
	behaviorOptions []client.DiaryBehaviorOption
	selectedGroup   *client.MyClassGroup

	// UI
	selectedStudentIdx int // -1 = none selected
	studentsList       *widget.List
	statusLabel        *widget.Label
	diligenceButtons   []*widget.Button
	refreshBtn         *widget.Button
}

// NewDiariesTab creates a new DiariesTab.
func NewDiariesTab(c Controller) *DiariesTab {
	dt := &DiariesTab{
		controller:        c,
		statusLabel:       widget.NewLabel("Загрузка..."),
		selectedStudentIdx: -1,
	}
	dt.buildUI()
	go dt.loadInitialData()
	return dt
}

// Container returns the root container for this tab.
func (dt *DiariesTab) Container() fyne.CanvasObject {
	return dt.container
}

// buildUI creates the full UI layout for the diaries tab.
func (dt *DiariesTab) buildUI() {
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

	dt.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go dt.loadInitialData()
	})

	headerRow := container.NewHBox(
		widget.NewLabelWithStyle("Дневник", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dt.refreshBtn,
	)

	placeholder := widget.NewLabelWithStyle(
		"Выберите ученика и нажмите кнопку прилежания:\n"+
			"«Отлично», «Отлично && Хорошо» и т.д.\n\n"+
			"Система автоматически заполнит дневник до текущей даты.",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	dt.container = container.NewBorder(
		container.NewVBox(headerRow, actionRow, widget.NewSeparator()),
		dt.statusLabel,
		nil,
		nil,
		placeholder,
	)
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

// loadInitialData loads groups, students, and behavior options automatically.
func (dt *DiariesTab) loadInitialData() {
	fyne.Do(func() {
		dt.statusLabel.SetText("Загрузка данных дневника...")
		dt.setDiligenceButtonsEnabled(false)
	})

	apiClient := dt.controller.GetClient()

	// 1. Load groups
	groups, errG := apiClient.GetMyClassGroups()
	if errG != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки классов: %v", errG))
		})
		return
	}

	if len(groups) == 0 {
		fyne.Do(func() {
			dt.statusLabel.SetText("Нет доступных классов")
		})
		return
	}

	dt.groups = groups

	// 2. Auto-select the first group
	dt.selectedGroup = &groups[0]

	// 3. Load students for the selected group
	students, errS := apiClient.GetMyClassStudents(dt.selectedGroup.ID)
	if errS != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки учеников: %v", errS))
		})
		return
	}

	dt.students = students

	// 4. Load behavior options (parallel with students — but we need them both)
	behaviorOpts, errB := apiClient.GetDiaryBehaviorOptions()
	if errB != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки вариантов поведения: %v", errB))
		})
		return
	}

	dt.behaviorOptions = behaviorOpts

	fyne.Do(func() {
		if len(students) == 0 {
			dt.statusLabel.SetText(fmt.Sprintf("Класс «%s» — нет учеников", dt.selectedGroup.Name))
			return
		}
		dt.statusLabel.SetText(fmt.Sprintf("Класс «%s» — %d учеников. Выберите ученика.", dt.selectedGroup.Name, len(students)))
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
			groupLabel := widget.NewLabel("")
			groupLabel.TextStyle = fyne.TextStyle{Italic: true}
			statusIcon := canvas.NewText("", color.NRGBA{R: 22, G: 163, B: 74, A: 255})
			statusIcon.TextSize = 14
			statusIcon.TextStyle = fyne.TextStyle{Bold: true}

			return container.NewBorder(nil, nil,
				container.NewHBox(numLabel, nameLabel),
				statusIcon,
				groupLabel,
			)
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			if id < 0 || id >= len(dt.students) {
				return
			}
			student := dt.students[id]

			border := cell.(*fyne.Container)
			leftBox := border.Objects[0].(*fyne.Container)
			statusIcon := border.Objects[1].(*canvas.Text)
			groupLabel := border.Objects[2].(*widget.Label)

			numLabel := leftBox.Objects[0].(*widget.Label)
			nameLabel := leftBox.Objects[1].(*widget.Label)

			numLabel.SetText(fmt.Sprintf("%d.", id+1))
			nameLabel.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
			groupLabel.SetText(student.GroupName)

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
	headerRow := container.NewHBox(
		widget.NewLabelWithStyle("Дневник", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dt.refreshBtn,
	)
	if dt.selectedGroup != nil {
		headerRow.Objects = append(headerRow.Objects,
			widget.NewLabel(fmt.Sprintf("— %s", dt.selectedGroup.Name)),
		)
	}
	actionRow := container.NewHBox()
	for _, btn := range dt.diligenceButtons {
		actionRow.Add(btn)
	}
	return container.NewVBox(headerRow, actionRow, widget.NewSeparator())
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

// findBehaviorID finds the behavior option ID matching a diligence name.
func (dt *DiariesTab) findBehaviorID(diligence string) int {
	for _, opt := range dt.behaviorOptions {
		if opt.Title == diligence {
			return opt.ID
		}
	}
	return 0
}

func (dt *DiariesTab) onDiligenceButton(comboIdx int) {
	if dt.selectedStudentIdx < 0 || dt.selectedStudentIdx >= len(dt.students) {
		dialog.ShowInformation("Внимание", "Сначала выберите ученика из списка", dt.controller.GetWindow())
		return
	}
	if dt.selectedGroup == nil {
		dialog.ShowInformation("Внимание", "Данные не загружены", dt.controller.GetWindow())
		return
	}

	combo := DiligenceCombos[comboIdx]
	student := dt.students[dt.selectedStudentIdx]

	confirmMsg := fmt.Sprintf(
		"Ученик: %s %s\n"+
			"Действие: %s\n"+
			"Описание: %s\n\n"+
			"Будут подписаны все недели до текущей даты.\n"+
			"Продолжить?",
		student.LastName, student.FirstName,
		combo.Label, combo.Description,
	)

	dialog.ShowConfirm("Заполнить дневник", confirmMsg, func(ok bool) {
		if !ok {
			return
		}
		go dt.executeDiligenceFill(dt.selectedStudentIdx, comboIdx)
	}, dt.controller.GetWindow())
}

// executeDiligenceFill signs the diary for the selected student with the appropriate
// behavior for all weeks up to the current date.
//
// Strategy:
// - For single-behavior combos (Отлично, Неудовлетворительный):
//   Sign each week (Monday-Saturday) from the quarter start to today.
//
// - For mixed-behavior combos (Отлично && Хорошо, Хорошо && Удовлетворительный):
//   Sign each day individually with a random behavior from the pool.
func (dt *DiariesTab) executeDiligenceFill(studentIdx, comboIdx int) {
	if studentIdx < 0 || studentIdx >= len(dt.students) {
		return
	}

	student := dt.students[studentIdx]
	combo := DiligenceCombos[comboIdx]
	apiClient := dt.controller.GetClient()
	today := time.Now()

	// Resolve behavior IDs for each diligence in the combo
	behaviorIDs := make([]int, len(combo.Diligences))
	for i, dil := range combo.Diligences {
		behaviorIDs[i] = dt.findBehaviorID(dil)
	}

	// Check if any behavior IDs are missing
	hasBehaviorID := false
	for _, id := range behaviorIDs {
		if id > 0 {
			hasBehaviorID = true
			break
		}
	}

	// If we couldn't get behavior IDs from the API, fall back to the old
	// CreateDiaryComment method (which uses /journal/comment)
	if !hasBehaviorID {
		dt.executeDiligenceFillLegacy(studentIdx, comboIdx)
		return
	}

	successCount := 0
	failCount := 0

	if len(combo.Diligences) == 1 {
		// Single behavior: sign week by week
		dt.signWeeks(apiClient, student.ID, behaviorIDs[0], today, &successCount, &failCount)
	} else {
		// Mixed behavior: sign day by day with random behavior from pool
		dt.signDays(apiClient, student.ID, behaviorIDs, today, &successCount, &failCount)
	}

	fyne.Do(func() {
		dt.statusLabel.SetText(fmt.Sprintf("Готово: %d подписей для %s %s (ошибок: %d)",
			successCount, student.LastName, student.FirstName, failCount))
	})
}

// signWeeks signs diary weeks from the quarter start to today with a single behavior.
func (dt *DiariesTab) signWeeks(apiClient *client.EdonishClient, studentID, behaviorID int, today time.Time, successCount, failCount *int) {
	// Calculate weeks from September 1 of the current school year to today
	startDate := dt.getSchoolYearStart(today)
	weekStart := startDate

	for !weekStart.After(today) {
		// Week: Monday to Saturday (6 days)
		weekEnd := weekStart.AddDate(0, 0, 5) // Saturday
		if weekEnd.After(today) {
			weekEnd = today
		}

		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Подписываю неделю %s ... %s (%d)",
				weekStart.Format("02.01"), weekEnd.Format("02.01"), *successCount+1))
		})

		err := apiClient.SignDiary(studentID, behaviorID,
			weekStart.Format("2006-01-02"),
			weekEnd.Format("2006-01-02"),
		)
		if err != nil {
			*failCount++
		} else {
			*successCount++
		}

		// Move to next week
		weekStart = weekStart.AddDate(0, 0, 7)
	}
}

// signDays signs diary entries day by day with random behaviors from the pool.
func (dt *DiariesTab) signDays(apiClient *client.EdonishClient, studentID int, behaviorIDs []int, today time.Time, successCount, failCount *int) {
	startDate := dt.getSchoolYearStart(today)
	current := startDate

	for !current.After(today) {
		// Skip Sundays
		if current.Weekday() == time.Sunday {
			current = current.AddDate(0, 0, 1)
			continue
		}

		// Pick a random behavior ID from the pool
		idx := RandomGradeInRange(0, len(behaviorIDs)-1)
		chosenID := behaviorIDs[idx]

		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Подписываю %s (%d)",
				current.Format("02.01"), *successCount+1))
		})

		dateStr := current.Format("2006-01-02")
		err := apiClient.SignDiary(studentID, chosenID, dateStr, dateStr)
		if err != nil {
			*failCount++
		} else {
			*successCount++
		}

		current = current.AddDate(0, 0, 1)
	}
}

// getSchoolYearStart returns September 1 of the current school year.
// In Tajikistan, the school year starts on September 1.
func (dt *DiariesTab) getSchoolYearStart(today time.Time) time.Time {
	year := today.Year()
	// If we're before September, the school year started last year
	if today.Month() < time.September {
		year--
	}
	return time.Date(year, time.September, 1, 0, 0, 0, 0, today.Location())
}

// ------------------------------------------
// LEGACY FALLBACK (uses /journal/comment)
// ------------------------------------------

// executeDiligenceFillLegacy uses the old /journal/comment API as fallback
// when behavior IDs are not available from the /myclass API.
func (dt *DiariesTab) executeDiligenceFillLegacy(studentIdx, comboIdx int) {
	// This fallback uses the Journal API which requires group/subject/quarter
	// We need to load journal options first
	apiClient := dt.controller.GetClient()

	opts, err := apiClient.GetJournalOptions()
	if err != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка (legacy): %v", err))
		})
		return
	}

	if len(opts.Groups) == 0 {
		fyne.Do(func() {
			dt.statusLabel.SetText("Нет классов для заполнения")
		})
		return
	}

	// Use the first group
	group := opts.Groups[0]
	if len(group.Subjects) == 0 || len(group.Quarters) == 0 {
		fyne.Do(func() {
			dt.statusLabel.SetText("Нет предметов или четвертей")
		})
		return
	}

	subject := group.Subjects[0]
	// Find current quarter
	var quarter *client.Quarter
	for i := range group.Quarters {
		if group.Quarters[i].CurrentQuarter {
			quarter = &group.Quarters[i]
			break
		}
	}
	if quarter == nil {
		q := group.Quarters[0]
		quarter = &q
	}

	student := dt.students[studentIdx]
	combo := DiligenceCombos[comboIdx]
	qID := quarter.ID

	// Load dates and journal students
	journalStudents, errS := apiClient.GetJournalStudents(group.ID, subject.SubjectID, qID)
	dates, errD := apiClient.GetJournalDates(group.ID, subject.SubjectID, qID)

	if errS != nil || errD != nil {
		fyne.Do(func() {
			dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v / %v", errS, errD))
		})
		return
	}

	// Find the student in journal students by matching student ID
	var journalStudent *client.Student
	for i := range journalStudents {
		if journalStudents[i].StudentID == student.ID {
			journalStudent = &journalStudents[i]
			break
		}
	}

	if journalStudent == nil && len(journalStudents) > 0 {
		// Fallback: use student index
		if studentIdx < len(journalStudents) {
			journalStudent = &journalStudents[studentIdx]
		}
	}

	today := time.Now().Format("2006-01-02")
	successCount := 0

	for _, date := range dates {
		if date.AssignmentDate > today {
			continue
		}

		// Check if already has a mark/comment
		if journalStudent != nil {
			hasMark := false
			for _, sm := range journalStudent.SubjectMarks {
				if sm.AssignmentDateID == date.AssignmentDateID && sm.ShortName != "" && sm.ShortName != "—" {
					hasMark = true
					break
				}
			}
			if hasMark {
				continue
			}
		}

		// Pick random diligence
		diligence := combo.Diligences[RandomGradeInRange(0, len(combo.Diligences)-1)]

		// Generate behavior comment
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
			dt.statusLabel.SetText(fmt.Sprintf("Ставлю «%s»: %s %s — %s (%d)",
				diligence, student.LastName, student.FirstName, date.AssignmentDate[5:], successCount+1))
		})

		err := apiClient.CreateDiaryComment(student.ID, date.AssignmentDateID, qID, comment)
		if err == nil {
			successCount++
		}
	}

	fyne.Do(func() {
		dt.statusLabel.SetText(fmt.Sprintf("Готово (legacy): %d записей для %s %s",
			successCount, student.LastName, student.FirstName))
	})
}

// Refresh updates the tab with new data.
func (dt *DiariesTab) Refresh() {
	go dt.loadInitialData()
}
