package ui

import (
        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"
        "image/color"
)

// Diligence marks
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

// Topic templates for random fill
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
        "Плохо": {
                "Повторение",
                "Подготовка к контрольной",
                "Работа над ошибками",
                "Консультация",
                "Резервный урок",
        },
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
        case "Неудовлетворительный":
                return color.NRGBA{R: 220, G: 38, B: 38, A: 255}
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
