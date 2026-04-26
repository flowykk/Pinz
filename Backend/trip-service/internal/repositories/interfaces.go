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

// TripEventPublisher — методы RedisRepository, используемые TripService (eventRepo может быть nil-интерфейсом).
type TripEventPublisher interface {
	PublishTripEvent(ctx context.Context, eventType string, tripID, userID string) error
	AddMLTask(ctx context.Context, tripID string) error
	// add-media flow: помечает контекст трипа, чтобы worker пропустил авто-теги для существующих пинов (ТЗ 5.3.4).
	SetMLContext(ctx context.Context, tripID, flow string, newPinIDs []string, ttl time.Duration) error
	AddMLTaskWithFlow(ctx context.Context, tripID, flow string, newPinIDs []string) error
	// statistics-service consumer — публикация в pinz:stats:events.
	PublishStatsEvent(ctx context.Context, eventType, tripID string, userIDs []string, payload map[string]any) error
	// Fan-out WS-события в pinz:trip:{id}:events и pinz:user:{uid}:events.
	PublishTripEventWS(ctx context.Context, tripID string, userIDs []string, eventType string, payload map[string]interface{}) error
	// DeleteTripEventStream удаляет per-trip WS-stream (вызывается из DeleteTrip).
	DeleteTripEventStream(ctx context.Context, tripID string) error
	// PublishPrivacyEvent публикует событие изменения per-user приватности
	// в pinz:trip:privacy:events для асинхронного fallback-пересчёта воркером.
	PublishPrivacyEvent(ctx context.Context, objectType, objectID, tripID, userID, privacyLevel string) error
}

type MediaRepositoryInterface interface {
	Create(m *models.Media) error
	// CommitInSession — атомарный INSERT в рамках add-media сессии с advisory
	// lock по session_id и проверкой лимитов (ErrMediaLimitExceeded /
	// ErrVideoLimitExceeded). Возвращает totalAfter/videosAfter для ответа
	// commit-upload.
	CommitInSession(ctx context.Context, m *models.Media, sessionID string, maxMedia, maxVideos int) (totalAfter, videosAfter int, err error)
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
	// ListByPinAdditionSession — media активной pin-add-media-сессии (pin_id=NULL,
	// pin_addition_session_id=$1). Используется в Process.
	ListByPinAdditionSession(sessionID string) ([]*models.Media, error)
	// DeleteOrphanByPinAdditionSession — orphan-cleanup при Cancel.
	DeleteOrphanByPinAdditionSession(sessionID string) ([]string, error)
	// ListByPinCreationSession — media активной pin-creation сессии (pin_id=NULL,
	// pin_creation_session_id=$1). Используется в Process / Finalize.
	ListByPinCreationSession(sessionID string) ([]*models.Media, error)
	// DeleteOrphanByPinCreationSession — orphan-cleanup при CancelPinCreation.
	DeleteOrphanByPinCreationSession(sessionID string) ([]string, error)
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
	ListRecommendationCandidates(locationID int, epsMeters float64) ([]*RecommendationPinCandidate, error)
}

// PinHiddenRepositoryInterface — управление soft-delete-for-self записями (ТЗ 4.5.2).
type PinHiddenRepositoryInterface interface {
	HidePinForUser(pinID, userID string) error
	ListHiddenPinIDsForUser(tripID, userID string) ([]string, error)
	IsHidden(pinID, userID string) (bool, error)
}

// PinMediaAdditionSessionRepositoryInterface — sessioned add-media-в-пин (ТЗ 4.2.2 + 4.12-4.14).
type PinMediaAdditionSessionRepositoryInterface interface {
	Create(ctx context.Context, tripID, pinID, userID string) (sessionID string, err error)
	GetByID(ctx context.Context, sessionID string) (*models.PinMediaAdditionSession, error)
	GetActiveForPin(ctx context.Context, pinID string) (*models.PinMediaAdditionSession, error)
	Touch(ctx context.Context, sessionID string) error
	SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error
	Close(ctx context.Context, sessionID, reason string) error
}

// PinCreationSessionRepositoryInterface — sessioned создание одиночного пина (ТЗ 4.1, 4.6-4.11).
type PinCreationSessionRepositoryInterface interface {
	Create(ctx context.Context, tripID, userID string) (sessionID string, err error)
	GetByID(ctx context.Context, sessionID string) (*models.PinCreationSession, error)
	GetActiveForTrip(ctx context.Context, tripID string) (*models.PinCreationSession, error)
	Touch(ctx context.Context, sessionID string) error
	SetDraftSnapshot(ctx context.Context, sessionID string, snapshot []byte) error
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
}

type FavouriteRepositoryInterface interface {
	Add(userID, tripID string) error
	Remove(userID, tripID string) error
	HasFavourite(userID, tripID string) (bool, error)
	HasFavouritesByOtherUsers(tripID, excludeUserID string) (bool, error)
	ListTripIDsByUserID(userID string, limit, offset int32) ([]string, error)
}

type GeoRegistryRepositoryInterface interface {
	EnsureLocationByName(ctx context.Context, countryName, cityName string) (countryID, cityID *int, displayName string, err error)
	UpsertTripLocations(ctx context.Context, tripID string, locationIDs []int) error
	// Используется рекомендательной системой (ТЗ 9): резолв страны/города по точному имени.
	FindCountryByName(ctx context.Context, name string) (int, error)
	FindCityByName(ctx context.Context, name string) (int, error)
	GetLocations(ctx context.Context, ids []int) ([]GeoLocation, error)
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
