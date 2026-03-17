package services

import (
	"fmt"
	"path"
	"strings"

	pb "pinz/backend/trip-service/pkg/proto"
)

// Limits for trip and media validation.
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
		"Отпуск":         true,
		"Командировка":   true,
		"Выходные":       true,
		"Активный отдых": true,
		"Образование":    true,
		"Другое":         true,
	}
	pinCategories = map[string]bool{
		"Достопримечательность": true,
		"Природа":       true,
		"Отдых":         true,
		"Жилье":         true,
		"Еда и напитки": true,
		"Шопинг":        true,
		"Транспорт":     true,
		"Развлечение":   true,
		"Мероприятие":   true,
		"Спорт":         true,
		"Рабочее место": true,
		"Другое":        true,
	}
	seasons = map[string]bool{
		"Зима":  true,
		"Весна": true,
		"Лето":  true,
		"Осень": true,
	}
	privacyLevels = map[string]bool{
		"Public":     true,
		"Private":    true,
		"Restricted": true,
	}

	// Allowed media extensions by logical type (image/video).
	allowedImageExt = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".heic": true,
	}
	allowedVideoExt = map[string]bool{
		".mov": true,
		".mp4": true,
	}

	allowedContentTypes = map[string]bool{
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
		"image/heic":      true,
		"video/mp4":       true,
		"video/quicktime": true,
	}
)

func validateCategory(c string) bool     { return categories[c] }
func validatePinCategory(c string) bool  { return pinCategories[c] }
func validateSeason(s string) bool       { return seasons[s] }
func validatePrivacyLevel(p string) bool { return privacyLevels[p] }

func validateContentType(ct string) bool { return allowedContentTypes[ct] }

// validateMediaMeta enforces basic constraints for incoming media:
// - type must be image or video
// - file extension must match allowed set for the type.
// Size and duration limits are not validated here because they are not present
// in MediaMeta; они должны проверяться на стороне клиента или Media Service.
func validateMediaMeta(m *pb.MediaMeta) error {
	mt := m.GetMediaType()
	if mt != "image" && mt != "video" {
		return fmt.Errorf("media_type must be 'image' or 'video'")
	}
	s3Key := strings.ToLower(strings.TrimSpace(m.GetS3Key()))
	if s3Key == "" {
		return fmt.Errorf("s3_key is required")
	}
	ext := strings.ToLower(path.Ext(s3Key))
	if mt == "image" {
		if !allowedImageExt[ext] {
			return fmt.Errorf("invalid image extension %s, allowed: .jpg, .jpeg, .png, .heic", ext)
		}
	} else {
		if !allowedVideoExt[ext] {
			return fmt.Errorf("invalid video extension %s, allowed: .mov, .mp4", ext)
		}
	}
	return nil
}
