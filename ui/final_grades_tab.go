package ui

import (
	"fmt"
	"math/rand"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/client"
)

// quarterHeaders defines the fixed column headers for the 7 final mark columns.
var quarterHeaders = []string{
	"Четверть 1",
	"Четверть 2",
	"Полугодие 1",
	"Четверть 3",
	"Четверть 4",
	"Полугодие 2",
	"Годовая",
}

// FinalGradesTab manages the Итоговые оценки tab with full CRUD.
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

// Container returns the root canvas object for this tab.
func (t *FinalGradesTab) Container() fyne.CanvasObject {
	return t.container
}

// buildUI creates the filter row (class, subject) + grades table placeholder.
func (t *FinalGradesTab) buildUI() {
	t.classSel = widget.NewSelect([]string{}, t.onClassSelected)
	t.classSel.PlaceHolder = "Класс..."

	t.subjectSel = widget.NewSelect([]string{}, t.onSubjectSelected)
	t.subjectSel.PlaceHolder = "Предмет..."

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		go t.loadData()
	})

	t.randomBtn = widget.NewButton("Рандомные итоговые оценки", t.showRandomFillDialog)
	t.randomBtn.Importance = widget.WarningImportance
	t.randomBtn.Disable()

	filterRow := container.NewHBox(
		widget.NewLabel("Фильтры:"),
		t.classSel,
		t.subjectSel,
		refreshBtn,
		t.randomBtn,
	)

	t.gradesContainer = container.NewStack(
		widget.NewLabelWithStyle(
			"Выберите класс и предмет для загрузки итоговых оценок",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
		),
	)

	t.container = container.NewBorder(
		container.NewVBox(filterRow, widget.NewSeparator()),
		t.statusLabel,
		nil,
		nil,
		t.gradesContainer,
	)
}

// loadJournalOptions loads classes and subjects from the API.
func (t *FinalGradesTab) loadJournalOptions() {
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
		t.statusLabel.SetText("Выберите класс и предмет")
		if len(classNames) > 0 {
			t.classSel.SetSelectedIndex(0)
		}
	})
}

// onClassSelected handles class filter selection.
func (t *FinalGradesTab) onClassSelected(selected string) {
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

	subjectNames := make([]string, len(group.Subjects))
	for i, s := range group.Subjects {
		subjectNames[i] = s.SubjectName
	}

	fyne.Do(func() {
		t.subjectSel.Options = subjectNames
		t.subjectSel.Refresh()
		t.subjectSel.ClearSelected()
		t.randomBtn.Disable()

		if len(subjectNames) > 0 {
			t.subjectSel.SetSelectedIndex(0)
		}
	})
}

// onSubjectSelected handles subject filter selection.
func (t *FinalGradesTab) onSubjectSelected(selected string) {
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
	if subject != nil {
		t.randomBtn.Enable()
		go t.loadData()
	} else {
		t.randomBtn.Disable()
	}
}

// loadData loads ALL final grades by querying each quarter and merging the data.
func (t *FinalGradesTab) loadData() {
	if t.selectedGroup == nil || t.selectedSubject == nil {
		return
	}

	fyne.Do(func() {
		t.statusLabel.SetText("Загрузка итоговых оценок (все четверти)...")
	})

	apiClient := t.controller.GetClient()

	// Use GetFinalGradesAll which queries each quarter and merges Q1-Q4, H1-H2, Year
	students, err := apiClient.GetFinalGradesAll(
		t.selectedGroup.ID,
		t.selectedSubject.SubjectID,
		t.selectedGroup.Quarters,
	)

	fyne.Do(func() {
		if err != nil {
			t.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки оценок: %v", err))
			dialog.ShowError(fmt.Errorf("Ошибка загрузки итоговых оценок: %v", err), t.controller.GetWindow())
			return
		}

		if len(students) == 0 {
			t.statusLabel.SetText("Нет учеников для выбранного класса/предмета")
			t.gradesContainer.Objects = []fyne.CanvasObject{
				widget.NewLabelWithStyle(
					"Нет данных об итоговых оценках.\nВозможно, для этого предмета ещё не выставлены оценки.",
					fyne.TextAlignCenter,
					fyne.TextStyle{Italic: true},
				),
			}
			t.gradesContainer.Refresh()
			return
		}

		t.students = students
		t.rebuildGradesTable()
		t.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников", len(students)))
	})
}

