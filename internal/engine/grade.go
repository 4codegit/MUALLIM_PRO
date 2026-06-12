// Package engine implements the grade automation engine with concurrent workers.
// v0.4.0: Smart grading based on student analysis, not random.
package engine

import (
        "fmt"
        "log"
        "math"
        "math/rand"
        "sync"
        "sync/atomic"
        "time"

        "github.com/4codegit/edonish-auto/internal/api"
        "github.com/4codegit/edonish-auto/internal/config"
)

// TaskStatus represents the current state of a grade task.
type TaskStatus int

const (
        StatusPending TaskStatus = iota
        StatusRunning
        StatusSuccess
        StatusError
        StatusSkipped
)

// TaskType represents what kind of grade task this is.
type TaskType int

const (
        TaskDaily    TaskType = iota // Daily mark (for a specific date)
        TaskQuarter                  // Quarter mark
        TaskSemester                 // Semester mark
        TaskYear                     // Year mark
        TaskSignature                // Signature/подпись for a date
)

// GradeTask represents a single grade creation task.
type GradeTask struct {
        StudentID         int
        StudentName       string
        AssignmentDateID  string
        DateStr           string
        QuarterPropertyID int
        Mark              int
        SubjectName       string
        GroupName         string
        Status            TaskStatus
        Error             string
        TaskType          TaskType
        // For quarter/semester/year marks
        SemesterPropertyID int
        YearPropertyID     int
}

// GradePlan represents a complete plan for grade creation.
type GradePlan struct {
        Tasks      []*GradeTask
        TotalTasks int
        Completed  int32
        Failed     int32
        Skipped    int32
        mu         sync.Mutex
}

// NewGradePlan creates an empty grade plan.
func NewGradePlan() *GradePlan {
        return &GradePlan{}
}

// AddTask adds a task to the plan.
func (p *GradePlan) AddTask(t *GradeTask) {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.Tasks = append(p.Tasks, t)
        p.TotalTasks = len(p.Tasks)
}

// Progress returns the current progress as a fraction (0.0 to 1.0).
func (p *GradePlan) Progress() float64 {
        if p.TotalTasks == 0 {
                return 0
        }
        done := int(atomic.LoadInt32(&p.Completed)) + int(atomic.LoadInt32(&p.Failed)) + int(atomic.LoadInt32(&p.Skipped))
        return float64(done) / float64(p.TotalTasks)
}

// PendingCount returns the number of tasks still pending.
func (p *GradePlan) PendingCount() int {
        p.mu.Lock()
        defer p.mu.Unlock()
        count := 0
        for _, t := range p.Tasks {
                if t.Status == StatusPending {
                        count++
                }
        }
        return count
}

// ProgressCallback is called when progress updates.
type ProgressCallback func(plan *GradePlan)

// LogCallback is called for log messages.
type LogCallback func(message, level string)

// Engine handles automated grade creation with parallel processing.
type Engine struct {
        api              *api.Client
        stopChan         chan struct{}
        progressCallback ProgressCallback
        logCallback      LogCallback
        running          atomic.Bool

        // Per-student grade limits: studentName → [min, max]
        StudentLimits map[string][2]int
}

// NewEngine creates a new grade engine.
func NewEngine(client *api.Client) *Engine {
        return &Engine{
                api:      client,
                stopChan: make(chan struct{}),
        }
}

// SetCallbacks sets the progress and log callbacks.
func (e *Engine) SetCallbacks(progressCB ProgressCallback, logCB LogCallback) {
        e.progressCallback = progressCB
        e.logCallback = logCB
}

// SetStudentLimits sets per-student grade limits (studentName → [min, max]).
func (e *Engine) SetStudentLimits(limits map[string][2]int) {
        e.StudentLimits = limits
}

// getStudentLimits returns the min/max grades for a student, using
// per-student overrides if set, otherwise the global defaults.
func (e *Engine) getStudentLimits(studentName string, globalMin, globalMax int) (int, int) {
        if e.StudentLimits != nil {
                if limits, ok := e.StudentLimits[studentName]; ok {
                        return limits[0], limits[1]
                }
        }
        return globalMin, globalMax
}

