package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

type TagRepository struct {
	q *sqlcdb.Queries
}

const (
	maxTagsPerPin = 10
	maxTagLength  = 15
)

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{q: sqlcdb.New(db)}
}

func pinNullUUID(pinID string) uuid.NullUUID {
	if pinID == "" {
		return uuid.NullUUID{}
	}
	id, err := uuid.Parse(pinID)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func (r *TagRepository) SetForPin(tripID, pinID string, tags []string) error {
	if len(tags) > maxTagsPerPin {
		tags = tags[:maxTagsPerPin]
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	if err := r.q.TagDeleteForPin(context.Background(), sqlcdb.TagDeleteForPinParams{
		TripID: tid,
		PinID:  pinNullUUID(pinID),
	}); err != nil {
		return err
	}
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if len(tag) > maxTagLength {
			tag = tag[:maxTagLength]
		}
		t := &models.Tag{TripID: tripID, PinID: pinID, Tag: tag}
		if err := r.Add(t); err != nil {
			return err
		}
	}
	return nil
}

func (r *TagRepository) Add(t *models.Tag) error {
	if t.Tag == "" {
		return nil
	}
	if len(t.Tag) > maxTagLength {
		t.Tag = t.Tag[:maxTagLength]
	}
	tid, err := uuid.Parse(t.TripID)
	if err != nil {
		return err
	}
	id, err := r.q.TagInsert(context.Background(), sqlcdb.TagInsertParams{
		TripID: tid,
		PinID:  pinNullUUID(t.PinID),
		Tag:    t.Tag,
	})
	if err != nil {
		return err
	}
	t.ID = id.String()
	return nil
}

func (r *TagRepository) GetByPinID(pinID string) ([]string, error) {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return nil, err
	}
	return r.q.TagListByPin(context.Background(), uuid.NullUUID{UUID: pid, Valid: true})
}

func (r *TagRepository) GetByTripID(tripID string) (map[string][]string, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.TagListByTrip(context.Background(), tid)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for _, row := range rows {
		pinKey := ""
		if row.PinID.Valid {
			pinKey = row.PinID.UUID.String()
		}
		out[pinKey] = append(out[pinKey], row.Tag)
	}
	return out, nil
}

func (r *TagRepository) DeleteForPin(pinID string) error {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return err
	}
	return r.q.TagDeleteAllForPin(context.Background(), uuid.NullUUID{UUID: pid, Valid: true})
}
