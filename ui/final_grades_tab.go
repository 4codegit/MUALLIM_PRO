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

// quarterHeaders defines the fixed column headers for the 7 quarter mark columns.
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
// Uses the real edonish.tj API:
//   - GET /journal/students/final with curriculum_property_id
//   - POST /journal/10_point_quarter_mark/create for quarter marks
//   - POST /journal/10_point_semester/create for semester marks
//   - POST /journal/10_point_year/create for year marks
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
	students        []client.Student

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

// loadData loads students with final grades using the correct API.
func (t *FinalGradesTab) loadData() {
	if t.selectedGroup == nil || t.selectedSubject == nil {
		return
	}

	fyne.Do(func() {
		t.statusLabel.SetText("Загрузка итоговых оценок...")
	})

	apiClient := t.controller.GetClient()

	// Use curriculum_property_id from the selected subject for the /final endpoint
	curriculumID := t.selectedSubject.CurriculumPropertyID
	students, err := apiClient.GetFinalGradesStudents(t.selectedGroup.ID, curriculumID)

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

// rebuildGradesTable builds the table with final grades.
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
					lbl.SetText("Средний балл")
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
				if student.AverageScore != "" {
					lbl.SetText(student.AverageScore)
					_, err := strconv.ParseFloat(student.AverageScore, 64)
					if err == nil {
						lbl.TextStyle = fyne.TextStyle{Bold: true}
					}
				} else {
					lbl.SetText("—")
				}
			default:
				// Quarter mark columns
				qIdx := id.Col - colQ1
				if qIdx >= 0 && qIdx < len(student.QuarterMarks) {
					qm := student.QuarterMarks[qIdx]
					if qm.ShortName != "" && qm.ShortName != "—" {
						lbl.SetText(qm.ShortName)
					} else {
						lbl.SetText("—")
					}
				}
			}
		},
	)

	// Column widths
	t.gradesTable.SetColumnWidth(colNumber, 40)
	t.gradesTable.SetColumnWidth(colName, 220)
	t.gradesTable.SetColumnWidth(colAvg, 100)
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
		markIdx := id.Col - colQ1
		t.onGradeCellTapped(studIdx, markIdx)
	}

	t.gradesContainer.Objects = []fyne.CanvasObject{t.gradesTable}
	t.gradesContainer.Refresh()
}

