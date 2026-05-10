package services

import (
	"context"
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

// pinUploadDraftSnapshot — JSON в pin_upload_sessions.draft_snapshot.
type pinUploadDraftSnapshot struct {
	// Заполнено только для creation (target_pin_id == nil).
	Suggested *pinSuggestedFields `json:"suggested,omitempty"`

	NewMediaIDs     []string                 `json:"new_media_ids"`
	NSFWMediaIDs    []string                 `json:"nsfw_media_ids"`
	DedupedMediaIDs []string                 `json:"deduped_media_ids"`
	PinIssues       []string                 `json:"pin_issues"`
	Similar         []pinUploadSimilarGroup  `json:"similar"`
}

type pinSuggestedFields struct {
	Name               string   `json:"name"`
	Category           string   `json:"category"`
	Tags               []string `json:"tags"`
	Latitude           *float64 `json:"latitude,omitempty"`
	Longitude          *float64 `json:"longitude,omitempty"`
	StartTimeUnix      *int64   `json:"start_time_unix,omitempty"`
	EndTimeUnix        *int64   `json:"end_time_unix,omitempty"`
}

type pinUploadSimilarGroup struct {
	MediaIDs []string `json:"media_ids"`
}

// PinUploadStart: target_pin_id=nil → creation, заполнен → addition. UNIQUE → 409.
func (s *TripService) PinUploadStart(ctx context.Context, req *pb.PinUploadStartRequest) (*pb.PinUploadStartResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	var targetPinID *string
	if req.TargetPinId != nil && *req.TargetPinId != "" {
		v := req.GetTargetPinId()
		targetPinID = &v
	}
	if targetPinID == nil {
		if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
			return nil, err
		}
	} else {
		if err := s.assertParticipantAndPinReady(ctx, tripID, *targetPinID, userID); err != nil {
			return nil, err
		}
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	if s.pinUploadSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin upload session repository not configured")
	}
	sessionID, err := s.pinUploadSessionRepo.Create(ctx, tripID, targetPinID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinUploadSessionActive) {
			return nil, status.Error(codes.FailedPrecondition, "another pin upload session is already active")
		}
		return nil, status.Error(codes.Internal, "failed to create pin upload session")
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	return &pb.PinUploadStartResponse{
		SessionId:  sessionID,
		UploadUrls: uploadUrls,
	}, nil
}

// RequestPinUploadUrls — догрузка presigned URLs.
func (s *TripService) RequestPinUploadUrls(ctx context.Context, req *pb.RequestPinUploadUrlsRequest) (*pb.RequestPinUploadUrlsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	if _, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	if err := s.assertTripCapacity(tripID, files); err != nil {
		return nil, err
	}
	uploadUrls, err := s.presignPinUploadUrls(ctx, tripID, files)
	if err != nil {
		return nil, err
	}
	_ = s.pinUploadSessionRepo.Touch(ctx, sessionID)
	return &pb.RequestPinUploadUrlsResponse{UploadUrls: uploadUrls}, nil
}

