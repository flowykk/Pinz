package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/models"
)

// TTL presigned GET URL для ML — должен покрывать максимальное время обработки на стороне ML.
const mlPresignTTL = 2 * time.Hour

type MLMediaPayload struct {
	MediaID        string   `json:"media_id"`
	MediaType      string   `json:"media_type"`
	S3Key          string   `json:"s3_key"`
	GetURL         string   `json:"get_url"`
	ContentType    string   `json:"content_type,omitempty"`
	CapturedAtUnix *int64   `json:"captured_at_unix,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
}

type MLPinPayload struct {
	PinID string           `json:"pin_id"`
	IsNew bool             `json:"is_new"`
	Media []MLMediaPayload `json:"media"`
}

type MLPinFetcher interface {
	ListByTripID(tripID string) ([]*models.Pin, error)
}

type MLMediaFetcher interface {
	ListByPinID(pinID string) ([]*models.Media, error)
}

// Для flow="add_media" в existing-пинах в payload идут только media из
// pendingExistingAttachments — исходные ML не пересчитывает. См. mlContract.md.
func BuildTripMLPayload(
	ctx context.Context,
	tripID, flow string,
	newPinIDs []string,
	pendingExistingAttachments []string,
	pinFetcher MLPinFetcher,
	mediaFetcher MLMediaFetcher,
	mediaURLs MediaURLResolver,
) (pinsJSON string, expiresAtUnix int64, mediaCount int, err error) {
	if pinFetcher == nil || mediaFetcher == nil {
		return "", 0, 0, errors.New("ml payload: pin/media fetcher required")
	}
	if mediaURLs == nil {
		return "", 0, 0, errors.New("ml payload: mediaURLs required")
	}
	if flow != mlFlowCreation && flow != mlFlowAddMedia {
		return "", 0, 0, fmt.Errorf("ml payload: unknown flow %q", flow)
	}

	pins, err := pinFetcher.ListByTripID(tripID)
	if err != nil {
		return "", 0, 0, fmt.Errorf("ml payload: list pins: %w", err)
	}

	newPinSet := make(map[string]struct{}, len(newPinIDs))
	for _, id := range newPinIDs {
		newPinSet[id] = struct{}{}
	}
	pendingSet := make(map[string]struct{}, len(pendingExistingAttachments))
	for _, id := range pendingExistingAttachments {
		pendingSet[id] = struct{}{}
	}

	out := make([]MLPinPayload, 0, len(pins))
	presignStart := time.Now()
	for _, pin := range pins {
		mediaList, err := mediaFetcher.ListByPinID(pin.ID)
		if err != nil {
			return "", 0, 0, fmt.Errorf("ml payload: list media for pin %s: %w", pin.ID, err)
		}
		isNew := true
		var filter map[string]struct{}
		if flow == mlFlowAddMedia {
			if _, ok := newPinSet[pin.ID]; ok {
				isNew = true
			} else {
				isNew = false
				filter = pendingSet
			}
		}
		mediaPayload := make([]MLMediaPayload, 0, len(mediaList))
		for _, m := range mediaList {
			if filter != nil {
				if _, ok := filter[m.ID]; !ok {
					continue
				}
			}
			item, err := buildMediaPayload(ctx, m, mediaURLs)
			if err != nil {
				return "", 0, 0, err
			}
			mediaPayload = append(mediaPayload, item)
		}
		if len(mediaPayload) == 0 {
			continue
		}
		mediaCount += len(mediaPayload)
		out = append(out, MLPinPayload{
			PinID: pin.ID,
			IsNew: isNew,
			Media: mediaPayload,
		})
	}
	metrics.ObserveMLPresignDuration(ctx, time.Since(presignStart).Seconds(), flow)

	b, err := json.Marshal(out)
	if err != nil {
		return "", 0, 0, fmt.Errorf("ml payload: marshal: %w", err)
	}
	expiresAtUnix = time.Now().Add(mlPresignTTL).Unix()
	metrics.MLPayloadSize(ctx, int64(len(b)), flow)
	metrics.MLPayloadMediaCount(ctx, int64(mediaCount), flow)
	return string(b), expiresAtUnix, mediaCount, nil
}

// existingMediaJSON возвращается пустой строкой, если pinMedia nil/пуст —
// caller не должен класть пустой массив в payload.
func BuildPinUploadMLPayload(
	ctx context.Context,
	flowLabel string,
	sessionMedia, pinMedia []*models.Media,
	mediaURLs MediaURLResolver,
) (newMediaJSON, existingMediaJSON string, expiresAtUnix int64, mediaCount int, err error) {
	if mediaURLs == nil {
		return "", "", 0, 0, errors.New("ml payload: mediaURLs required")
	}

	presignStart := time.Now()
	newItems := make([]MLMediaPayload, 0, len(sessionMedia))
	for _, m := range sessionMedia {
		item, err := buildMediaPayload(ctx, m, mediaURLs)
		if err != nil {
			return "", "", 0, 0, err
		}
		newItems = append(newItems, item)
	}
	existingItems := make([]MLMediaPayload, 0, len(pinMedia))
	for _, m := range pinMedia {
		item, err := buildMediaPayload(ctx, m, mediaURLs)
		if err != nil {
			return "", "", 0, 0, err
		}
		existingItems = append(existingItems, item)
	}
	metrics.ObserveMLPresignDuration(ctx, time.Since(presignStart).Seconds(), flowLabel)

	nb, err := json.Marshal(newItems)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("ml payload: marshal new_media: %w", err)
	}
	newMediaJSON = string(nb)

	if len(existingItems) > 0 {
		eb, err := json.Marshal(existingItems)
		if err != nil {
			return "", "", 0, 0, fmt.Errorf("ml payload: marshal existing_media: %w", err)
		}
		existingMediaJSON = string(eb)
	}

	mediaCount = len(newItems) + len(existingItems)
	expiresAtUnix = time.Now().Add(mlPresignTTL).Unix()
	metrics.MLPayloadSize(ctx, int64(len(newMediaJSON)+len(existingMediaJSON)), flowLabel)
	metrics.MLPayloadMediaCount(ctx, int64(mediaCount), flowLabel)
	return newMediaJSON, existingMediaJSON, expiresAtUnix, mediaCount, nil
}

func buildMediaPayload(ctx context.Context, m *models.Media, mediaURLs MediaURLResolver) (MLMediaPayload, error) {
	if m == nil {
		return MLMediaPayload{}, errors.New("ml payload: nil media")
	}
	getURL, err := mediaURLs.ReadURLWithTTL(ctx, m.S3Key, mlPresignTTL)
	if err != nil {
		return MLMediaPayload{}, fmt.Errorf("ml payload: presign for media %s: %w", m.ID, err)
	}
	item := MLMediaPayload{
		MediaID:   m.ID,
		MediaType: m.MediaType,
		S3Key:     m.S3Key,
		GetURL:    getURL,
	}
	if m.CapturedAt != nil {
		ts := m.CapturedAt.Unix()
		item.CapturedAtUnix = &ts
	}
	if m.Latitude != nil {
		lat := *m.Latitude
		item.Latitude = &lat
	}
	if m.Longitude != nil {
		lon := *m.Longitude
		item.Longitude = &lon
	}
	return item, nil
}

const (
	mlFlowCreation        = "creation"
	mlFlowAddMedia        = "add_media"
	MLFlowCreation        = mlFlowCreation
	MLFlowAddMedia        = mlFlowAddMedia
	MLFlowPinUploadCreate = "pin_upload_creation"
	MLFlowPinUploadAddTo  = "pin_upload_addition"
)
