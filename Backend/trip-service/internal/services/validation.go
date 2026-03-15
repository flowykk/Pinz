package services

// Limits from task.txt (4.1.2)
const (
	MaxNameLength        = 100
	MaxDescriptionLength = 5000
	MaxMediaPerTrip      = 500
	MaxVideosPerTrip     = 50
	MaxTagsPerPin        = 10
	MaxTagLength         = 15
)

// Clustering params (task 3.9: ~50-100m, time diff <10 min)
const (
	ClusterRadiusMeters = 75
	TimeClusterMinutes  = 10
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
