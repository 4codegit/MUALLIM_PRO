package ui

import (
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/client"
)

// ------------------------------------------
// COLOR PALETTE — Modern Dark-Accent Design
// ------------------------------------------

var (
	colorNavBG      = color.NRGBA{R: 15, G: 23, B: 42, A: 255}  // Slate-900
	colorCardBlue   = color.NRGBA{R: 37, G: 99, B: 235, A: 255}  // Blue-600
	colorCardOrange = color.NRGBA{R: 234, G: 88, B: 12, A: 255}  // Orange-600
	colorCardPurple = color.NRGBA{R: 124, G: 58, B: 237, A: 255} // Violet-600
	colorAccent     = color.NRGBA{R: 56, G: 189, B: 248, A: 255} // Sky-400
	colorSurface    = color.NRGBA{R: 248, G: 250, B: 252, A: 255} // Slate-50
)

// ------------------------------------------
// DASHBOARD
// ------------------------------------------

type Dashboard struct {
	controller  Controller
	container   *fyne.Container
	statusLabel *widget.Label

	// Navigation state
	homePage     *fyne.Container
	contentStack *fyne.Container
	currentPage  fyne.CanvasObject
	navStack     []fyne.CanvasObject

	// Filters state
	classSel   *widget.Select
	subjectSel *widget.Select
	quarterSel *widget.Select
	refreshBtn *widget.Button

	selectedGroup   *client.JournalGroup
	selectedSubject *client.Subject
	selectedQuarter *client.Quarter

	journalOpts *client.JournalOptions
	dates       []client.Day
	students    []client.Student

	// Tab content objects
	gradesTable     *widget.Table
	gradesContainer *fyne.Container

	// Tab objects
	diariesTab     *DiariesTab
	finalGradesTab *FinalGradesTab
}

func NewDashboard(c Controller) *Dashboard {
	d := &Dashboard{
		controller:  c,
		statusLabel: widget.NewLabel(""),
	}
	d.buildUI()
	go d.loadJournalOptions()
	return d
}

func (d *Dashboard) Container() fyne.CanvasObject {
	return d.container
}

// ------------------------------------------
// UI BUILD
// ------------------------------------------