// Column indices
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

// rebuildGradesTable builds the table with final grades.
// Maps the 7 mark columns correctly:
//   - colQ1 = QuarterMarks[0], colQ2 = QuarterMarks[1]
//   - colH1 = SemesterMarks[0]
//   - colQ3 = QuarterMarks[2], colQ4 = QuarterMarks[3]
//   - colH2 = SemesterMarks[1]
//   - colYear = YearMark
func (t *FinalGradesTab) rebuildGradesTable() {
	rowCount := len(t.students) + 1 // +1 for header

	t.gradesTable = widget.NewTable(
		func() (int, int) {
			return rowCount, totalCols
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.TextStyle = fyne.TextStyle{}
			lbl.Alignment = fyne.TextAlignCenter
			lbl.Wrapping = fyne.TextWrapOff
			return container.NewMax(lbl)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			c := cell.(*fyne.Container)
			lbl := c.Objects[0].(*widget.Label)
			lbl.TextStyle = fyne.TextStyle{}
			lbl.SetText("—")
			lbl.Alignment = fyne.TextAlignCenter

			// Header row
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				switch id.Col {
				case colNumber:
					lbl.SetText("№")
				case colName:
					lbl.SetText("ФИО ученика")
					lbl.Alignment = fyne.TextAlignLeading
				case colAvg:
					lbl.SetText("Ср. балл")
				default:
					qIdx := id.Col - colQ1
					if qIdx >= 0 && qIdx < len(quarterHeaders) {
						lbl.SetText(quarterHeaders[qIdx])
					}
				}
				return
			}

			// Student rows
			studIdx := id.Row - 1
			if studIdx < 0 || studIdx >= len(t.students) {
				return
			}
			student := t.students[studIdx]

			switch id.Col {
			case colNumber:
				lbl.SetText(strconv.Itoa(studIdx + 1))
			case colName:
				lbl.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
				lbl.Alignment = fyne.TextAlignLeading
			case colAvg:
				if student.AverageScore != "" && student.AverageScore != "0.0" {
					lbl.SetText(student.AverageScore)
				} else {
					// Calculate average from quarter marks
					avg := t.calculateAverage(student)
					if avg > 0 {
						lbl.SetText(fmt.Sprintf("%.1f", avg))
						lbl.TextStyle = fyne.TextStyle{Bold: true}
					} else {
						lbl.SetText("—")
					}
				}
			case colQ1:
				lbl.SetText(markOrDash(student.QuarterMarks[0].ShortName))
			case colQ2:
				lbl.SetText(markOrDash(student.QuarterMarks[1].ShortName))
			case colH1:
				lbl.SetText(markOrDash(student.SemesterMarks[0].ShortName))
			case colQ3:
				lbl.SetText(markOrDash(student.QuarterMarks[2].ShortName))
			case colQ4:
				lbl.SetText(markOrDash(student.QuarterMarks[3].ShortName))
			case colH2:
				lbl.SetText(markOrDash(student.SemesterMarks[1].ShortName))
			case colYear:
				if student.YearMark != nil && student.YearMark.ShortName != "" {
					lbl.SetText(student.YearMark.ShortName)
				} else {
					lbl.SetText("—")
				}
			}
		},
	)

	// Column widths
	t.gradesTable.SetColumnWidth(colNumber, 40)
	t.gradesTable.SetColumnWidth(colName, 220)
	t.gradesTable.SetColumnWidth(colAvg, 70)
	for i := colQ1; i < totalCols; i++ {
		t.gradesTable.SetColumnWidth(i, 65)
	}

	t.gradesTable.OnSelected = func(id widget.TableCellID) {
		t.gradesTable.Unselect(id)
		// Skip header, number, name, and average columns
		if id.Row == 0 || id.Col < colQ1 {
			return
		}
		studIdx := id.Row - 1
		markColIdx := id.Col - colQ1 // 0-6 mapping to quarterHeaders
		t.onGradeCellTapped(studIdx, markColIdx)
	}

	t.gradesContainer.Objects = []fyne.CanvasObject{t.gradesTable}
	t.gradesContainer.Refresh()
}

