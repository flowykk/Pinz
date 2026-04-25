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
	SetSimilarGroupID(mediaIDs []string, groupID string) error
	CountByTripID(tripID string) (total int, videos int, err error)
	ClusterIDsByLocation(tripID string, radiusMeters float64) (map[string]int, error)
	ListByPinID(pinID string) ([]*models.Media, error)
	TopMediaByTripIDs(tripIDs []string, limitPerTrip int) (map[string][]*FeedMedia, error)
	PickRandomForBattle(tripID string, limit int) ([]*models.Media, error)
	IncrementBattleRating(mediaID string) (int32, error)
	ListWithPositiveBattleRating(tripID string) ([]*models.Media, error)
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
	Update(p *models.Pin) error
	Delete(id string) error
	DeleteByTripID(tripID string) error
	ListPublishedPinsByTripIDs(tripIDs []string) (map[string][]*FeedPin, error)
	SearchByUserID(userID, query string, limit, offset int32) ([]*models.Pin, error)
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
}
