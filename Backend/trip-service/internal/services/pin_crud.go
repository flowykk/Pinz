package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

// pinAdditionDraftSnapshot — JSON-структура для сохранения результата
// ProcessPinMediaAddition между этапами Process и Finalize. Хранится в
// pin_media_addition_sessions.draft_snapshot и читается при GetReview/Finalize.
type pinAdditionDraftSnapshot struct {
	NewMediaIDs     []string                  `json:"new_media_ids"`
	NSFWMediaIDs    []string                  `json:"nsfw_media_ids"`
	DedupedMediaIDs []string                  `json:"deduped_media_ids"`
	PinIssues       []string                  `json:"pin_issues"`
	Similar         []pinAdditionSimilarGroup `json:"similar"`
}

type pinAdditionSimilarGroup struct {
	MediaIDs []string `json:"media_ids"`
}

// pinIssueMissingCoordinates / pinIssueMissingDates — коды проблем для review (ТЗ 4.7.3-4.7.4).
const (
	pinIssueMissingCoordinates = "MISSING_COORDINATES"
	pinIssueMissingDates       = "MISSING_DATES"
)

// =============================================================================
// GetPin (ТЗ 4.3)
// =============================================================================

// GetPin возвращает все поля пина. Доступ — participant трипа или favourite-юзер.
// Если пин скрыт для caller'а через pin_hidden_by_user (ТЗ 4.5.2 soft-delete-for-self),
// возвращается NotFound, чтобы клиент не отличал «нет пина» от «спрятан».
func (s *TripService) GetPin(ctx context.Context, req *pb.GetPinRequest) (*pb.GetPinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	if _, err := s.assertCanReadPin(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}
	mediaList, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.GetPinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// assertCanReadPin — общий guard для GetPin и read-операций над пином: участник
// или favourite-юзер; pin_hidden для текущего юзера → NotFound.
func (s *TripService) assertCanReadPin(ctx context.Context, tripID, pinID, userID string) (bool, error) {
	if s.pinHiddenRepo != nil {
		hidden, err := s.pinHiddenRepo.IsHidden(pinID, userID)
		if err == nil && hidden {
			return false, status.Error(codes.NotFound, "pin not found")
		}
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return false, status.Error(codes.Internal, "failed to check participant")
	}
	if participant {
		return true, nil
	}
	if s.favouriteRepo != nil {
		hasFav, err := s.favouriteRepo.HasFavourite(userID, tripID)
		if err == nil && hasFav {
			return true, nil
		}
	}
	_ = ctx
	return false, status.Error(codes.PermissionDenied, "not a participant")
}

// =============================================================================
// UpdatePin (ТЗ 4.2.1, 4.2.4-4.2.9)
// =============================================================================

// UpdatePin применяет правки полей пина на READY-трипе. Любой participant.
// tags применяются как replace-all только при tags_set=true (даже если массив пуст).
// При смене координат — reverse geocoding и upsert в trip_locations (статистика).
func (s *TripService) UpdatePin(ctx context.Context, req *pb.UpdatePinRequest) (*pb.UpdatePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusReady {
		return nil, errWrongStatus(models.TripStatusReady, trip.Status)
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}

	// валидация и применение полей
	coordsChanged := false
	if req.Name != nil {
		name := req.GetName()
		if len(name) == 0 {
			return nil, status.Error(codes.InvalidArgument, "name must not be empty")
		}
		if len(name) > MaxNameLength {
			return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
		}
		pin.Name = name
	}
	if req.Description != nil {
		desc := req.GetDescription()
		if len(desc) > MaxDescriptionLength {
			return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
		}
		pin.Description = desc
	}
	if req.Category != nil {
		pin.Category = ValidatePinCategory(req.GetCategory())
	}
	if req.Latitude != nil || req.Longitude != nil {
		if req.Latitude == nil || req.Longitude == nil {
			return nil, status.Error(codes.InvalidArgument, "latitude and longitude must be provided together")
		}
		lat := req.GetLatitude()
		lon := req.GetLongitude()
		if lat < -90 || lat > 90 {
			return nil, status.Error(codes.InvalidArgument, "latitude must be in [-90, 90]")
		}
		if lon < -180 || lon > 180 {
			return nil, status.Error(codes.InvalidArgument, "longitude must be in [-180, 180]")
		}
		pin.Latitude = &lat
		pin.Longitude = &lon
		coordsChanged = true
	}
	if req.StartTimeUnix != nil {
		t := time.Unix(req.GetStartTimeUnix(), 0)
		pin.StartTime = &t
	}
	if req.EndTimeUnix != nil {
		t := time.Unix(req.GetEndTimeUnix(), 0)
		pin.EndTime = &t
	}
	if pin.StartTime != nil && pin.EndTime != nil && pin.EndTime.Before(*pin.StartTime) {
		return nil, status.Error(codes.InvalidArgument, "end_time must be greater than or equal to start_time")
	}
	if req.GetTagsSet() {
		if err := validateTags(req.GetTags()); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if err := s.pinRepo.Update(pin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to update pin")
	}
	if req.GetTagsSet() {
		if err := s.tagRepo.SetForPin(tripID, pinID, req.GetTags()); err != nil {
			return nil, status.Error(codes.Internal, "failed to update pin tags")
		}
	}
	if coordsChanged && pin.Latitude != nil && pin.Longitude != nil && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, []repositories.GeoRequestPin{{
			PinID:     pinID,
			Latitude:  *pin.Latitude,
			Longitude: *pin.Longitude,
		}})
	}
	mediaList, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.UpdatePinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// =============================================================================
// DeletePin (ТЗ 4.5.1 / 4.5.2)
// =============================================================================

// DeletePin — full delete если трип не в избранном у других пользователей,
// иначе soft-delete-for-self через pin_hidden_by_user. Любой participant.
// Защита: запрет удаления при активной pin_media_addition_session, чтобы не было
// orphan media с pin_addition_session_id, ссылающимся на удалённый пин.
func (s *TripService) DeletePin(ctx context.Context, req *pb.DeletePinRequest) (*pb.DeletePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return nil, status.Error(codes.NotFound, "pin not found")
	}
	// блокируем delete если идёт сессия добавления медиа в этот пин.
	if s.pinAddSessionRepo != nil {
		if _, err := s.pinAddSessionRepo.GetActiveForPin(ctx, pinID); err == nil {
			return nil, status.Error(codes.FailedPrecondition, "pin has an active media addition session; cancel it first")
		} else if !errors.Is(err, repositories.ErrPinAdditionSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to check active pin media session")
		}
	}
	inOthersFav := false
	if s.favouriteRepo != nil {
		v, err := s.favouriteRepo.HasFavouritesByOtherUsers(tripID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check favourites")
		}
		inOthersFav = v
	}
	if inOthersFav {
		// soft-delete-for-self (ТЗ 4.5.2)
		if s.pinHiddenRepo == nil {
			return nil, status.Error(codes.Internal, "pin hidden repository not configured")
		}
		if err := s.pinHiddenRepo.HidePinForUser(pinID, userID); err != nil {
			return nil, status.Error(codes.Internal, "failed to hide pin")
		}
		return &pb.DeletePinResponse{DeletionMode: "soft_for_user"}, nil
	}
	// full delete (ТЗ 4.5.1): ссылка media.pin_id = ON DELETE SET NULL, поэтому
	// каскадим media явно — DeleteByPinID возвращает s3_keys для best-effort
	// S3 cleanup. tagRepo.DeleteForPin зеркалирует CASCADE FK для прозрачности.
	s3Keys, err := s.mediaRepo.DeleteByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete pin media")
	}
	if s.mediaURLs != nil {
		for _, key := range s3Keys {
			if err := s.mediaURLs.DeleteObject(ctx, key); err != nil {
				slog.WarnContext(ctx, "DeletePin: s3 delete failed (best-effort)", "pin_id", pinID, "key", key, "err", err)
			}
		}
	}
	if err := s.tagRepo.DeleteForPin(pinID); err != nil {
		slog.WarnContext(ctx, "DeletePin: tag delete failed", "pin_id", pinID, "err", err)
	}
	if err := s.pinRepo.Delete(pinID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete pin")
	}
	return &pb.DeletePinResponse{DeletionMode: "full"}, nil
}

