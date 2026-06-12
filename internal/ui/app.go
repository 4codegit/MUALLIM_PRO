// Package ui implements the Fyne-based user interface.
package ui

import (
        "encoding/json"
        "fmt"
        "os"
        "sync"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/app"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/internal/api"
        "github.com/4codegit/edonish-auto/internal/config"
        "github.com/4codegit/edonish-auto/internal/engine"
)

// App is the main application state coordinator.
type App struct {
        fyneApp    fyne.App
        mainWindow fyne.Window
        apiClient  *api.Client
        engine     *engine.Engine

        // Root container for goroutine-safe page switching.
        rootContainer *fyne.Container

        // Data
        journalOptions  interface{}
        groupsData      []map[string]interface{}
        quartersData    []map[string]interface{}
        teacherSubjects []map[string]interface{}

        // State
        loggedIn    bool
        currentPlan *engine.GradePlan
        loadingData bool

        // UI Components
        loginPage  *LoginPage
        autoPage   *AutoGradePage
        journalPg  *JournalPage
        logsPage   *LogsPage
        schoolPage *SchoolPage

        // Tab container for navigation
        tabs *container.AppTabs

        // Status bar
        statusLabel *widget.Label
        schoolLabel *widget.Label

        // School selector in header
        schoolSelect *widget.Select

        // Log buffer
        logMutex  sync.Mutex
        logBuffer []string

        // Dark mode state
        isDarkTheme bool
}

// NewApp creates a new application instance.
func NewApp() *App {
        a := &App{
                fyneApp:   app.NewWithID("com.edonish.auto"),
                apiClient: api.NewClient(),
                logBuffer: make([]string, 0),
        }
        a.engine = engine.NewEngine(a.apiClient)
        a.engine.SetCallbacks(a.onProgress, a.onLog)

        // Set up the main window
        a.mainWindow = a.fyneApp.NewWindow(fmt.Sprintf("%s v%s", config.AppName, config.AppVersion))
        a.mainWindow.Resize(fyne.NewSize(1280, 820))
        a.mainWindow.SetMaster()

        // Initialize pages
        a.logsPage = NewLogsPage(a)
        a.autoPage = NewAutoGradePage(a)
        a.journalPg = NewJournalPage(a)
        a.loginPage = NewLoginPage(a)
        a.schoolPage = NewSchoolPage(a)

        // Initialize the root container with the login page
        loginContent := a.loginPage.Build()
        a.rootContainer = container.NewStack(loginContent)
        a.mainWindow.SetContent(a.rootContainer)

        return a
}

// Run starts the application event loop.
func (a *App) Run() {
        go func() {
                fyne.Do(func() {
                        a.mainWindow.Canvas().Focus(a.loginPage.loginEntry)
                        a.loginPage.LoadSession()
                })
        }()
        a.mainWindow.ShowAndRun()
}

// setPage swaps the visible page in the root container.
func (a *App) setPage(obj fyne.CanvasObject) {
        a.rootContainer.Objects = []fyne.CanvasObject{obj}
        a.rootContainer.Refresh()
}

// showLogin displays the login screen.
func (a *App) showLogin() {
        a.setPage(a.loginPage.Build())
        fyne.Do(func() {
                a.mainWindow.Canvas().Focus(a.loginPage.loginEntry)
                a.loginPage.LoadSession()
        })
}

// onLoginSuccess handles the post-login flow.
func (a *App) onLoginSuccess(userInfo *api.UserInfo) {
        if a.apiClient.HasMultipleSchools() {
                a.showSchoolSelection()
        } else {
                a.showDashboard(userInfo)
        }
}

// showSchoolSelection displays the school selection screen.
func (a *App) showSchoolSelection() {
        schools := a.apiClient.GetSchools()
        content := a.schoolPage.SetSchools(schools)
        a.setPage(content)
        a.LogMessage(fmt.Sprintf("Найдено %d школ — выберите школу", len(schools)), "info")
}

