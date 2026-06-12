// Package api provides the eDonish API client for authentication and data operations.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/4codegit/edonish-auto/internal/config"
)

// AuthenticationError is returned when login fails.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// School represents a school associated with the user's account.
type School struct {
	ID         int
	Name       string
	Role       string
	RolePrefix string
}

// Client is the eDonish API client.
type Client struct {
	httpClient   *http.Client
	JWTToken     string
	RefreshToken string
	UserInfo     *UserInfo
	SchoolID     int
	Role         string
	RolePrefix   string
	UID          string
	Schools      []School // All schools available to this user
}

// UserInfo holds authenticated user data.
type UserInfo struct {
	UID       string `json:"uid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// NewClient creates a new API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// headers returns the authentication headers.
func (c *Client) headers() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + c.JWTToken,
		"Content-Type":  "application/json",
	}
}

// url builds the full API URL with role prefix.
func (c *Client) url(endpoint string, useRolePrefix bool) string {
	prefix := ""
	if useRolePrefix {
		prefix = c.RolePrefix
	}
	return config.APIBase + prefix + endpoint
}

// doRequest performs an authenticated API request.
func (c *Client) doRequest(method, url string, body interface{}) (interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		if c.RefreshAuth() {
			for k, v := range c.headers() {
				req.Header.Set(k, v)
			}
			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("retry request: %w", err)
			}
			defer resp.Body.Close()
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}

// doRequestWithParams performs an authenticated API request with query parameters.
func (c *Client) doRequestWithParams(method, url string, params map[string]interface{}, body interface{}) (interface{}, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}

	q := req.URL.Query()
	for k, v := range params {
		switch val := v.(type) {
		case int:
			q.Set(k, fmt.Sprintf("%d", val))
		case string:
			q.Set(k, val)
		case bool:
			if val {
				q.Set(k, "true")
			} else {
				q.Set(k, "false")
			}
		default:
			q.Set(k, fmt.Sprintf("%v", val))
		}
	}
	req.URL.RawQuery = q.Encode()

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.ContentLength = int64(len(data))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		if c.RefreshAuth() {
			for k, v := range c.headers() {
				req.Header.Set(k, v)
			}
			req.Body = nil
			if body != nil {
				data, _ := json.Marshal(body)
				req.Body = io.NopCloser(bytes.NewReader(data))
			}
			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Login authenticates with the eDonish API.
func (c *Client) Login(loginID, password string) (*UserInfo, error) {
	body := map[string]string{
		"login":    loginID,
		"password": password,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", config.APILogin, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &AuthenticationError{Message: fmt.Sprintf("Ошибка сети: %v", err)}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &AuthenticationError{Message: "Неверный ответ сервера"}
	}

	if statusCode, ok := result["status_code"].(float64); ok && statusCode != 0 {
		return nil, &AuthenticationError{Message: fmt.Sprintf("Ошибка входа: код %v", statusCode)}
	}

	jwt, ok := result["jwt_token"].(string)
	if !ok {
		return nil, &AuthenticationError{Message: "Не получен токен авторизации"}
	}
	c.JWTToken = jwt

	if rt, ok := result["refresh_token"].(string); ok {
		c.RefreshToken = rt
	}
	if uid, ok := result["uid"].(string); ok {
		c.UID = uid
	}

	c.UserInfo = &UserInfo{
		UID:       c.UID,
		FirstName: stringField(result, "first_name"),
		LastName:  stringField(result, "last_name"),
	}

	if err := c.resolveRoleAndSchool(); err != nil {
		log.Printf("Warning: could not resolve role: %v", err)
		c.Role = "teacher"
		c.RolePrefix = config.APIPrefixes["teacher"]
	}

	log.Printf("Logged in as %s %s (role: %s, school: %d)",
		c.UserInfo.LastName, c.UserInfo.FirstName, c.Role, c.SchoolID)

	return c.UserInfo, nil
}

// resolveRoleAndSchool determines the user role and school ID.
func (c *Client) resolveRoleAndSchool() error {
	req, err := http.NewRequest("GET", config.APIHeaderInfo, nil)
	if err != nil {
		return err
	}
	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}
	q := req.URL.Query()
	q.Set("lang", fmt.Sprintf("%d", config.LangRU))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var roles []interface{}
	if err := json.Unmarshal(body, &roles); err != nil {
		return err
	}

	if len(roles) == 0 {
		c.Role = "teacher"
		c.RolePrefix = config.APIPrefixes["teacher"]
		c.Schools = []School{{ID: 0, Name: "Школа по умолчанию", Role: "teacher", RolePrefix: "/teacher/v1"}}
		return nil
	}

	// Collect unique schools from all roles
	schoolSet := make(map[int]School)
	var order []int

	for _, r := range roles {
		if roleData, ok := r.(map[string]interface{}); ok {
			schoolID := intField(roleData, "schoolId")
			roleName := stringField(roleData, "name")
			schoolName := stringField(roleData, "schoolName")
			if schoolName == "" {
				schoolName = fmt.Sprintf("Школа ID: %d", schoolID)
			}

			rolePrefix := "/teacher/v1"
			if prefix, ok := config.APIPrefixes[roleName]; ok {
				rolePrefix = prefix
			}

			if _, exists := schoolSet[schoolID]; !exists {
				schoolSet[schoolID] = School{
					ID:         schoolID,
					Name:       schoolName,
					Role:       roleName,
					RolePrefix: rolePrefix,
				}
				order = append(order, schoolID)
			}
		}
	}

	// Build ordered school list
	c.Schools = make([]School, 0, len(schoolSet))
	for _, id := range order {
		c.Schools = append(c.Schools, schoolSet[id])
	}

	// Prefer teacher or classroom-teacher role for the default selection
	chosen := false
	for _, school := range c.Schools {
		if school.Role == "teacher" || school.Role == "classroom-teacher" {
			c.SchoolID = school.ID
			c.Role = school.Role
			c.RolePrefix = school.RolePrefix
			chosen = true
			break
		}
	}
	if !chosen && len(c.Schools) > 0 {
		c.SchoolID = c.Schools[0].ID
		c.Role = c.Schools[0].Role
		c.RolePrefix = c.Schools[0].RolePrefix
	}

	return nil
}

// GetSchools returns all schools available to the current user.
func (c *Client) GetSchools() []School {
	return c.Schools
}

// SetSchool switches the active school and updates role info.
func (c *Client) SetSchool(schoolID int) bool {
	for _, school := range c.Schools {
		if school.ID == schoolID {
			c.SchoolID = school.ID
			c.Role = school.Role
			c.RolePrefix = school.RolePrefix
			return true
		}
	}
	return false
}

// HasMultipleSchools returns true if the user has more than one school.
func (c *Client) HasMultipleSchools() bool {
	return len(c.Schools) > 1
}

// RefreshAuth refreshes the JWT token.
func (c *Client) RefreshAuth() bool {
	if c.RefreshToken == "" {
		return false
	}
	req, err := http.NewRequest("GET", config.APIRefresh, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.RefreshToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}

	if jwt, ok := result["jwt_token"].(string); ok {
		c.JWTToken = jwt
	}
	if rt, ok := result["refresh_token"].(string); ok {
		c.RefreshToken = rt
	}
	return true
}

// GetJournalOptions returns available classes, subjects, and subgroups.
func (c *Client) GetJournalOptions() (interface{}, error) {
	return c.doRequestWithParams("OPTIONS", c.url(config.JournalOptions, true), map[string]interface{}{
		"lang":      config.LangRU,
		"school_id": c.SchoolID,
	}, nil)
}

// GetQuarters returns quarter periods for the school.
func (c *Client) GetQuarters() ([]interface{}, error) {
	result, err := c.doRequestWithParams("GET",
		config.APIBase+"/school_admin/v1"+config.PeriodQuarters,
		map[string]interface{}{
			"school_id": c.SchoolID,
			"rank_id":   2,
			"lang":      config.LangRU,
		}, nil)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, nil
}

// GetJournalDates returns dates for a specific group/subject/quarter.
func (c *Client) GetJournalDates(groupID, subjectID, quarterPropertyID int) (interface{}, error) {
	return c.doRequestWithParams("GET", c.url(config.JournalDates, true), map[string]interface{}{
		"group_id":            groupID,
		"subject_id":          subjectID,
		"quarter_property_id": quarterPropertyID,
		"school_id":           c.SchoolID,
		"lang":                config.LangRU,
	}, nil)
}

// GetJournalStudents returns students with their marks.
func (c *Client) GetJournalStudents(groupID, subjectID, quarterPropertyID int) (interface{}, error) {
	return c.doRequestWithParams("GET", c.url(config.JournalStudents, true), map[string]interface{}{
		"group_id":            groupID,
		"subject_id":          subjectID,
		"quarter_property_id": quarterPropertyID,
		"school_id":           c.SchoolID,
		"lang":                config.LangRU,
	}, nil)
}

// CreateMark creates a mark for a student on a specific date.
func (c *Client) CreateMark(studentID int, assignmentDateID string, mark, markTypeID, quarterPropertyID int, description string) (interface{}, error) {
	body := map[string]interface{}{
		"mark_type_id":                  markTypeID,
		"group_subgroup_student_id":     studentID,
		"schedule_date_id":              assignmentDateID,
		"quarter_property_id":           quarterPropertyID,
		"mark":                          mark,
		"signature":                     description,
	}
	return c.doRequestWithParams("POST", c.url(config.JournalMarkCreate, true), map[string]interface{}{
		"school_id": c.SchoolID,
		"lang":      config.LangRU,
	}, body)
}

// DeleteMark deletes a mark by its ID.
func (c *Client) DeleteMark(markID string) (interface{}, error) {
	return c.doRequestWithParams("POST", c.url(config.JournalMarkDelete, true), map[string]interface{}{
		"mark_id":   markID,
		"school_id": c.SchoolID,
	}, nil)
}

// CreateQuarterMark creates a quarter mark.
func (c *Client) CreateQuarterMark(studentID, quarterPropertyID, mark int) (interface{}, error) {
	body := map[string]interface{}{
		"group_subgroup_student_id": studentID,
		"quarter_property_id":       quarterPropertyID,
		"mark":                      mark,
	}
	return c.doRequestWithParams("POST", c.url(config.JournalQuarterCreate, true), map[string]interface{}{
		"school_id": c.SchoolID,
	}, body)
}

// CreateSemesterMark creates a semester mark.
func (c *Client) CreateSemesterMark(studentID, semesterPropertyID, mark int) (interface{}, error) {
	body := map[string]interface{}{
		"group_subgroup_student_id": studentID,
		"semester_property_id":      semesterPropertyID,
		"mark":                      mark,
	}
	return c.doRequestWithParams("POST", c.url(config.JournalSemesterCreate, true), map[string]interface{}{
		"school_id": c.SchoolID,
	}, body)
}

// CreateYearMark creates a year mark.
func (c *Client) CreateYearMark(studentID, yearPropertyID, mark int) (interface{}, error) {
	body := map[string]interface{}{
		"group_subgroup_student_id": studentID,
		"year_property_id":          yearPropertyID,
		"mark":                      mark,
	}
	return c.doRequestWithParams("POST", c.url(config.JournalYearCreate, true), map[string]interface{}{
		"school_id": c.SchoolID,
	}, body)
}

// Helper functions for extracting fields from maps

func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func intField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