// =============================================================================
// AddMediaToPin* — sessioned флоу добавления медиа в существующий пин (ТЗ 4.2.2)
// =============================================================================

// AddMediaToPinStart создаёт сессию (UNIQUE per-pin), возвращает session_id и
// presigned URLs. Conflict — если для пина уже идёт сессия (FailedPrecondition).
func (s *TripService) AddMediaToPinStart(ctx context.Context, req *pb.AddMediaToPinStartRequest) (*pb.AddMediaToPinStartResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	if s.pinAddSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin media addition repository not configured")
	}
	sessionID, err := s.pinAddSessionRepo.Create(ctx, tripID, pinID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinAdditionSessionActive) {
			return nil, status.Error(codes.FailedPrecondition, "another pin media addition session is already active for this pin")
		}
		return nil, status.Error(codes.Internal, "failed to create pin media session")
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	return &pb.AddMediaToPinStartResponse{
		SessionId: sessionID,
		UploadUrls: uploadUrls,
	}, nil
}

// RequestPinMediaUploadUrls — догрузка presigned URLs к активной сессии.
// Не меняет состояние; валидирует сессию + лимиты.
func (s *TripService) RequestPinMediaUploadUrls(ctx context.Context, req *pb.RequestPinMediaUploadUrlsRequest) (*pb.RequestPinMediaUploadUrlsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID); err != nil {
		return nil, err
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	_ = s.pinAddSessionRepo.Touch(ctx, sessionID)
	return &pb.RequestPinMediaUploadUrlsResponse{UploadUrls: uploadUrls}, nil
}

