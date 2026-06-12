package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/4codegit/edonish-auto/internal/config"
)

// LoginPage holds the login screen UI components.
type LoginPage struct {
	app         *App
	loginEntry  *widget.Entry
	passEntry   *widget.Entry
	rememberChk *widget.Check
	statusLabel *widget.Label
	loginBtn    *widget.Button
}

// NewLoginPage creates a new login page.
func NewLoginPage(app *App) *LoginPage {
	return &LoginPage{app: app}
}

// Build creates the login view and returns the root container.
func (p *LoginPage) Build() fyne.CanvasObject {
	p.loginEntry = widget.NewEntry()
	p.loginEntry.SetPlaceHolder("Логин (ID телефона)")

	p.passEntry = widget.NewPasswordEntry()
	p.passEntry.SetPlaceHolder("Пароль")
	p.passEntry.OnSubmitted = func(_ string) { p.doLogin() }

	p.rememberChk = widget.NewCheck("Запомнить меня", nil)

	p.statusLabel = widget.NewLabel("")
	p.statusLabel.Wrapping = fyne.TextWrapWord

	p.loginBtn = widget.NewButton("Войти", func() {
		p.doLogin()
	})
	p.loginBtn.Importance = widget.HighImportance

	// Logo icon
	icon := canvas.NewImageFromResource(theme.ComputerIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(96, 96))

	// Title and subtitle
	title := widget.NewLabelWithStyle("eDonish Auto", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Автоматизация электронного журнала edonish.tj")
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	versionLabel := widget.NewLabel(fmt.Sprintf("v%s", config.AppVersion))
	versionLabel.Alignment = fyne.TextAlignCenter
	versionLabel.TextStyle = fyne.TextStyle{Italic: true}

	shortcut := widget.NewLabel("Enter — быстрый вход")
	shortcut.Alignment = fyne.TextAlignCenter
	shortcut.TextStyle = fyne.TextStyle{Italic: true}

	// Login form card
	formCard := widget.NewCard("Авторизация", "", container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.AccountIcon()),
			widget.NewLabelWithStyle("Логин", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		p.loginEntry,
		container.NewHBox(
			widget.NewIcon(theme.VisibilityIcon()),
			widget.NewLabelWithStyle("Пароль", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		p.passEntry,
		widget.NewSeparator(),
		p.rememberChk,
		p.loginBtn,
		container.NewCenter(shortcut),
		p.statusLabel,
	))

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(
			container.NewVBox(
				container.NewCenter(icon),
				container.NewCenter(title),
				container.NewCenter(subtitle),
				container.NewCenter(versionLabel),
				widget.NewSeparator(),
				formCard,
			),
		),
		layout.NewSpacer(),
	)

	return content
}

// LoadSession loads saved session data into the form.
func (p *LoginPage) LoadSession() {
	loginID, password, remember, _ := p.app.LoadSessionData()
	if loginID != "" {
		p.loginEntry.SetText(loginID)
	}
	if remember && password != "" {
		p.passEntry.SetText(password)
		p.rememberChk.SetChecked(true)
	}
}

// doLogin handles the login button press.
func (p *LoginPage) doLogin() {
	loginID := p.loginEntry.Text
	password := p.passEntry.Text

	if loginID == "" || password == "" {
		p.statusLabel.SetText("Введите логин и пароль")
		return
	}

	p.loginBtn.Disable()
	p.loginBtn.SetText("Вход...")
	p.statusLabel.SetText("Подключение к серверу...")

	// Save session
	p.app.SaveSession(loginID, password, p.rememberChk.Checked)

	go func() {
		userInfo, err := p.app.apiClient.Login(loginID, password)
		if err != nil {
			fyne.Do(func() {
				p.loginBtn.Enable()
				p.loginBtn.SetText("Войти")
				p.statusLabel.SetText(err.Error())
			})
			return
		}

		// Apply saved school selection if available
		_, _, _, savedSchoolID := p.app.LoadSessionData()
		if savedSchoolID > 0 && p.app.apiClient.HasMultipleSchools() {
			p.app.apiClient.SetSchool(savedSchoolID)
		}

		fyne.Do(func() {
			p.app.onLoginSuccess(userInfo)
		})
	}()
}
