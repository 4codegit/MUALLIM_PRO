package ui

import (
        "fmt"
        "image/color"
        "strconv"
        "strings"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

type Dashboard struct {
        controller  Controller
        container   *fyne.Container
        statusLabel *widget.Label

        // Navigation state
        homePage     *fyne.Container
        contentStack *fyne.Container // stack that holds the current page (only last visible)
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
        gradesTable       *widget.Table
        scheduleList      *widget.List
        homeworkList      *widget.List
        gradesContainer   *fyne.Container
        scheduleContainer *fyne.Container
        homeworkContainer *fyne.Container

        // New tab objects
        topicsTab      *TopicsTab
        diariesTab     *DiariesTab
        finalGradesTab *FinalGradesTab
}

func NewDashboard(c Controller) *Dashboard {
        d := &Dashboard{
                controller:  c,
                statusLabel: widget.NewLabel("Готово"),
        }
        d.buildUI()
        go d.loadJournalOptions()
        return d
}

func (d *Dashboard) Container() fyne.CanvasObject {
        return d.container
}

func (d *Dashboard) buildUI() {
        header := d.buildHeader()
        filters := d.buildFilters()

        // Build individual tab placeholders
        d.gradesContainer = container.NewStack(widget.NewLabelWithStyle("Выберите фильтры для загрузки оценок", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
        d.scheduleContainer = container.NewStack(widget.NewLabelWithStyle("Выберите фильтры для загрузки расписания", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
        d.homeworkContainer = container.NewStack(widget.NewLabelWithStyle("Выберите фильтры для загрузки ДЗ", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))

        // Initialize new tabs
        d.topicsTab = NewTopicsTab(d.controller)
        d.diariesTab = NewDiariesTab(d.controller)
        d.finalGradesTab = NewFinalGradesTab(d.controller)

        // Build the home page with 4 navigation cards
        d.homePage = d.buildHomePage()
        d.currentPage = d.homePage

        // Content stack holds the current page; swapping Objects[0] navigates
        d.contentStack = container.NewStack(d.homePage)

        // Fixed header + filters at top, status at bottom, content fills remaining space
        topSection := container.NewVBox(header, filters, widget.NewSeparator())

        d.container = container.NewBorder(
                topSection,
                d.statusLabel,
                nil,
                nil,
                d.contentStack,
        )
}

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

        // App title on the left
        appTitle := canvas.NewText("eDonish Auto v4.6", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
        appTitle.TextStyle = fyne.TextStyle{Bold: true}
        appTitle.TextSize = 18

        userLabel := canvas.NewText(userText, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
        userLabel.TextStyle = fyne.TextStyle{Bold: true}
        userLabel.TextSize = 14

        roleLabel := canvas.NewText(fmt.Sprintf("%s — %s", roleText, schoolName), color.NRGBA{R: 200, G: 200, B: 200, A: 255})
        roleLabel.TextSize = 11

        themeBtn := widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), d.controller.ToggleTheme)
        themeBtn.Importance = widget.LowImportance

        logoutBtn := widget.NewButton("Выйти", d.controller.Logout)
        logoutBtn.Importance = widget.DangerImportance

        leftBox := container.NewVBox(appTitle)
        userInfoBox := container.NewVBox(userLabel, roleLabel)
        rightBox := container.NewHBox(userInfoBox, themeBtn, logoutBtn)

        // Background rectangle for the navbar
        bgColor := color.NRGBA{R: 30, G: 58, B: 95, A: 255} // Dark blue navbar
        bg := canvas.NewRectangle(bgColor)
        bg.SetMinSize(fyne.NewSize(0, 52))

        navbar := container.NewBorder(nil, nil, leftBox, rightBox, bg)
        overlay := container.NewStack(bg, container.NewPadded(navbar))

        return overlay
}

func (d *Dashboard) buildFilters() *fyne.Container {
        d.classSel = widget.NewSelect([]string{}, d.onClassSelected)
        d.classSel.PlaceHolder = "Класс..."

        d.subjectSel = widget.NewSelect([]string{}, d.onSubjectSelected)
        d.subjectSel.PlaceHolder = "Предмет..."

        d.quarterSel = widget.NewSelect([]string{}, d.onQuarterSelected)
        d.quarterSel.PlaceHolder = "Четверть..."

        d.refreshBtn = widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
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
// HOME PAGE WITH NAVIGATION CARDS
// ------------------------------------------

// navCard creates a clickable card with icon, title and subtitle.
func navCard(icon, title, subtitle string, accent color.Color, onTap func()) *fyne.Container {
        iconText := canvas.NewText(icon, color.White)
        iconText.TextSize = 32
        iconText.Alignment = fyne.TextAlignCenter

        titleText := canvas.NewText(title, color.White)
        titleText.TextStyle = fyne.TextStyle{Bold: true}
        titleText.TextSize = 16
        titleText.Alignment = fyne.TextAlignCenter

        subText := canvas.NewText(subtitle, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
        subText.TextSize = 11
        subText.Alignment = fyne.TextAlignCenter

        content := container.NewVBox(
                container.NewCenter(iconText),
                container.NewCenter(titleText),
                container.NewCenter(subText),
        )

        bg := canvas.NewRectangle(accent)
        bg.SetMinSize(fyne.NewSize(180, 140))
        bg.CornerRadius = 12

        cardStack := container.NewStack(bg, container.NewPadded(content))

        // Tappable overlay to handle clicks
        tapOverlay := newTapOverlay(onTap)

        return container.NewStack(cardStack, tapOverlay)
}

func (d *Dashboard) buildHomePage() *fyne.Container {
        welcomeText := canvas.NewText("eDonish Auto", color.NRGBA{R: 30, G: 58, B: 95, A: 255})
        welcomeText.TextStyle = fyne.TextStyle{Bold: true}
        welcomeText.TextSize = 22
        welcomeText.Alignment = fyne.TextAlignCenter

        subtitleText := canvas.NewText("Выберите раздел для работы", theme.DisabledColor())
        subtitleText.TextSize = 13
        subtitleText.Alignment = fyne.TextAlignCenter

        headerSection := container.NewVBox(
                container.NewCenter(welcomeText),
                container.NewCenter(subtitleText),
                widget.NewSeparator(),
        )

        // Colors for each section card
        blue := color.NRGBA{R: 37, G: 99, B: 235, A: 255}    // Журнал
        green := color.NRGBA{R: 22, G: 163, B: 74, A: 255}   // Темы и ДЗ
        orange := color.NRGBA{R: 217, G: 119, B: 6, A: 255}  // Дневник
        purple := color.NRGBA{R: 124, G: 58, B: 237, A: 255} // Итоговые

        cardJournal := navCard("📋", "Журнал", "Оценки и посещаемость", blue, func() {
                d.navigateTo(d.buildJournalPage())
        })
        cardTopics := navCard("📝", "Темы и ДЗ", "Темы уроков и задания", green, func() {
                d.navigateTo(d.buildTopicsPage())
        })
        cardDiary := navCard("📓", "Дневник", "Подписи в дневнике", orange, func() {
                d.navigateTo(d.buildDiariesPage())
        })
        cardFinal := navCard("🏆", "Итоговые", "Итоговые оценки", purple, func() {
                d.navigateTo(d.buildFinalGradesPage())
        })

        // 2x2 grid of cards
        row1 := container.NewGridWithColumns(2, cardJournal, cardTopics)
        row2 := container.NewGridWithColumns(2, cardDiary, cardFinal)
        cardsGrid := container.NewVBox(row1, row2)

        return container.NewVBox(
                headerSection,
                container.NewCenter(cardsGrid),
        )
}

// navigateTo pushes a new page onto the navigation stack and displays it.
func (d *Dashboard) navigateTo(page fyne.CanvasObject) {
        d.navStack = append(d.navStack, d.currentPage)
        d.currentPage = page
        d.contentStack.Objects = []fyne.CanvasObject{page}
        d.contentStack.Refresh()
}

// navigateBack returns to the previous page on the navigation stack.
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

// makeSubPage wraps content with a back button header.
func (d *Dashboard) makeSubPage(title string, content fyne.CanvasObject) *fyne.Container {
        backBtn := widget.NewButtonWithIcon("← Назад", theme.NavigateBackIcon(), func() {
                d.navigateBack()
        })
        backBtn.Importance = widget.LowImportance

        titleLabel := canvas.NewText(title, color.NRGBA{R: 30, G: 58, B: 95, A: 255})
        titleLabel.TextStyle = fyne.TextStyle{Bold: true}
        titleLabel.TextSize = 16

        pageHeader := container.NewHBox(backBtn, titleLabel)
        return container.NewBorder(pageHeader, nil, nil, nil, content)
}

// ------------------------------------------
// SUB-PAGE BUILDERS
// ------------------------------------------

func (d *Dashboard) buildJournalPage() *fyne.Container {
        // Journal has its own sub-tabs: Оценки, Расписание, ДЗ
        journalTabs := container.NewAppTabs(
                container.NewTabItem("Оценки", d.gradesContainer),
                container.NewTabItem("Расписание", d.scheduleContainer),
                container.NewTabItem("ДЗ", d.homeworkContainer),
        )
        return d.makeSubPage("📋 Журнал", journalTabs)
}

func (d *Dashboard) buildTopicsPage() *fyne.Container {
        return d.makeSubPage("📝 Темы и ДЗ", d.topicsTab.Container())
}

func (d *Dashboard) buildDiariesPage() *fyne.Container {
        return d.makeSubPage("📓 Дневник", d.diariesTab.Container())
}

func (d *Dashboard) buildFinalGradesPage() *fyne.Container {
        return d.makeSubPage("🏆 Итоговые оценки", d.finalGradesTab.Container())
}

func (d *Dashboard) loadJournalOptions() {
        d.statusLabel.SetText("Загрузка классов и предметов...")
        opts, err := d.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        d.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки настроек журнала: %v", err))
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
                d.statusLabel.SetText("Выберите класс")
                if len(classNames) > 0 {
                        d.classSel.SetSelectedIndex(0)
                }
        })
}

func (d *Dashboard) onClassSelected(selected string) {
        if d.journalOpts == nil {
                return
        }

        var group *client.JournalGroup
        for i, g := range d.journalOpts.Groups {
                gName := fmt.Sprintf("%d %s", g.Number, g.Name)
                if gName == selected {
                        group = &d.journalOpts.Groups[i]
                        break
                }
        }

        if group == nil {
                return
        }

        d.selectedGroup = group
        d.selectedSubject = nil
        d.selectedQuarter = nil

        subjectNames := make([]string, len(group.Subjects))
        for i, s := range group.Subjects {
                subjectNames[i] = s.SubjectName
        }

        quarterNames := make([]string, len(group.Quarters))
        for i, q := range group.Quarters {
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

                // Auto select first subject
                if len(subjectNames) > 0 {
                        d.subjectSel.SetSelectedIndex(0)
                }
                // Auto select current quarter if exists
                for i, q := range group.Quarters {
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

        var subject *client.Subject
        for i, s := range d.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        subject = &d.selectedGroup.Subjects[i]
                        break
                }
        }

        d.selectedSubject = subject
        d.checkFilterCompletion()
}

func (d *Dashboard) onQuarterSelected(selected string) {
        if d.selectedGroup == nil {
                return
        }

        var quarter *client.Quarter
        for i, q := range d.selectedGroup.Quarters {
                if q.Name == selected {
                        quarter = &d.selectedGroup.Quarters[i]
                        break
                }
        }

        d.selectedQuarter = quarter
        d.checkFilterCompletion()
}

func (d *Dashboard) checkFilterCompletion() {
        if d.selectedGroup != nil && d.selectedSubject != nil && d.selectedQuarter != nil {
                d.refreshBtn.Enable()
                go d.loadData()
        } else {
                d.refreshBtn.Disable()
        }
}

func (d *Dashboard) loadData() {
        if d.selectedGroup == nil || d.selectedSubject == nil || d.selectedQuarter == nil {
                return
        }

        fyne.Do(func() {
                d.statusLabel.SetText("Загрузка данных...")
        })

        apiClient := d.controller.GetClient()
        gID := d.selectedGroup.ID
        sID := d.selectedSubject.SubjectID
        qID := d.selectedQuarter.ID

        // Load dates
        dates, errDates := apiClient.GetJournalDates(gID, sID, qID)
        // Load students
        students, errStudents := apiClient.GetJournalStudents(gID, sID, qID)

        fyne.Do(func() {
                if errDates != nil {
                        d.statusLabel.SetText(fmt.Sprintf("Ошибка дат: %v", errDates))
                        return
                }
                if errStudents != nil {
                        d.statusLabel.SetText(fmt.Sprintf("Ошибка учеников: %v", errStudents))
                        return
                }

                d.dates = dates
                d.students = students

                d.rebuildGradesTab()
                d.rebuildScheduleTab()
                d.rebuildHomeworkTab()

                // Refresh new tabs with current filter context
                d.topicsTab.Refresh(d.dates, d.selectedGroup, d.selectedSubject)
                d.diariesTab.Refresh(d.students, d.selectedGroup, d.selectedSubject, d.selectedQuarter)
                d.finalGradesTab.Refresh(d.students, d.selectedGroup, d.selectedSubject, d.selectedQuarter)

                d.statusLabel.SetText(fmt.Sprintf("Загружено: %d учеников, %d дат", len(students), len(dates)))
        })
}

// ------------------------------------------
// GRADES TAB LOGIC
// ------------------------------------------

func (d *Dashboard) rebuildGradesTab() {
        colCount := 1 + len(d.dates) + 1 // Student Name, Dates..., Avg
        rowCount := len(d.students) + 1

        d.gradesTable = widget.NewTable(
                func() (int, int) {
                        return rowCount, colCount
                },
                func() fyne.CanvasObject {
                        lbl := widget.NewLabel("")
                        lbl.TextStyle = fyne.TextStyle{}
                        lbl.Alignment = fyne.TextAlignCenter
                        lbl.Wrapping = fyne.TextWrapOff
                        return container.NewMax(lbl)
                },
                func(id widget.TableCellID, cell fyne.CanvasObject) {
                        cellContainer := cell.(*fyne.Container)
                        lbl := cellContainer.Objects[0].(*widget.Label)
                        lbl.TextStyle = fyne.TextStyle{}
                        lbl.SetText("—")

                        // Table Header Row
                        if id.Row == 0 {
                                lbl.TextStyle = fyne.TextStyle{Bold: true}
                                if id.Col == 0 {
                                        lbl.SetText("Ученик")
                                        lbl.Alignment = fyne.TextAlignLeading
                                } else if id.Col == colCount-1 {
                                        lbl.SetText("Средняя")
                                        lbl.Alignment = fyne.TextAlignCenter
                                } else {
                                        dateIdx := id.Col - 1
                                        if dateIdx >= 0 && dateIdx < len(d.dates) {
                                                fullDate := d.dates[dateIdx].AssignmentDate
                                                if len(fullDate) >= 10 {
                                                        lbl.SetText(fullDate[5:10]) // MM-DD
                                                } else {
                                                        lbl.SetText(fullDate)
                                                }
                                        }
                                        lbl.Alignment = fyne.TextAlignCenter
                                }
                                return
                        }

                        // Student rows
                        studIdx := id.Row - 1
                        if studIdx < 0 || studIdx >= len(d.students) {
                                return
                        }
                        student := d.students[studIdx]

                        if id.Col == 0 {
                                lbl.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
                                lbl.Alignment = fyne.TextAlignLeading
                        } else if id.Col == colCount-1 {
                                // Calculate Average
                                avg := d.calculateAverage(student)
                                if avg > 0 {
                                        lbl.SetText(fmt.Sprintf("%.1f", avg))
                                        lbl.TextStyle = fyne.TextStyle{Bold: true}
                                } else {
                                        lbl.SetText("—")
                                }
                                lbl.Alignment = fyne.TextAlignCenter
                        } else {
                                // Mark cells
                                dateIdx := id.Col - 1
                                if dateIdx >= 0 && dateIdx < len(d.dates) {
                                        dateID := d.dates[dateIdx].AssignmentDateID
                                        mark := d.findMark(student, dateID)
                                        if mark != nil {
                                                lbl.SetText(mark.ShortName)
                                                // Color not supported on widget.Label; using theme default
                                        } else {
                                                lbl.SetText("—")
                                        }
                                }
                                lbl.Alignment = fyne.TextAlignCenter
                        }
                },
        )

        // Column Width Adjustments
        d.gradesTable.SetColumnWidth(0, 220)
        for i := 1; i < colCount-1; i++ {
                d.gradesTable.SetColumnWidth(i, 55)
        }
        d.gradesTable.SetColumnWidth(colCount-1, 75)

        d.gradesTable.OnSelected = func(id widget.TableCellID) {
                d.gradesTable.Unselect(id)
                if id.Row == 0 || id.Col == 0 || id.Col == colCount-1 {
                        return
                }
                d.onGradeCellTapped(id.Row-1, id.Col-1)
        }

        d.gradesContainer.Objects = []fyne.CanvasObject{d.gradesTable}
        d.gradesContainer.Refresh()
}

func (d *Dashboard) calculateAverage(student client.Student) float64 {
        sum := 0.0
        count := 0.0
        for _, m := range student.SubjectMarks {
                if m.ShortName != "" && m.ShortName != "ғ/у" && m.ShortName != "Н/А" {
                        val, err := strconv.Atoi(m.ShortName)
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

func (d *Dashboard) findMark(student client.Student, dateID string) *client.SubjectMark {
        for _, m := range student.SubjectMarks {
                if m.AssignmentDateID == dateID {
                        return &m
                }
        }
        return nil
}

func (d *Dashboard) onGradeCellTapped(studIdx, dateIdx int) {
        if studIdx < 0 || studIdx >= len(d.students) || dateIdx < 0 || dateIdx >= len(d.dates) {
                return
        }

        student := d.students[studIdx]
        date := d.dates[dateIdx]
        currentMark := d.findMark(student, date.AssignmentDateID)

        dialogTitle := fmt.Sprintf("Оценка: %s %s", student.LastName, student.FirstName)
        infoText := fmt.Sprintf("Дата: %s\nТема: %s", date.AssignmentDate, date.Topic)

        // Build grade quick select options
        var dlg dialog.Dialog

        buttons := container.NewGridWithColumns(6)
        // 1-10 grades
        for i := 1; i <= 10; i++ {
                gradeVal := i
                btn := widget.NewButton(strconv.Itoa(gradeVal), func() {
                        go d.setGrade(student.StudentID, date.AssignmentDateID, gradeVal)
                        dlg.Hide()
                })
                buttons.Add(btn)
        }

        // Absent button
        absentBtn := widget.NewButton("Н/А", func() {
                go d.setGrade(student.StudentID, date.AssignmentDateID, 0) // 0 is absent
                dlg.Hide()
        })
        absentBtn.Importance = widget.WarningImportance
        buttons.Add(absentBtn)

        // Delete button (only if mark exists)
        deleteBtn := widget.NewButton("Удалить оценку", func() {
                if currentMark != nil && currentMark.AssignmentMarkID != "" {
                        go d.deleteGrade(currentMark.AssignmentMarkID)
                }
                dlg.Hide()
        })
        deleteBtn.Importance = widget.DangerImportance
        if currentMark == nil || currentMark.AssignmentMarkID == "" {
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

        dlg = dialog.NewCustom(dialogTitle, "Отмена", content, d.controller.GetWindow())
        dlg.Show()
}

func (d *Dashboard) setGrade(studentID int, dateID string, mark int) {
        fyne.Do(func() {
                d.statusLabel.SetText("Установка оценки...")
        })

        err := d.controller.GetClient().CreateMark(studentID, dateID, d.selectedQuarter.ID, mark)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка создания оценки: %v", err), d.controller.GetWindow())
                        d.statusLabel.SetText("Ошибка создания оценки")
                } else {
                        d.statusLabel.SetText("Оценка успешно сохранена")
                        go d.loadData()
                }
        })
}

func (d *Dashboard) deleteGrade(markID string) {
        fyne.Do(func() {
                d.statusLabel.SetText("Удаление оценки...")
        })

        err := d.controller.GetClient().DeleteMark(markID)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка удаления оценки: %v", err), d.controller.GetWindow())
                        d.statusLabel.SetText("Ошибка удаления оценки")
                } else {
                        d.statusLabel.SetText("Оценка удалена")
                        go d.loadData()
                }
        })
}

func getGradeColor(mark int) color.Color {
        switch {
        case mark >= 9:
                return color.NRGBA{R: 22, G: 163, B: 74, A: 255} // Green
        case mark >= 7:
                return color.NRGBA{R: 37, G: 99, B: 235, A: 255} // Blue
        case mark >= 5:
                return color.NRGBA{R: 217, G: 119, B: 6, A: 255} // Orange
        default:
                return color.NRGBA{R: 220, G: 38, B: 38, A: 255} // Red
        }
}

// ------------------------------------------
// SCHEDULE TAB LOGIC
// ------------------------------------------

func (d *Dashboard) rebuildScheduleTab() {
        if len(d.dates) == 0 {
                d.scheduleContainer.Objects = []fyne.CanvasObject{widget.NewLabelWithStyle("Расписание на эту четверть отсутствует", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})}
                d.scheduleContainer.Refresh()
                return
        }

        d.scheduleList = widget.NewList(
                func() int {
                        return len(d.dates)
                },
                func() fyne.CanvasObject {
                        // Title, Day of week, Topic Description
                        dateText := widget.NewLabel("")
                        dateText.TextStyle = fyne.TextStyle{Bold: true}
                        dayText := widget.NewLabel("")

                        topicLabel := widget.NewLabel("Тема урока...")
                        topicLabel.Wrapping = fyne.TextWrapWord

                        left := container.NewVBox(dateText, dayText)
                        rowContent := container.NewBorder(nil, nil, left, nil, topicLabel)

                        return container.NewPadded(rowContent)
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        day := d.dates[id]

                        pad := cell.(*fyne.Container)
                        border := pad.Objects[0].(*fyne.Container)
                        left := border.Objects[0].(*fyne.Container)
                        topicLabel := border.Objects[1].(*widget.Label)

                        dateText := left.Objects[0].(*widget.Label)
                        dayText := left.Objects[1].(*widget.Label)

                        dateText.SetText(day.AssignmentDate)
                        dayText.SetText(day.WeekdayName)

                        if day.Topic != "" {
                                topicLabel.SetText(day.Topic)
                                topicLabel.TextStyle = fyne.TextStyle{}
                        } else {
                                topicLabel.SetText("Нажмите для ввода темы урока...")
                                topicLabel.TextStyle = fyne.TextStyle{Italic: true}
                        }
                        topicLabel.Refresh()
                },
        )

        d.scheduleList.OnSelected = func(id widget.ListItemID) {
                d.scheduleList.Unselect(id)
                d.showEditTopicDialog(id)
        }

        d.scheduleContainer.Objects = []fyne.CanvasObject{d.scheduleList}
        d.scheduleContainer.Refresh()
}

func (d *Dashboard) showEditTopicDialog(idx int) {
        day := d.dates[idx]

        entry := widget.NewMultiLineEntry()
        entry.SetText(day.Topic)
        entry.SetPlaceHolder("Введите тему урока...")

        var dlg dialog.Dialog
        content := container.NewVBox(
                widget.NewLabel(fmt.Sprintf("Редактирование темы урока на %s (%s)", day.AssignmentDate, day.WeekdayName)),
                entry,
        )

        dlg = dialog.NewCustom("Тема урока", "Сохранить", content, d.controller.GetWindow())
        // Use confirm logic or manual save
        dlg.SetOnClosed(func() {
                // Save topic if it was edited
                newTopic := strings.TrimSpace(entry.Text)
                if newTopic != day.Topic {
                        go d.updateAssignmentTopic(day.AssignmentDateID, newTopic, day.HomeWork)
                }
        })
        dlg.Show()
}

func (d *Dashboard) updateAssignmentTopic(dateID, topic, homework string) {
        fyne.Do(func() {
                d.statusLabel.SetText("Сохранение темы...")
        })

        err := d.controller.GetClient().UpdateAssignment(dateID, topic, homework)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка сохранения темы: %v", err), d.controller.GetWindow())
                        d.statusLabel.SetText("Ошибка сохранения темы")
                } else {
                        d.statusLabel.SetText("Тема успешно сохранена")
                        go d.loadData()
                }
        })
}

