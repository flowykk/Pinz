package repositories

import (
	"context"
	"time"

	"pinz/backend/trip-service/internal/models"
)

// Интерфейсы для DI и автогенерируемых моков (см. internal/mocks).

type TripRepositoryInterface interface {
	Create(t *models.Trip) error
	GetByID(id string) (*models.Trip, error)
	ListByUserID(userID string, limit, offset int32) ([]*models.Trip, error)
	ListSummariesByUserID(userID string) ([]*TripSummary, error)
	Update(t *models.Trip) error
	Delete(id string) error
	SetStatus(tripID, status string) error
	SetSoftDeleted(tripID string) error
	UpdateCoverURL(tripID, s3Key string) error
	SetPrivacyLevel(tripID, level string) error
	ListFeed(limit, offset int32, category, season string, locationIDs []int, sortBy string) ([]*models.Trip, error)
	// выборки для notification-service scheduler'а.
	ListAnniversaryCandidates(today int64) ([]*NotificationTripCandidate, error)
	ListEndedMonthAgoCandidates(today int64) ([]*NotificationTripCandidate, error)
	ListAbandonedGenerated(minAge time.Duration, limit int) ([]string, error)
}

type TripParticipantRepositoryInterface interface {
	Add(p *models.TripParticipant) error
	GetByTripID(tripID string) ([]*models.TripParticipant, error)
	IsParticipant(tripID, userID string) (bool, error)
	IsAdmin(tripID, userID string) (bool, error)
	Remove(tripID, userID string) error
	RemoveAllByTripID(tripID string) error
	SetAdmin(tripID, userID string) error
}

// AddMediaSessionRepositoryInterface — интерфейс add-media session repo для DI и моков.
type AddMediaSessionRepositoryInterface interface {
	Create(ctx context.Context, tripID string, existingMediaIDs []string) (sessionID string, err error)
	GetExistingMediaIDs(ctx context.Context, sessionID string) ([]string, string, error)
	Exists(ctx context.Context, tripID, sessionID string) (bool, error)
	GetActive(ctx context.Context, tripID string) (*models.AddMediaSession, error)
	SetInitiator(ctx context.Context, sessionID, userID string, at time.Time) error
	Touch(ctx context.Context, sessionID string, at time.Time) error
	Close(ctx context.Context, sessionID, reason string, at time.Time) (tripID string, err error)
	ListAbandoned(ctx context.Context, threshold time.Time) ([]AbandonedSession, error)
}

type InvitationLinkRepositoryInterface interface {
	Create(link *models.InvitationLink) error
	GetByToken(token string) (*models.InvitationLink, error)
}

type TripSettingsRepositoryInterface interface {
	EnsureDefaultSettings(tripID, userID string) error
	UpdateNotifications(tripID, userID string, enabled bool) error
	// выборка notifications_enabled для списка участников трипа.
	GetByTripAndUsers(tripID string, userIDs []string) (map[string]bool, error)
}