func (d *Dashboard) buildUI() {
	header := d.buildHeader()
	filters := d.buildFilters()

	d.gradesContainer = container.NewStack(
		widget.NewLabelWithStyle("Выберите фильтры для загрузки оценок", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)

	d.diariesTab = NewDiariesTab(d.controller)
	d.finalGradesTab = NewFinalGradesTab(d.controller)

	d.homePage = d.buildHomePage()
	d.currentPage = d.homePage
	d.contentStack = container.NewStack(d.homePage)

	topSection := container.NewVBox(header, filters, widget.NewSeparator())

	d.container = container.NewBorder(
		topSection,
		d.statusLabel,
		nil,
		nil,
		d.contentStack,
	)
}

// ------------------------------------------
// HEADER — Sleek minimal navbar
// ------------------------------------------

func (d *Dashboard) buildHeader() *fyne.Container {
	apiClient := d.controller.GetClient()

	userText := ""
	if apiClient.UserInfo != nil {
		userText = fmt.Sprintf("%s %s", apiClient.UserInfo.LastName, apiClient.UserInfo.FirstName)
	}

	roleText := apiClient.Role
	schoolName := ""
	for _, sch := range apiClient.Schools {
		if sch.SchoolID == apiClient.SchoolID {
			schoolName = sch.SchoolName
			break
		}
	}

	appTitle := canvas.NewText("eDonish Auto", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	appTitle.TextStyle = fyne.TextStyle{Bold: true}
	appTitle.TextSize = 18

	versionTag := canvas.NewText("v5.0", colorAccent)
	versionTag.TextSize = 11
	versionTag.TextStyle = fyne.TextStyle{Bold: true}

	userLabel := canvas.NewText(userText, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	userLabel.TextStyle = fyne.TextStyle{Bold: true}
	userLabel.TextSize = 14

	roleLabel := canvas.NewText(fmt.Sprintf("%s — %s", roleText, schoolName), color.NRGBA{R: 160, G: 174, B: 192, A: 255})
	roleLabel.TextSize = 11

	logoutBtn := widget.NewButton("Выйти", d.controller.Logout)
	logoutBtn.Importance = widget.DangerImportance

	leftBox := container.NewHBox(appTitle, versionTag)
	userInfoBox := container.NewVBox(userLabel, roleLabel)
	rightBox := container.NewHBox(userInfoBox, logoutBtn)

	bg := canvas.NewRectangle(colorNavBG)
	bg.SetMinSize(fyne.NewSize(0, 52))

	navbar := container.NewBorder(nil, nil, leftBox, rightBox, bg)
	return container.NewStack(bg, container.NewPadded(navbar))
}

// ------------------------------------------
// FILTERS — Clean row
// ------------------------------------------

func (d *Dashboard) buildFilters() *fyne.Container {
	d.classSel = widget.NewSelect([]string{}, d.onClassSelected)
	d.classSel.PlaceHolder = "Класс..."

	d.subjectSel = widget.NewSelect([]string{}, d.onSubjectSelected)
	d.subjectSel.PlaceHolder = "Предмет..."

	d.quarterSel = widget.NewSelect([]string{}, d.onQuarterSelected)
	d.quarterSel.PlaceHolder = "Четверть..."

	d.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go d.loadData()
	})
	d.refreshBtn.Disable()

	return container.NewHBox(
		widget.NewLabel("Фильтры:"),
		d.classSel,
		d.subjectSel,
		d.quarterSel,
		d.refreshBtn,
	)
}

// ------------------------------------------
// HOME PAGE — 3 modern cards
// ------------------------------------------

func (d *Dashboard) buildHomePage() *fyne.Container {
	welcomeText := canvas.NewText("eDonish Auto", colorNavBG)
	welcomeText.TextStyle = fyne.TextStyle{Bold: true}
	welcomeText.TextSize = 24
	welcomeText.Alignment = fyne.TextAlignCenter

	subtitleText := canvas.NewText("Выберите раздел для работы", color.NRGBA{R: 100, G: 116, B: 139, A: 255})
	subtitleText.TextSize = 13
	subtitleText.Alignment = fyne.TextAlignCenter

	headerSection := container.NewVBox(
		container.NewCenter(welcomeText),
		container.NewCenter(subtitleText),
		widget.NewSeparator(),
	)

	cardJournal := modernCard("\U0001F4CB", "Журнал", "Оценки и посещаемость", colorCardBlue, func() {
		d.navigateTo(d.buildJournalPage())
	})
	cardDiary := modernCard("\U0001F4D3", "Дневник", "Подписи и комментарии", colorCardOrange, func() {
		d.navigateTo(d.buildDiariesPage())
	})
	cardFinal := modernCard("\U0001F3C6", "Итоговые", "Четвертные и годовые", colorCardPurple, func() {
		d.navigateTo(d.buildFinalGradesPage())
	})

	row1 := container.NewGridWithColumns(2, cardJournal, cardDiary)
	row2 := container.NewGridWithColumns(1, cardFinal)

	cardsGrid := container.NewVBox(row1, row2)

	return container.NewVBox(
		headerSection,
		container.NewCenter(cardsGrid),
	)
}

// modernCard creates a sleek clickable card with rounded corners.
func modernCard(icon, title, subtitle string, accent color.Color, onTap func()) *fyne.Container {
	iconText := canvas.NewText(icon, color.White)
	iconText.TextSize = 36
	iconText.Alignment = fyne.TextAlignCenter

	titleText := canvas.NewText(title, color.White)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 17
	titleText.Alignment = fyne.TextAlignCenter

	subText := canvas.NewText(subtitle, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
	subText.TextSize = 12
	subText.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		container.NewCenter(iconText),
		container.NewCenter(titleText),
		container.NewCenter(subText),
	)

	bg := canvas.NewRectangle(accent)
	bg.SetMinSize(fyne.NewSize(220, 150))
	bg.CornerRadius = 16

	cardStack := container.NewStack(bg, container.NewPadded(content))
	tapOverlay := newTapOverlay(onTap)

	return container.NewStack(cardStack, tapOverlay)
}

// ------------------------------------------
// NAVIGATION
// ------------------------------------------

func (d *Dashboard) navigateTo(page fyne.CanvasObject) {
	d.navStack = append(d.navStack, d.currentPage)
	d.currentPage = page
	d.contentStack.Objects = []fyne.CanvasObject{page}
	d.contentStack.Refresh()
}

func (d *Dashboard) navigateBack() {
	if len(d.navStack) == 0 {
		return
	}
	prev := d.navStack[len(d.navStack)-1]
	d.navStack = d.navStack[:len(d.navStack)-1]
	d.currentPage = prev
	d.contentStack.Objects = []fyne.CanvasObject{prev}
	d.contentStack.Refresh()
}

func (d *Dashboard) makeSubPage(title string, content fyne.CanvasObject) *fyne.Container {
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		d.navigateBack()
	})
	backBtn.Importance = widget.LowImportance

	titleLabel := canvas.NewText(title, colorNavBG)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.TextSize = 16

	pageHeader := container.NewHBox(backBtn, titleLabel)
	return container.NewBorder(pageHeader, nil, nil, nil, content)
}

