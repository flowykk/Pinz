package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

// Должен покрывать максимальное время обработки на стороне ML.
const mlPresignTTL = 2 * time.Hour

const (
	MLFlowCreation          = repositories.MLFlowCreation
	MLFlowAddMedia          = repositories.MLFlowAddMedia
	MLFlowPinUploadCreation = repositories.MLFlowPinUploadCreation
	MLFlowPinUploadAddition = repositories.MLFlowPinUploadAddition
)

type MLPinFetcher interface {
	ListByTripID(tripID string) ([]*models.Pin, error)
}

type MLMediaFetcher interface {
	ListByPinID(pinID string) ([]*models.Media, error)
}

// Для add_media existing-пины несут всё медиа: новые is_new=true, существующие false.
func BuildMLTaskMessageForTrip(
	ctx context.Context,
	tripID, flow string,
	sessionID string,
	newPinIDs []string,
	pendingExistingAttachments []string,
	pinFetcher MLPinFetcher,
	mediaFetcher MLMediaFetcher,
	mediaURLs MediaURLResolver,
) (msg repositories.MLTaskMessage, mediaCount int, err error) {
	if pinFetcher == nil || mediaFetcher == nil {
		return repositories.MLTaskMessage{}, 0, errors.New("ml payload: pin/media fetcher required")
	}
	if mediaURLs == nil {
		return repositories.MLTaskMessage{}, 0, errors.New("ml payload: mediaURLs required")
	}
	if flow != MLFlowCreation && flow != MLFlowAddMedia {
		return repositories.MLTaskMessage{}, 0, fmt.Errorf("ml payload: unknown trip flow %q", flow)
	}

	pins, err := pinFetcher.ListByTripID(tripID)
	if err != nil {
		return repositories.MLTaskMessage{}, 0, fmt.Errorf("ml payload: list pins: %w", err)
	}

	newPinSet := make(map[string]struct{}, len(newPinIDs))
	for _, id := range newPinIDs {
		newPinSet[id] = struct{}{}
	}
	pendingSet := make(map[string]struct{}, len(pendingExistingAttachments))
	for _, id := range pendingExistingAttachments {
		pendingSet[id] = struct{}{}
	}

	out := make([]repositories.MLPinPayload, 0, len(pins))
	presignStart := time.Now()
	for _, pin := range pins {
		mediaList, err := mediaFetcher.ListByPinID(pin.ID)
		if err != nil {
			return repositories.MLTaskMessage{}, 0, fmt.Errorf("ml payload: list media for pin %s: %w", pin.ID, err)
		}
		isNewPin := true
		if flow == MLFlowAddMedia {
			if _, ok := newPinSet[pin.ID]; !ok {
				isNewPin = false
			}
		}

		mediaPayload := make([]repositories.MLMediaPayload, 0, len(mediaList))
		for _, m := range mediaList {
			item, err := buildMediaPayload(ctx, m, mediaURLs)
			if err != nil {
				return repositories.MLTaskMessage{}, 0, err
			}
			if isNewPin {
				item.IsNew = true
			} else if _, ok := pendingSet[m.ID]; ok {
				item.IsNew = true
			}
			mediaPayload = append(mediaPayload, item)
		}
		if len(mediaPayload) == 0 {
			continue
		}
		mediaCount += len(mediaPayload)
		out = append(out, repositories.MLPinPayload{
			PinID:       pin.ID,
			IsNew:       isNewPin,
			Description: pin.Description,
			Media:       mediaPayload,
		})
	}
	metrics.ObserveMLPresignDuration(ctx, time.Since(presignStart).Seconds(), flow)

	msg = repositories.MLTaskMessage{
		Flow:          flow,
		TripID:        tripID,
		SessionID:     sessionID,
		Pins:          out,
		ExpiresAtUnix: time.Now().Add(mlPresignTTL).Unix(),
	}
	metrics.MLPayloadMediaCount(ctx, int64(mediaCount), flow)
	return msg, mediaCount, nil
}

func BuildMLTaskMessageForPinUpload(
	ctx context.Context,
	flow string,
	tripID, sessionID, targetPinID, targetPinDescription string,
	sessionMedia, pinMedia []*models.Media,
	mediaURLs MediaURLResolver,
) (msg repositories.MLTaskMessage, mediaCount int, err error) {
	if mediaURLs == nil {
		return repositories.MLTaskMessage{}, 0, errors.New("ml payload: mediaURLs required")
	}
	if flow != MLFlowPinUploadCreation && flow != MLFlowPinUploadAddition {
		return repositories.MLTaskMessage{}, 0, fmt.Errorf("ml payload: unknown pin_upload flow %q", flow)
	}

	presignStart := time.Now()
	media := make([]repositories.MLMediaPayload, 0, len(sessionMedia)+len(pinMedia))
	for _, m := range sessionMedia {
		item, err := buildMediaPayload(ctx, m, mediaURLs)
		if err != nil {
			return repositories.MLTaskMessage{}, 0, err
		}
		item.IsNew = true
		media = append(media, item)
	}
	for _, m := range pinMedia {
		item, err := buildMediaPayload(ctx, m, mediaURLs)
		if err != nil {
			return repositories.MLTaskMessage{}, 0, err
		}
		item.IsNew = false
		media = append(media, item)
	}
	metrics.ObserveMLPresignDuration(ctx, time.Since(presignStart).Seconds(), flow)

	pinPayload := repositories.MLPinPayload{
		PinID:       targetPinID,
		IsNew:       flow == MLFlowPinUploadCreation,
		Description: targetPinDescription,
		Media:       media,
	}
	msg = repositories.MLTaskMessage{
		Flow:          flow,
		TripID:        tripID,
		SessionID:     sessionID,
		TargetPinID:   targetPinID,
		Pins:          []repositories.MLPinPayload{pinPayload},
		ExpiresAtUnix: time.Now().Add(mlPresignTTL).Unix(),
	}
	mediaCount = len(media)
	metrics.MLPayloadMediaCount(ctx, int64(mediaCount), flow)
	return msg, mediaCount, nil
}

func buildMediaPayload(ctx context.Context, m *models.Media, mediaURLs MediaURLResolver) (repositories.MLMediaPayload, error) {
	if m == nil {
		return repositories.MLMediaPayload{}, errors.New("ml payload: nil media")
	}
	getURL, err := mediaURLs.ReadURLWithTTL(ctx, m.S3Key, mlPresignTTL)
	if err != nil {
		return repositories.MLMediaPayload{}, fmt.Errorf("ml payload: presign for media %s: %w", m.ID, err)
	}
	item := repositories.MLMediaPayload{
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