// GeoRequestPin — координаты пина для PIN_LOCATIONS_REQUESTED.
type GeoRequestPin struct {
	PinID     string  `json:"pin_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TripEventPublisher — методы RedisRepository, используемые TripService (eventRepo может быть nil-интерфейсом).
type TripEventPublisher interface {
	PublishTripEvent(ctx context.Context, eventType string, tripID, userID string) error
	AddMLTask(ctx context.Context, tripID string) error
	// add-media flow: помечает контекст трипа, чтобы worker пропустил авто-теги для существующих пинов (ТЗ 5.3.4).
	SetMLContext(ctx context.Context, tripID, flow string, newPinIDs []string, ttl time.Duration) error
	AddMLTaskWithFlow(ctx context.Context, tripID, flow string, newPinIDs []string) error
	// statistics-service consumer — публикация в pinz:stats:events (LIKE/DISLIKE/TRIP_DELETED/BATTLE_FINISHED).
	PublishStatsEvent(ctx context.Context, eventType, tripID string, userIDs []string, payload map[string]any) error
	// PublishGeoRequest публикует PIN_LOCATIONS_REQUESTED — обратное направление
	// репликации: statistics-service вызовет BigDataCloud, заполнит master
	// geo_registry/trip_locations и пришлёт PIN_LOCATIONS_RESOLVED обратно
	// в pinz:trip:geo_events. См. vkr.txt §2.5.4.
	PublishGeoRequest(ctx context.Context, tripID string, pins []GeoRequestPin) error
	PublishTripEventWS(ctx context.Context, tripID string, eventType string, payload map[string]interface{}) error
	// DeleteTripEventStream удаляет per-trip WS-stream (вызывается из DeleteTrip).
	DeleteTripEventStream(ctx context.Context, tripID string) error
	// PublishPrivacyEvent публикует событие изменения per-user приватности
	// в pinz:trip:privacy:events для асинхронного fallback-пересчёта воркером.
	PublishPrivacyEvent(ctx context.Context, objectType, objectID, tripID, userID, privacyLevel string) error
	AddPinUploadTask(ctx context.Context, tripID, sessionID string, targetPinID *string, initiatorUserID string) error
}

type MediaRepositoryInterface interface {
	Create(m *models.Media) error
	// CommitInSession — атомарный INSERT в рамках add-media сессии с advisory
	// lock по session_id и проверкой лимитов (ErrMediaLimitExceeded /
	// ErrVideoLimitExceeded). Возвращает totalAfter/videosAfter для ответа
	// commit-upload.
	CommitInSession(ctx context.Context, m *models.Media, sessionID string, maxMedia, maxVideos int) (totalAfter, videosAfter int, err error)
	// CommitInUploadSession — то же, но для pin_upload_sessions.
	CommitInUploadSession(ctx context.Context, m *models.Media, sessionID string, maxMedia, maxVideos int) (totalAfter, videosAfter int, err error)
	GetByID(id string) (*models.Media, error)
	ListByTripID(tripID string) ([]*models.Media, error)
	UpdatePinID(mediaID, pinID string) error
	UpdatePinIDByIDs(mediaIDs []string, pinID string) error
	DeleteByIDs(ids []string) error
	// удалить неприкреплённые медиа текущей add-media сессии.
	DeleteOrphanSessionMedia(tripID string, existingMediaIDs []string) ([]string, error)
	// DeleteByPinID — full delete пина (ТЗ 4.5.1): удаляет все media пина и
	// возвращает s3_keys для S3 cleanup.
	DeleteByPinID(pinID string) ([]string, error)
	// ListByUploadSession — media унифицированной pin-upload сессии (pin_id=NULL,
	// upload_session_id=$1).
	ListByUploadSession(sessionID string) ([]*models.Media, error)
	// DeleteOrphanByUploadSession — orphan-cleanup при CancelPinUpload.
	DeleteOrphanByUploadSession(sessionID string) ([]string, error)
	SetSimilarGroupID(mediaIDs []string, groupID string) error
	MarkNSFW(mediaIDs []string) error
	CountByTripID(tripID string) (total int, videos int, err error)
	ClusterIDsByLocation(tripID string, radiusMeters float64) (map[string]int, error)
	ListByPinID(pinID string) ([]*models.Media, error)
	TopMediaByTripIDs(tripIDs []string, limitPerTrip int) (map[string][]*FeedMedia, error)
	TopMediaByPinIDs(pinIDs []string, limitPerPin int) (map[string][]*FeedMedia, error)
	PickRandomForBattle(tripID string, limit int) ([]*models.Media, error)
	IncrementBattleRating(mediaID string) (int32, error)
	ListWithPositiveBattleRating(tripID string) ([]*models.Media, error)
	SetPrivacyLevel(mediaID, level string) error
}

// MediaBattleRepositoryInterface — сессии фотобатла (ТЗ 8.1).
type MediaBattleRepositoryInterface interface {
	Create(b *models.MediaBattle) error
	GetByID(id string) (*models.MediaBattle, error)
	SetWinner(battleID, winnerMediaID string) error
}

type PinRepositoryInterface interface {
	Create(p *models.Pin) error
	GetByID(id string) (*models.Pin, error)
	ListByTripID(tripID string) ([]*models.Pin, error)
	// ListByTripIDExcludingHidden — список пинов за вычетом скрытых для userID
	// через pin_hidden_by_user (ТЗ 4.5.2 soft-delete-for-self).
	ListByTripIDExcludingHidden(tripID, userID string) ([]*models.Pin, error)
	Update(p *models.Pin) error
	Delete(id string) error
	DeleteByTripID(tripID string) error
	SetPrivacyLevel(pinID, level string) error
	// IncMediaCount атомарно увеличивает/уменьшает media_count на пине.
	// Используется AddMediaToPin (delta=+N после finalize) и RemoveMediaFromPin (delta=-1).
	IncMediaCount(pinID string, delta int) error
	ListPublishedPinsByTripIDs(tripIDs []string) (map[string][]*FeedPin, error)
	SearchByUserID(userID, query string, limit, offset int32) ([]*models.Pin, error)
	// ListRecommendationCandidates — выборка кандидатов для рекомендательной системы (ТЗ 9):
	// топ-50 опубликованных трипов региона за 2 года, их пины с координатами и
	// cluster_id из ST_ClusterDBSCAN (партиция по category, eps в метрах).
	ListRecommendationCandidates(locationID int, epsMeters float64, tripCategory, tripSeason string) ([]*RecommendationPinCandidate, error)
	GetByIDs(ids []string) ([]*models.Pin, error)
}

// PinHiddenRepositoryInterface — управление soft-delete-for-self записями (ТЗ 4.5.2).
type PinHiddenRepositoryInterface interface {
	HidePinForUser(pinID, userID string) error
	ListHiddenPinIDsForUser(tripID, userID string) ([]string, error)
	IsHidden(pinID, userID string) (bool, error)
}

// PinUploadSessionRepositoryInterface — pin upload session (creation/addition).
type PinUploadSessionRepositoryInterface interface {
	Create(ctx context.Context, tripID string, targetPinID *string, userID string) (sessionID string, err error)
	GetByID(ctx context.Context, sessionID string) (*models.PinUploadSession, error)
	GetActiveCreationForTrip(ctx context.Context, tripID string) (*models.PinUploadSession, error)
	GetActiveAdditionForPin(ctx context.Context, pinID string) (*models.PinUploadSession, error)
	Touch(ctx context.Context, sessionID string) error
	SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error
	SetProcessingStatus(ctx context.Context, sessionID, expected, next string) error
	Close(ctx context.Context, sessionID, reason string) error
}

type TagRepositoryInterface interface {
	SetForPin(tripID, pinID string, tags []string) error
	Add(t *models.Tag) error
	GetByPinID(pinID string) ([]string, error)
	GetByTripID(tripID string) (map[string][]string, error)
	DeleteForPin(pinID string) error
}

type SocialRepositoryInterface interface {
	// Возвращает oldReaction ("", "Like", "Dislike") — для публикации статистических событий.
	SetReaction(userID, tripID, reaction string) (oldReaction string, err error)
	GetReaction(userID, tripID string) (string, error)
	// GetReactionsByUserAndTrips bulk-фетч реакций пользователя по списку trip_id.
	// В мапе только те трипы, по которым у пользователя есть запись; reaction — "Like" или "Dislike".
	GetReactionsByUserAndTrips(userID string, tripIDs []string) (map[string]string, error)
}

type FavouriteRepositoryInterface interface {
	Add(userID, tripID string) error
	Remove(userID, tripID string) error
	HasFavourite(userID, tripID string) (bool, error)
	HasFavouritesByOtherUsers(tripID, excludeUserID string) (bool, error)
	ListTripIDsByUserID(userID string, limit, offset int32) ([]string, error)
	// FavouritesByUserAndTrips возвращает множество tripIDs из переданного списка,
	// которые сохранены пользователем. Для O(1) проверки в ленте.
	FavouritesByUserAndTrips(userID string, tripIDs []string) (map[string]struct{}, error)
}

type GeoRegistryRepositoryInterface interface {
	// MirrorByID идемпотентно зеркалит запись master geo_registry в локальную
	// реплику trip-service (по id из statistics-service).
	MirrorByID(ctx context.Context, row GeoLocation) error
	// UpsertTripLocations пишет факт «trip T содержит локацию L» в локальную
	// реплику trip_locations (нужно для read-heavy фильтрации ленты).
	UpsertTripLocations(ctx context.Context, tripID string, locationIDs []int) error
	// Используется рекомендательной системой (ТЗ 9): резолв страны/города по точному имени.
	FindCountryByName(ctx context.Context, name string) (int, error)
	FindCityByName(ctx context.Context, name string) (int, error)
	GetLocations(ctx context.Context, ids []int) ([]GeoLocation, error)
	TripIDsAtLocation(ctx context.Context, locationID int, tripIDs []string) (map[string]struct{}, error)
}

// Per-user приватность: каждый участник выставляет свой уровень для trip/pin/media.
// Эффективный privacy_level пересчитывается из этих записей через AggregatePrivacyLevel.
type TripPrivacyRepositoryInterface interface {
	Upsert(ctx context.Context, tripID, userID, privacyLevel string) error
	GetByTripID(ctx context.Context, tripID string) ([]PrivacyEntry, error)
}

type PinPrivacyRepositoryInterface interface {
	Upsert(ctx context.Context, pinID, userID, privacyLevel string) error
	GetByPinID(ctx context.Context, pinID string) ([]PrivacyEntry, error)
}

type MediaPrivacyRepositoryInterface interface {
	Upsert(ctx context.Context, mediaID, userID, privacyLevel string) error
	GetByMediaID(ctx context.Context, mediaID string) ([]PrivacyEntry, error)
}

type RecommendationSnapshotRepositoryInterface interface {
	Save(ctx context.Context, token string, snap *RecommendationSnapshot, ttl time.Duration) error
	Get(ctx context.Context, token string) (*RecommendationSnapshot, bool, error)
	Delete(ctx context.Context, token string) error
}
