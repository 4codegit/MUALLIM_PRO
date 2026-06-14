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
        Name              string `json:"name"`
        QuarterPropertyID int    `json:"quarterPropertyId"`
        Days              []Day  `json:"days"`
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
        StudentID    int           `json:"studentId"`
        LastName     string        `json:"lastName"`
        FirstName    string        `json:"firstName"`
        MiddleName   string        `json:"middleName"`
        SubjectMarks []SubjectMark `json:"subjectMarks"`
        QuarterMarks []QuarterMark `json:"quarterMark"`
        AverageScore string        `json:"averageScore"`
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

type CreateMarkRequest struct {
        MarkTypeID             int    `json:"mark_type_id"`
        GroupSubgroupStudentID int    `json:"group_subgroup_student_id"`
        ScheduleDateID         string `json:"schedule_date_id"`
        QuarterPropertyID      int    `json:"quarter_property_id"`
        Mark                   int    `json:"mark"`
}

type UpdateAssignmentRequest struct {
        ScheduleDateID string `json:"schedule_date_id"`
        Topic          string `json:"topic,omitempty"`
        HomeWork       string `json:"homeWork,omitempty"`
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
        return &EdonishClient{
                httpClient: &http.Client{
                        Timeout: 15 * time.Second,
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

        if len(qDates) == 0 {
                return []Day{}, nil
        }
        return qDates[0].Days, nil
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

// UpdateAssignment updates the lesson topic and homework.
func (c *EdonishClient) UpdateAssignment(dateID string, topic, homework string) error {
        reqBody := UpdateAssignmentRequest{
                ScheduleDateID: dateID,
                Topic:          topic,
                HomeWork:       homework,
        }

        u := c.buildURL("/journal/assignment/update", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
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
type SemesterMarkCreateRequest struct {
        GroupSubgroupStudentID int `json:"group_subgroup_student_id"`
        SemesterPropertyID     int `json:"semester_property_id"`
        Mark                   int `json:"mark"`
}

// YearMarkCreateRequest is the body for creating a year mark.
// Uses POST /journal/10_point_year/create
type YearMarkCreateRequest struct {
        GroupSubgroupStudentID int `json:"group_subgroup_student_id"`
        YearPropertyID         int `json:"year_property_id"`
        Mark                   int `json:"mark"`
}

// TopicEntry represents a topic for a date.
type TopicEntry struct {
        ScheduleDateID string `json:"scheduleDateId"`
        AssignmentDate string `json:"assignmentDate"`
        Topic          string `json:"topic"`
        HomeWork       string `json:"homeWork"`
        WeekdayName    string `json:"weekdayName"`
        SubjectID      int    `json:"subjectId"`
        SubjectName    string `json:"subjectName"`
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

// --- Final Grades methods ---

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
func (c *EdonishClient) CreateSemesterMark(studentID, semesterPropertyID, mark int) error {
        reqBody := SemesterMarkCreateRequest{
                GroupSubgroupStudentID: studentID,
                SemesterPropertyID:     semesterPropertyID,
                Mark:                   mark,
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
func (c *EdonishClient) CreateYearMark(studentID, yearPropertyID, mark int) error {
        reqBody := YearMarkCreateRequest{
                GroupSubgroupStudentID: studentID,
                YearPropertyID:         yearPropertyID,
                Mark:                   mark,
        }

        u := c.buildURL("/journal/10_point_year/create", nil)
        req, err := http.NewRequest("POST", u, nil)
        if err != nil {
                return err
        }

        _, _, err = c.doRequest(req, reqBody)
        return err
}

// GetTopics fetches topic entries for a group/subject/quarter.
func (c *EdonishClient) GetTopics(groupID, subjectID, quarterID int) ([]TopicEntry, error) {
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

        var topics []TopicEntry
        for _, qd := range qDates {
                for _, d := range qd.Days {
                        topics = append(topics, TopicEntry{
                                ScheduleDateID: d.AssignmentDateID,
                                AssignmentDate: d.AssignmentDate,
                                Topic:          d.Topic,
                                HomeWork:       d.HomeWork,
                                WeekdayName:    d.WeekdayName,
                                SubjectID:      d.SubjectID,
                                SubjectName:    d.SubjectName,
                        })
                }
        }
        return topics, nil
}

// UpdateTopic updates the topic for a schedule date.
func (c *EdonishClient) UpdateTopic(dateID, topic string) error {
        reqBody := struct {
                ScheduleDateID string `json:"schedule_date_id"`
                Topic          string `json:"topic"`
        }{
                ScheduleDateID: dateID,
                Topic:          topic,
        }

        u := c.buildURL("/journal/assignment/update", nil)
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