// IsRunning returns whether the engine is currently executing.
func (e *Engine) IsRunning() bool {
        return e.running.Load()
}

// Stop signals the engine to stop processing.
func (e *Engine) Stop() {
        e.running.Store(false)
        close(e.stopChan)
        e.log("Остановка...", "warning")
        e.stopChan = make(chan struct{})
}

func (e *Engine) log(message, level string) {
        if e.logCallback != nil {
                e.logCallback(message, level)
        }
        if level == "error" {
                log.Printf("[ERROR] %s", message)
        } else if level == "warning" {
                log.Printf("[WARN] %s", message)
        } else {
                log.Printf("[INFO] %s", message)
        }
}

func (e *Engine) updateProgress(plan *GradePlan) {
        if e.progressCallback != nil {
                e.progressCallback(plan)
        }
}

// ─── Smart Grade Calculation ─────────────────────────────────────

// StudentAnalysis holds the analysis of a student's existing grades.
type StudentAnalysis struct {
        StudentID   int
        StudentName string
        ExistingGrades []int  // All existing numeric grades
        Average     float64  // Average of existing grades
        GradeCount  int      // Number of existing grades
        Min         int      // Minimum existing grade
        Max         int      // Maximum existing grade
        Missing     int      // Number of missing grades
}

// SmartGrade calculates a grade for a student based on their existing performance.
// Instead of random, it uses the student's average as the base and adds controlled variation.
// The grade is guaranteed to be within [minGrade, maxGrade].
func SmartGrade(analysis *StudentAnalysis, minGrade, maxGrade int) int {
        if analysis.GradeCount == 0 {
                // No existing grades — use the middle of the range with slight variation
                mid := (minGrade + maxGrade) / 2
                variation := rand.Intn(2) // 0 or 1
                if rand.Intn(2) == 0 {
                        variation = -variation
                }
                grade := mid + variation
                return clampGrade(grade, minGrade, maxGrade)
        }

        // Base the grade on the student's average
        avg := analysis.Average

        // Add controlled variation: ±1 or ±2 around the average
        // This makes grades look natural but consistent with the student's level
        variation := 0
        r := rand.Intn(100)
        switch {
        case r < 40: // 40% chance: exact average
                variation = 0
        case r < 70: // 30% chance: +1
                variation = 1
        case r < 85: // 15% chance: -1
                variation = -1
        case r < 93: // 8% chance: +2
                variation = 2
        case r < 98: // 5% chance: -2
                variation = -2
        default: // 2% chance: bigger swing
                variation = rand.Intn(3) - 1 // -1, 0, 1
        }

        grade := int(math.Round(avg)) + variation
        return clampGrade(grade, minGrade, maxGrade)
}

// CalculateQuarterMark calculates the quarter mark based on daily grades.
// It rounds the average and ensures it's within the specified range.
func CalculateQuarterMark(analysis *StudentAnalysis, minGrade, maxGrade int) int {
        if analysis.GradeCount == 0 {
                // No daily grades — use the middle of the range
                return (minGrade + maxGrade) / 2
        }

        // Round the average to nearest integer
        grade := int(math.Round(analysis.Average))

        // Slight bump up (quarter marks tend to be rounded up)
        if analysis.Average-float64(grade) >= -0.3 {
                grade = int(math.Ceil(analysis.Average))
        }

        return clampGrade(grade, minGrade, maxGrade)
}

// CalculateSemesterMark calculates the semester mark from quarter averages.
func CalculateSemesterMark(quarterAverages []float64, minGrade, maxGrade int) int {
        if len(quarterAverages) == 0 {
                return (minGrade + maxGrade) / 2
        }
        sum := 0.0
        for _, q := range quarterAverages {
                sum += q
        }
        avg := sum / float64(len(quarterAverages))
        grade := int(math.Round(avg))
        return clampGrade(grade, minGrade, maxGrade)
}

// CalculateYearMark calculates the year mark from semester averages.
func CalculateYearMark(semesterAverages []float64, minGrade, maxGrade int) int {
        if len(semesterAverages) == 0 {
                return (minGrade + maxGrade) / 2
        }
        sum := 0.0
        for _, q := range semesterAverages {
                sum += q
        }
        avg := sum / float64(len(semesterAverages))
        grade := int(math.Round(avg))
        return clampGrade(grade, minGrade, maxGrade)
}