// CommitPinMediaUpload фиксирует загрузку файла в S3: создаёт media с
// pin_id=NULL и pin_addition_session_id=session. Лимиты трипа считаются на
// общем COUNT (включая draft media — защищает от over-upload в S3).
func (s *TripService) CommitPinMediaUpload(ctx context.Context, req *pb.CommitPinMediaUploadRequest) (*pb.CommitPinMediaUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	if req.GetS3Key() == "" || req.GetMediaType() == "" {
		return nil, status.Error(codes.InvalidArgument, "s3_key and media_type are required")
	}
	if req.GetMediaType() != "image" && req.GetMediaType() != "video" {
		return nil, status.Error(codes.InvalidArgument, "media_type must be 'image' or 'video'")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	total, videos, _ := s.mediaRepo.CountByTripID(tripID)
	if total+1 > MaxMediaPerTrip {
		return nil, errLimitExceeded("media", MaxMediaPerTrip, total+1)
	}
	if req.GetMediaType() == "video" && videos+1 > MaxVideosPerTrip {
		return nil, errLimitExceeded("video", MaxVideosPerTrip, videos+1)
	}
	m := &models.Media{
		TripID: tripID,
		S3Key: req.GetS3Key(),
		MediaType: req.GetMediaType(),
		PrivacyLevel: trip.PrivacyLevel,
		PinAdditionSessionID: &sessionID,
		UploadedBy: &userID,
	}
	if req.CapturedAtUnix != nil {
		t := time.Unix(req.GetCapturedAtUnix(), 0)
		m.CapturedAt = &t
	}
	if req.Latitude != nil && req.Longitude != nil {
		lat := req.GetLatitude()
		lon := req.GetLongitude()
		m.Latitude = &lat
		m.Longitude = &lon
	}
	if err := s.mediaRepo.Create(m); err != nil {
		return nil, status.Error(codes.Internal, "failed to save media")
	}
	_ = s.pinAddSessionRepo.Touch(ctx, sessionID)
	mediaInSession, _ := s.mediaRepo.ListByPinAdditionSession(sessionID)
	return &pb.CommitPinMediaUploadResponse{
		MediaId: m.ID,
		MediaCountInSession: int32(len(mediaInSession)),
	}, nil
}

// ProcessPinMediaAddition — синхронный ML-stub: хеш-дедуп (4.7.6.a), NSFW (4.7.5),
// similar (4.7.7), pin issues (4.7.3-4.7.4). Snapshot → draft_snapshot для повторного
// чтения через GetPinMediaAdditionReview. Дедуплицированные media удаляются физически
// (DeleteByIDs + S3 cleanup).
func (s *TripService) ProcessPinMediaAddition(ctx context.Context, req *pb.ProcessPinMediaAdditionRequest) (*pb.ProcessPinMediaAdditionResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID); err != nil {
		return nil, err
	}

	// 1. Текущие медиа сессии + уже привязанные к пину (для дедупа кросс-сессии).
	sessionMedia, err := s.mediaRepo.ListByPinAdditionSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list session media")
	}
	pinMedia, _ := s.mediaRepo.ListByPinID(pinID)

	// 2. Хеш-дедуп: если у session-media совпадает content_hash с (a) другой
	// session-media или (b) уже привязанной к пину — кандидат на удаление.
	existingHashes := map[string]struct{}{}
	for _, m := range pinMedia {
		if m.ContentHash != nil {
			existingHashes[*m.ContentHash] = struct{}{}
		}
	}
	seen := map[string]string{} // hash → first kept media id in session
	var deduped []*models.Media
	for _, m := range sessionMedia {
		if m.ContentHash == nil {
			continue
		}
		if _, dup := existingHashes[*m.ContentHash]; dup {
			deduped = append(deduped, m)
			continue
		}
		if _, dup := seen[*m.ContentHash]; dup {
			deduped = append(deduped, m)
			continue
		}
		seen[*m.ContentHash] = m.ID
	}
	dedupedIDs := make([]string, 0, len(deduped))
	dedupedKeys := make([]string, 0, len(deduped))
	for _, m := range deduped {
		dedupedIDs = append(dedupedIDs, m.ID)
		if m.S3Key != "" {
			dedupedKeys = append(dedupedKeys, m.S3Key)
		}
	}
	if len(dedupedIDs) > 0 {
		if err := s.mediaRepo.DeleteByIDs(dedupedIDs); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete duplicates")
		}
		if s.mediaURLs != nil {
			for _, k := range dedupedKeys {
				_ = s.mediaURLs.DeleteObject(ctx, k)
			}
		}
	}

	// 3. Перечитать оставшиеся session media после дедупа.
	remaining, err := s.mediaRepo.ListByPinAdditionSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload session media")
	}

	// 4. NSFW и similar — заглушки до подключения реального ML (паритет
	// с finalizeProcessingStub в creation-флоу).
	var nsfwIDs []string
	var similar []pinAdditionSimilarGroup

	// 5. Pin issues: симулируем привязку remaining к пину и считаем,
	// есть ли captured_at и координаты у объединённого набора (pinMedia + remaining).
	hasDate := false
	hasCoords := false
	for _, m := range pinMedia {
		if m.CapturedAt != nil {
			hasDate = true
		}
		if m.Latitude != nil && m.Longitude != nil {
			hasCoords = true
		}
	}
	for _, m := range remaining {
		if m.CapturedAt != nil {
			hasDate = true
		}
		if m.Latitude != nil && m.Longitude != nil {
			hasCoords = true
		}
	}
	var issues []string
	if !hasCoords {
		issues = append(issues, pinIssueMissingCoordinates)
	}
	if !hasDate {
		issues = append(issues, pinIssueMissingDates)
	}

	// 6. Snapshot и сохранение.
	newIDs := make([]string, 0, len(remaining))
	for _, m := range remaining {
		newIDs = append(newIDs, m.ID)
	}
	snap := pinAdditionDraftSnapshot{
		NewMediaIDs: newIDs,
		NSFWMediaIDs: nsfwIDs,
		DedupedMediaIDs: dedupedIDs,
		PinIssues: issues,
		Similar: similar,
	}
	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode snapshot")
	}
	if err := s.pinAddSessionRepo.SetDraftSnapshot(ctx, sessionID, snapBytes); err != nil {
		return nil, status.Error(codes.Internal, "failed to save snapshot")
	}
	return &pb.ProcessPinMediaAdditionResponse{
		SessionId: sessionID,
		Draft: snapshotToDraftProto(ctx, s, snap, remaining),
		Similar: snapshotToSimilarProto(snap),
	}, nil
}