// markOrDash returns the mark text or "—" if empty.
func markOrDash(shortName string) string {
	if shortName != "" && shortName != "—" {
		return shortName
	}
	return "—"
}

// calculateAverage computes the average from quarter marks.
func (t *FinalGradesTab) calculateAverage(student client.FinalGradeStudent) float64 {
	sum := 0.0
	count := 0.0
	for _, qm := range student.QuarterMarks {
		if qm.ShortName != "" && qm.ShortName != "—" {
			val, err := strconv.Atoi(qm.ShortName)
			if err == nil && val > 0 {
				sum += float64(val)
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

// onGradeCellTapped shows a dialog to create/edit a final grade.
// markColIdx: 0=Q1, 1=Q2, 2=H1, 3=Q3, 4=Q4, 5=H2, 6=Year
func (t *FinalGradesTab) onGradeCellTapped(studIdx, markColIdx int) {
	if studIdx < 0 || studIdx >= len(t.students) {
		return
	}
	if markColIdx < 0 || markColIdx >= len(quarterHeaders) {
		return
	}

	student := t.students[studIdx]

	// Determine current mark info based on column type
	var currentShortName string
	var currentMarkID string
	var markType string // "quarter", "semester", "year"

	switch markColIdx {
	case 0: // Q1
		currentShortName = student.QuarterMarks[0].ShortName
		currentMarkID = student.QuarterMarks[0].QuarterMarkID
		markType = "quarter"
	case 1: // Q2
		currentShortName = student.QuarterMarks[1].ShortName
		currentMarkID = student.QuarterMarks[1].QuarterMarkID
		markType = "quarter"
	case 2: // H1
		currentShortName = student.SemesterMarks[0].ShortName
		currentMarkID = student.SemesterMarks[0].SemesterMarkID
		markType = "semester"
	case 3: // Q3
		currentShortName = student.QuarterMarks[2].ShortName
		currentMarkID = student.QuarterMarks[2].QuarterMarkID
		markType = "quarter"
	case 4: // Q4
		currentShortName = student.QuarterMarks[3].ShortName
		currentMarkID = student.QuarterMarks[3].QuarterMarkID
		markType = "quarter"
	case 5: // H2
		currentShortName = student.SemesterMarks[1].ShortName
		currentMarkID = student.SemesterMarks[1].SemesterMarkID
		markType = "semester"
	case 6: // Year
		if student.YearMark != nil {
			currentShortName = student.YearMark.ShortName
			currentMarkID = student.YearMark.YearMarkID
		}
		markType = "year"
	}

	dialogTitle := fmt.Sprintf("Итоговая оценка: %s %s", student.LastName, student.FirstName)
	infoText := fmt.Sprintf("Период: %s\nУченик ID: %d\nТип: %s", quarterHeaders[markColIdx], student.StudentID, markType)
	if currentShortName != "" && currentShortName != "—" {
		infoText += fmt.Sprintf("\nТекущая оценка: %s", currentShortName)
	} else {
		infoText += "\nТекущая оценка: не выставлена"
	}

	var dlg dialog.Dialog

	// Quick select buttons 1-10
	buttons := container.NewGridWithColumns(5)
	for i := 1; i <= 10; i++ {
		gradeVal := i
		btn := widget.NewButton(strconv.Itoa(gradeVal), func() {
			dlg.Hide()
			go t.createFinalMark(student.StudentID, gradeVal, studIdx, markColIdx, markType)
		})
		buttons.Add(btn)
	}

	// Delete button
	deleteBtn := widget.NewButton("Удалить оценку", func() {
		dlg.Hide()
		if currentMarkID != "" {
			go t.deleteFinalGrade(currentMarkID, studIdx, markColIdx, markType)
		}
	})
	deleteBtn.Importance = widget.DangerImportance
	if currentMarkID == "" {
		deleteBtn.Disable()
	}

	content := container.NewVBox(
		widget.NewLabel(infoText),
		widget.NewSeparator(),
		widget.NewLabel("Выберите оценку:"),
		buttons,
		widget.NewSeparator(),
		deleteBtn,
	)

	dlg = dialog.NewCustom(dialogTitle, "Отмена", content, t.controller.GetWindow())
	dlg.Show()
}

// createFinalMark creates a new final grade using the correct API based on mark type.
func (t *FinalGradesTab) createFinalMark(studentID, mark, studIdx, markColIdx int, markType string) {
	if t.selectedGroup == nil || t.selectedSubject == nil {
		return
	}

	fyne.Do(func() {
		t.statusLabel.SetText(fmt.Sprintf("Установка оценки %d для ученика %d...", mark, studentID))
	})

	apiClient := t.controller.GetClient()

	var err error

	switch markType {
	case "quarter":
		// Find the quarter property ID for this mark column
		quarterIdx := -1
		switch markColIdx {
		case 0:
			quarterIdx = 0 // Q1
		case 1:
			quarterIdx = 1 // Q2
		case 3:
			quarterIdx = 2 // Q3
		case 4:
			quarterIdx = 3 // Q4
		}
		if quarterIdx >= 0 && quarterIdx < len(t.selectedGroup.Quarters) {
			quarterID := t.selectedGroup.Quarters[quarterIdx].ID
			err = apiClient.CreateQuarterMark(
				studentID,
				quarterID,
				mark,
				t.selectedSubject.SubjectID,
				t.selectedSubject.CurriculumPropertyID,
			)
		} else {
			err = fmt.Errorf("не найдена четверть для колонки %d", markColIdx)
		}

	case "semester":
		// For semester marks, we need semester_property_id
		// Semesters are derived from quarters: H1 = average of Q1+Q2, H2 = average of Q3+Q4
		// The semester_property_id comes from the API's period data
		// For now, try using the quarter IDs as semester reference
		// H1 (markColIdx=2) uses Q2's quarter property ID as a reference
		// H2 (markColIdx=5) uses Q4's quarter property ID as a reference
		var semesterPropertyID int
		if markColIdx == 2 && len(t.selectedGroup.Quarters) >= 2 {
			// Semester 1 — use Q2's ID as semester property reference
			semesterPropertyID = t.selectedGroup.Quarters[1].ID
		} else if markColIdx == 5 && len(t.selectedGroup.Quarters) >= 4 {
			// Semester 2 — use Q4's ID as semester property reference
			semesterPropertyID = t.selectedGroup.Quarters[3].ID
		}
		if semesterPropertyID > 0 {
			err = apiClient.CreateSemesterMark(studentID, semesterPropertyID, mark)
		} else {
			err = fmt.Errorf("не найден полугодие ID для колонки %d", markColIdx)
		}

	case "year":
		// For year marks, we need year_property_id
		// Use the last quarter's ID as a reference
		if len(t.selectedGroup.Quarters) > 0 {
			yearPropertyID := t.selectedGroup.Quarters[len(t.selectedGroup.Quarters)-1].ID
			err = apiClient.CreateYearMark(studentID, yearPropertyID, mark)
		} else {
			err = fmt.Errorf("нет четвертей для определения годовой оценки")
		}
	}

	fyne.Do(func() {
		if err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка создания оценки: %v", err), t.controller.GetWindow())
			t.statusLabel.SetText("Ошибка создания оценки")
		} else {
			t.statusLabel.SetText("Итоговая оценка сохранена")
			// Reload to get fresh data
			go t.loadData()
		}
	})
}

// deleteFinalGrade calls the API to delete a final grade.
func (t *FinalGradesTab) deleteFinalGrade(markID string, studIdx, markColIdx int, markType string) {
	fyne.Do(func() {
		t.statusLabel.SetText("Удаление итоговой оценки...")
	})

	var err error
	apiClient := t.controller.GetClient()

	switch markType {
	case "quarter":
		err = apiClient.DeleteFinalGrade(markID)
	case "semester":
		// Use semester delete endpoint
		params := map[string]string{}
		_ = params // TODO: implement semester delete
		err = apiClient.DeleteFinalGrade(markID)
	case "year":
		err = apiClient.DeleteFinalGrade(markID)
	default:
		err = apiClient.DeleteFinalGrade(markID)
	}

	fyne.Do(func() {
		if err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка удаления итоговой оценки: %v", err), t.controller.GetWindow())
			t.statusLabel.SetText("Ошибка удаления оценки")
		} else {
			t.statusLabel.SetText("Итоговая оценка удалена")
			go t.loadData()
		}
	})
}

// showRandomFillDialog shows a dialog for batch random fill of final grades.
func (t *FinalGradesTab) showRandomFillDialog() {
	if t.selectedGroup == nil || t.selectedSubject == nil || len(t.students) == 0 {
		dialog.ShowInformation("Внимание", "Сначала выберите класс и предмет", t.controller.GetWindow())
		return
	}

	gradeComboSel := widget.NewSelect(
		[]string{
			"Хорошо и Отлично (7-10)",
			"Хорошо и Плохо (4-8)",
			"Удовлетворительно и Плохо (3-6)",
		},
		nil,
	)
	gradeComboSel.PlaceHolder = "Выберите диапазон оценок..."
	gradeComboSel.SetSelectedIndex(0)

	periodSel := widget.NewSelect(
		[]string{"Полугодие 1 (Q1,Q2,H1)", "Полугодие 2 (Q3,Q4,H2)", "Весь год (все оценки)"},
		nil,
	)
	periodSel.PlaceHolder = "Выберите период заполнения..."
	periodSel.SetSelectedIndex(2)

	var dlg dialog.Dialog

	fillBtn := widget.NewButton("Заполнить", func() {
		dlg.Hide()
		go t.performRandomFill(gradeComboSel.SelectedIndex(), periodSel.SelectedIndex())
	})
	fillBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Отмена", func() {
		dlg.Hide()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Рандомное заполнение итоговых оценок", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Диапазон оценок:"),
		gradeComboSel,
		widget.NewSeparator(),
		widget.NewLabel("Период заполнения:"),
		periodSel,
		widget.NewSeparator(),
		widget.NewLabel("Заполняются только пустые итоговые оценки."),
		container.NewHBox(fillBtn, cancelBtn),
	)

	dlg = dialog.NewCustom("Рандомные итоговые оценки", "", content, t.controller.GetWindow())
	dlg.Show()
}