// clampGrade ensures grade is within [min, max].
func clampGrade(grade, min, max int) int {
        if grade < min {
                return min
        }
        if grade > max {
                return max
        }
        return grade
}

// AnalyzeStudent creates an analysis of a student's existing grades.
func AnalyzeStudent(student map[string]interface{}) *StudentAnalysis {
        a := &StudentAnalysis{
                StudentID:   intField(student, "studentId"),
                StudentName: fmt.Sprintf("%s %s", stringField(student, "lastName"), stringField(student, "firstName")),
        }

        existingMarks := ExtractExistingMarksWithValues(student)
        for _, val := range existingMarks {
                if val > 0 {
                        a.ExistingGrades = append(a.ExistingGrades, val)
                }
        }

        a.GradeCount = len(a.ExistingGrades)
        if a.GradeCount > 0 {
                sum := 0
                a.Min = a.ExistingGrades[0]
                a.Max = a.ExistingGrades[0]
                for _, g := range a.ExistingGrades {
                        sum += g
                        if g < a.Min {
                                a.Min = g
                        }
                        if g > a.Max {
                                a.Max = g
                        }
                }
                a.Average = float64(sum) / float64(a.GradeCount)
        }

        return a
}

// ─── Build Complete Grade Plan ──────────────────────────────────

// BuildGradePlan builds a complete plan of grades to create.
// Uses smart grading based on student analysis instead of random.
func (e *Engine) BuildGradePlan(
        groups []map[string]interface{},
        subjects []map[string]interface{},
        quarters []map[string]interface{},
        minGrade, maxGrade int,
        fillEmptyOnly bool,
) *GradePlan {
        plan := NewGradePlan()
        e.log("Построение плана оценок (умный режим)...", "info")

        for _, group := range groups {
                groupID := intField(group, "id")
                groupName := fmt.Sprintf("%s%s", stringField(group, "number"), stringField(group, "name"))

                for _, subject := range subjects {
                        subjectID := intField(subject, "subjectId")
                        if subjectID == 0 {
                                subjectID = intField(subject, "id")
                        }
                        subjectName := stringField(subject, "subjectName")
                        if subjectName == "" {
                                subjectName = stringField(subject, "name")
                        }

                        for _, quarter := range quarters {
                                qpropID := intField(quarter, "qpropId")
                                quarterName := stringField(quarter, "name")
                                if quarterName == "" {
                                        quarterName = fmt.Sprintf("Четверть %d", qpropID)
                                }

                                e.log(fmt.Sprintf("%s | %s | %s", groupName, subjectName, quarterName), "info")

                                // Get dates
                                datesData, err := e.api.GetJournalDates(groupID, subjectID, qpropID)
                                if err != nil {
                                        e.log(fmt.Sprintf("  Ошибка дат: %v", err), "error")
                                        continue
                                }
                                days := ExtractDays(datesData)
                                if len(days) == 0 {
                                        e.log("  Нет дат", "info")
                                        continue
                                }

                                // Get students
                                studentsData, err := e.api.GetJournalStudents(groupID, subjectID, qpropID)
                                if err != nil {
                                        e.log(fmt.Sprintf("  Ошибка студентов: %v", err), "error")
                                        continue
                                }
                                students := ExtractStudents(studentsData)
                                if len(students) == 0 {
                                        e.log("  Нет студентов", "info")
                                        continue
                                }

                                // Plan grades for each student
                                for _, student := range students {
                                        analysis := AnalyzeStudent(student)
                                        existingMarks := ExtractExistingMarks(student)
                                        sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)

                                        for _, day := range days {
                                                dateID := stringField(day, "assignmentDateId")
                                                dateStr := stringField(day, "assignmentDate")

                                                if fillEmptyOnly {
                                                        if _, hasMark := existingMarks[dateID]; hasMark {
                                                                task := &GradeTask{
                                                                        StudentID:         analysis.StudentID,
                                                                        StudentName:       analysis.StudentName,
                                                                        AssignmentDateID:  dateID,
                                                                        DateStr:           dateStr,
                                                                        QuarterPropertyID: qpropID,
                                                                        Mark:              0,
                                                                        SubjectName:       subjectName,
                                                                        GroupName:         groupName,
                                                                        Status:            StatusSkipped,
                                                                        TaskType:          TaskDaily,
                                                                }
                                                                plan.AddTask(task)
                                                                atomic.AddInt32(&plan.Skipped, 1)
                                                                continue
                                                        }
                                                }

                                                // Smart grade based on student analysis
                                                grade := SmartGrade(analysis, sMin, sMax)
                                                task := &GradeTask{
                                                        StudentID:         analysis.StudentID,
                                                        StudentName:       analysis.StudentName,
                                                        AssignmentDateID:  dateID,
                                                        DateStr:           dateStr,
                                                        QuarterPropertyID: qpropID,
                                                        Mark:              grade,
                                                        SubjectName:       subjectName,
                                                        GroupName:         groupName,
                                                        Status:            StatusPending,
                                                        TaskType:          TaskDaily,
                                                }
                                                plan.AddTask(task)
                                        }
                                }
                        }
                }
        }

        e.log(fmt.Sprintf("План: %d задач (%d пропущено)", plan.TotalTasks, int(atomic.LoadInt32(&plan.Skipped))), "info")
        return plan
}

