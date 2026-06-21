// Package client contains the API client logic for interacting with edonish.tj.
package client

import (
        "bytes"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "net/url"
        "strconv"
        "time"
)

const (
        APIBase       = "https://api.edonish.tj"
        APILogin      = APIBase + "/auth/v1/login"
        APIRefresh    = APIBase + "/auth/v1/refresh_token"
        APIHeaderInfo = APIBase + "/auth/v1/header/info"
        LangRU        = 2
)

var RolePrefixMap = map[string]string{
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

// Structs for API responses & requests

type UserInfo struct {
        UID       string `json:"uid"`
        FirstName string `json:"first_name"`
        LastName  string `json:"last_name"`
}

type LoginResponse struct {
        JWTToken     string `json:"jwt_token"`
        RefreshToken string `json:"refresh_token"`
        UID          string `json:"uid"`
        FirstName    string `json:"first_name"`
        LastName     string `json:"last_name"`
        StatusCode   int    `json:"status_code"`
}

type School struct {
        SchoolID   int    `json:"schoolId"`
        Name       string `json:"name"` // Role name, e.g. "teacher"
        SchoolName string `json:"schoolName"`
}

type JournalOptions struct {
        Groups []JournalGroup `json:"groups"`
}

type JournalGroup struct {
        ID       int       `json:"id"`
        Number   int       `json:"number"`
        Name     string    `json:"name"` // Letter, e.g. "Б"
        Subjects []Subject `json:"subjects"`
        Quarters []Quarter `json:"quarters"`
}

type Subject struct {
        SubjectID            int    `json:"subjectId"`
        SubjectName          string `json:"subjectName"`
        CurriculumPropertyID int    `json:"curriculumPropertyId"`
}

type Quarter struct {
        ID             int    `json:"id"`
        Name           string `json:"name"`
        StartDate      string `json:"startDate"`
        EndDate        string `json:"endDate"`
        CurrentQuarter bool   `json:"currentQuarter"`
}

type QuarterDates struct {
        Name              string           `json:"name"`
        QuarterPropertyID int              `json:"quarterPropertyId"`
        StartDate         string           `json:"startDate"`
        EndDate           string           `json:"endDate"`
        CurrentDate       string           `json:"currentDate"`
        Semester          []SemesterInfo   `json:"semester"`
        EducationYear     []EducationYear  `json:"educationYear"`
        Days              []Day            `json:"days"`
}

// SemesterInfo is the semester metadata embedded in the /journal/dates response.
// Each quarter response includes the semester that the quarter belongs to.
type SemesterInfo struct {
        SemesterName        string `json:"semesterName"`
        SemesterPropertyID  int    `json:"semesterPropertyId"`
}

// EducationYear is the school-year metadata embedded in the /journal/dates response.
type EducationYear struct {
        EducationYearName string `json:"educationYearName"`
        EducationYearID   int    `json:"educationYearId"`
}

type Day struct {
        AssignmentDate   string `json:"assignmentDate"`
        AssignmentDateID string `json:"assignmentDateId"`
        WeekdayName      string `json:"weekdayName"`
        WeekdayShortName string `json:"weekdayShortName"`
        SubjectID        int    `json:"subjectId"`
        SubjectName      string `json:"subjectName"`
        Topic            string `json:"topic"`
        HomeWork         string `json:"homeWork"`
}

type Student struct {
        StudentID     int            `json:"studentId"`
        LastName      string         `json:"lastName"`
        FirstName     string         `json:"firstName"`
        MiddleName    string         `json:"middleName"`
        GroupID       int            `json:"groupId"`
        GroupName     string         `json:"groupName"`
        SubjectMarks  []SubjectMark  `json:"subjectMarks"`
        QuarterMarks  []QuarterMark  `json:"quarterMark"`
        SemesterMarks []SemesterMark `json:"semesterMark"`
        YearMarks     []YearMark     `json:"yearMark"`
        AverageScore  string         `json:"averageScore"`
        Access        []AccessInfo   `json:"access"`
}

// GetYearMark returns the first year mark if available.
func (s *Student) GetYearMark() *YearMark {
        if len(s.YearMarks) > 0 {
                return &s.YearMarks[0]
        }
        return nil
}

type SubjectMark struct {
        AssignmentDateID string `json:"assignmentDateId"`
        AssignmentMarkID string `json:"assignmentMarkId"` // String UUID
        ShortName        string `json:"shortName"`        // e.g. "10", "ғ/у" (absent)
        IsNum            bool   `json:"isNum"`
        Comment          string `json:"comment"`
}

type QuarterMark struct {
        QuarterMarkID string `json:"quarterMarkId"`
        ShortName     string `json:"shortName"`
}

type SemesterMark struct {
        SemesterMarkID string `json:"semesterMarkId"`
        ShortName      string `json:"shortName"`
}

type YearMark struct {
        YearMarkID string `json:"yearMarkId"`
        ShortName  string `json:"shortName"`
}

type AccessInfo struct {
        Edit bool `json:"edit"`
}

type CreateMarkRequest struct {
        MarkTypeID             int    `json:"mark_type_id"`
        GroupSubgroupStudentID int    `json:"group_subgroup_student_id"`
        ScheduleDateID         string `json:"schedule_date_id"`
        QuarterPropertyID      int    `json:"quarter_property_id"`
        Mark                   int    `json:"mark"`
}



// EdonishClient manages HTTP operations and session state.
type EdonishClient struct {
        httpClient   *http.Client
        JWTToken     string
        RefreshToken string
        UserInfo     *UserInfo
        SchoolID     int
        Role         string
        RolePrefix   string
        Schools      []School
}

func NewEdonishClient() *EdonishClient {
        // Custom transport with connection pooling tuned for many concurrent
        // requests to the same host (edonish.tj). The default http.Transport
        // has MaxIdleConnsPerHost=2, which effectively serialises concurrent
        // requests to the same host through only 2 keep-alive sockets — every
        // additional in-flight request either opens a fresh TCP+TLS connection
        // (~300-500ms) or waits for a free slot. Bumping MaxIdleConnsPerHost
        // to 16 lets the worker pool fire 8-16 parallel POSTs without ever
        // re-handshaking TLS.
        tr := &http.Transport{
                MaxIdleConns:        50,
                MaxIdleConnsPerHost: 16,
                IdleConnTimeout:     90 * time.Second,
                TLSHandshakeTimeout: 10 * time.Second,
                ResponseHeaderTimeout: 15 * time.Second,
                ExpectContinueTimeout: 1 * time.Second,
                ForceAttemptHTTP2:   true,
        }
        return &EdonishClient{
                httpClient: &http.Client{
                        Timeout:   30 * time.Second,
                        Transport: tr,
                },
        }
}

// Login authenticates with the API.
func (c *EdonishClient) Login(login, password string) error {
        body := map[string]string{
                "login":    login,
                "password": password,
        }
        jsonBody, err := json.Marshal(body)
        if err != nil {
                return err
        }

        resp, err := c.httpClient.Post(APILogin, "application/json", bytes.NewBuffer(jsonBody))
        if err != nil {
                return err
        }
        defer resp.Body.Close()

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
                return err
        }

        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("ошибка входа (код %d): %s", resp.StatusCode, string(respBody))
        }

        var lr LoginResponse
        if err := json.Unmarshal(respBody, &lr); err != nil {
                return err
        }

        if lr.StatusCode != 0 {
                return fmt.Errorf("ошибка авторизации (код %d)", lr.StatusCode)
        }

        c.JWTToken = lr.JWTToken
        c.RefreshToken = lr.RefreshToken
        c.UserInfo = &UserInfo{
                UID:       lr.UID,
                FirstName: lr.FirstName,
                LastName:  lr.LastName,
        }

        return nil
}

