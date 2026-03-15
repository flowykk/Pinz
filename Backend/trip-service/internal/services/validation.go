package services

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 5000
)

var (
	Categories = map[string]bool{
		"Отпуск": true, "Командировка": true, "Выходные": true,
		"Активный отдых": true, "Образование": true, "Другое": true,
	}
	Seasons = map[string]bool{
		"Зима": true, "Весна": true, "Лето": true, "Осень": true,
	}
	PrivacyLevels = map[string]bool{
		"Public": true, "Private": true, "Restricted": true,
	}
)

func validateCategory(c string) bool { return Categories[c] }
func validateSeason(s string) bool   { return Seasons[s] }
func validatePrivacyLevel(p string) bool {
	if p == "" {
		return true
	}
	return PrivacyLevels[p]
}