// GetPinMediaAdditionReview читает snapshot из сессии и возвращает review-снимок.
// Если Process ещё не вызывался — снапшот пустой (issues/similar пустые).
func (s *TripService) GetPinMediaAdditionReview(ctx context.Context, req *pb.GetPinMediaAdditionReviewRequest) (*pb.GetPinMediaAdditionReviewResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	session, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	var snap pinAdditionDraftSnapshot
	if len(session.DraftSnapshot) > 0 {
		_ = json.Unmarshal(session.DraftSnapshot, &snap)
	}
	remaining, _ := s.mediaRepo.ListByPinAdditionSession(sessionID)
	return &pb.GetPinMediaAdditionReviewResponse{
		SessionId: sessionID,
		Draft: snapshotToDraftProto(ctx, s, snap, remaining),
		Similar: snapshotToSimilarProto(snap),
	}, nil
}

// FinalizePinMediaAddition применяет media_to_delete (orphan-cleanup), привязывает
// оставшиеся media сессии к пину (UpdatePinIDByIDs), пересчитывает start/end/lat/lon,
// делает reverse-geocoding (если у пина появились координаты впервые/изменились),
// закрывает сессию.
func (s *TripService) FinalizePinMediaAddition(ctx context.Context, req *pb.FinalizePinMediaAdditionRequest) (*pb.FinalizePinMediaAdditionResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	if _, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID); err != nil {
		return nil, err
	}

	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	prevHadCoords := pin.Latitude != nil && pin.Longitude != nil

	// 1. Применить media_to_delete (только media текущей сессии).
	if del := req.GetMediaToDelete(); len(del) > 0 {
		toDelete, s3Keys, derr := s.collectDeletableSessionMedia(sessionID, del)
		if derr != nil {
			return nil, derr
		}
		if len(toDelete) > 0 {
			if err := s.mediaRepo.DeleteByIDs(toDelete); err != nil {
				return nil, status.Error(codes.Internal, "failed to delete media")
			}
			if s.mediaURLs != nil {
				for _, k := range s3Keys {
					_ = s.mediaURLs.DeleteObject(ctx, k)
				}
			}
		}
	}

	// 2. Привязать оставшиеся media сессии к пину.
	remaining, err := s.mediaRepo.ListByPinAdditionSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload session media")
	}
	if len(remaining) > 0 {
		ids := make([]string, 0, len(remaining))
		for _, m := range remaining {
			ids = append(ids, m.ID)
		}
		if err := s.mediaRepo.UpdatePinIDByIDs(ids, pinID); err != nil {
			return nil, status.Error(codes.Internal, "failed to attach media to pin")
		}
		if err := s.pinRepo.IncMediaCount(pinID, len(ids)); err != nil {
			slog.WarnContext(ctx, "FinalizePinMediaAddition: media_count update failed", "pin_id", pinID, "err", err)
		}
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pinID)
	}

	// 3. Reverse geocoding если у пина теперь есть координаты впервые. Async через
	// statistics-service: PIN_LOCATIONS_REQUESTED → PIN_LOCATIONS_RESOLVED.
	pin, err = s.pinRepo.GetByID(pinID)
	if err == nil && pin.Latitude != nil && pin.Longitude != nil && !prevHadCoords && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, []repositories.GeoRequestPin{{
			PinID:     pinID,
			Latitude:  *pin.Latitude,
			Longitude: *pin.Longitude,
		}})
	}

	// 4. Закрыть сессию.
	if err := s.pinAddSessionRepo.Close(ctx, sessionID, models.PinMediaAdditionSessionCloseReasonConfirmed); err != nil {
		if !errors.Is(err, repositories.ErrPinAdditionSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}

	mediaList, _ := s.mediaRepo.ListByPinID(pinID)
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.FinalizePinMediaAdditionResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// CancelPinMediaAddition удаляет orphan media сессии (pin_id=NULL,
// pin_addition_session_id=session) + S3 cleanup, закрывает сессию.
func (s *TripService) CancelPinMediaAddition(ctx context.Context, req *pb.CancelPinMediaAdditionRequest) (*pb.CancelPinMediaAdditionResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, sessionID := req.GetTripId(), req.GetPinId(), req.GetSessionId()
	if tripID == "" || pinID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, session_id are required")
	}
	participant, perr := s.participantRepo.IsParticipant(tripID, userID)
	if perr != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.assertActivePinSession(ctx, tripID, pinID, sessionID, userID); err != nil {
		return nil, err
	}
	s3Keys, err := s.mediaRepo.DeleteOrphanByPinAdditionSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cleanup orphan media")
	}
	if s.mediaURLs != nil {
		for _, k := range s3Keys {
			_ = s.mediaURLs.DeleteObject(ctx, k)
		}
	}
	if err := s.pinAddSessionRepo.Close(ctx, sessionID, models.PinMediaAdditionSessionCloseReasonCancelled); err != nil {
		if !errors.Is(err, repositories.ErrPinAdditionSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}
	return &pb.CancelPinMediaAdditionResponse{Status: "cancelled"}, nil
}