// FetchHeaderInfo gets roles/schools list.
func (c *EdonishClient) FetchHeaderInfo() error {
        u, _ := url.Parse(APIHeaderInfo)
        q := u.Query()
        q.Set("lang", strconv.Itoa(LangRU))
        u.RawQuery = q.Encode()

        req, err := http.NewRequest("GET", u.String(), nil)
        if err != nil {
                return err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return err
        }

        var schools []School
        if err := json.Unmarshal(respBody, &schools); err != nil {
                return err
        }

        c.Schools = schools
        return nil
}

// SelectSchool activates a specific school and role prefix.
func (c *EdonishClient) SelectSchool(schoolID int) error {
        for _, s := range c.Schools {
                if s.SchoolID == schoolID {
                        c.SchoolID = schoolID
                        c.Role = s.Name
                        if prefix, ok := RolePrefixMap[s.Name]; ok {
                                c.RolePrefix = prefix
                        } else {
                                c.RolePrefix = "/teacher/v1"
                        }
                        return nil
                }
        }
        return fmt.Errorf("школа с ID %d не найдена", schoolID)
}

// RefreshJWT gets a new JWT token.
func (c *EdonishClient) RefreshJWT() error {
        if c.RefreshToken == "" {
                return fmt.Errorf("нет refresh-токена")
        }

        req, err := http.NewRequest("GET", APIRefresh, nil)
        if err != nil {
                return err
        }
        req.Header.Set("Authorization", "Bearer "+c.RefreshToken)

        resp, err := c.httpClient.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
                return err
        }

        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("ошибка обновления токена (%d): %s", resp.StatusCode, string(respBody))
        }

        var res struct {
                JWTToken     string `json:"jwt_token"`
                RefreshToken string `json:"refresh_token"`
        }
        if err := json.Unmarshal(respBody, &res); err != nil {
                return err
        }

        c.JWTToken = res.JWTToken
        if res.RefreshToken != "" {
                c.RefreshToken = res.RefreshToken
        }
        return nil
}