// BuildCompletePlan builds a FULL plan: daily + quarter + semester + year marks.
// This fills EVERYTHING for the selected groups/subjects/quarters.
func (e *Engine) BuildCompletePlan(
        groups []map[string]interface{},
        subjects []map[string]interface{},
        quarters []map[string]interface{},
        minGrade, maxGrade int,
        fillEmptyOnly bool,
        includeDaily bool,
        includeQuarter bool,
        includeSemester bool,
        includeYear bool,
) *GradePlan {
        plan := NewGradePlan()
        e.log("Построение полного плана (дневные + четвертные + семестровые + годовые)...", "info")

        for _, group := range groups {
                groupID := intField(group, "id")
                groupName := fmt.Sprintf("%s%s", stringField(group, "number"), stringField(group, "name"))

                for _, subject := range subjects {
                        subjectID := intField(subject, "subjectId")
                        if subjectID == 0 {
                                subjectID = intField(subject, "id")
                        }
                        subjectName := stringField(subject, "subjectName")
                        if subjectName == "" {
                                subjectName = stringField(subject, "name")
                        }

                        for _, quarter := range quarters {
                                qpropID := intField(quarter, "qpropId")
                                quarterName := stringField(quarter, "name")
                                if quarterName == "" {
                                        quarterName = fmt.Sprintf("Четверть %d", qpropID)
                                }

                                e.log(fmt.Sprintf("%s | %s | %s", groupName, subjectName, quarterName), "info")

                                datesData, err := e.api.GetJournalDates(groupID, subjectID, qpropID)
                                if err != nil {
                                        e.log(fmt.Sprintf("  Ошибка дат: %v", err), "error")
                                        continue
                                }
                                days := ExtractDays(datesData)

                                studentsData, err := e.api.GetJournalStudents(groupID, subjectID, qpropID)
                                if err != nil {
                                        e.log(fmt.Sprintf("  Ошибка студентов: %v", err), "error")
                                        continue
                                }
                                students := ExtractStudents(studentsData)

                                if len(students) == 0 {
                                        continue
                                }

                                // ── Daily marks ──
                                if includeDaily && len(days) > 0 {
                                        for _, student := range students {
                                                analysis := AnalyzeStudent(student)
                                                existingMarks := ExtractExistingMarks(student)
                                                sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)

                                                for _, day := range days {
                                                        dateID := stringField(day, "assignmentDateId")
                                                        dateStr := stringField(day, "assignmentDate")

                                                        if fillEmptyOnly {
                                                                if _, hasMark := existingMarks[dateID]; hasMark {
                                                                        task := &GradeTask{
                                                                                StudentID:         analysis.StudentID,
                                                                                StudentName:       analysis.StudentName,
                                                                                AssignmentDateID:  dateID,
                                                                                DateStr:           dateStr,
                                                                                QuarterPropertyID: qpropID,
                                                                                Mark:              0,
                                                                                SubjectName:       subjectName,
                                                                                GroupName:         groupName,
                                                                                Status:            StatusSkipped,
                                                                                TaskType:          TaskDaily,
                                                                        }
                                                                        plan.AddTask(task)
                                                                        atomic.AddInt32(&plan.Skipped, 1)
                                                                        continue
                                                                }
                                                        }

                                                        grade := SmartGrade(analysis, sMin, sMax)
                                                        task := &GradeTask{
                                                                StudentID:         analysis.StudentID,
                                                                StudentName:       analysis.StudentName,
                                                                AssignmentDateID:  dateID,
                                                                DateStr:           dateStr,
                                                                QuarterPropertyID: qpropID,
                                                                Mark:              grade,
                                                                SubjectName:       subjectName,
                                                                GroupName:         groupName,
                                                                Status:            StatusPending,
                                                                TaskType:          TaskDaily,
                                                        }
                                                        plan.AddTask(task)
                                                }
                                        }
                                }

                                // ── Quarter marks ──
                                if includeQuarter {
                                        for _, student := range students {
                                                analysis := AnalyzeStudent(student)
                                                sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)

                                                if fillEmptyOnly && hasQuarterMark(student) {
                                                        continue
                                                }

                                                grade := CalculateQuarterMark(analysis, sMin, sMax)
                                                task := &GradeTask{
                                                        StudentID:         analysis.StudentID,
                                                        StudentName:       analysis.StudentName,
                                                        QuarterPropertyID: qpropID,
                                                        DateStr:           quarterName,
                                                        Mark:              grade,
                                                        SubjectName:       subjectName,
                                                        GroupName:         groupName,
                                                        Status:            StatusPending,
                                                        TaskType:          TaskQuarter,
                                                }
                                                plan.AddTask(task)
                                        }
                                }

                                // ── Semester marks ──
                                if includeSemester {
                                        semesterPropID := intField(quarter, "spropId")
                                        if semesterPropID == 0 {
                                                // Try to extract from quarter data
                                                semesterPropID = intField(quarter, "semesterPropertyId")
                                        }
                                        if semesterPropID > 0 {
                                                for _, student := range students {
                                                        analysis := AnalyzeStudent(student)
                                                        sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)
                                                        grade := CalculateQuarterMark(analysis, sMin, sMax)
                                                        task := &GradeTask{
                                                                StudentID:          analysis.StudentID,
                                                                StudentName:        analysis.StudentName,
                                                                SemesterPropertyID: semesterPropID,
                                                                QuarterPropertyID:  qpropID,
                                                                DateStr:            "Семестр",
                                                                Mark:               grade,
                                                                SubjectName:        subjectName,
                                                                GroupName:          groupName,
                                                                Status:             StatusPending,
                                                                TaskType:           TaskSemester,
                                                        }
                                                        plan.AddTask(task)
                                                }
                                        }
                                }

                                // ── Year marks ──
                                if includeYear {
                                        yearPropID := intField(quarter, "ypropId")
                                        if yearPropID == 0 {
                                                yearPropID = intField(quarter, "yearPropertyId")
                                        }
                                        if yearPropID > 0 {
                                                for _, student := range students {
                                                        analysis := AnalyzeStudent(student)
                                                        sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)
                                                        grade := CalculateQuarterMark(analysis, sMin, sMax)
                                                        task := &GradeTask{
                                                                StudentID:         analysis.StudentID,
                                                                StudentName:       analysis.StudentName,
                                                                YearPropertyID:    yearPropID,
                                                                QuarterPropertyID: qpropID,
                                                                DateStr:           "Год",
                                                                Mark:              grade,
                                                                SubjectName:       subjectName,
                                                                GroupName:         groupName,
                                                                Status:            StatusPending,
                                                                TaskType:          TaskYear,
                                                        }
                                                        plan.AddTask(task)
                                                }
                                        }
                                }
                        }
                }
        }

        e.log(fmt.Sprintf("Полный план: %d задач (%d пропущено)", plan.TotalTasks, int(atomic.LoadInt32(&plan.Skipped))), "info")
        return plan
}