// =============================================================================
// RemoveMediaFromPin — sessionless удаление одного медиа (ТЗ 4.2.3)
// =============================================================================

// RemoveMediaFromPin удаляет одно медиа из пина с пересчётом агрегатов.
// Защита: пин не может остаться пустым (ТЗ 2.2.9 — set of media обязателен).
func (s *TripService) RemoveMediaFromPin(ctx context.Context, req *pb.RemoveMediaFromPinRequest) (*pb.RemoveMediaFromPinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, pinID, mediaID := req.GetTripId(), req.GetPinId(), req.GetMediaId()
	if tripID == "" || pinID == "" || mediaID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id, pin_id, media_id are required")
	}
	if err := s.assertParticipantAndPinReady(ctx, tripID, pinID, userID); err != nil {
		return nil, err
	}
	m, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Error(codes.Internal, "failed to get media")
	}
	if m.TripID != tripID || m.PinID == nil || *m.PinID != pinID {
		return nil, status.Error(codes.NotFound, "media not found")
	}
	// Пин не должен остаться без медиа.
	pinMedia, err := s.mediaRepo.ListByPinID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pin media")
	}
	if len(pinMedia) <= 1 {
		return nil, status.Error(codes.FailedPrecondition, "pin must contain at least one media; delete pin instead")
	}
	if err := s.mediaRepo.DeleteByIDs([]string{mediaID}); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete media")
	}
	if s.mediaURLs != nil && m.S3Key != "" {
		_ = s.mediaURLs.DeleteObject(ctx, m.S3Key)
	}
	if err := s.pinRepo.IncMediaCount(pinID, -1); err != nil {
		slog.WarnContext(ctx, "RemoveMediaFromPin: media_count update failed", "pin_id", pinID, "err", err)
	}
	updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pinID)
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	mediaList, _ := s.mediaRepo.ListByPinID(pinID)
	tags, _ := s.tagRepo.GetByPinID(pinID)
	if tags == nil {
		tags = []string{}
	}
	return &pb.RemoveMediaFromPinResponse{Pin: s.pinWithMediaToProto(ctx, pin, mediaList, tags)}, nil
}