// showDashboard displays the main dashboard after login.
func (a *App) showDashboard(userInfo *api.UserInfo) {
        a.loggedIn = true

        // Build tab pages
        a.tabs = container.NewAppTabs(
                container.NewTabItemWithIcon("Авто-оценки", theme.MediaPlayIcon(), a.autoPage.Build()),
                container.NewTabItemWithIcon("Журнал", theme.DocumentIcon(), a.journalPg.Build()),
                container.NewTabItemWithIcon("Логи", theme.ListIcon(), a.logsPage.Build()),
        )

        // App bar with user info and actions
        userName := fmt.Sprintf("%s %s", userInfo.LastName, userInfo.FirstName)
        roleDisplay := a.apiClient.Role
        roleDisplay = translateRole(roleDisplay)

        // ── Modern header with colored title ──────────────────
        appTitle := canvas.NewText(config.AppName, hexColor(config.ColorPrimary))
        appTitle.TextStyle = fyne.TextStyle{Bold: true}
        appTitle.TextSize = 18

        versionText := canvas.NewText(fmt.Sprintf("v%s", config.AppVersion), hexColor(config.ColorMuted))
        versionText.TextSize = 11
        versionText.TextStyle = fyne.TextStyle{Italic: true}

        a.schoolLabel = widget.NewLabel(fmt.Sprintf("Школа: %d", a.apiClient.SchoolID))

        // Colored role label
        roleColor := hexColor(config.ColorInfo) // cyan default
        switch a.apiClient.Role {
        case "school_admin":
                roleColor = hexColor(config.ColorError) // red for admin
        case "director":
                roleColor = hexColor(config.ColorWarning) // orange for director
        case "classroom-teacher":
                roleColor = hexColor(config.ColorSuccess) // green for class teacher
        case "teacher":
                roleColor = hexColor(config.ColorPrimary) // blue for teacher
        }
        roleText := canvas.NewText(roleDisplay, roleColor)
        roleText.TextStyle = fyne.TextStyle{Bold: true}

        // Status bar with colored text
        a.statusLabel = widget.NewLabelWithStyle("Готов", fyne.TextAlignLeading, fyne.TextStyle{Bold: false})

        // Theme toggle
        themeBtn := widget.NewButtonWithIcon("Тема", theme.ColorPaletteIcon(), func() {
                a.isDarkTheme = !a.isDarkTheme
                if a.isDarkTheme {
                        a.fyneApp.Settings().SetTheme(theme.DarkTheme())
                } else {
                        a.fyneApp.Settings().SetTheme(theme.DefaultTheme())
                }
        })

        // Logout button
        logoutBtn := widget.NewButtonWithIcon("Выйти", theme.LogoutIcon(), func() {
                a.onLogout()
        })

        // User avatar initial
        initial := "?"
        if len(userInfo.FirstName) > 0 {
                initial = string([]rune(userInfo.FirstName)[0])
        } else if len(userInfo.LastName) > 0 {
                initial = string([]rune(userInfo.LastName)[0])
        }
        avatarText := canvas.NewText(initial, hexColor(config.ColorPrimary))
        avatarText.TextStyle = fyne.TextStyle{Bold: true}
        avatarText.Alignment = fyne.TextAlignCenter
        avatarText.TextSize = 20

        // Header layout
        headerLeft := container.NewHBox(
                widget.NewIcon(theme.ComputerIcon()),
                appTitle,
                versionText,
                widget.NewSeparator(),
                container.NewCenter(avatarText),
                widget.NewLabel(userName),
                roleText,
                a.schoolLabel,
        )

        headerRight := container.NewHBox(
                a.statusLabel,
                widget.NewSeparator(),
                themeBtn,
                logoutBtn,
        )

        // School selector (only if multiple schools)
        if a.apiClient.HasMultipleSchools() {
                schoolOpts := make([]string, len(a.apiClient.GetSchools()))
                for i, s := range a.apiClient.GetSchools() {
                        schoolOpts[i] = s.Name
                }
                a.schoolSelect = widget.NewSelect(schoolOpts, func(selected string) {
                        a.onSchoolChange(selected)
                })
                for i, s := range a.apiClient.GetSchools() {
                        if s.ID == a.apiClient.SchoolID {
                                a.schoolSelect.SetSelectedIndex(i)
                                break
                        }
                }
                headerLeft.Add(widget.NewSeparator())
                headerLeft.Add(widget.NewLabel("Школа:"))
                headerLeft.Add(a.schoolSelect)
        }

        header := container.NewBorder(nil, nil, headerLeft, headerRight)
        header.Resize(fyne.NewSize(1280, 48))

        content := container.NewBorder(header, nil, nil, nil, a.tabs)
        a.setPage(content)

        // Load initial data
        a.LogMessage("Загрузка данных журнала...", "info")
        go a.loadInitialData()
}

