// Package config holds all application constants and configuration.
package config

import (
        "os"
        "path/filepath"
)

// Application metadata
const (
        AppName    = "eDonish Auto"
        AppVersion = "0.5.0"
        Signature  = "eDonish Auto by 4code"
)

// API base configuration
const (
        APIBase       = "https://api.edonish.tj"
        APILogin      = APIBase + "/auth/v1/login"
        APIRefresh    = APIBase + "/auth/v1/refresh_token"
        APIHeaderInfo = APIBase + "/auth/v1/header/info"
)

// Role-based API prefixes
var APIPrefixes = map[string]string{
        "teacher":           "/teacher/v1",
        "classroom-teacher": "/teacher/v1",
        "school_admin":      "/school_admin/v1",
        "director":          "/director/v1",
        "headteacher":       "/headteacher/v1",
        "chief_curator":     "/chief_curator/v1",
        "regional_curator":  "/regional_curator/v1",
        "parent":            "/parent/v1",
        "student":           "/student/v1",
}

// Journal API endpoints (relative to role prefix)
const (
        JournalOptions        = "/journal"
        JournalDates          = "/journal/dates"
        JournalStudents       = "/journal/students"
        JournalStudentsFinal  = "/journal/students/final"
        JournalDatesFinal     = "/journal/dates/final"
        JournalMarkCreate     = "/journal/10_point_mark/create"
        JournalMarkDelete     = "/journal/mark/delete"
        JournalQuarterCreate  = "/journal/10_point_quarter_mark/create"
        JournalSemesterCreate = "/journal/10_point_semester/create"
        JournalYearCreate     = "/journal/10_point_year/create"
        PeriodQuarters        = "/period/quaters"
        GroupsList            = "/groups/list"
        TeacherSubject        = "/teacher/subject"
        Subgroups             = "/subgroups"
)

// Language codes
const (
        LangTJ = 1
        LangRU = 2
        LangEN = 3
)

// Grade settings
const (
        MinGrade       = 8
        MaxGrade       = 10
        DefaultWorkers = 4
)

// UI Colors
const (
        ColorPrimary = "#2563EB" // Blue
        ColorSuccess = "#16A34A" // Green
        ColorError   = "#DC2626" // Red
        ColorWarning = "#D97706" // Orange/Amber
        ColorInfo    = "#0891B2" // Cyan
        ColorMuted   = "#6B7280" // Grey
)

// SessionFile returns the path to the session persistence file.
func SessionFile() string {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, ".edonish_session.json")
}

// LogFile returns the path to the log file.
func LogFile() string {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, ".edonish_auto.log")
}