// performRandomFill fills empty quarter marks with random grades.
func (t *FinalGradesTab) performRandomFill(comboIdx, periodIdx int) {
	fyne.Do(func() {
		t.statusLabel.SetText("Заполнение рандомными оценками...")
	})

	var minGrade, maxGrade int
	switch comboIdx {
	case 0:
		minGrade, maxGrade = 7, 10
	case 1:
		minGrade, maxGrade = 4, 8
	case 2:
		minGrade, maxGrade = 3, 6
	default:
		minGrade, maxGrade = 7, 10
	}

	apiClient := t.controller.GetClient()
	errorCount := 0
	fillCount := 0

	// Determine which quarter indices to fill
	var quarterIndices []int // 0-3 for Q1-Q4
	var semesterIndices []int // 0-1 for H1-H2
	fillYear := false

	switch periodIdx {
	case 0: // Semester 1
		quarterIndices = []int{0, 1}
		semesterIndices = []int{0}
	case 1: // Semester 2
		quarterIndices = []int{2, 3}
		semesterIndices = []int{1}
	case 2: // Full year
		quarterIndices = []int{0, 1, 2, 3}
		semesterIndices = []int{0, 1}
		fillYear = true
	}

	for si := range t.students {
		student := &t.students[si]

		// Fill quarter marks
		for _, qi := range quarterIndices {
			if qi >= len(t.selectedGroup.Quarters) {
				continue
			}
			qm := student.QuarterMarks[qi]
			if qm.ShortName == "" || qm.ShortName == "—" {
				randomGrade := minGrade + rand.Intn(maxGrade-minGrade+1)
				quarterID := t.selectedGroup.Quarters[qi].ID
				err := apiClient.CreateQuarterMark(
					student.StudentID, quarterID, randomGrade,
					t.selectedSubject.SubjectID, t.selectedSubject.CurriculumPropertyID,
				)
				if err != nil {
					errorCount++
				} else {
					fillCount++
				}
			}
		}

		// Fill semester marks
		for _, semi := range semesterIndices {
			sm := student.SemesterMarks[semi]
			if sm.ShortName == "" || sm.ShortName == "—" {
				randomGrade := minGrade + rand.Intn(maxGrade-minGrade+1)
				var semesterPropertyID int
				if semi == 0 && len(t.selectedGroup.Quarters) >= 2 {
					semesterPropertyID = t.selectedGroup.Quarters[1].ID
				} else if semi == 1 && len(t.selectedGroup.Quarters) >= 4 {
					semesterPropertyID = t.selectedGroup.Quarters[3].ID
				}
				if semesterPropertyID > 0 {
					err := apiClient.CreateSemesterMark(student.StudentID, semesterPropertyID, randomGrade)
					if err != nil {
						errorCount++
					} else {
						fillCount++
					}
				}
			}
		}

		// Fill year mark
		if fillYear {
			ym := student.YearMark
			if ym == nil || ym.ShortName == "" || ym.ShortName == "—" {
				randomGrade := minGrade + rand.Intn(maxGrade-minGrade+1)
				if len(t.selectedGroup.Quarters) > 0 {
					yearPropertyID := t.selectedGroup.Quarters[len(t.selectedGroup.Quarters)-1].ID
					err := apiClient.CreateYearMark(student.StudentID, yearPropertyID, randomGrade)
					if err != nil {
						errorCount++
					} else {
						fillCount++
					}
				}
			}
		}
	}

	fyne.Do(func() {
		if errorCount > 0 {
			t.statusLabel.SetText(fmt.Sprintf("Заполнено: %d оценок, ошибок: %d", fillCount, errorCount))
		} else {
			t.statusLabel.SetText(fmt.Sprintf("Заполнено: %d итоговых оценок", fillCount))
		}
		// Reload to get fresh data from API
		go t.loadData()
	})
}

// Refresh updates the tab with new data from the dashboard context.
func (t *FinalGradesTab) Refresh(students []client.Student, group *client.JournalGroup, subject *client.Subject, quarter *client.Quarter) {
	needReload := false
	if group != nil && (t.selectedGroup == nil || t.selectedGroup.ID != group.ID) {
		t.selectedGroup = group
		needReload = true
	}
	if subject != nil && (t.selectedSubject == nil || t.selectedSubject.SubjectID != subject.SubjectID) {
		t.selectedSubject = subject
		needReload = true
	}
	if needReload && t.selectedGroup != nil && t.selectedSubject != nil {
		go t.loadData()
	}
}
