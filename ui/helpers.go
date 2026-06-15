package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ------------------------------------------
// GRADE COMBOS — for random fill
// ------------------------------------------

// Diligence marks — these match the edonish.tj API enum values.
var DiligenceMarks = []string{"Отличный", "Хорошо", "Удовлетворительный", "Неудовлетворительно"}

// GradeCombo defines a named range for random grade generation.
type GradeCombo struct {
	Name   string
	MinVal int
	MaxVal int
}

var GradeCombos = []GradeCombo{
	{Name: "Хорошо и Отлично", MinVal: 7, MaxVal: 10},
	{Name: "Хорошо и Плохо", MinVal: 4, MaxVal: 8},
	{Name: "Удовлетворительно и Плохо", MinVal: 3, MaxVal: 6},
	{Name: "Отлично только", MinVal: 9, MaxVal: 10},
	{Name: "Хорошо только", MinVal: 7, MaxVal: 8},
}

// WeightPeriods defines period options for fill operations.
var WeightPeriods = []string{"Полугодие 1", "Полугодие 2", "Весь год", "До текущей даты"}

// ------------------------------------------
// GRADE CALCULATION UTILITIES
// ------------------------------------------

// AverageToGrade converts a floating-point average score to a 10-point grade.
// Uses standard rounding thresholds.
func AverageToGrade(avg float64) int {
	switch {
	case avg >= 9.5:
		return 10
	case avg >= 8.5:
		return 9
	case avg >= 7.5:
		return 8
	case avg >= 6.5:
		return 7
	case avg >= 5.5:
		return 6
	case avg >= 4.5:
		return 5
	case avg >= 3.5:
		return 4
	case avg >= 2.5:
		return 3
	default:
		return 2
	}
}

// ClassAverageToCategory determines the sign category and comment based on class average.
// Returns (diligence, comment).
func ClassAverageToCategory(avg float64) (string, string) {
	switch {
	case avg >= 8.5:
		return "Отличный", "Хорошо и отлично"
	case avg >= 6.5:
		return "Хорошо", "Хорошо и удовлетворительно"
	default:
		return "Удовлетворительный", "Плохо и удовлетворительно"
	}
}