// =============================================================================
// helpers
// =============================================================================

// assertParticipantAndPinReady — общий guard: caller — participant, trip в READY,
// pin принадлежит trip. PermissionDenied/NotFound/FailedPrecondition.
func (s *TripService) assertParticipantAndPinReady(ctx context.Context, tripID, pinID, userID string) error {
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Error(codes.NotFound, "trip not found")
		}
		return status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusReady {
		return errWrongStatus(models.TripStatusReady, trip.Status)
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Error(codes.NotFound, "pin not found")
		}
		return status.Error(codes.Internal, "failed to get pin")
	}
	if pin.TripID != tripID {
		return status.Error(codes.NotFound, "pin not found")
	}
	_ = ctx
	return nil
}

// assertActivePinSession — сессия существует, не закрыта, принадлежит этому пину
// и инициатор = caller. Возвращает session для дальнейшего использования.
func (s *TripService) assertActivePinSession(ctx context.Context, tripID, pinID, sessionID, userID string) (*models.PinMediaAdditionSession, error) {
	if s.pinAddSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin media addition repository not configured")
	}
	session, err := s.pinAddSessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinAdditionSessionNotFound) {
			return nil, status.Error(codes.NotFound, "pin media session not found")
		}
		return nil, status.Error(codes.Internal, "failed to load pin media session")
	}
	if session.ClosedAt != nil {
		return nil, status.Error(codes.FailedPrecondition, "pin media session is closed")
	}
	if session.TripID != tripID || session.PinID != pinID {
		return nil, status.Error(codes.PermissionDenied, "session does not belong to this pin")
	}
	if session.InitiatorUserID != userID {
		return nil, status.Error(codes.PermissionDenied, "only session initiator can act on it")
	}
	return session, nil
}