// translateRole converts role API names to Russian display names.
func translateRole(role string) string {
        switch role {
        case "classroom-teacher":
                return "Кл. руководитель"
        case "teacher":
                return "Учитель"
        case "school_admin":
                return "Админ"
        case "director":
                return "Директор"
        case "headteacher":
                return "Завуч"
        default:
                return role
        }
}

// onSchoolChange handles the school selector change.
func (a *App) onSchoolChange(selected string) {
        schools := a.apiClient.GetSchools()
        for _, s := range schools {
                if s.Name == selected {
                        if s.ID == a.apiClient.SchoolID {
                                return
                        }
                        a.apiClient.SetSchool(s.ID)
                        a.LogMessage(fmt.Sprintf("Переключение на школу: %s (ID: %d)", s.Name, s.ID), "info")
                        a.SaveSessionSchool(s.ID)

                        if a.schoolLabel != nil {
                                a.schoolLabel.SetText(fmt.Sprintf("Школа: %d (%s)", s.ID, translateRole(s.Role)))
                        }

                        if a.engine.IsRunning() {
                                a.engine.Stop()
                        }
                        a.currentPlan = nil

                        a.groupsData = nil
                        a.quartersData = nil
                        a.teacherSubjects = nil
                        a.autoPage.UpdateDropdowns()
                        a.journalPg.UpdateDropdowns()

                        go a.loadInitialData()
                        return
                }
        }
}

