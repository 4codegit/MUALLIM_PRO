package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/client"
)

// Controller interface decouples the UI from the application controller implementation
type Controller interface {
	Login(username, password string) error
	GetClient() *client.EdonishClient
	SelectSchool(schoolID int) error
	GetWindow() fyne.Window
	ShowDashboard()
	SaveSession(login, password string, schoolID int, remember bool)
	GetSavedSession() (string, string, bool)
	Logout()
	ToggleTheme()
}

type LoginScreen struct {
	controller  Controller
	container   *fyne.Container
	loginEntry  *widget.Entry
	passEntry   *widget.Entry
	remember    *widget.Check
	loginBtn    *widget.Button
	schoolSel   *widget.Select
	schoolLabel *canvas.Text
	schoolBox   *fyne.Container
	formBox     *fyne.Container
}

func NewLoginScreen(c Controller) *LoginScreen {
	ls := &LoginScreen{controller: c}
	ls.buildUI()
	return ls
}

func (ls *LoginScreen) buildUI() {
	// App Header Title
	titleText := canvas.NewText("eDonish Auto", nil)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 28

	subtitleText := canvas.NewText("Автоматизация электронного журнала edonish.tj", nil)
	subtitleText.TextSize = 14

	ls.loginEntry = widget.NewEntry()
	ls.loginEntry.SetPlaceHolder("Введите логин")

	ls.passEntry = widget.NewPasswordEntry()
	ls.passEntry.SetPlaceHolder("Введите пароль")

	ls.remember = widget.NewCheck("Запомнить меня", nil)

	ls.loginBtn = widget.NewButton("Войти", ls.onLoginTapped)
	ls.loginBtn.Importance = widget.HighImportance

	// Setup return submissions
	ls.passEntry.OnSubmitted = func(string) { ls.loginBtn.OnTapped() }
	ls.loginEntry.OnSubmitted = func(string) { ls.passEntry.FocusGained() }

	// Autofill from saved session
	savedLogin, savedPass, rememberChecked := ls.controller.GetSavedSession()
	if savedLogin != "" {
		ls.loginEntry.SetText(savedLogin)
		if rememberChecked {
			ls.passEntry.SetText(savedPass)
			ls.remember.SetChecked(true)
		}
	}

	ls.formBox = container.NewVBox(
		widget.NewForm(
			&widget.FormItem{Text: "Логин", Widget: ls.loginEntry},
			&widget.FormItem{Text: "Пароль", Widget: ls.passEntry},
		),
		ls.remember,
		ls.loginBtn,
	)

	// School selection components (hidden initially)
	ls.schoolLabel = canvas.NewText("Выберите школу/роль:", nil)
	ls.schoolLabel.TextStyle = fyne.TextStyle{Bold: true}
	ls.schoolLabel.TextSize = 16
	ls.schoolLabel.Hide()

	ls.schoolSel = widget.NewSelect([]string{}, ls.onSchoolSelected)
	ls.schoolSel.PlaceHolder = "Выберите школу..."
	ls.schoolSel.Hide()

	ls.schoolBox = container.NewVBox(
		ls.schoolLabel,
		ls.schoolSel,
	)
	ls.schoolBox.Hide()

	ls.container = container.NewPadded(
		container.NewCenter(
			container.NewVBox(
				container.NewHBox(layout.NewSpacer(), titleText, layout.NewSpacer()),
				container.NewHBox(layout.NewSpacer(), subtitleText, layout.NewSpacer()),
				widget.NewSeparator(),
				ls.formBox,
				ls.schoolBox,
				layout.NewSpacer(),
			),
		),
	)
}

func (ls *LoginScreen) Container() fyne.CanvasObject {
	return ls.container
}

func (ls *LoginScreen) GetLoginEntry() *widget.Entry {
	return ls.loginEntry
}

func (ls *LoginScreen) onLoginTapped() {
	login := ls.loginEntry.Text
	password := ls.passEntry.Text

	if login == "" || password == "" {
		dialog.ShowError(fmt.Errorf("Введите логин и пароль"), ls.controller.GetWindow())
		return
	}

	ls.loginBtn.Disable()
	ls.loginBtn.SetText("Вход...")

	go func() {
		err := ls.controller.Login(login, password)

		fyne.Do(func() {
			ls.loginBtn.Enable()
			ls.loginBtn.SetText("Войти")

			if err != nil {
				dialog.ShowError(err, ls.controller.GetWindow())
				return
			}

			// Hide form and show school selection
			ls.formBox.Hide()

			apiClient := ls.controller.GetClient()
			schoolNames := make([]string, len(apiClient.Schools))
			for i, sch := range apiClient.Schools {
				schoolNames[i] = fmt.Sprintf("%s (%s)", sch.SchoolName, sch.Name)
			}
			ls.schoolSel.Options = schoolNames
			ls.schoolSel.Refresh()

			ls.schoolLabel.Show()
			ls.schoolSel.Show()
			ls.schoolBox.Show()

			// Auto-select if only 1 option
			if len(apiClient.Schools) == 1 {
				ls.schoolSel.SetSelectedIndex(0)
			}
		})
	}()
}

func (ls *LoginScreen) onSchoolSelected(selected string) {
	if selected == "" {
		return
	}

	idx := -1
	for i, opt := range ls.schoolSel.Options {
		if opt == selected {
			idx = i
			break
		}
	}

	if idx < 0 {
		return
	}

	apiClient := ls.controller.GetClient()
	selectedSchool := apiClient.Schools[idx]

	err := ls.controller.SelectSchool(selectedSchool.SchoolID)
	if err != nil {
		dialog.ShowError(err, ls.controller.GetWindow())
		return
	}

	// Save session details
	ls.controller.SaveSession(
		ls.loginEntry.Text,
		ls.passEntry.Text,
		selectedSchool.SchoolID,
		ls.remember.Checked,
	)

	// Proceed to Dashboard
	ls.controller.ShowDashboard()
}
