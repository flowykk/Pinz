package services

// Limits from task.txt
const (
	MaxNameLength        = 100
	MaxDescriptionLength = 5000
)

var (
	categories = map[string]bool{
		"Отпуск": true, "Командировка": true, "Выходные": true,
		"Активный отдых": true, "Образование": true, "Другое": true,
	}
	seasons = map[string]bool{
		"Зима": true, "Весна": true, "Лето": true, "Осень": true,
	}
	privacyLevels = map[string]bool{
		"Public": true, "Private": true, "Restricted": true,
	}
)

func validateCategory(c string) bool     { return categories[c] }
func validateSeason(s string) bool       { return seasons[s] }
func validatePrivacyLevel(p string) bool { return privacyLevels[p] }
