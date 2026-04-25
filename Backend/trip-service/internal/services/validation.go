package services

import "fmt"

// Limits for trip and media validation.
const (
	MaxNameLength = 100
	MaxDescriptionLength = 5000
	MaxMediaPerTrip = 500
	MaxVideosPerTrip = 50
	MaxTagsPerPin = 10
	MaxTagLength = 15
)

// Clustering params (task 3.9: ~50-100m, time diff <10 min)
const (
	ClusterRadiusMeters = 75
	TimeClusterMinutes = 10
)

var (
	categories = map[string]bool{
		"Отпуск": true,
		"Командировка": true,
		"Выходные": true,
		"Активный отдых": true,
		"Образование": true,
		"Другое": true,
	}
	seasons = map[string]bool{
		"Зима": true,
		"Весна": true,
		"Лето": true,
		"Осень": true,
	}
	privacyLevels = map[string]bool{
		"Public": true,
		"Private": true,
		"Restricted": true,
	}
	// userPrivacyLevels — уровни, доступные пользователю. Restricted (ТЗ 6.3 «постоянно приватный») ставит только система.
	userPrivacyLevels = map[string]bool{
		"Public": true,
		"Private": true,
	}
)

// Pin categories per ТЗ 2.2.4.
var pinCategories = map[string]bool{
	"Достопримечательность": true,
	"Природа": true,
	"Отдых": true,
	"Жилье": true,
	"Еда и напитки": true,
	"Шопинг": true,
	"Транспорт": true,
	"Развлечение": true,
	"Мероприятие": true,
	"Спорт": true,
	"Рабочее место": true,
	"Другое": true,
}

const PinCategoryDefault = "Другое"

func ValidatePinCategory(c string) string {
	if pinCategories[c] {
		return c
	}
	return PinCategoryDefault
}

var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg": true,
	"image/png": true,
	"image/heic": true,
	"video/mp4": true,
	"video/quicktime": true,
}

func validateCategory(c string) bool { return categories[c] }
func validateSeason(s string) bool { return seasons[s] }
func validatePrivacyLevel(p string) bool { return privacyLevels[p] }
func validateUserPrivacyLevel(p string) bool { return userPrivacyLevels[p] }
func validateContentType(ct string) bool { return allowedContentTypes[ct] }

func validateTags(tags []string) error {
	if len(tags) > MaxTagsPerPin {
		return fmt.Errorf("at most %d tags per pin", MaxTagsPerPin)
	}
	for _, t := range tags {
		if len(t) > MaxTagLength {
			return fmt.Errorf("tag must be at most %d characters", MaxTagLength)
		}
	}
	return nil
}
