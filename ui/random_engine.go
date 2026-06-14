package ui

import (
	"math/rand"
	"strings"
)

// Note: In Go 1.20+, the global random generator is automatically
// seeded with a random value. No explicit rand.Seed() or init() needed.

// RandomDiligenceMark picks a random diligence mark from the pool.
// combo can be: "Отличный", "Хорошо", "Удовлетворительный", "Неудовлетворительно"
// or "random" for fully random.
func RandomDiligenceMark(combo string) string {
	if combo == "random" || combo == "" {
		return DiligenceMarks[rand.Intn(len(DiligenceMarks))]
	}
	return combo
}

// RandomGradeInRange returns a random grade between min and max (inclusive).
func RandomGradeInRange(minVal, maxVal int) int {
	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}
	return minVal + rand.Intn(maxVal-minVal+1)
}

// RandomGradeForCombo returns a random grade for a named combo.
func RandomGradeForCombo(comboName string) int {
	for _, c := range GradeCombos {
		if c.Name == comboName {
			return RandomGradeInRange(c.MinVal, c.MaxVal)
		}
	}
	// Default: Good and Excellent
	return RandomGradeInRange(7, 10)
}

// RandomTopicForDiligence picks a random topic template for a given diligence level.
func RandomTopicForDiligence(diligence string) string {
	topics, ok := TopicTemplates[diligence]
	if !ok || len(topics) == 0 {
		for _, t := range TopicTemplates {
			topics = append(topics, t...)
		}
	}
	if len(topics) == 0 {
		return "Урок"
	}
	return topics[rand.Intn(len(topics))]
}

// SequentialTopicForDiligence picks a topic template sequentially for a given diligence level.
// idx is the sequential index; it cycles through the pool using modulo.
func SequentialTopicForDiligence(diligence string, idx int) string {
	topics, ok := TopicTemplates[diligence]
	if !ok || len(topics) == 0 {
		for _, t := range TopicTemplates {
			topics = append(topics, t...)
		}
	}
	if len(topics) == 0 {
		return "Урок"
	}
	return topics[idx%len(topics)]
}

// RandomDiligenceCombo picks a random diligence combination.
// Returns one of: "Отличный", "Хорошо", "Удовлетворительный", "Неудовлетворительно"
func RandomDiligenceCombo() string {
	weights := []int{35, 35, 20, 10} // Отличный, Хорошо, Удовлетворительный, Неудовлетворительно
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return DiligenceMarks[i]
		}
	}
	return DiligenceMarks[1] // default "Хорошо"
}

// ------------------------------------------
// BEHAVIOR COMMENT GENERATION
// ------------------------------------------

// RandomBehaviorComment picks a random behavior comment for a given category.
func RandomBehaviorComment(category BehaviorCategory) string {
	templates, ok := BehaviorTemplates[category]
	if !ok || len(templates) == 0 {
		return "Комментарий учителя"
	}
	return templates[rand.Intn(len(templates))]
}

// SequentialBehaviorComment picks a behavior comment sequentially for a given category.
// idx cycles through the pool using modulo.
func SequentialBehaviorComment(category BehaviorCategory, idx int) string {
	templates, ok := BehaviorTemplates[category]
	if !ok || len(templates) == 0 {
		return "Комментарий учителя"
	}
	return templates[idx%len(templates)]
}

// RandomBehaviorCategory picks a random behavior category.
// Weighted towards praise and mixed.
func RandomBehaviorCategory() BehaviorCategory {
	weights := []int{40, 15, 30, 15} // Прахвала, Жалоба, Смешанный, Нейтральный
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	cumulative := 0
	cats := []BehaviorCategory{BehaviorPraise, BehaviorComplaint, BehaviorMixed, BehaviorNeutral}
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return cats[i]
		}
	}
	return BehaviorNeutral
}

// GenerateBehaviorWithDiligence returns a behavior comment and corresponding diligence mark
// for a given category. This is the main function used for batch diary filling.
func GenerateBehaviorWithDiligence(category BehaviorCategory, idx int) (comment string, diligence string) {
	comment = SequentialBehaviorComment(category, idx)
	diligence = BehaviorToDiligence[category]
	return
}

// ShouldFillDate determines if a date should be filled based on the weight period.
func ShouldFillDate(period, quarterName, currentDate, assignmentDate string) bool {
	switch period {
	case "Полугодие 1":
		return quarterName == "Четверть 1" || quarterName == "Четверть 2" || quarterName == "Полугодие 1"
	case "Полугодие 2":
		return quarterName == "Четверть 3" || quarterName == "Четверть 4" || quarterName == "Полугодие 2"
	case "До текущей даты":
		return assignmentDate <= currentDate
	case "Весь год":
		fallthrough
	default:
		return true
	}
}

// GenerateTopicLine generates a complete topic line with subject context.
func GenerateTopicLine(subject, topicBase string, lineNum int) string {
	if strings.TrimSpace(topicBase) == "" {
		return topicBase
	}
	return topicBase
}

// BatchRandomGrades generates a map of studentID -> random grade for a given combo.
func BatchRandomGrades(studentIDs []int, comboName string) map[int]int {
	result := make(map[int]int)
	for _, id := range studentIDs {
		result[id] = RandomGradeForCombo(comboName)
	}
	return result
}

// PickRandomComboName returns a random combo name from GradeCombos.
func PickRandomComboName() string {
	if len(GradeCombos) == 0 {
		return "Хорошо и Отлично"
	}
	return GradeCombos[rand.Intn(len(GradeCombos))].Name
}