// ------------------------------------------
// SUB-PAGE BUILDERS
// ------------------------------------------

func (d *Dashboard) buildJournalPage() *fyne.Container {
	return d.makeSubPage("Журнал — оценки и посещаемость", d.gradesContainer)
}

func (d *Dashboard) buildDiariesPage() *fyne.Container {
	return d.makeSubPage("Дневник — подписи и комментарии", d.diariesTab.Container())
}

func (d *Dashboard) buildFinalGradesPage() *fyne.Container {
	return d.makeSubPage("Итоговые оценки", d.finalGradesTab.Container())
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

func (d *Dashboard) loadJournalOptions() {
	d.statusLabel.SetText("Загрузка настроек журнала...")
	opts, err := d.controller.GetClient().GetJournalOptions()
	if err != nil {
		fyne.Do(func() {
			d.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v", err))
		})
		return
	}

	d.journalOpts = opts

	classNames := make([]string, len(opts.Groups))
	for i, g := range opts.Groups {
		classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
	}

	fyne.Do(func() {
		d.classSel.Options = classNames
		d.classSel.Refresh()
		d.statusLabel.SetText("")
		if len(classNames) > 0 {
			d.classSel.SetSelectedIndex(0)
		}
	})
}

// ------------------------------------------
// FILTER HANDLERS
// ------------------------------------------

func (d *Dashboard) onClassSelected(selected string) {
	if d.journalOpts == nil {
		return
	}
	for i, g := range d.journalOpts.Groups {
		if fmt.Sprintf("%d %s", g.Number, g.Name) == selected {
			d.selectedGroup = &d.journalOpts.Groups[i]
			break
		}
	}
	if d.selectedGroup == nil {
		return
	}

	d.selectedSubject = nil
	d.selectedQuarter = nil

	subjectNames := make([]string, len(d.selectedGroup.Subjects))
	for i, s := range d.selectedGroup.Subjects {
		subjectNames[i] = s.SubjectName
	}

	quarterNames := make([]string, len(d.selectedGroup.Quarters))
	for i, q := range d.selectedGroup.Quarters {
		quarterNames[i] = q.Name
	}

	fyne.Do(func() {
		d.subjectSel.Options = subjectNames
		d.subjectSel.Refresh()
		d.subjectSel.ClearSelected()
		d.quarterSel.Options = quarterNames
		d.quarterSel.Refresh()
		d.quarterSel.ClearSelected()
		d.refreshBtn.Disable()

		if len(subjectNames) > 0 {
			d.subjectSel.SetSelectedIndex(0)
		}
		for i, q := range d.selectedGroup.Quarters {
			if q.CurrentQuarter {
				d.quarterSel.SetSelectedIndex(i)
				break
			}
		}
	})
}

func (d *Dashboard) onSubjectSelected(selected string) {
	if d.selectedGroup == nil {
		return
	}
	for i, s := range d.selectedGroup.Subjects {
		if s.SubjectName == selected {
			d.selectedSubject = &d.selectedGroup.Subjects[i]
			break
		}
	}
	d.checkFilterCompletion()
}

func (d *Dashboard) onQuarterSelected(selected string) {
	if d.selectedGroup == nil {
		return
	}
	for i, q := range d.selectedGroup.Quarters {
		if q.Name == selected {
			d.selectedQuarter = &d.selectedGroup.Quarters[i]
			break
		}
	}
	d.checkFilterCompletion()
}

func (d *Dashboard) checkFilterCompletion() {
	ready := d.selectedGroup != nil && d.selectedSubject != nil && d.selectedQuarter != nil
	d.refreshBtn.SetEnabled(ready)
	if ready {
		go d.loadData()
	}
}

func (d *Dashboard) loadData() {
	if d.selectedGroup == nil || d.selectedSubject == nil || d.selectedQuarter == nil {
		return
	}

	fyne.Do(func() {
		d.statusLabel.SetText("Загрузка данных журнала...")
	})

	apiClient := d.controller.GetClient()
	gID := d.selectedGroup.ID
	sID := d.selectedSubject.SubjectID
	qID := d.selectedQuarter.ID

	students, errS := apiClient.GetJournalStudents(gID, sID, qID)
	dates, errD := apiClient.GetJournalDates(gID, sID, qID)

	fyne.Do(func() {
		if errS != nil || errD != nil {
			msg := ""
			if errS != nil {
				msg = fmt.Sprintf("Ошибка учеников: %v", errS)
			} else {
				msg = fmt.Sprintf("Ошибка дат: %v", errD)
			}
			d.statusLabel.SetText(msg)
			dialog.ShowError(fmt.Errorf(msg), d.controller.GetWindow())
			return
		}

		d.students = students
		d.dates = dates

		if len(students) == 0 {
			d.statusLabel.SetText("Нет данных")
		} else {
			d.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников, %d дат", len(students), len(dates)))
		}

		d.rebuildGradesTable()
	})
}

// ------------------------------------------
// GRADES TABLE — with double-click random popup
// ------------------------------------------

func (d *Dashboard) rebuildGradesTable() {
	if len(d.students) == 0 || len(d.dates) == 0 {
		d.gradesContainer.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Нет данных для отображения", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		}
		d.gradesContainer.Refresh()
		return
	}

	// Columns: № | ФИО | date1 | date2 | ... | dateN | Ср.балл
	numDateCols := len(d.dates)
	totalCols := 2 + numDateCols + 1 // №, ФИО, dates, avg
	rowCount := len(d.students) + 1   // +1 header

	d.gradesTable = widget.NewTable(
		func() (int, int) { return rowCount, totalCols },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Alignment = fyne.TextAlignCenter
			lbl.Wrapping = fyne.TextWrapOff
			return container.NewMax(lbl)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			c := cell.(*fyne.Container)
			lbl := c.Objects[0].(*widget.Label)
			lbl.SetText("—")
			lbl.Alignment = fyne.TextAlignCenter
			lbl.TextStyle = fyne.TextStyle{}

			// Header row
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				switch id.Col {
				case 0:
					lbl.SetText("№")
				case 1:
					lbl.SetText("ФИО ученика")
					lbl.Alignment = fyne.TextAlignLeading
				case totalCols - 1:
					lbl.SetText("Ср.")
				default:
					dateIdx := id.Col - 2
					if dateIdx >= 0 && dateIdx < len(d.dates) {
						day := d.dates[dateIdx]
						lbl.SetText(day.WeekdayShortName + "\n" + day.AssignmentDate[5:])
					}
				}
				return
			}

			// Data rows
			studentIdx := id.Row - 1
			if studentIdx >= len(d.students) {
				return
			}
			student := d.students[studentIdx]

			switch id.Col {
			case 0:
				lbl.SetText(strconv.Itoa(studentIdx + 1))
			case 1:
				lbl.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
				lbl.Alignment = fyne.TextAlignLeading
			case totalCols - 1:
				if student.AverageScore != "" && student.AverageScore != "0.0" {
					lbl.SetText(student.AverageScore)
				}
			default:
				dateIdx := id.Col - 2
				if dateIdx >= 0 && dateIdx < len(d.dates) {
					dateID := d.dates[dateIdx].AssignmentDateID
					for _, sm := range student.SubjectMarks {
						if sm.AssignmentDateID == dateID {
							lbl.SetText(sm.ShortName)
							break
						}
					}
				}
			}
		},
	)

	// Set column widths
	d.gradesTable.SetColumnWidth(0, 40)  // №
	d.gradesTable.SetColumnWidth(1, 180) // ФИО
	for i := 0; i < numDateCols; i++ {
		d.gradesTable.SetColumnWidth(2+i, 50) // date columns
	}
	d.gradesTable.SetColumnWidth(totalCols-1, 50) // avg

	// Double-click on a cell → random grade popup
	clickCount := 0
	var lastCellID widget.TableCellID
	d.gradesTable.OnSelected = func(id widget.TableCellID) {
		if id == lastCellID {
			clickCount++
		} else {
			clickCount = 1
			lastCellID = id
		}

		d.gradesTable.Unselect(id)

		if clickCount >= 2 && id.Row > 0 && id.Col >= 2 && id.Col < totalCols-1 {
			clickCount = 0
			studentIdx := id.Row - 1
			dateIdx := id.Col - 2
			if studentIdx < len(d.students) && dateIdx < len(d.dates) {
				d.showRandomGradePopup(studentIdx, dateIdx)
			}
		}
	}

	d.gradesContainer.Objects = []fyne.CanvasObject{d.gradesTable}
	d.gradesContainer.Refresh()
}