// buildURL helper to construct API endpoints.
func (c *EdonishClient) buildURL(endpoint string, params map[string]string) string {
        u, _ := url.Parse(APIBase + c.RolePrefix + endpoint)
        q := u.Query()
        q.Set("school_id", strconv.Itoa(c.SchoolID))
        q.Set("lang", strconv.Itoa(LangRU))
        for k, v := range params {
                q.Set(k, v)
        }
        u.RawQuery = q.Encode()
        return u.String()
}

// doRequest performs request and auto-retries on 401/403 with token refresh.
//
// Body handling: when `body` is non-nil, it's JSON-marshalled and assigned to
// req.Body. We ALSO set req.ContentLength explicitly — without this, the
// http.Client would use Transfer-Encoding: chunked (because http.NewRequest
// was called with nil body, leaving ContentLength=0). Some servers parse
// chunked bodies inconsistently — particularly for non-first fields — which
// manifested as "topic saves but ДЗ doesn't" on /journal/assignment/update.
func (c *EdonishClient) doRequest(req *http.Request, body interface{}) ([]byte, int, error) {
        if req.Header.Get("Authorization") == "" && c.JWTToken != "" {
                req.Header.Set("Authorization", "Bearer "+c.JWTToken)
        }
        if body != nil {
                jsonData, err := json.Marshal(body)
                if err != nil {
                        return nil, 0, err
                }
                req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
                req.ContentLength = int64(len(jsonData))
                req.Header.Set("Content-Type", "application/json")
        }

        resp, err := c.httpClient.Do(req)
        if err != nil {
                return nil, 0, err
        }
        defer resp.Body.Close()

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
                return nil, resp.StatusCode, err
        }

        // Token expired: retry once
        if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
                if refreshErr := c.RefreshJWT(); refreshErr == nil {
                        req.Header.Set("Authorization", "Bearer "+c.JWTToken)
                        if body != nil {
                                jsonData, _ := json.Marshal(body)
                                req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
                                req.ContentLength = int64(len(jsonData))
                        }
                        resp2, err2 := c.httpClient.Do(req)
                        if err2 == nil {
                                defer resp2.Body.Close()
                                respBody2, errRead := io.ReadAll(resp2.Body)
                                return respBody2, resp2.StatusCode, errRead
                        }
                }
        }

        if resp.StatusCode >= 400 {
                return respBody, resp.StatusCode, fmt.Errorf("сервер вернул ошибку (код %d): %s", resp.StatusCode, string(respBody))
        }

        return respBody, resp.StatusCode, nil
}