// ------------------------------------------
// HOMEWORK TAB LOGIC
// ------------------------------------------

func (d *Dashboard) rebuildHomeworkTab() {
        if len(d.dates) == 0 {
                d.homeworkContainer.Objects = []fyne.CanvasObject{widget.NewLabelWithStyle("Задания отсутствуют", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})}
                d.homeworkContainer.Refresh()
                return
        }

        d.homeworkList = widget.NewList(
                func() int {
                        return len(d.dates)
                },
                func() fyne.CanvasObject {
                        dateText := widget.NewLabel("")
                        dateText.TextStyle = fyne.TextStyle{Bold: true}

                        hwLabel := widget.NewLabel("Домашнее задание...")
                        hwLabel.Wrapping = fyne.TextWrapWord

                        rowContent := container.NewBorder(nil, nil, dateText, nil, hwLabel)
                        return container.NewPadded(rowContent)
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        day := d.dates[id]

                        pad := cell.(*fyne.Container)
                        border := pad.Objects[0].(*fyne.Container)
                        dateText := border.Objects[0].(*widget.Label)
                        hwLabel := border.Objects[1].(*widget.Label)

                        dateText.SetText(day.AssignmentDate)
                        if day.HomeWork != "" {
                                hwLabel.SetText(day.HomeWork)
                                hwLabel.TextStyle = fyne.TextStyle{}
                        } else {
                                hwLabel.SetText("Нажмите для ввода домашнего задания...")
                                hwLabel.TextStyle = fyne.TextStyle{Italic: true}
                        }
                        hwLabel.Refresh()
                },
        )

        d.homeworkList.OnSelected = func(id widget.ListItemID) {
                d.homeworkList.Unselect(id)
                d.showEditHomeworkDialog(id)
        }

        d.homeworkContainer.Objects = []fyne.CanvasObject{d.homeworkList}
        d.homeworkContainer.Refresh()
}

