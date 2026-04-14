package repositories

import (
	"context"

	"pinz/backend/trip-service/internal/models"
)

// Интерфейсы для DI и автогенерируемых моков (см. internal/mocks).

type TripRepositoryInterface interface {
	Create(t *models.Trip) error
	GetByID(id string) (*models.Trip, error)
	ListByUserID(userID string, limit, offset int32) ([]*models.Trip, error)
	Update(t *models.Trip) error
	Delete(id string) error
	SetStatus(tripID, status string) error
	SetSoftDeleted(tripID string) error
	ListFeed(limit, offset int32, category, season string, locationIDs []int, sortBy string) ([]*models.Trip, error)
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

type InvitationLinkRepositoryInterface interface {
	Create(link *models.InvitationLink) error
	GetByToken(token string) (*models.InvitationLink, error)
}

type TripSettingsRepositoryInterface interface {
	EnsureDefaultSettings(tripID, userID string) error
	UpdateNotifications(tripID, userID string, enabled bool) error
}

// TripEventPublisher — методы RedisRepository, используемые TripService (eventRepo может быть nil-интерфейсом).
type TripEventPublisher interface {
	PublishTripEvent(ctx context.Context, eventType string, tripID, userID string) error
	AddMLTask(ctx context.Context, tripID string) error
}

type MediaRepositoryInterface interface {
	Create(m *models.Media) error
	GetByID(id string) (*models.Media, error)
	ListByTripID(tripID string) ([]*models.Media, error)
	UpdatePinID(mediaID, pinID string) error
	UpdatePinIDByIDs(mediaIDs []string, pinID string) error
	DeleteByIDs(ids []string) error
	SetSimilarGroupID(mediaIDs []string, groupID string) error
	CountByTripID(tripID string) (total int, videos int, err error)
	ClusterIDsByLocation(tripID string, radiusMeters float64) (map[string]int, error)
	ListByPinID(pinID string) ([]*models.Media, error)
}

type PinRepositoryInterface interface {
	Create(p *models.Pin) error
	GetByID(id string) (*models.Pin, error)
	ListByTripID(tripID string) ([]*models.Pin, error)
	Update(p *models.Pin) error
	Delete(id string) error
	DeleteByTripID(tripID string) error
}

type TagRepositoryInterface interface {
	SetForPin(tripID, pinID string, tags []string) error
	Add(t *models.Tag) error
	GetByPinID(pinID string) ([]string, error)
	GetByTripID(tripID string) (map[string][]string, error)
	DeleteForPin(pinID string) error
}

type SocialRepositoryInterface interface {
	SetReaction(userID, tripID, reaction string) error
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