// hasQuarterMark checks if a student already has a quarter mark.
func hasQuarterMark(student map[string]interface{}) bool {
        if qm := getMapField(student, "quarterMark"); qm != nil {
                if arr, ok := qm.([]interface{}); ok && len(arr) > 0 {
                        if first, ok := arr[0].(map[string]interface{}); ok {
                                if sn := stringField(first, "shortName"); sn != "" {
                                        return true
                                }
                        }
                }
        }
        return false
}

// BuildGradePlanForQuarterMarks builds a plan for quarter marks only.
// Uses smart grading based on student's daily grades average.
func (e *Engine) BuildGradePlanForQuarterMarks(
        groups []map[string]interface{},
        subjects []map[string]interface{},
        quarters []map[string]interface{},
        minGrade, maxGrade int,
        fillEmptyOnly bool,
) *GradePlan {
        plan := NewGradePlan()
        e.log("Построение плана четвертных оценок (умный режим)...", "info")

        for _, group := range groups {
                groupID := intField(group, "id")
                groupName := fmt.Sprintf("%s%s", stringField(group, "number"), stringField(group, "name"))

                for _, subject := range subjects {
                        subjectID := intField(subject, "subjectId")
                        if subjectID == 0 {
                                subjectID = intField(subject, "id")
                        }
                        subjectName := stringField(subject, "subjectName")
                        if subjectName == "" {
                                subjectName = stringField(subject, "name")
                        }

                        for _, quarter := range quarters {
                                qpropID := intField(quarter, "qpropId")
                                quarterName := stringField(quarter, "name")

                                studentsData, err := e.api.GetJournalStudents(groupID, subjectID, qpropID)
                                if err != nil {
                                        e.log(fmt.Sprintf("  Ошибка: %v", err), "error")
                                        continue
                                }
                                students := ExtractStudents(studentsData)

                                for _, student := range students {
                                        analysis := AnalyzeStudent(student)
                                        sMin, sMax := e.getStudentLimits(analysis.StudentName, minGrade, maxGrade)

                                        if fillEmptyOnly && hasQuarterMark(student) {
                                                continue
                                        }

                                        // Calculate quarter mark from daily grades average
                                        grade := CalculateQuarterMark(analysis, sMin, sMax)

                                        task := &GradeTask{
                                                StudentID:         analysis.StudentID,
                                                StudentName:       analysis.StudentName,
                                                QuarterPropertyID: qpropID,
                                                DateStr:           quarterName,
                                                Mark:              grade,
                                                SubjectName:       subjectName,
                                                GroupName:         groupName,
                                                Status:            StatusPending,
                                                TaskType:          TaskQuarter,
                                        }
                                        plan.AddTask(task)
                                }
                        }
                }
        }

        e.log(fmt.Sprintf("План: %d четвертных оценок", plan.TotalTasks), "info")
        return plan
}

