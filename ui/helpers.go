package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// Diligence marks — these match the edonish.tj API enum values.
var DiligenceMarks = []string{"Отличный", "Хорошо", "Удовлетворительный", "Неудовлетворительный"}

// Grade combinations for random fill
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

// Weight period options
var WeightPeriods = []string{"Полугодие 1", "Полугодие 2", "Весь год", "До текущей даты"}

// Topic templates for sequential fill — keyed by quality level names
var TopicTemplates = map[string][]string{
	"Отличный": {
		"Повторение материала",
		"Решение задач повышенной сложности",
		"Контрольная работа",
		"Практическая работа",
		"Обобщение и систематизация знаний",
	},
	"Хорошо": {
		"Изучение нового материала",
		"Закрепление пройденного",
		"Самостоятельная работа",
		"Работа с упражнениями",
		"Проверка знаний",
	},
	"Удовлетворительно": {
		"Объяснение новой темы",
		"Работа с учебником",
		"Устный опрос",
		"Комбинированный урок",
		"Беседа по теме",
	},
	"Неудовлетворительно": {
		"Повторение",
		"Подготовка к контрольной",
		"Работа над ошибками",
		"Консультация",
		"Резервный урок",
	},
}

// ------------------------------------------
// BEHAVIOR COMMENT TEMPLATES
// (Teacher/parent notes in real school diaries)
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
// These are used like real diary entries: teacher writes a note about the student,
// parent reads it and signs.
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
// This determines what diligence mark is set alongside the behavior comment.
var BehaviorToDiligence = map[BehaviorCategory]string{
	BehaviorPraise:    "Отличный",
	BehaviorComplaint: "Неудовлетворительно",
	BehaviorMixed:     "Удовлетворительный",
	BehaviorNeutral:   "Хорошо",
}

// DiligenceToBehaviorComment returns a default behavior comment for a given diligence mark.
// This is used when filling diaries with only a diligence mark selected.
var DiligenceToBehaviorComment = map[string]string{
	"Отличный":            "Превосходная успеваемость и поведение. Молодец!",
	"Хорошо":              "Хорошо учится и ведёт себя на уроках.",
	"Удовлетворительный":  "Удовлетворительная успеваемость, есть над чем работать.",
	"Неудовлетворительно": "Неудовлетворительная успеваемость, требуется внимание родителей.",
}

// getDiligenceColor returns color for diligence mark
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

// MakeFixedHeader creates a fixed header bar
func MakeFixedHeader(content fyne.CanvasObject) *fyne.Container {
	bg := canvas.NewRectangle(color.NRGBA{R: 245, G: 245, B: 245, A: 255})
	return container.NewStack(bg, container.NewPadded(content))
}

// FormatSignedStatus returns colored text for signed status
func FormatSignedStatus(signed bool) (string, color.Color) {
	if signed {
		return "Подписано", color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	}
	return "Не подписано", color.NRGBA{R: 220, G: 38, B: 38, A: 255}
}

// ------------------------------------------
// TAPPABLE OVERLAY (clickable transparent area)
// ------------------------------------------

// tapOverlay is a transparent widget that handles tap events.
type tapOverlay struct {
	widget.BaseWidget
	onTap func()
}

// newTapOverlay creates a transparent tappable area.
func newTapOverlay(onTap func()) *tapOverlay {
	t := &tapOverlay{onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer returns a minimal renderer (empty/transparent).
func (t *tapOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 0}))
}

// Tapped handles a tap event by calling the onTap callback.
func (t *tapOverlay) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

// TappedSecondary is a no-op for secondary taps.
func (t *tapOverlay) TappedSecondary(*fyne.PointEvent) {}