// onGradeCellTapped shows a dialog to create/edit a quarter/final grade.
func (t *FinalGradesTab) onGradeCellTapped(studIdx, markIdx int) {
	if studIdx < 0 || studIdx >= len(t.students) {
		return
	}
	if markIdx < 0 || markIdx >= len(quarterHeaders) {
		return
	}

	student := t.students[studIdx]

	// Get current mark info
	var currentMark *client.QuarterMark
	if markIdx < len(student.QuarterMarks) {
		currentMark = &student.QuarterMarks[markIdx]
	}

	dialogTitle := fmt.Sprintf("Итоговая оценка: %s %s", student.LastName, student.FirstName)
	infoText := fmt.Sprintf("Период: %s\nУченик ID: %d", quarterHeaders[markIdx], student.StudentID)
	if currentMark != nil && currentMark.ShortName != "" && currentMark.ShortName != "—" {
		infoText += fmt.Sprintf("\nТекущая оценка: %s (ID: %s)", currentMark.ShortName, currentMark.QuarterMarkID)
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
			go t.createQuarterMark(student.StudentID, gradeVal, studIdx, markIdx)
		})
		buttons.Add(btn)
	}

	// Delete button (only if there's a mark)
	deleteBtn := widget.NewButton("Удалить оценку", func() {
		dlg.Hide()
		if currentMark != nil && currentMark.QuarterMarkID != "" {
			go t.deleteFinalGrade(currentMark.QuarterMarkID, studIdx, markIdx)
		}
	})
	deleteBtn.Importance = widget.DangerImportance
	if currentMark == nil || currentMark.QuarterMarkID == "" {
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

// createQuarterMark creates a new quarter mark using the real API.
func (t *FinalGradesTab) createQuarterMark(studentID, mark, studIdx, markIdx int) {
	if t.selectedGroup == nil || t.selectedSubject == nil {
		return
	}

	fyne.Do(func() {
		t.statusLabel.SetText(fmt.Sprintf("Установка оценки %d для ученика %d...", mark, studentID))
	})

	apiClient := t.controller.GetClient()

	// Find the quarter property ID for this mark index
	// markIdx 0=Q1, 1=Q2, 2=H1, 3=Q3, 4=Q4, 5=H2, 6=Year
	var quarterID int
	if t.selectedGroup != nil && markIdx < len(t.selectedGroup.Quarters) {
		quarterID = t.selectedGroup.Quarters[markIdx].ID
	}

	// Try using CreateQuarterMark API
	err := apiClient.CreateQuarterMark(
		studentID,
		quarterID,
		mark,
		t.selectedSubject.SubjectID,
		t.selectedSubject.CurriculumPropertyID,
	)

	fyne.Do(func() {
		if err != nil {
			// Fallback: try the old update method if create fails
			dialog.ShowError(fmt.Errorf("Ошибка создания оценки: %v", err), t.controller.GetWindow())
			t.statusLabel.SetText("Ошибка создания оценки")
		} else {
			// Update local state
			if studIdx >= 0 && studIdx < len(t.students) && markIdx >= 0 && markIdx < len(t.students[studIdx].QuarterMarks) {
				t.students[studIdx].QuarterMarks[markIdx].ShortName = strconv.Itoa(mark)
			}
			t.statusLabel.SetText("Итоговая оценка сохранена")
			t.rebuildGradesTable()
		}
	})
}

// deleteFinalGrade calls the API to delete a final grade.
func (t *FinalGradesTab) deleteFinalGrade(markID string, studIdx, markIdx int) {
	fyne.Do(func() {
		t.statusLabel.SetText("Удаление итоговой оценки...")
	})

	err := t.controller.GetClient().DeleteFinalGrade(markID)

	fyne.Do(func() {
		if err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка удаления итоговой оценки: %v", err), t.controller.GetWindow())
			t.statusLabel.SetText("Ошибка удаления оценки")
		} else {
			if studIdx >= 0 && studIdx < len(t.students) && markIdx >= 0 && markIdx < len(t.students[studIdx].QuarterMarks) {
				t.students[studIdx].QuarterMarks[markIdx].ShortName = ""
				t.students[studIdx].QuarterMarks[markIdx].QuarterMarkID = ""
			}
			t.statusLabel.SetText("Итоговая оценка удалена")
			t.rebuildGradesTable()
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

	weightSel := widget.NewSelect(
		[]string{"Полугодие 1", "Полугодие 2", "Весь год"},
		nil,
	)
	weightSel.PlaceHolder = "Выберите период заполнения..."
	weightSel.SetSelectedIndex(2)

	var dlg dialog.Dialog

	fillBtn := widget.NewButton("Заполнить", func() {
		dlg.Hide()
		go t.performRandomFill(gradeComboSel.SelectedIndex(), weightSel.SelectedIndex())
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
		weightSel,
		widget.NewSeparator(),
		widget.NewLabel("Заполняются только пустые итоговые оценки."),
		container.NewHBox(fillBtn, cancelBtn),
	)

	dlg = dialog.NewCustom("Рандомные итоговые оценки", "", content, t.controller.GetWindow())
	dlg.Show()
}

// performRandomFill fills empty quarter marks with random grades.
func (t *FinalGradesTab) performRandomFill(comboIdx, weightIdx int) {
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

	var markIndices []int
	switch weightIdx {
	case 0:
		markIndices = []int{0, 1, 2}
	case 1:
		markIndices = []int{3, 4, 5}
	case 2:
		markIndices = []int{0, 1, 2, 3, 4, 5, 6}
	default:
		markIndices = []int{0, 1, 2, 3, 4, 5, 6}
	}

	apiClient := t.controller.GetClient()
	errorCount := 0
	fillCount := 0

	for si := range t.students {
		student := &t.students[si]
		for _, mi := range markIndices {
			if mi >= len(student.QuarterMarks) {
				continue
			}
			qm := &student.QuarterMarks[mi]
			if qm.ShortName == "" || qm.ShortName == "—" {
				if qm.QuarterMarkID == "" {
					// Try creating via CreateQuarterMark
					var quarterID int
					if t.selectedGroup != nil && mi < len(t.selectedGroup.Quarters) {
						quarterID = t.selectedGroup.Quarters[mi].ID
					}
					randomGrade := minGrade + rand.Intn(maxGrade-minGrade+1)
					err := apiClient.CreateQuarterMark(
						student.StudentID, quarterID, randomGrade,
						t.selectedSubject.SubjectID, t.selectedSubject.CurriculumPropertyID,
					)
					if err != nil {
						errorCount++
					} else {
						qm.ShortName = strconv.Itoa(randomGrade)
						fillCount++
					}
					continue
				}
				// Has mark ID — update it
				randomGrade := minGrade + rand.Intn(maxGrade-minGrade+1)
				err := apiClient.UpdateFinalGrade(qm.QuarterMarkID, randomGrade)
				if err != nil {
					errorCount++
				} else {
					qm.ShortName = strconv.Itoa(randomGrade)
					fillCount++
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
		t.rebuildGradesTable()
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