// ─── Execute Plans ──────────────────────────────────────────────

// ExecutePlan executes the grade plan with parallel workers.
func (e *Engine) ExecutePlan(plan *GradePlan, numWorkers int, taskDelay time.Duration) {
        e.running.Store(true)
        defer e.running.Store(false)

        tasks := make([]*GradeTask, 0)
        for _, t := range plan.Tasks {
                if t.Status == StatusPending {
                        tasks = append(tasks, t)
                }
        }

        if len(tasks) == 0 {
                e.log("Нет задач для выполнения", "info")
                return
        }

        e.log(fmt.Sprintf("Запуск %d задач с %d воркерами...", len(tasks), numWorkers), "info")

        workerTasks := make([][]*GradeTask, numWorkers)
        for i, t := range tasks {
                workerIdx := i % numWorkers
                workerTasks[workerIdx] = append(workerTasks[workerIdx], t)
        }

        var wg sync.WaitGroup
        for i, tasks := range workerTasks {
                if len(tasks) == 0 {
                        continue
                }
                wg.Add(1)
                go func(workerID int, tasks []*GradeTask) {
                        defer wg.Done()
                        for _, task := range tasks {
                                select {
                                case <-e.stopChan:
                                        task.Status = StatusSkipped
                                        continue
                                default:
                                }

                                task.Status = StatusRunning
                                e.updateProgress(plan)

                                var result interface{}
                                var err error

                                switch task.TaskType {
                                case TaskDaily:
                                        result, err = e.api.CreateMark(
                                                task.StudentID,
                                                task.AssignmentDateID,
                                                task.Mark,
                                                8, // mark_type_id
                                                task.QuarterPropertyID,
                                                config.Signature,
                                        )
                                case TaskQuarter:
                                        result, err = e.api.CreateQuarterMark(
                                                task.StudentID,
                                                task.QuarterPropertyID,
                                                task.Mark,
                                        )
                                case TaskSemester:
                                        result, err = e.api.CreateSemesterMark(
                                                task.StudentID,
                                                task.SemesterPropertyID,
                                                task.Mark,
                                        )
                                case TaskYear:
                                        result, err = e.api.CreateYearMark(
                                                task.StudentID,
                                                task.YearPropertyID,
                                                task.Mark,
                                        )
                                default:
                                        result, err = e.api.CreateMark(
                                                task.StudentID,
                                                task.AssignmentDateID,
                                                task.Mark,
                                                8,
                                                task.QuarterPropertyID,
                                                config.Signature,
                                        )
                                }

                                if err != nil {
                                        task.Status = StatusError
                                        task.Error = err.Error()
                                        atomic.AddInt32(&plan.Failed, 1)
                                        e.log(fmt.Sprintf("  [%d] %s: %v", workerID, task.StudentName, err), "error")
                                } else if resultMap, ok := result.(map[string]interface{}); ok {
                                        if errMsg, exists := resultMap["error"]; exists && errMsg != nil {
                                                task.Status = StatusError
                                                task.Error = fmt.Sprintf("%v", errMsg)
                                                atomic.AddInt32(&plan.Failed, 1)
                                                e.log(fmt.Sprintf("  [%d] %s: %v", workerID, task.StudentName, errMsg), "error")
                                        } else {
                                                task.Status = StatusSuccess
                                                atomic.AddInt32(&plan.Completed, 1)
                                                typeLabel := taskTypeLabel(task.TaskType)
                                                e.log(fmt.Sprintf("  [%d] %s -> %d (%s, %s)", workerID, task.StudentName, task.Mark, task.DateStr, typeLabel), "info")
                                        }
                                } else {
                                        task.Status = StatusSuccess
                                        atomic.AddInt32(&plan.Completed, 1)
                                        typeLabel := taskTypeLabel(task.TaskType)
                                        e.log(fmt.Sprintf("  [%d] %s -> %d (%s, %s)", workerID, task.StudentName, task.Mark, task.DateStr, typeLabel), "info")
                                }

                                e.updateProgress(plan)
                                if e.running.Load() {
                                        time.Sleep(taskDelay)
                                }
                        }
                }(i+1, tasks)
        }

        wg.Wait()

        completed := int(atomic.LoadInt32(&plan.Completed))
        failed := int(atomic.LoadInt32(&plan.Failed))
        skipped := int(atomic.LoadInt32(&plan.Skipped))
        e.log(fmt.Sprintf("Завершено! %d успешно, %d ошибок, %d пропущено", completed, failed, skipped), "info")
        e.updateProgress(plan)
}