// ------------------------------------------
// RANDOM GRADE POPUP — appears on double-click
// ------------------------------------------

func (d *Dashboard) showRandomGradePopup(studentIdx, dateIdx int) {
	student := d.students[studentIdx]
	date := d.dates[dateIdx]

	minEntry := widget.NewEntry()
	minEntry.SetPlaceHolder("2")
	minEntry.SetText("7")

	maxEntry := widget.NewEntry()
	maxEntry.SetPlaceHolder("10")
	maxEntry.SetText("10")

	comboSel := widget.NewSelect(comboNames(), func(sel string) {
		for _, c := range GradeCombos {
			if c.Name == sel {
				minEntry.SetText(strconv.Itoa(c.MinVal))
				maxEntry.SetText(strconv.Itoa(c.MaxVal))
				break
			}
		}
	})
	comboSel.PlaceHolder = "Быстрый выбор..."

	header := fmt.Sprintf("%s %s — %s (%s)",
		student.LastName, student.FirstName,
		date.WeekdayShortName, date.AssignmentDate[5:])

	content := container.NewVBox(
		widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Быстрый диапазон:"),
		comboSel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewLabel("Мин. оценка:"),
			minEntry,
			widget.NewLabel("Макс. оценка:"),
			maxEntry,
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Будет выставлена случайная оценка в заданном диапазоне",
			fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)

	dialog.ShowForm("Рандомная оценка", "Поставить", "Отмена", []*widget.FormItem{
		widget.NewFormItem("", content),
	}, func(ok bool) {
		if !ok {
			return
		}
		minVal, err1 := strconv.Atoi(minEntry.Text)
		maxVal, err2 := strconv.Atoi(maxEntry.Text)
		if err1 != nil || err2 != nil {
			dialog.ShowError(fmt.Errorf("Введите корректные числа"), d.controller.GetWindow())
			return
		}
		grade := RandomGradeInRange(minVal, maxVal)

		go func() {
			err := d.controller.GetClient().CreateMark(
				student.StudentID,
				date.AssignmentDateID,
				d.selectedQuarter.ID,
				grade,
			)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(fmt.Errorf("Ошибка: %v", err), d.controller.GetWindow())
				} else {
					d.statusLabel.SetText(fmt.Sprintf("Оценка %d поставлена: %s %s — %s",
						grade, student.LastName, student.FirstName, date.AssignmentDate[5:]))
					go d.loadData()
				}
			})
		}()
	}, d.controller.GetWindow())
}

// comboNames returns list of grade combo names for UI selector.
func comboNames() []string {
	names := make([]string, len(GradeCombos))
	for i, c := range GradeCombos {
		names[i] = c.Name
	}
	return names
}