// GetJournalOptions fetches classes, subjects and quarters.
func (c *EdonishClient) GetJournalOptions() (*JournalOptions, error) {
        u := c.buildURL("/journal", nil)
        req, err := http.NewRequest("OPTIONS", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var opts JournalOptions
        if err := json.Unmarshal(respBody, &opts); err != nil {
                return nil, err
        }
        return &opts, nil
}

// GetJournalDates fetches dates and assignments.
func (c *EdonishClient) GetJournalDates(groupID, subjectID, quarterID int) ([]Day, error) {
        qDates, err := c.GetJournalDatesFull(groupID, subjectID, quarterID)
        if err != nil {
                return nil, err
        }
        if len(qDates) == 0 {
                return []Day{}, nil
        }
        return qDates[0].Days, nil
}

// GetJournalDatesFull fetches dates and assignments WITH the full QuarterDates
// metadata (including Semester and EducationYear info needed for
// CreateSemesterMark / CreateYearMark).
//
// Use this instead of GetJournalDates when you need semesterPropertyId or
// educationYearId — without them, CreateSemesterMark / CreateYearMark will
// fail with FK constraint violations (server returns 409).
func (c *EdonishClient) GetJournalDatesFull(groupID, subjectID, quarterID int) ([]QuarterDates, error) {
        params := map[string]string{
                "group_id":            strconv.Itoa(groupID),
                "subject_id":          strconv.Itoa(subjectID),
                "quarter_property_id": strconv.Itoa(quarterID),
        }
        u := c.buildURL("/journal/dates", params)
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var qDates []QuarterDates
        if err := json.Unmarshal(respBody, &qDates); err != nil {
                return nil, err
        }
        return qDates, nil
}

// GetJournalStudents fetches students and their marks.
func (c *EdonishClient) GetJournalStudents(groupID, subjectID, quarterID int) ([]Student, error) {
        params := map[string]string{
                "group_id":            strconv.Itoa(groupID),
                "subject_id":          strconv.Itoa(subjectID),
                "quarter_property_id": strconv.Itoa(quarterID),
        }
        u := c.buildURL("/journal/students", params)
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var students []Student
        if err := json.Unmarshal(respBody, &students); err != nil {
                return nil, err
        }
        return students, nil
}

// CreateMark creates or updates a mark.
func (c *EdonishClient) CreateMark(studentID int, dateID string, quarterID, mark int) error {
        markTypeID := mark
        if mark == 0 { // absent
                markTypeID = 1
        }

        reqBody := CreateMarkRequest{
                MarkTypeID:             markTypeID,
                GroupSubgroupStudentID: studentID,
                ScheduleDateID:         dateID,
                QuarterPropertyID:      quarterID,
                Mark:                   mark,
        }

        u := c.buildURL("/journal/10_point_mark/create", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// DeleteMark deletes a mark.
func (c *EdonishClient) DeleteMark(markID string) error {
        params := map[string]string{
                "mark_id": markID,
        }
        u := c.buildURL("/journal/mark/delete", params)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, nil)
        return err
}



// --- Diary types ---

// DiaryCommentRequest is the body for creating a diary comment (signature/note).
// This uses the real edonish.tj API: POST /journal/comment
type DiaryCommentRequest struct {
        GroupSubgroupStudentID int    `json:"group_subgroup_student_id"`
        ScheduleDateID         string `json:"schedule_date_id"`
        QuarterPropertyID      int    `json:"quarter_property_id"`
        Comment                string `json:"comment"` // Teacher's note, diligence mark, or signature
}

// QuarterMarkCreateRequest is the body for creating a quarter mark.
// Uses POST /journal/10_point_quarter_mark/create
type QuarterMarkCreateRequest struct {
        GroupSubgroupStudentID int    `json:"group_subgroup_student_id"`
        QuarterPropertyID      int    `json:"quarter_property_id"`
        Mark                   int    `json:"mark"`
        MarkID                 int    `json:"mark_id"` // In 10-point system, mark_id == mark
        SubjectID              int    `json:"subject_id"`
        CurriculumPropertyID   int    `json:"curriculum_property_id"`
}

// SemesterMarkCreateRequest is the body for creating a semester mark.
// Uses POST /journal/10_point_semester/create
// As of 2026, the edonish API requires mark_id, subject_id, and curriculum_property_id
// (same as QuarterMarkCreateRequest). Without them the server returns 422.
type SemesterMarkCreateRequest struct {
        GroupSubgroupStudentID int `json:"group_subgroup_student_id"`
        SemesterPropertyID     int `json:"semester_property_id"`
        Mark                   int `json:"mark"`
        MarkID                 int `json:"mark_id"` // In 10-point system, mark_id == mark
        SubjectID              int `json:"subject_id"`
        CurriculumPropertyID   int `json:"curriculum_property_id"`
}

// YearMarkCreateRequest is the body for creating a year mark.
// Uses POST /journal/10_point_year/create
// As of 2026, the edonish API requires mark_id, subject_id, and curriculum_property_id.
type YearMarkCreateRequest struct {
        GroupSubgroupStudentID int `json:"group_subgroup_student_id"`
        YearPropertyID         int `json:"year_property_id"`
        Mark                   int `json:"mark"`
        MarkID                 int `json:"mark_id"` // In 10-point system, mark_id == mark
        SubjectID              int `json:"subject_id"`
        CurriculumPropertyID   int `json:"curriculum_property_id"`
}

// FinalGrade represents an итоговая оценка (quarter/semester/year).
type FinalGrade struct {
        QuarterMarkID string `json:"quarterMarkId"`
        StudentID     int    `json:"studentId"`
        ShortName     string `json:"shortName"`
        MarkType      string `json:"markType"` // "quarter", "semester", "year"
        QuarterName   string `json:"quarterName"`
}

// UpdateFinalGradeRequest is the body for updating a final grade.
type UpdateFinalGradeRequest struct {
        MarkID  string `json:"mark_id"`
        NewMark int    `json:"new_mark"`
}

// --- Topic & HomeWork methods ---

// UpdateAssignmentRequest is the body for updating topic and/or homework on a date.
//
// Field name history:
//   - v5.4.1: `home_work` (snake_case) — matched `schedule_date_id` / `quarter_property_id`
//   - v5.4.2: `homeWork` (camelCase)   — matched GET /journal/dates response field
//   - v5.4.5: BOTH sent simultaneously — server expects `home_work` (snake_case);
//             `homeWork` is also sent as a fallback. The server picks whichever
//             it recognises and ignores the others.
//
// The user reported "тема сохраняется а дз не сохраняется" in v5.4.1 (Python used
// `homeWork`) AND in v5.4.4 (Go used `homeWork`). So `homeWork` alone is wrong.
// `home_work` is the most likely correct name because every other POST body field
// on this endpoint uses snake_case. We send all three plausible variants to be safe.
type UpdateAssignmentRequest struct {
        ScheduleDateID    string `json:"schedule_date_id"`
        Topic             string `json:"topic"`
        HomeWork          string `json:"homeWork"`  // camelCase (matches GET response)
        HomeWorkSnake     string `json:"home_work"` // snake_case (matches other POST fields)
        HomeWorkLower     string `json:"homework"`  // lowercase (simple form)
        QuarterPropertyID int    `json:"quarter_property_id"`
}

// UpdateAssignment updates the topic and/or homework for a specific date.
// Uses POST /journal/assignment/update
//
// The server treats this as an upsert — existing topic and ДЗ are overwritten.
// All three homework field-name variants are sent in the body so the server
// picks whichever its parser recognises.
func (c *EdonishClient) UpdateAssignment(scheduleDateID, topic, homeWork string, quarterPropertyID int) error {
        reqBody := UpdateAssignmentRequest{
                ScheduleDateID:    scheduleDateID,
                Topic:             topic,
                HomeWork:          homeWork,
                HomeWorkSnake:     homeWork,
                HomeWorkLower:     homeWork,
                QuarterPropertyID: quarterPropertyID,
        }

        u := c.buildURL("/journal/assignment/update", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// --- Diary (Comment) methods ---
// The edonish.tj platform uses /journal/comment for diary notes and signatures.
// Diary data comes from /journal/students — each student has marks and comments per date.

// CreateDiaryComment creates a comment (teacher note, diligence, or signature) for a student on a date.
// This is the real edonish.tj API: POST /journal/comment
func (c *EdonishClient) CreateDiaryComment(studentID int, scheduleDateID string, quarterPropertyID int, comment string) error {
        reqBody := DiaryCommentRequest{
                GroupSubgroupStudentID: studentID,
                ScheduleDateID:         scheduleDateID,
                QuarterPropertyID:      quarterPropertyID,
                Comment:                comment,
        }

        u := c.buildURL("/journal/comment", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// --- Diary (MyClass) methods ---
// These use the /myclass/* endpoints, NOT /journal/*.
// The diary (Дневник) on edonish.tj uses a completely separate API:
//   OPTIONS /myclass                              → groups list
//   GET     /myclass/students?group_id=...        → students list
//   OPTIONS /myclass/student/diary/signature      → behavior (diligence) options
//   GET     /myclass/student/diary/signature      → current signature data
//   POST    /myclass/student/diary/signature      → sign with behavior_id

// MyClassGroup represents a class group from the /myclass API.
type MyClassGroup struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
}

// MyClassStudent represents a student from the /myclass/students API.
type MyClassStudent struct {
        ID        int    `json:"id"`
        LastName  string `json:"lastName"`
        FirstName string `json:"firstName"`
        GroupID   int    `json:"groupId"`
        GroupName string `json:"groupName"`
}

// DiaryBehaviorOption represents a behavior/diligence option for diary signatures.
type DiaryBehaviorOption struct {
        ID    int    `json:"id"`
        Title string `json:"title"`
}

// DiaryDay represents a single day in the diary data.
type DiaryDay struct {
        Date      string `json:"date"`
        DayName   string `json:"dayName"`
        IsSunday  bool   `json:"isSunday"`
        Signed    bool   `json:"signed"`
        Behavior  string `json:"behavior"`
        Comment   string `json:"comment"`
}

// unwrapResponse tries to unwrap a server response that may be either a raw
// JSON array OR a wrapped object whose value at some key is the array.
//
// The edonish.tj /myclass endpoints return objects like:
//   {"data":[...]}            (common REST envelope)
//   {"groups":[...]}          (resource-named envelope)
//   {"classes":[...]}         (alternative naming)
//   {"result":{"groups":[...]}} (nested — handled by recursion)
//
// Hardcoding a key list was too brittle — we now scan ALL top-level keys
// and return the first one whose value is a JSON array. If none of the
// top-level values is an array, we recurse one level into any nested
// object values looking for an array. This catches shapes like
// {"result":{"groups":[...]}}.
//
// Returns the JSON bytes of the array portion (or the original bytes if no
// array is found anywhere — caller's unmarshal will then fail with a
// helpful error including the response body).
func unwrapResponse(respBody []byte) []byte {
        // Quick check: if it starts with '[', it's already an array.
        trimmed := bytes.TrimLeft(respBody, " \t\r\n")
        if len(trimmed) > 0 && trimmed[0] == '[' {
                return respBody
        }

        // Parse as a generic object
        var wrapper map[string]json.RawMessage
        if err := json.Unmarshal(respBody, &wrapper); err != nil {
                return respBody
        }

        // Pass 1: try a few well-known envelope keys first (preserves intent).
        for _, key := range []string{"data", "result", "results", "items", "list", "groups", "classes", "students", "options", "days"} {
                if raw, ok := wrapper[key]; ok {
                        t := bytes.TrimLeft(raw, " \t\r\n")
                        if len(t) > 0 && t[0] == '[' {
                                return raw
                        }
                }
        }

        // Pass 2: scan ALL remaining keys for any value that's a JSON array.
        for _, raw := range wrapper {
                t := bytes.TrimLeft(raw, " \t\r\n")
                if len(t) > 0 && t[0] == '[' {
                        return raw
                }
        }

        // Pass 3: recurse one level into nested objects (e.g. {"result":{"groups":[...]}}).
        for _, raw := range wrapper {
                t := bytes.TrimLeft(raw, " \t\r\n")
                if len(t) > 0 && t[0] == '{' {
                        if inner := unwrapResponse(raw); len(inner) > 0 {
                                it := bytes.TrimLeft(inner, " \t\r\n")
                                if len(it) > 0 && it[0] == '[' {
                                        return inner
                                }
                        }
                }
        }

        return respBody
}

// truncateForError returns up to n bytes of body, safe-printable, for inclusion
// in error messages so the user (and developer) can see what the server
// actually returned when unmarshal fails.
func truncateForError(body []byte, n int) string {
        if len(body) <= n {
                return string(body)
        }
        return string(body[:n]) + "...(truncated)"
}

// GetMyClassGroups fetches the list of class groups for the diary.
// Uses OPTIONS /myclass?lang=...&school_id=...
//
// The server response may be wrapped as {"data":[...]} (object) or be a
// raw array. We handle both via unwrapResponse.
func (c *EdonishClient) GetMyClassGroups() ([]MyClassGroup, error) {
        u := c.buildURL("/myclass", nil)
        req, err := http.NewRequest("OPTIONS", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var groups []MyClassGroup
        if err := json.Unmarshal(unwrapResponse(respBody), &groups); err != nil {
                return nil, fmt.Errorf("не удалось разобрать список классов: %v (ответ сервера: %s)",
                        err, truncateForError(respBody, 300))
        }
        return groups, nil
}

// GetMyClassStudents fetches students for a specific class group.
// Uses GET /myclass/students?group_id=...&school_id=...
//
// Server response may be wrapped as {"data":[...]} or be a raw array.
func (c *EdonishClient) GetMyClassStudents(groupID int) ([]MyClassStudent, error) {
        params := map[string]string{
                "group_id": strconv.Itoa(groupID),
        }
        u := c.buildURL("/myclass/students", params)
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var students []MyClassStudent
        if err := json.Unmarshal(unwrapResponse(respBody), &students); err != nil {
                return nil, fmt.Errorf("не удалось разобрать список учеников: %v (ответ сервера: %s)",
                        err, truncateForError(respBody, 300))
        }
        return students, nil
}

// GetDiaryBehaviorOptions fetches available behavior/diligence options for diary signatures.
// Uses OPTIONS /myclass/student/diary/signature?lang=...&school_id=...
//
// Server response may be wrapped as {"data":[...]} or be a raw array.
func (c *EdonishClient) GetDiaryBehaviorOptions() ([]DiaryBehaviorOption, error) {
        u := c.buildURL("/myclass/student/diary/signature", nil)
        req, err := http.NewRequest("OPTIONS", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var options []DiaryBehaviorOption
        if err := json.Unmarshal(unwrapResponse(respBody), &options); err != nil {
                return nil, fmt.Errorf("не удалось разобрать варианты поведения: %v (ответ сервера: %s)",
                        err, truncateForError(respBody, 300))
        }
        return options, nil
}

// GetDiaryData fetches diary data for a student in a date range.
// Uses GET /myclass/diary?lang=...&start_date=...&end_date=...&group_id=...&school_student_id=...&school_id=...
//
// Server response may be wrapped as {"data":[...]} or be a raw array.
func (c *EdonishClient) GetDiaryData(groupID, studentID int, startDate, endDate string) ([]DiaryDay, error) {
        params := map[string]string{
                "group_id":          strconv.Itoa(groupID),
                "school_student_id": strconv.Itoa(studentID),
                "start_date":        startDate,
                "end_date":          endDate,
        }
        u := c.buildURL("/myclass/diary", params)
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, _, err := c.doRequest(req, nil)
        if err != nil {
                return nil, err
        }

        var days []DiaryDay
        if err := json.Unmarshal(unwrapResponse(respBody), &days); err != nil {
                // The API might return null for empty
                return []DiaryDay{}, nil
        }
        return days, nil
}

// SignDiary signs the diary for a student with a specific behavior in a date range.
// Uses POST /myclass/student/diary/signature?behavior_id=...&start_date=...&end_date=...&student_id=...&school_id=...
// If behaviorID is 0, it signs without setting a behavior.
func (c *EdonishClient) SignDiary(studentID, behaviorID int, startDate, endDate string) error {
        params := map[string]string{
                "student_id": strconv.Itoa(studentID),
                "start_date": startDate,
                "end_date":   endDate,
        }
        if behaviorID > 0 {
                params["behavior_id"] = strconv.Itoa(behaviorID)
        }

        u := c.buildURL("/myclass/student/diary/signature", params)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, nil)
        return err
}

// --- Final Grades methods ---

// FinalGradeStudent holds merged final grade data for one student across all periods.
type FinalGradeStudent struct {
        StudentID     int
        LastName      string
        FirstName     string
        MiddleName    string
        AverageScore  string
        QuarterMarks  [4]QuarterMark  // Q1-Q4
        SemesterMarks [2]SemesterMark // H1-H2
        YearMark      *YearMark       // Year
}

// GetAverageScore implements ui.AvgScorer interface.
func (f *FinalGradeStudent) GetAverageScore() float64 {
        if f.AverageScore == "" || f.AverageScore == "0.0" {
                return 0
        }
        v, err := strconv.ParseFloat(f.AverageScore, 64)
        if err != nil {
                return 0
        }
        return v
}

// Student GetAverageScore implements ui.AvgScorer interface.
func (s *Student) GetAverageScore() float64 {
        if s.AverageScore == "" || s.AverageScore == "0.0" {
                return 0
        }
        v, err := strconv.ParseFloat(s.AverageScore, 64)
        if err != nil {
                return 0
        }
        return v
}

// GetFinalGradesStudents fetches students with final marks using the correct API.
// Uses GET /journal/students/final with curriculum_property_id instead of quarter_property_id.
func (c *EdonishClient) GetFinalGradesStudents(groupID, curriculumPropertyID int) ([]Student, error) {
        params := map[string]string{
                "group_id":               strconv.Itoa(groupID),
                "curriculum_property_id": strconv.Itoa(curriculumPropertyID),
        }
        u := c.buildURL("/journal/students/final", params)
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
                return nil, err
        }

        respBody, statusCode, err := c.doRequest(req, nil)
        if err != nil {
                return nil, fmt.Errorf("ошибка загрузки итоговых оценок (код %d): %v", statusCode, err)
        }

        var students []Student
        if err := json.Unmarshal(respBody, &students); err != nil {
                return nil, fmt.Errorf("ошибка разбора ответа: %v (первые 200 символов: %s)", err, string(respBody[:min(len(respBody), 200)]))
        }
        return students, nil
}

// GetFinalGradesAll loads students with ALL final marks by querying each quarter separately.
// Returns merged FinalGradeStudent structs with Q1-Q4, H1-H2, and Year marks populated.
func (c *EdonishClient) GetFinalGradesAll(groupID, subjectID int, quarters []Quarter) ([]FinalGradeStudent, error) {
        if len(quarters) == 0 {
                return nil, fmt.Errorf("нет четвертей для загрузки итоговых оценок")
        }

        // studentID -> FinalGradeStudent
        merged := make(map[int]*FinalGradeStudent)
        // Track order of first appearance
        var order []int

        for qi, q := range quarters {
                students, err := c.GetJournalStudents(groupID, subjectID, q.ID)
                if err != nil {
                        // If one quarter fails, continue with others
                        continue
                }

                for _, s := range students {
                        fgs, exists := merged[s.StudentID]
                        if !exists {
                                fgs = &FinalGradeStudent{
                                        StudentID:    s.StudentID,
                                        LastName:     s.LastName,
                                        FirstName:    s.FirstName,
                                        MiddleName:   s.MiddleName,
                                        AverageScore: s.AverageScore,
                                }
                                merged[s.StudentID] = fgs
                                order = append(order, s.StudentID)
                        }

                        // Store quarter mark (index 0-3)
                        if qi < 4 && len(s.QuarterMarks) > 0 {
                                fgs.QuarterMarks[qi] = s.QuarterMarks[0]
                        }

                        // Store semester marks
                        // Q1 (qi=0) and Q2 (qi=1) responses may contain semester 1 mark
                        // Q3 (qi=2) and Q4 (qi=3) responses may contain semester 2 mark
                        if len(s.SemesterMarks) > 0 {
                                if qi <= 1 {
                                        fgs.SemesterMarks[0] = s.SemesterMarks[0]
                                } else {
                                        fgs.SemesterMarks[1] = s.SemesterMarks[0]
                                }
                        }

                        // Store year mark (from any quarter that has it)
                        ym := s.GetYearMark()
                        if ym != nil && fgs.YearMark == nil {
                                fgs.YearMark = ym
                        }
                }
        }

        // Build result in original order
        result := make([]FinalGradeStudent, 0, len(merged))
        for _, sid := range order {
                result = append(result, *merged[sid])
        }
        return result, nil
}

// CreateQuarterMark creates or updates a quarter mark for a student.
// Uses POST /journal/10_point_quarter_mark/create
func (c *EdonishClient) CreateQuarterMark(studentID, quarterPropertyID, mark, subjectID, curriculumPropertyID int) error {
        reqBody := QuarterMarkCreateRequest{
                GroupSubgroupStudentID: studentID,
                QuarterPropertyID:      quarterPropertyID,
                Mark:                   mark,
                MarkID:                 mark, // In 10-point system, mark_id == mark
                SubjectID:              subjectID,
                CurriculumPropertyID:   curriculumPropertyID,
        }

        u := c.buildURL("/journal/10_point_quarter_mark/create", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// CreateSemesterMark creates or updates a semester mark for a student.
// Uses POST /journal/10_point_semester/create
// subjectID and curriculumPropertyID are required by the edonish API (since 2026).
func (c *EdonishClient) CreateSemesterMark(studentID, semesterPropertyID, mark, subjectID, curriculumPropertyID int) error {
        reqBody := SemesterMarkCreateRequest{
                GroupSubgroupStudentID: studentID,
                SemesterPropertyID:     semesterPropertyID,
                Mark:                   mark,
                MarkID:                 mark, // 10-point system: mark_id == mark
                SubjectID:              subjectID,
                CurriculumPropertyID:   curriculumPropertyID,
        }

        u := c.buildURL("/journal/10_point_semester/create", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// CreateYearMark creates or updates a year mark for a student.
// Uses POST /journal/10_point_year/create
// subjectID and curriculumPropertyID are required by the edonish API (since 2026).
func (c *EdonishClient) CreateYearMark(studentID, yearPropertyID, mark, subjectID, curriculumPropertyID int) error {
        reqBody := YearMarkCreateRequest{
                GroupSubgroupStudentID: studentID,
                YearPropertyID:         yearPropertyID,
                Mark:                   mark,
                MarkID:                 mark, // 10-point system: mark_id == mark
                SubjectID:              subjectID,
                CurriculumPropertyID:   curriculumPropertyID,
        }

        u := c.buildURL("/journal/10_point_year/create", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}


// UpdateFinalGrade updates a quarter/semester/year mark via the generic quarter_mark/update endpoint.
func (c *EdonishClient) UpdateFinalGrade(markID string, newMark int) error {
        reqBody := UpdateFinalGradeRequest{
                MarkID:  markID,
                NewMark: newMark,
        }

        u := c.buildURL("/journal/quarter_mark/update", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// DeleteFinalGrade deletes a quarter/semester/year mark.
func (c *EdonishClient) DeleteFinalGrade(markID string) error {
        params := map[string]string{
                "mark_id": markID,
        }
        u := c.buildURL("/journal/quarter_mark/delete", params)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, nil)
        return err
}