// assertTripCapacity — лимит трипа (ТЗ §1.6: ≤500 media, ≤50 видео) учитывая
// планируемые files. Использует CountByTripID который считает все media трипа,
// включая draft из активных pin-add-сессий — намеренно, чтобы не дать over-upload в S3.
func (s *TripService) assertTripCapacity(tripID string, files []*pb.FileToUpload) error {
	for _, f := range files {
		if !validateContentType(f.GetContentType()) {
			return status.Errorf(codes.InvalidArgument, "unsupported content type: %s", f.GetContentType())
		}
	}
	total, videos, _ := s.mediaRepo.CountByTripID(tripID)
	newVideos := 0
	for _, f := range files {
		if ct := f.GetContentType(); ct == "video/mp4" || ct == "video/quicktime" {
			newVideos++
		}
	}
	if total+len(files) > MaxMediaPerTrip {
		return errLimitExceeded("media", MaxMediaPerTrip, total+len(files))
	}
	if videos+newVideos > MaxVideosPerTrip {
		return errLimitExceeded("video", MaxVideosPerTrip, videos+newVideos)
	}
	return nil
}

// presignPinUploadUrls — выдача presigned PUT URLs (паттерн AddMediaRequestUploadUrls).
// s3_key включает trip_id для упорядоченности; client_id используется как имя файла,
// что достаточно для уникальности в рамках одного клиента.
func (s *TripService) presignPinUploadUrls(ctx context.Context, tripID string, files []*pb.FileToUpload) ([]*pb.UploadUrl, error) {
	uploadUrls := make([]*pb.UploadUrl, 0, len(files))
	for _, f := range files {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + tripID + "/pins/" + f.GetClientId() + ext
		url := ""
		if s.mediaURLs != nil {
			var perr error
			url, perr = s.mediaURLs.PresignedUploadURL(ctx, s3Key, f.GetContentType())
			if perr != nil {
				slog.ErrorContext(ctx, "trip_service: S3 presign upload failed (pin add)", "trip_id", tripID, "client_id", f.GetClientId(), "s3_key", s3Key, "err", perr)
				return nil, status.Error(codes.Internal, "failed to presign upload url")
			}
		}
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key: s3Key,
			Url: url,
		})
	}
	return uploadUrls, nil
}

// collectDeletableSessionMedia — фильтрует ids: только media текущей сессии (pin_id=NULL,
// pin_addition_session_id=$session). Возвращает allowed ids + s3 keys для cleanup.
func (s *TripService) collectDeletableSessionMedia(sessionID string, ids []string) ([]string, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	idSet := map[string]struct{}{}
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	sessionMedia, err := s.mediaRepo.ListByPinAdditionSession(sessionID)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "failed to list session media")
	}
	var allowed []string
	var keys []string
	for _, m := range sessionMedia {
		if _, ok := idSet[m.ID]; !ok {
			continue
		}
		allowed = append(allowed, m.ID)
		if m.S3Key != "" {
			keys = append(keys, m.S3Key)
		}
	}
	return allowed, keys, nil
}

// snapshotToDraftProto собирает PinAdditionDraft proto из снимка и текущего списка media
// (с presigned URLs).
func snapshotToDraftProto(ctx context.Context, s *TripService, snap pinAdditionDraftSnapshot, media []*models.Media) *pb.PinAdditionDraft {
	out := &pb.PinAdditionDraft{
		PinIssues: snap.PinIssues,
		NsfwMediaIds: snap.NSFWMediaIDs,
		DedupedMediaIds: snap.DedupedMediaIDs,
	}
	for _, m := range media {
		out.NewMedia = append(out.NewMedia, &pb.ReviewPinMedia{
			MediaId: m.ID,
			Url: s.presignedReadURL(ctx, m.S3Key),
			PrivacyLevel: m.PrivacyLevel,
		})
	}
	return out
}

func snapshotToSimilarProto(snap pinAdditionDraftSnapshot) []*pb.MediaSimilarGroup {
	out := make([]*pb.MediaSimilarGroup, 0, len(snap.Similar))
	for _, g := range snap.Similar {
		out = append(out, &pb.MediaSimilarGroup{MediaIds: g.MediaIDs})
	}
	return out
}