// loadInitialData loads journal options and populates dropdowns.
// Quarters are extracted from journal_options groups (with correct qpropId),
// falling back to GetQuarters() API only if journal_options has no quarters.
func (a *App) loadInitialData() {
        a.loadingData = true
        defer func() { a.loadingData = false }()

        options, err := a.apiClient.GetJournalOptions()
        if err != nil {
                a.LogMessage(fmt.Sprintf("Ошибка загрузки: %v", err), "error")
                return
        }
        a.journalOptions = options

        a.groupsData = nil
        a.quartersData = nil
        a.teacherSubjects = nil
        subjectsSet := make(map[string]map[string]interface{})

        // Extract quarters from journal_options groups (like Python does),
        // deduplicated by name. This gives correct qpropId values per group.
        quartersByName := make(map[string]map[string]interface{})

        if optionsMap, ok := options.(map[string]interface{}); ok {
                if groups, ok := optionsMap["groups"].([]interface{}); ok {
                        for _, g := range groups {
                                if gm, ok := g.(map[string]interface{}); ok {
                                        groupName := fmt.Sprintf("%s%s", mapStr(gm, "number"), mapStr(gm, "name"))

                                        // Build group-specific quarters from journal_options
                                        var groupQuarters []interface{}
                                        if qList, ok := gm["quarters"].([]interface{}); ok {
                                                for _, q := range qList {
                                                        if qm, ok := q.(map[string]interface{}); ok {
                                                                qname := mapStr(qm, "name")
                                                                if qname != "" {
                                                                        // Add to global deduplicated quarters map
                                                                        if _, exists := quartersByName[qname]; !exists {
                                                                                quartersByName[qname] = map[string]interface{}{
                                                                                        "qpropId":        qm["id"],
                                                                                        "name":           qname,
                                                                                        "startDate":      qm["startDate"],
                                                                                        "endDate":        qm["endDate"],
                                                                                        "currentQuarter": qm["currentQuarter"],
                                                                                }
                                                                        }
                                                                }
                                                                // Build group-specific quarter entry
                                                                groupQuarters = append(groupQuarters, map[string]interface{}{
                                                                        "qpropId":        qm["id"],
                                                                        "name":           qname,
                                                                        "startDate":      qm["startDate"],
                                                                        "endDate":        qm["endDate"],
                                                                        "currentQuarter": qm["currentQuarter"],
                                                                })
                                                        }
                                                }
                                        }

                                        // Build group-specific subjects from journal_options
                                        var groupSubjects []interface{}
                                        if sList, ok := gm["subjects"].([]interface{}); ok {
                                                for _, s := range sList {
                                                        if sm, ok := s.(map[string]interface{}); ok {
                                                                groupSubjects = append(groupSubjects, map[string]interface{}{
                                                                        "subjectId":             sm["subjectId"],
                                                                        "subjectName":           sm["subjectName"],
                                                                        "curriculumPropertyId": sm["curriculumPropertyId"],
                                                                })
                                                        }
                                                }
                                        }

                                        a.groupsData = append(a.groupsData, map[string]interface{}{
                                                "id":       gm["id"],
                                                "name":     groupName,
                                                "number":   gm["number"],
                                                "group":    gm["name"],
                                                "edit":     gm["edit"],
                                                "myClass":  gm["myClass"],
                                                "subjects": groupSubjects,
                                                "quarters": groupQuarters,
                                        })

                                        // Collect unique subjects across all groups
                                        if subjects, ok := gm["subjects"].([]interface{}); ok {
                                                for _, s := range subjects {
                                                        if sm, ok := s.(map[string]interface{}); ok {
                                                                key := fmt.Sprintf("%v", sm["subjectId"])
                                                                if _, exists := subjectsSet[key]; !exists {
                                                                        subjectsSet[key] = map[string]interface{}{
                                                                                "subjectId":   sm["subjectId"],
                                                                                "subjectName": sm["subjectName"],
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        for _, s := range subjectsSet {
                a.teacherSubjects = append(a.teacherSubjects, s)
        }

        // Use quarters extracted from journal_options (correct qpropId values)
        for _, q := range quartersByName {
                a.quartersData = append(a.quartersData, q)
        }

        // Fallback: if no quarters found from journal_options, try GetQuarters() API
        if len(a.quartersData) == 0 {
                a.LogMessage("Четверти не найдены в journal_options, пробуем API...", "warning")
                quarters, err := a.apiClient.GetQuarters()
                if err != nil {
                        a.LogMessage(fmt.Sprintf("Ошибка загрузки четвертей: %v", err), "error")
                } else {
                        for _, q := range quarters {
                                if qm, ok := q.(map[string]interface{}); ok {
                                        a.quartersData = append(a.quartersData, qm)
                                }
                        }
                }
        }

        msg := fmt.Sprintf("Загружено: %d классов, %d предметов, %d четвертей", len(a.groupsData), len(a.teacherSubjects), len(a.quartersData))
        a.LogMessage(msg, "info")

        fyne.Do(func() {
                a.autoPage.UpdateDropdowns()
                a.journalPg.UpdateDropdowns()
                if a.statusLabel != nil {
                        a.statusLabel.SetText(msg)
                }
        })
}

// onLogout handles user logout.
func (a *App) onLogout() {
        if a.engine.IsRunning() {
                a.engine.Stop()
        }
        a.loggedIn = false
        a.currentPlan = nil
        a.apiClient = api.NewClient()
        a.engine = engine.NewEngine(a.apiClient)
        a.engine.SetCallbacks(a.onProgress, a.onLog)
        a.groupsData = nil
        a.quartersData = nil
        a.teacherSubjects = nil
        a.showLogin()
}

// onProgress handles progress updates from the engine.
func (a *App) onProgress(plan *engine.GradePlan) {
        fyne.Do(func() {
                a.autoPage.UpdateProgress(plan)
        })
}

// onLog handles log messages from the engine.
func (a *App) onLog(message, level string) {
        a.LogMessage(message, level)
}

// LogMessage adds a message to the log buffer and updates the logs page.
func (a *App) LogMessage(message, level string) {
        timestamp := time.Now().Format("2006-01-02 15:04:05")
        entry := fmt.Sprintf("%s [%s] %s", timestamp, level, message)

        a.logMutex.Lock()
        a.logBuffer = append(a.logBuffer, entry)
        a.logMutex.Unlock()

        if a.logsPage != nil {
                a.logsPage.AppendLog(entry)
        }
}

// GetLogText returns all log entries as a single string.
func (a *App) GetLogText() string {
        a.logMutex.Lock()
        defer a.logMutex.Unlock()
        result := ""
        for _, line := range a.logBuffer {
                result += line + "\n"
        }
        return result
}

// ClearLogs clears all log entries.
func (a *App) ClearLogs() {
        a.logMutex.Lock()
        a.logBuffer = make([]string, 0)
        a.logMutex.Unlock()
        if a.logsPage != nil {
                a.logsPage.Clear()
        }
}

// SaveSession saves login credentials to disk.
func (a *App) SaveSession(loginID, password string, remember bool) {
        data := map[string]interface{}{
                "login_id":  loginID,
                "password":  password,
                "remember":  remember,
                "school_id": a.apiClient.SchoolID,
        }
        if !remember {
                data["login_id"] = ""
                data["password"] = ""
        }
        fileData, _ := json.MarshalIndent(data, "", "  ")
        _ = os.WriteFile(config.SessionFile(), fileData, 0600)
}

// SaveSessionSchool saves the selected school ID to the session file.
func (a *App) SaveSessionSchool(schoolID int) {
        data, err := os.ReadFile(config.SessionFile())
        if err != nil {
                session := map[string]interface{}{
                        "school_id": schoolID,
                }
                fileData, _ := json.MarshalIndent(session, "", "  ")
                _ = os.WriteFile(config.SessionFile(), fileData, 0600)
                return
        }

        var session map[string]interface{}
        if err := json.Unmarshal(data, &session); err != nil {
                session = make(map[string]interface{})
        }
        session["school_id"] = schoolID
        fileData, _ := json.MarshalIndent(session, "", "  ")
        _ = os.WriteFile(config.SessionFile(), fileData, 0600)
}

// LoadSessionData loads saved session from disk.
func (a *App) LoadSessionData() (loginID, password string, remember bool, schoolID int) {
        data, err := os.ReadFile(config.SessionFile())
        if err != nil {
                return "", "", false, 0
        }
        var session map[string]interface{}
        if err := json.Unmarshal(data, &session); err != nil {
                return "", "", false, 0
        }
        loginID, _ = session["login_id"].(string)
        password, _ = session["password"].(string)
        remember, _ = session["remember"].(bool)
        if sid, ok := session["school_id"].(float64); ok {
                schoolID = int(sid)
        }
        return
}

// mapStr extracts a string field from a map[string]interface{}.
func mapStr(m map[string]interface{}, key string) string {
        if v, ok := m[key].(string); ok {
                return v
        }
        if v, ok := m[key].(float64); ok {
                return fmt.Sprintf("%.0f", v)
        }
        return ""
}

// mapInt extracts an int field from a map[string]interface{}.
func mapInt(m map[string]interface{}, key string) int {
        if v, ok := m[key].(float64); ok {
                return int(v)
        }
        if v, ok := m[key].(int); ok {
                return v
        }
        return 0
}

// makeHeaderLabel creates a styled section header label.
func makeHeaderLabel(text string) *widget.Label {
        return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// makeInfoCard creates a card with a title and content using consistent styling.
func makeInfoCard(title string, content fyne.CanvasObject) *widget.Card {
        return widget.NewCard(title, "", content)
}
