package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"github.com/4codegit/edonish-auto/client"
	"github.com/4codegit/edonish-auto/ui"
)

type SessionData struct {
	LoginID  string `json:"login_id"`
	Password string `json:"password"`
	SchoolID int    `json:"school_id"`
	Remember bool   `json:"remember"`
}

type AppController struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	apiClient  *client.EdonishClient
	isDark     bool
	session    *SessionData
}

func NewAppController() *AppController {
	ac := &AppController{
		fyneApp:   app.NewWithID("tj.edonish.auto"),
		apiClient: client.NewEdonishClient(),
		isDark:    true,
	}

	ac.mainWindow = ac.fyneApp.NewWindow("eDonish Auto")
	ac.mainWindow.Resize(fyne.NewSize(1200, 800))
	ac.mainWindow.CenterOnScreen()

	ac.applyTheme()
	ac.loadSession()

	return ac
}

func (ac *AppController) Run() {
	ac.ShowLogin()
	ac.mainWindow.ShowAndRun()
}

func (ac *AppController) ShowLogin() {
	loginScreen := ui.NewLoginScreen(ac)
	ac.mainWindow.SetContent(loginScreen.Container())
	ac.mainWindow.Canvas().Focus(loginScreen.GetLoginEntry())
}

func (ac *AppController) ShowDashboard() {
	dashboard := ui.NewDashboard(ac)
	ac.mainWindow.SetContent(dashboard.Container())
}

// Controller interface implementation

func (ac *AppController) Login(username, password string) error {
	err := ac.apiClient.Login(username, password)
	if err != nil {
		return err
	}
	return ac.apiClient.FetchHeaderInfo()
}

func (ac *AppController) GetClient() *client.EdonishClient {
	return ac.apiClient
}

func (ac *AppController) SelectSchool(schoolID int) error {
	return ac.apiClient.SelectSchool(schoolID)
}

func (ac *AppController) GetWindow() fyne.Window {
	return ac.mainWindow
}

func (ac *AppController) Logout() {
	ac.apiClient = client.NewEdonishClient()
	ac.ShowLogin()
}

func (ac *AppController) ToggleTheme() {
	ac.isDark = !ac.isDark
	ac.applyTheme()
}

func (ac *AppController) applyTheme() {
	if ac.isDark {
		ac.fyneApp.Settings().SetTheme(theme.DefaultTheme())
	} else {
		ac.fyneApp.Settings().SetTheme(theme.DefaultTheme())
	}
}

// Session management

func sessionFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".edonish_session.json")
}

func (ac *AppController) loadSession() {
	path := sessionFilePath()
	if path == "" {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var sd SessionData
	if err := json.Unmarshal(data, &sd); err == nil {
		ac.session = &sd
	}
}

func (ac *AppController) SaveSession(login, password string, schoolID int, remember bool) {
	sd := SessionData{
		LoginID:  login,
		SchoolID: schoolID,
		Remember: remember,
	}

	if remember {
		sd.Password = password
	}

	path := sessionFilePath()
	if path == "" {
		return
	}

	data, err := json.Marshal(sd)
	if err == nil {
		_ = os.WriteFile(path, data, 0600)
	}
}

func (ac *AppController) GetSavedSession() (string, string, bool) {
	if ac.session != nil {
		return ac.session.LoginID, ac.session.Password, ac.session.Remember
	}
	return "", "", false
}