func (d *Dashboard) showEditHomeworkDialog(idx int) {
        day := d.dates[idx]

        entry := widget.NewMultiLineEntry()
        entry.SetText(day.HomeWork)
        entry.SetPlaceHolder("Введите домашнее задание...")

        var dlg dialog.Dialog
        content := container.NewVBox(
                widget.NewLabel(fmt.Sprintf("Редактирование ДЗ на %s (Тема: %s)", day.AssignmentDate, day.Topic)),
                entry,
        )

        dlg = dialog.NewCustom("Домашнее задание", "Сохранить", content, d.controller.GetWindow())
        dlg.SetOnClosed(func() {
                newHW := strings.TrimSpace(entry.Text)
                if newHW != day.HomeWork {
                        go d.updateAssignmentHomework(day.AssignmentDateID, day.Topic, newHW)
                }
        })
        dlg.Show()
}

func (d *Dashboard) updateAssignmentHomework(dateID, topic, homework string) {
        fyne.Do(func() {
                d.statusLabel.SetText("Сохранение домашнего задания...")
        })

        err := d.controller.GetClient().UpdateAssignment(dateID, topic, homework)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка сохранения домашнего задания: %v", err), d.controller.GetWindow())
                        d.statusLabel.SetText("Ошибка сохранения домашнего задания")
                } else {
                        d.statusLabel.SetText("Домашнее задание успешно сохранено")
                        go d.loadData()
                }
        })
}