// ExecuteQuarterMarks executes quarter marks plan sequentially.
func (e *Engine) ExecuteQuarterMarks(plan *GradePlan, taskDelay time.Duration) {
        e.running.Store(true)
        defer e.running.Store(false)

        tasks := make([]*GradeTask, 0)
        for _, t := range plan.Tasks {
                if t.Status == StatusPending {
                        tasks = append(tasks, t)
                }
        }

        if len(tasks) == 0 {
                e.log("Нет четвертных оценок", "info")
                return
        }

        for _, task := range tasks {
                select {
                case <-e.stopChan:
                        return
                default:
                }

                task.Status = StatusRunning
                result, err := e.api.CreateQuarterMark(task.StudentID, task.QuarterPropertyID, task.Mark)

                if err != nil {
                        task.Status = StatusError
                        task.Error = err.Error()
                        atomic.AddInt32(&plan.Failed, 1)
                        e.log(fmt.Sprintf("  %s: %v", task.StudentName, err), "error")
                } else if result != nil {
                        task.Status = StatusSuccess
                        atomic.AddInt32(&plan.Completed, 1)
                        e.log(fmt.Sprintf("  %s -> %d (%s)", task.StudentName, task.Mark, task.DateStr), "info")
                } else {
                        task.Status = StatusError
                        task.Error = "empty response"
                        atomic.AddInt32(&plan.Failed, 1)
                }

                e.updateProgress(plan)
                time.Sleep(taskDelay)
        }

        completed := int(atomic.LoadInt32(&plan.Completed))
        failed := int(atomic.LoadInt32(&plan.Failed))
        e.log(fmt.Sprintf("Четвертные: %d успешно, %d ошибок", completed, failed), "info")
}