// CommitPinUpload — INSERT media с upload_session_id.
func (s *TripService) CommitPinUpload(ctx context.Context, req *pb.CommitPinUploadRequest) (*pb.CommitPinUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if req.GetS3Key() == "" || req.GetMediaType() == "" {
		return nil, status.Error(codes.InvalidArgument, "s3_key and media_type are required")
	}
	if req.GetMediaType() != "image" && req.GetMediaType() != "video" {
		return nil, status.Error(codes.InvalidArgument, "media_type must be 'image' or 'video'")
	}
	if _, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	m := &models.Media{
		TripID:          tripID,
		S3Key:           req.GetS3Key(),
		MediaType:       req.GetMediaType(),
		PrivacyLevel:    trip.PrivacyLevel,
		UploadSessionID: &sessionID,
		UploadedBy:      &userID,
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
	totalAfter, _, err := s.mediaRepo.CommitInUploadSession(ctx, m, sessionID, MaxMediaPerTrip, MaxVideosPerTrip)
	if err != nil {
		if errors.Is(err, repositories.ErrMediaLimitExceeded) {
			return nil, errLimitExceeded("media", MaxMediaPerTrip, totalAfter+1)
		}
		if errors.Is(err, repositories.ErrVideoLimitExceeded) {
			return nil, errLimitExceeded("video", MaxVideosPerTrip, totalAfter+1)
		}
		return nil, status.Error(codes.Internal, "failed to save media")
	}
	mediaInSession, _ := s.mediaRepo.ListByUploadSession(sessionID)
	return &pb.CommitPinUploadResponse{
		MediaId:             m.ID,
		MediaCountInSession: int32(len(mediaInSession)),
	}, nil
}

// ProcessPinUpload — CAS UPLOADING→PROCESSING + AddPinUploadTask в стрим.
func (s *TripService) ProcessPinUpload(ctx context.Context, req *pb.ProcessPinUploadRequest) (*pb.ProcessPinUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	session, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.ProcessingStatus != models.PinUploadProcessingStatusUploading {
		return nil, status.Errorf(codes.FailedPrecondition,
			"pin upload session is in %q state, expected %q",
			session.ProcessingStatus, models.PinUploadProcessingStatusUploading)
	}
	if err := s.pinUploadSessionRepo.SetProcessingStatus(ctx, sessionID,
		models.PinUploadProcessingStatusUploading,
		models.PinUploadProcessingStatusProcessing); err != nil {
		if errors.Is(err, repositories.ErrPinUploadSessionWrongState) {
			return nil, status.Error(codes.FailedPrecondition, "pin upload session already processing")
		}
		return nil, status.Error(codes.Internal, "failed to transition session state")
	}
	if s.eventRepo != nil {
		if perr := s.eventRepo.AddPinUploadTask(ctx, tripID, sessionID, session.TargetPinID, userID); perr != nil {
			return nil, status.Error(codes.Internal, "failed to schedule pin upload processing")
		}
	}
	return &pb.ProcessPinUploadResponse{
		SessionId:        sessionID,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil
}

// GetPinUploadReview — draft заполнен только в READY_FOR_REVIEW.
func (s *TripService) GetPinUploadReview(ctx context.Context, req *pb.GetPinUploadReviewRequest) (*pb.GetPinUploadReviewResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	session, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	resp := &pb.GetPinUploadReviewResponse{
		SessionId:        sessionID,
		ProcessingStatus: session.ProcessingStatus,
	}
	if session.ProcessingStatus == models.PinUploadProcessingStatusReadyForReview {
		var snap pinUploadDraftSnapshot
		if len(session.DraftSnapshot) > 0 {
			_ = json.Unmarshal(session.DraftSnapshot, &snap)
		}
		remaining, _ := s.mediaRepo.ListByUploadSession(sessionID)
		resp.Draft = snapshotToPinUploadDraftProto(ctx, s, snap, remaining)
		resp.Similar = snapshotToPinUploadSimilarProto(snap)
	}
	return resp, nil
}

// FinalizePinUpload — guard READY_FOR_REVIEW; ветвление по target_pin_id.
func (s *TripService) FinalizePinUpload(ctx context.Context, req *pb.FinalizePinUploadRequest) (*pb.FinalizePinUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	session, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.ProcessingStatus != models.PinUploadProcessingStatusReadyForReview {
		return nil, status.Errorf(codes.FailedPrecondition,
			"pin upload session is in %q state, expected %q (call ProcessPinUpload first and wait for completion)",
			session.ProcessingStatus, models.PinUploadProcessingStatusReadyForReview)
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}

	var snap pinUploadDraftSnapshot
	if len(session.DraftSnapshot) > 0 {
		_ = json.Unmarshal(session.DraftSnapshot, &snap)
	}

	// Применяем media_to_delete (только media текущей сессии).
	if del := req.GetMediaToDelete(); len(del) > 0 {
		toDelete, s3Keys, derr := s.collectDeletableUploadSessionMedia(sessionID, del)
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
	remaining, err := s.mediaRepo.ListByUploadSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload session media")
	}

	if session.TargetPinID == nil {
		return s.finalizePinUploadCreation(ctx, trip, session, &snap, remaining, req, userID)
	}
	return s.finalizePinUploadAddition(ctx, session, remaining, req, userID)
}

// CancelPinUpload — orphan-cleanup + close.
func (s *TripService) CancelPinUpload(ctx context.Context, req *pb.CancelPinUploadRequest) (*pb.CancelPinUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID, sessionID := req.GetTripId(), req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	if _, err := s.assertActivePinUploadSession(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	s3Keys, err := s.mediaRepo.DeleteOrphanByUploadSession(sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cleanup orphan media")
	}
	if s.mediaURLs != nil {
		for _, k := range s3Keys {
			_ = s.mediaURLs.DeleteObject(ctx, k)
		}
	}
	if err := s.pinUploadSessionRepo.Close(ctx, sessionID, models.PinUploadSessionCloseReasonCancelled); err != nil {
		if !errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			return nil, status.Error(codes.Internal, "failed to close session")
		}
	}
	return &pb.CancelPinUploadResponse{Status: "cancelled"}, nil
}

// =============================================================================
// helpers
// =============================================================================

func (s *TripService) assertParticipantAndTripReady(ctx context.Context, tripID, userID string) error {
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return status.Error(codes.Internal, "failed to get trip")
	}
	if trip.IsGenerated {
		return errGeneratedReadOnly
	}
	if trip.Status != models.TripStatusReady {
		return errWrongStatus(models.TripStatusReady, trip.Status)
	}
	_ = ctx
	return nil
}

// assertTripCapacity: ≤500 media, ≤50 видео per trip с учётом files.
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

// presignPinUploadUrls — пресайн PUT URLs для загрузки в S3.
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
				slog.ErrorContext(ctx, "trip_service: S3 presign upload failed", "trip_id", tripID, "client_id", f.GetClientId(), "s3_key", s3Key, "err", perr)
				return nil, status.Error(codes.Internal, "failed to presign upload url")
			}
		}
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key:    s3Key,
			Url:      url,
		})
	}
	return uploadUrls, nil
}