// ParseAverageScore safely parses average score string to float64.
func ParseAverageScore(s string) float64 {
	if s == "" || s == "0.0" || s == "—" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// CalcClassAverage computes the average of all student average scores.
func CalcClassAverage(students []AvgScorer) float64 {
	if len(students) == 0 {
		return 0
	}
	var total float64
	var count int
	for _, s := range students {
		avg := s.GetAverageScore()
		if avg > 0 {
			total += avg
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// AvgScorer interface for objects that have an average score.
type AvgScorer interface {
	GetAverageScore() float64
}

// ------------------------------------------
// BEHAVIOR COMMENT TEMPLATES
// ------------------------------------------

// BehaviorCategory defines the type of behavior note.
type BehaviorCategory string

const (
	BehaviorPraise    BehaviorCategory = "Похвала"
	BehaviorComplaint BehaviorCategory = "Жалоба"
	BehaviorMixed     BehaviorCategory = "Смешанный"
	BehaviorNeutral   BehaviorCategory = "Нейтральный"
)

// BehaviorCategories is the list of all categories for UI selectors.
var BehaviorCategories = []string{
	string(BehaviorPraise),
	string(BehaviorComplaint),
	string(BehaviorMixed),
	string(BehaviorNeutral),
}

// BehaviorTemplates maps a category to a pool of ready-made teacher comments.
var BehaviorTemplates = map[BehaviorCategory][]string{
	BehaviorPraise: {
		"Учится отлично, молодец! Так держать!",
		"Активно работает на уроках, показывает высокие результаты.",
		"Внимателен и старателен, домашние задания выполняет на отлично.",
		"Показывает глубокие знания предмета, активно участвует в обсуждениях.",
		"Очень старательный ученик, всегда готов к урокам.",
		"Проявляет большой интерес к предмету, задаёт правильные вопросы.",
		"Отличная успеваемость, пример для других учеников.",
		"Внимательно слушает объяснения, быстро усваивает материал.",
	},
	BehaviorComplaint: {
		"Не выполняет домашние задания, необходимо усилить контроль.",
		"Нарушает дисциплину на уроках, отвлекает других учеников.",
		"Невнимателен на уроках, часто отвлекается.",
		"Пропускает занятия без уважительной причины.",
		"Не готов к урокам, необходимо больше заниматься дома.",
		"Не участвует в работе на уроке, пассивен.",
		"Систематически не выполняет задания, нужна помощь родителей.",
		"Слабая подготовка к урокам, необходимо дополнительное занятие.",
	},
	BehaviorMixed: {
		"Способный ученик, но недостаточно старается на уроках.",
		"Может учиться лучше, но часто отвлекается и не доделывает задания.",
		"Хорошие знания, но нестабильная успеваемость — нужно больше усилий.",
		"Старается, но результаты пока не соответствуют способностям.",
		"Показывает хорошие результаты, но иногда ленится.",
		"Активен на уроках, но домашние задания выполняет нерегулярно.",
		"Понимает материал, но не всегда применяет знания на практике.",
		"Есть прогресс, но необходимо больше самостоятельной работы.",
	},
	BehaviorNeutral: {
		"Уроки посещает регулярно, ведёт тетрадь.",
		"Задания выполняет в среднем объёме.",
		"Работает на уроках на среднем уровне.",
		"Присутствует на занятиях, участие в работе среднее.",
		"Домашние задания выполняет, но без особого старания.",
		"Материал усваивает на базовом уровне.",
	},
}

// BehaviorToDiligence maps a behavior category to the corresponding diligence mark.
var BehaviorToDiligence = map[BehaviorCategory]string{
	BehaviorPraise:    "Отличный",
	BehaviorComplaint: "Неудовлетворительно",
	BehaviorMixed:     "Удовлетворительный",
	BehaviorNeutral:   "Хорошо",
}

// DiligenceToBehaviorComment returns a default behavior comment for a given diligence mark.
var DiligenceToBehaviorComment = map[string]string{
	"Отличный":            "Превосходная успеваемость и поведение. Молодец!",
	"Хорошо":              "Хорошо учится и ведёт себя на уроках.",
	"Удовлетворительный":  "Удовлетворительная успеваемость, есть над чем работать.",
	"Неудовлетворительно": "Неудовлетворительная успеваемость, требуется внимание родителей.",
}

// ------------------------------------------
// SIGN COMMENT TEMPLATES (for final grades tab)
// ------------------------------------------

// SignCommentTemplates maps category to comments for auto-signing.
var SignCommentTemplates = map[string][]string{
	"Хорошо и отлично": {
		"Высокая успеваемость класса. Рекомендуется продолжать в том же духе.",
		"Отличные результаты по итогам периода. Класс показывает стабильные знания.",
		"Успеваемость класса на высоком уровне. Большинство учеников справляются отлично.",
	},
	"Хорошо и удовлетворительно": {
		"Успеваемость класса на среднем уровне. Есть потенциал для роста.",
		"Результаты неоднородные, часть учеников показывает хорошие знания.",
		"Средний уровень успеваемости. Требуется дополнительная работа с отстающими.",
	},
	"Плохо и удовлетворительно": {
		"Низкая успеваемость класса. Необходимы дополнительные занятия.",
		"Многие ученики не справляются с программой. Требуется внимание родителей.",
		"Успеваемость ниже среднего. Рекомендуется индивидуальная работа.",
	},
}

// RandomSignComment picks a random sign comment for a category.
func RandomSignComment(category string) string {
	templates, ok := SignCommentTemplates[category]
	if !ok || len(templates) == 0 {
		return "Подпись классного руководителя"
	}
	return templates[time.Now().Nanosecond()%len(templates)]
}

// ------------------------------------------
// COLOR HELPERS
// ------------------------------------------

// getDiligenceColor returns color for diligence mark.
func getDiligenceColor(mark string) color.Color {
	switch mark {
	case "Отличный":
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	case "Хорошо":
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	case "Удовлетворительный":
		return color.NRGBA{R: 217, G: 119, B: 6, A: 255}
	case "Неудовлетворительно":
		return color.NRGBA{R: 220, G: 38, B: 38, A: 255}
	default:
		return theme.DisabledColor()
	}
}

// getBehaviorColor returns color for a behavior category.
func getBehaviorColor(cat BehaviorCategory) color.Color {
	switch cat {
	case BehaviorPraise:
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	case BehaviorComplaint:
		return color.NRGBA{R: 220, G: 38, B: 38, A: 255}
	case BehaviorMixed:
		return color.NRGBA{R: 217, G: 119, B: 6, A: 255}
	case BehaviorNeutral:
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	default:
		return theme.DisabledColor()
	}
}

// GradeColor returns a color for a numeric grade.
func GradeColor(grade int) color.Color {
	switch {
	case grade >= 9:
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}  // Green
	case grade >= 7:
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}   // Blue
	case grade >= 5:
		return color.NRGBA{R: 217, G: 119, B: 6, A: 255}   // Orange
	default:
		return color.NRGBA{R: 220, G: 38, B: 38, A: 255}   // Red
	}
}

// ------------------------------------------
// UI HELPERS
// ------------------------------------------

// MakeFixedHeader creates a fixed header bar.
func MakeFixedHeader(content fyne.CanvasObject) *fyne.Container {
	bg := canvas.NewRectangle(color.NRGBA{R: 245, G: 245, B: 245, A: 255})
	return container.NewStack(bg, container.NewPadded(content))
}

// FormatSignedStatus returns colored text for signed status.
func FormatSignedStatus(signed bool) (string, color.Color) {
	if signed {
		return "Подписано", color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	}
	return "Не подписано", color.NRGBA{R: 220, G: 38, B: 38, A: 255}
}

// FormatStudentName returns "LastName FirstName M." format.
func FormatStudentName(last, first, middle string) string {
	name := fmt.Sprintf("%s %s", last, first)
	if middle != "" {
		name += " " + string([]rune(middle)[:1]) + "."
	}
	return name
}

// ------------------------------------------
// TAPPABLE OVERLAY (clickable transparent area)
// ------------------------------------------

// tapOverlay is a transparent widget that handles tap events.
type tapOverlay struct {
	widget.BaseWidget
	onTap func()
}

func newTapOverlay(onTap func()) *tapOverlay {
	t := &tapOverlay{onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 0}))
}

func (t *tapOverlay) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tapOverlay) TappedSecondary(*fyne.PointEvent) {}