func taskTypeLabel(t TaskType) string {
        switch t {
        case TaskDaily:
                return "дневная"
        case TaskQuarter:
                return "четвертная"
        case TaskSemester:
                return "семестровая"
        case TaskYear:
                return "годовая"
        case TaskSignature:
                return "подпись"
        default:
                return "оценка"
        }
}

// ─── Data extraction helpers ─────────────────────────────────────────

// ExtractDays extracts days list from API dates response.
func ExtractDays(data interface{}) []map[string]interface{} {
        if data == nil {
                return nil
        }
        if arr, ok := data.([]interface{}); ok && len(arr) > 0 {
                if first, ok := arr[0].(map[string]interface{}); ok {
                        if daysRaw, ok := first["days"].([]interface{}); ok {
                                var result []map[string]interface{}
                                for _, d := range daysRaw {
                                        if dm, ok := d.(map[string]interface{}); ok {
                                                result = append(result, dm)
                                        }
                                }
                                return result
                        }
                }
        }
        return nil
}

// ExtractStudents extracts students list from API students response.
func ExtractStudents(data interface{}) []map[string]interface{} {
        if data == nil {
                return nil
        }
        if arr, ok := data.([]interface{}); ok {
                var result []map[string]interface{}
                for _, s := range arr {
                        if sm, ok := s.(map[string]interface{}); ok {
                                result = append(result, sm)
                        }
                }
                return result
        }
        return nil
}

// ExtractExistingMarks extracts existing marks indexed by assignmentDateId.
func ExtractExistingMarks(student map[string]interface{}) map[string]bool {
        marks := make(map[string]bool)
        if subjectMarks, ok := student["subjectMarks"].([]interface{}); ok {
                for _, m := range subjectMarks {
                        if mm, ok := m.(map[string]interface{}); ok {
                                dateID := stringField(mm, "assignmentDateId")
                                if dateID != "" {
                                        marks[dateID] = true
                                }
                        }
                }
        }
        return marks
}

// ExtractExistingMarksWithValues extracts existing marks with their numeric values.
func ExtractExistingMarksWithValues(student map[string]interface{}) map[string]int {
        marks := make(map[string]int)
        if subjectMarks, ok := student["subjectMarks"].([]interface{}); ok {
                for _, m := range subjectMarks {
                        if mm, ok := m.(map[string]interface{}); ok {
                                dateID := stringField(mm, "assignmentDateId")
                                markVal := intField(mm, "mark")
                                if dateID != "" && markVal > 0 {
                                        marks[dateID] = markVal
                                }
                        }
                }
        }
        return marks
}

// ParseGradeDisplay converts API shortName to display text.
func ParseGradeDisplay(shortName string, markValue int) string {
        if shortName == "" {
                return ""
        }
        if markValue == 0 {
                return "отсутствует"
        }
        var num, den int
        if _, err := fmt.Sscanf(shortName, "%d/%d", &num, &den); err == nil && den > 0 {
                if num < config.MinGrade {
                        return "отсутствует"
                }
                return shortName
        }
        return shortName
}

func stringField(m map[string]interface{}, key string) string {
        if v, ok := m[key].(string); ok {
                return v
        }
        if v, ok := m[key].(float64); ok {
                return fmt.Sprintf("%.0f", v)
        }
        return ""
}

func intField(m map[string]interface{}, key string) int {
        if v, ok := m[key].(float64); ok {
                return int(v)
        }
        if v, ok := m[key].(int); ok {
                return v
        }
        return 0
}

func getMapField(m map[string]interface{}, key string) interface{} {
        return m[key]
}