// assertActivePinUploadSession: сессия активна, принадлежит трипу, caller — инициатор;
// participant + trip-ready (или pin-ready при target_pin_id) проверяется внутри.
func (s *TripService) assertActivePinUploadSession(ctx context.Context, tripID, sessionID, userID string) (*models.PinUploadSession, error) {
	if s.pinUploadSessionRepo == nil {
		return nil, status.Error(codes.Internal, "pin upload session repository not configured")
	}
	session, err := s.pinUploadSessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			return nil, status.Error(codes.NotFound, "pin upload session not found")
		}
		return nil, status.Error(codes.Internal, "failed to load pin upload session")
	}
	if session.ClosedAt != nil {
		return nil, status.Error(codes.FailedPrecondition, "pin upload session is closed")
	}
	if session.TripID != tripID {
		return nil, status.Error(codes.PermissionDenied, "session does not belong to this trip")
	}
	if session.InitiatorUserID != userID {
		return nil, status.Error(codes.PermissionDenied, "only session initiator can act on it")
	}
	if session.TargetPinID == nil {
		if err := s.assertParticipantAndTripReady(ctx, tripID, userID); err != nil {
			return nil, err
		}
	} else {
		if err := s.assertParticipantAndPinReady(ctx, tripID, *session.TargetPinID, userID); err != nil {
			return nil, err
		}
	}
	return session, nil
}

func (s *TripService) collectDeletableUploadSessionMedia(sessionID string, ids []string) ([]string, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	idSet := map[string]struct{}{}
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	sessionMedia, err := s.mediaRepo.ListByUploadSession(sessionID)
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

func snapshotToPinUploadDraftProto(ctx context.Context, s *TripService, snap pinUploadDraftSnapshot, media []*models.Media) *pb.PinUploadDraft {
	out := &pb.PinUploadDraft{
		PinIssues:       snap.PinIssues,
		NsfwMediaIds:    snap.NSFWMediaIDs,
		DedupedMediaIds: snap.DedupedMediaIDs,
	}
	if snap.Suggested != nil {
		out.Suggested = &pb.PinSuggestedFields{
			Name:     snap.Suggested.Name,
			Category: snap.Suggested.Category,
			Tags:     snap.Suggested.Tags,
		}
		if snap.Suggested.Latitude != nil {
			out.Suggested.Latitude = snap.Suggested.Latitude
		}
		if snap.Suggested.Longitude != nil {
			out.Suggested.Longitude = snap.Suggested.Longitude
		}
		if snap.Suggested.StartTimeUnix != nil {
			out.Suggested.StartTimeUnix = snap.Suggested.StartTimeUnix
		}
		if snap.Suggested.EndTimeUnix != nil {
			out.Suggested.EndTimeUnix = snap.Suggested.EndTimeUnix
		}
	}
	for _, m := range media {
		out.Media = append(out.Media, &pb.ReviewPinMedia{
			MediaId:      m.ID,
			Url:          s.presignedReadURL(ctx, m.S3Key),
			PrivacyLevel: m.PrivacyLevel,
		})
	}
	return out
}

func snapshotToPinUploadSimilarProto(snap pinUploadDraftSnapshot) []*pb.MediaSimilarGroup {
	out := make([]*pb.MediaSimilarGroup, 0, len(snap.Similar))
	for _, g := range snap.Similar {
		out = append(out, &pb.MediaSimilarGroup{MediaIds: g.MediaIDs})
	}
	return out
}
