package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

const defaultInviteExpiresInSec = 86400 // 24 hours

// сколько ведущий может не делать мутаций в DRAFT_FINAL_REVIEW, прежде чем
// следующий participant может перехватить роль неявно (первое мутирующее действие).
// Cron порог 72 часа для abandoned-sessions живёт в worker/session_cleanup.go.
const initiatorTakeoverTimeout = time.Hour

type TripService struct {
	pb.UnimplementedTripServiceServer
	tripRepo repositories.TripRepositoryInterface
	participantRepo repositories.TripParticipantRepositoryInterface
	inviteRepo repositories.InvitationLinkRepositoryInterface
	settingsRepo repositories.TripSettingsRepositoryInterface
	eventRepo repositories.TripEventPublisher
	mediaRepo repositories.MediaRepositoryInterface
	mediaURLs MediaURLResolver
	pinRepo repositories.PinRepositoryInterface
	tagRepo repositories.TagRepositoryInterface
	socialRepo repositories.SocialRepositoryInterface
	favouriteRepo repositories.FavouriteRepositoryInterface
	geoRepo repositories.GeoRegistryRepositoryInterface
	addMediaSessionRepo repositories.AddMediaSessionRepositoryInterface
	battleRepo repositories.MediaBattleRepositoryInterface
	tripPrivacyRepo repositories.TripPrivacyRepositoryInterface
	pinPrivacyRepo repositories.PinPrivacyRepositoryInterface
	mediaPrivacyRepo repositories.MediaPrivacyRepositoryInterface
	pinHiddenRepo repositories.PinHiddenRepositoryInterface
	pinAddSessionRepo repositories.PinMediaAdditionSessionRepositoryInterface
	pinCreationSessionRepo repositories.PinCreationSessionRepositoryInterface
}

func NewTripService(
	tripRepo repositories.TripRepositoryInterface,
	participantRepo repositories.TripParticipantRepositoryInterface,
	inviteRepo repositories.InvitationLinkRepositoryInterface,
	settingsRepo repositories.TripSettingsRepositoryInterface,
	eventRepo repositories.TripEventPublisher,
	mediaRepo repositories.MediaRepositoryInterface,
	mediaURLs MediaURLResolver,
	pinRepo repositories.PinRepositoryInterface,
	tagRepo repositories.TagRepositoryInterface,
	socialRepo repositories.SocialRepositoryInterface,
	favouriteRepo repositories.FavouriteRepositoryInterface,
	geoRepo repositories.GeoRegistryRepositoryInterface,
	addMediaSessionRepo repositories.AddMediaSessionRepositoryInterface,
	battleRepo repositories.MediaBattleRepositoryInterface,
	tripPrivacyRepo repositories.TripPrivacyRepositoryInterface,
	pinPrivacyRepo repositories.PinPrivacyRepositoryInterface,
	mediaPrivacyRepo repositories.MediaPrivacyRepositoryInterface,
	pinHiddenRepo repositories.PinHiddenRepositoryInterface,
	pinAddSessionRepo repositories.PinMediaAdditionSessionRepositoryInterface,
	pinCreationSessionRepo repositories.PinCreationSessionRepositoryInterface,
) *TripService {
	return &TripService{
		tripRepo: tripRepo,
		participantRepo: participantRepo,
		inviteRepo: inviteRepo,
		settingsRepo: settingsRepo,
		eventRepo: eventRepo,
		mediaRepo: mediaRepo,
		mediaURLs: mediaURLs,
		pinRepo: pinRepo,
		tagRepo: tagRepo,
		socialRepo: socialRepo,
		favouriteRepo: favouriteRepo,
		geoRepo: geoRepo,
		addMediaSessionRepo: addMediaSessionRepo,
		battleRepo: battleRepo,
		tripPrivacyRepo: tripPrivacyRepo,
		pinPrivacyRepo: pinPrivacyRepo,
		mediaPrivacyRepo: mediaPrivacyRepo,
		pinHiddenRepo: pinHiddenRepo,
		pinAddSessionRepo: pinAddSessionRepo,
		pinCreationSessionRepo: pinCreationSessionRepo,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	// Use authenticated user as owner; ignore request owner for security.
	name := req.GetName()
	if len(name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if len(name) > MaxNameLength {
		return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
	}
	if len(req.GetDescription()) > MaxDescriptionLength {
		return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
	}
	category := req.GetCategory()
	if category == "" || !validateCategory(category) {
		return nil, status.Error(codes.InvalidArgument, "category must be one of: Отпуск, Командировка, Выходные, Активный отдых, Образование, Другое")
	}
	season := req.GetSeason()
	if season == "" || !validateSeason(season) {
		return nil, status.Error(codes.InvalidArgument, "season must be one of: Зима, Весна, Лето, Осень")
	}
	for _, f := range req.GetFilesToUpload() {
		if !validateContentType(f.GetContentType()) {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported content type: %s", f.GetContentType())
		}
	}

	tripStatus := "Created"
	if len(req.GetFilesToUpload()) > 0 {
		tripStatus = "UPLOADING"
	}
	// privacy_level — DEFAULT 'Private' в БД; ставим явно для надёжности.
	// Per-user уровень хранится в trip_privacy и меняется через UpsertTripPrivacy (ТЗ 6.4).
	trip := &models.Trip{
		OwnerUserID: userID,
		Name: name,
		Description: req.GetDescription(),
		Category: category,
		Season: season,
		Status: tripStatus,
		PrivacyLevel: "Private",
		LikesCount: 0,
		DislikesCount: 0,
		IsPublished: false,
		IsGenerated: false,
	}
	if err := s.tripRepo.Create(trip); err != nil {
		return nil, status.Error(codes.Internal, "failed to create trip")
	}
	participant := &models.TripParticipant{TripID: trip.ID, UserID: userID, IsAdmin: true}
	if err := s.participantRepo.Add(participant); err != nil {
		return nil, status.Error(codes.Internal, "failed to add owner as participant")
	}
	// дефолтный per-user уровень владельца — Private (ТЗ 6.6: один Private → Private).
	if s.tripPrivacyRepo != nil {
		if err := s.tripPrivacyRepo.Upsert(ctx, trip.ID, userID, "Private"); err != nil {
			slog.WarnContext(ctx, "trip_service: default trip_privacy upsert failed", "trip_id", trip.ID, "user_id", userID, "err", err)
		}
	}
	uploadUrls := make([]*pb.UploadUrl, 0, len(req.GetFilesToUpload()))
	for _, f := range req.GetFilesToUpload() {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + trip.ID + "/" + f.GetClientId() + ext
		url := ""
		if s.mediaURLs != nil {
			var err error
			url, err = s.mediaURLs.PresignedUploadURL(ctx, s3Key, f.GetContentType())
			if err != nil {
				slog.Error("trip_service: S3 presign upload failed", "trip_id", trip.ID, "client_id", f.GetClientId(), "s3_key", s3Key, "err", err)
				return nil, status.Error(codes.Internal, "failed to presign upload url")
			}
		}
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key: s3Key,
			Url: url,
		})
	}
	return &pb.CreateTripResponse{
		TripId: trip.ID,
		Status: trip.Status,
		UploadUrls: uploadUrls,
	}, nil
}

func (s *TripService) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	ok, err = s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if ok {
		resp, err := s.getTripResponseWithPins(ctx, trip, userID)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	// allow access if user has trip in favourites (e.g. after soft delete)
	hasFav, err := s.favouriteRepo.HasFavourite(userID, tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check favourite")
	}
	if hasFav {
		resp, err := s.getTripResponseWithPins(ctx, trip, userID)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	// ТЗ 3.4: любой залогиненный пользователь может открыть опубликованный трип
	// по share-ссылке. Отдаём read-only view: только публичные пины (выбранные
	// при публикации) и публичные медиа в них; participants/settings/active-сессии
	// не передаются — они для участников.
	if trip.IsPublished {
		return s.getSharedTripResponse(ctx, trip)
	}
	return nil, status.Error(codes.PermissionDenied, "not a participant")
}

// getSharedTripResponse собирает публичную read-only выборку опубликованного трипа
// для не-участника (ТЗ 3.4 + 6.1). В выборку попадают только пины с
// is_published_in_feed=true и privacy_level=Public; в каждом пине — только медиа
// с privacy_level=Public.
func (s *TripService) getSharedTripResponse(ctx context.Context, trip *models.Trip) (*pb.GetTripResponse, error) {
	pins, err := s.pinRepo.ListByTripID(trip.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	tagsByPin, _ := s.tagRepo.GetByTripID(trip.ID)
	if tagsByPin == nil {
		tagsByPin = make(map[string][]string)
	}
	outPins := make([]*pb.TripPin, 0, len(pins))
	for _, pin := range pins {
		if !pin.IsPublishedInFeed || pin.PrivacyLevel != "Public" {
			continue
		}
		mediaList, err := s.mediaRepo.ListByPinID(pin.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list pin media")
		}
		publicMedia := make([]*models.Media, 0, len(mediaList))
		for _, m := range mediaList {
			if m.PrivacyLevel == "Public" {
				publicMedia = append(publicMedia, m)
			}
		}
		tags := tagsByPin[pin.ID]
		if tags == nil {
			tags = []string{}
		}
		outPins = append(outPins, s.pinWithMediaToProto(ctx, pin, publicMedia, tags))
	}
	return &pb.GetTripResponse{
		Trip: s.tripToProto(ctx, trip),
		Pins: outPins,
	}, nil
}

// getTripResponseWithPins builds GetTripResponse with trip, pins (each pin with its media),
// participants (с per-user privacy_level и ролью) и current_user_settings (notifications +
// per-user privacy_level вызывающего юзера). также догружает active_add_media_session —
// клиент использует его для роутинга на экран сессии без дополнительных запросов.
func (s *TripService) getTripResponseWithPins(ctx context.Context, trip *models.Trip, callerUserID string) (*pb.GetTripResponse, error) {
	// ТЗ 4.5.2: пины, скрытые caller'ом через pin_hidden_by_user (soft-delete-for-self),
	// не возвращаются. Если pinHiddenRepo не подключён (старый DI/тесты), fallback —
	// обычный ListByTripID.
	var pins []*models.Pin
	var err error
	if s.pinHiddenRepo != nil && callerUserID != "" {
		pins, err = s.pinRepo.ListByTripIDExcludingHidden(trip.ID, callerUserID)
	} else {
		pins, err = s.pinRepo.ListByTripID(trip.ID)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	tagsByPin, _ := s.tagRepo.GetByTripID(trip.ID)
	if tagsByPin == nil {
		tagsByPin = make(map[string][]string)
	}
	outPins := make([]*pb.TripPin, 0, len(pins))
	for _, pin := range pins {
		mediaList, err := s.mediaRepo.ListByPinID(pin.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list pin media")
		}
		tags := tagsByPin[pin.ID]
		if tags == nil {
			tags = []string{}
		}
		outPins = append(outPins, s.pinWithMediaToProto(ctx, pin, mediaList, tags))
	}
	activeProto, err := s.buildActiveSessionProto(ctx, trip.ID)
	if err != nil {
		return nil, err
	}
	participants, settings := s.buildParticipantsAndSettings(ctx, trip.ID, callerUserID)
	return &pb.GetTripResponse{
		Trip: s.tripToProto(ctx, trip),
		Pins: outPins,
		ActiveAddMediaSession: activeProto,
		Participants: participants,
		CurrentUserSettings: settings,
	}, nil
}

// buildParticipantsAndSettings — участники трипа с per-user privacy_level/role и
// per-user настройки вызывающего юзера. Read-only сборка не должна валить GetTrip:
// при ошибке любой из подсистем возвращаем пустой список / дефолты, лог в Warn.
// Default privacy_level — "Private" (соответствует AggregatePrivacyLevel(empty)).
func (s *TripService) buildParticipantsAndSettings(ctx context.Context, tripID, callerUserID string) ([]*pb.TripParticipant, *pb.TripSettings) {
	const defaultPrivacy = "Private"
	settings := &pb.TripSettings{NotificationsEnabled: true, PrivacyLevel: defaultPrivacy}

	parts, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		slog.WarnContext(ctx, "trip_service: GetByTripID participants failed", "trip_id", tripID, "err", err)
		parts = nil
	}

	levelByUser := map[string]string{}
	if s.tripPrivacyRepo != nil {
		entries, err := s.tripPrivacyRepo.GetByTripID(ctx, tripID)
		if err != nil {
			slog.WarnContext(ctx, "trip_service: GetByTripID trip_privacy failed", "trip_id", tripID, "err", err)
		} else {
			for _, e := range entries {
				levelByUser[e.UserID] = e.PrivacyLevel
			}
		}
	}

	out := make([]*pb.TripParticipant, 0, len(parts))
	for _, p := range parts {
		if p == nil {
			continue
		}
		level := levelByUser[p.UserID]
		if level == "" {
			level = defaultPrivacy
		}
		role := "member"
		if p.IsAdmin {
			role = "admin"
		}
		out = append(out, &pb.TripParticipant{
			UserId: p.UserID,
			PrivacyLevel: level,
			Role: role,
		})
	}

	if callerUserID != "" {
		if level := levelByUser[callerUserID]; level != "" {
			settings.PrivacyLevel = level
		}
		if s.settingsRepo != nil {
			settingsMap, err := s.settingsRepo.GetByTripAndUsers(tripID, []string{callerUserID})
			if err != nil {
				slog.WarnContext(ctx, "trip_service: GetByTripAndUsers failed", "trip_id", tripID, "user_id", callerUserID, "err", err)
			} else if v, ok := settingsMap[callerUserID]; ok {
				settings.NotificationsEnabled = v
			}
		}
	}

	return out, settings
}

// buildActiveSessionProto — в GetTrip отдаём активную add-media сессию как метаданные.
// Возвращает nil, если активной сессии нет, или если репозиторий не инициализирован
// (graceful degradation — основной GetTrip не должен падать из-за add-media).
// media_count_in_session считается как total_media − existing_media_ids.
func (s *TripService) buildActiveSessionProto(ctx context.Context, tripID string) (*pb.ActiveAddMediaSession, error) {
	if s.addMediaSessionRepo == nil {
		return nil, nil
	}
	active, err := s.addMediaSessionRepo.GetActive(ctx, tripID)
	if err != nil {
		if errors.Is(err, repositories.ErrNoActiveSession) {
			return nil, nil
		}
		slog.WarnContext(ctx, "buildActiveSessionProto: GetActive failed", "trip_id", tripID, "err", err)
		return nil, nil
	}
	if s.mediaRepo == nil {
		return &pb.ActiveAddMediaSession{SessionId: active.SessionID}, nil
	}
	total, _, err := s.mediaRepo.CountByTripID(tripID)
	if err != nil {
		slog.WarnContext(ctx, "buildActiveSessionProto: CountByTripID failed", "trip_id", tripID, "err", err)
		total = 0
	}
	mediaInSession := int32(total - len(active.ExistingMediaIDs))
	if mediaInSession < 0 {
		mediaInSession = 0
	}
	out := &pb.ActiveAddMediaSession{
		SessionId: active.SessionID,
		MediaCountInSession: mediaInSession,
	}
	if active.CurrentInitiatorUserID != nil {
		v := *active.CurrentInitiatorUserID
		out.CurrentInitiatorUserId = &v
	}
	if active.InitiatorAssignedAt != nil {
		ts := active.InitiatorAssignedAt.Unix()
		out.InitiatorAssignedAtUnix = &ts
		takeover := active.InitiatorAssignedAt.Add(initiatorTakeoverTimeout).Unix()
		out.TakeoverAvailableAtUnix = &takeover
	}
	return out, nil
}

// validateActiveSession проверяет, что переданный session_id соответствует активной
// add-media сессии трипа. Возвращает саму сессию или FailedPrecondition (api-gateway
// маппит в 410 Gone для устаревшего session_id).
func (s *TripService) validateActiveSession(ctx context.Context, tripID, sessionID string) (*models.AddMediaSession, error) {
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	active, err := s.addMediaSessionRepo.GetActive(ctx, tripID)
	if err != nil {
		if errors.Is(err, repositories.ErrNoActiveSession) {
			return nil, errNoActiveSession(tripID)
		}
		return nil, status.Error(codes.Internal, "failed to load session")
	}
	if active.SessionID != sessionID {
		return nil, errSessionStale(tripID, sessionID, active.SessionID)
	}
	return active, nil
}

// publishTripStatusChanged — хелпер для . Публикует TRIP_STATUS_CHANGED
// всем participant'ам через per-trip WS-канал. Reason помогает клиенту определить
// конкретный переход (add_media_started, add_media_rollback, add_media_processing и т.п.).
func (s *TripService) publishTripStatusChanged(ctx context.Context, tripID, newStatus, reason string) {
	if s.eventRepo == nil || s.participantRepo == nil {
		return
	}
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		slog.WarnContext(ctx, "publishTripStatusChanged: GetByTripID failed", "trip_id", tripID, "err", err)
		return
	}
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventTripStatusChanged, map[string]interface{}{
		"trip_id": tripID,
		"new_status": newStatus,
		"reason": reason,
	})
}

// ensureInitiator — проверяет, что мутация в ADD_MEDIA_DRAFT_FINAL_REVIEW
// разрешена userID'у. Модель ведущего:
// - userID == current_initiator → OK (таймер обновляется вызывающей стороной через Touch).
// - current_initiator == nil → FailedPrecondition (Apply ещё не делали).
// - now()-initiator_assigned_at > initiatorTakeoverTimeout → неявный перехват:
// переназначаем ведущего на userID, публикуем WS ADD_MEDIA_INITIATOR_CHANGED,
// возвращаем обновлённую сессию.
// - иначе → PermissionDenied (api-gateway маппит в 403 для клиента).
func (s *TripService) ensureInitiator(ctx context.Context, tripID, sessionID, userID string) (*models.AddMediaSession, error) {
	active, err := s.validateActiveSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	if active.CurrentInitiatorUserID == nil {
		return nil, errNotInitiator("", time.Time{})
	}
	if *active.CurrentInitiatorUserID == userID {
		return active, nil
	}
	if active.InitiatorAssignedAt == nil {
		return nil, errNotInitiator(*active.CurrentInitiatorUserID, time.Time{})
	}
	takeoverAt := active.InitiatorAssignedAt.Add(initiatorTakeoverTimeout)
	if time.Since(*active.InitiatorAssignedAt) < initiatorTakeoverTimeout {
		return nil, errNotInitiator(*active.CurrentInitiatorUserID, takeoverAt)
	}
	// Неявный перехват — fallback для Confirm/Cancel, если клиент срезал угол
	// и не вызвал явный /takeover. Штатный UI-флоу: сначала /takeover, потом мутация.
	now, err := s.executeTakeover(ctx, tripID, sessionID, *active.CurrentInitiatorUserID, userID)
	if err != nil {
		return nil, err
	}
	newUser := userID
	newAt := now
	active.CurrentInitiatorUserID = &newUser
	active.InitiatorAssignedAt = &newAt
	return active, nil
}

// executeTakeover выполняет SetInitiator + публикацию ADD_MEDIA_INITIATOR_CHANGED.
// Используется и из ensureInitiator (неявный fallback на мутации), и из
// AddMediaTakeover (явный запрос пользователя).
func (s *TripService) executeTakeover(ctx context.Context, tripID, sessionID, previousUserID, newUserID string) (time.Time, error) {
	now := time.Now()
	if err := s.addMediaSessionRepo.SetInitiator(ctx, sessionID, newUserID, now); err != nil {
		return time.Time{}, status.Error(codes.Internal, "failed to reassign initiator")
	}
	if s.eventRepo != nil && s.participantRepo != nil {
		participants, _ := s.participantRepo.GetByTripID(tripID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		takeoverAt := now.Add(initiatorTakeoverTimeout).Unix()
		_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventAddMediaInitiatorChanged, map[string]interface{}{
			"session_id": sessionID,
			"previous_initiator_user_id": previousUserID,
			"current_initiator_user_id": newUserID,
			"takeover_available_at_unix": takeoverAt,
		})
	}
	return now, nil
}

func (s *TripService) pinWithMediaToProto(ctx context.Context, pin *models.Pin, mediaList []*models.Media, tags []string) *pb.TripPin {
	out := &pb.TripPin{
		Id: pin.ID,
		TripId: pin.TripID,
		Name: pin.Name,
		Description: pin.Description,
		Category: pin.Category,
		PrivacyLevel: pin.PrivacyLevel,
		Tags: tags,
	}
	if pin.Latitude != nil {
		out.Latitude = pin.Latitude
	}
	if pin.Longitude != nil {
		out.Longitude = pin.Longitude
	}
	if pin.StartTime != nil {
		out.StartTimeUnix = pin.StartTime.Unix()
	}
	if pin.EndTime != nil {
		out.EndTimeUnix = pin.EndTime.Unix()
	}
	for _, m := range mediaList {
		pm := &pb.TripPinMedia{
			MediaId: m.ID,
			Url: s.presignedReadURL(ctx, m.S3Key),
			MediaType: m.MediaType,
			PrivacyLevel: m.PrivacyLevel,
		}
		if m.CapturedAt != nil {
			pm.CapturedAtUnix = m.CapturedAt.Unix()
		}
		out.Media = append(out.Media, pm)
	}
	return out
}

func (s *TripService) ListUserTrips(ctx context.Context, req *pb.ListUserTripsRequest) (*pb.ListUserTripsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}
	trips, err := s.tripRepo.ListByUserID(userID, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list trips")
	}
	out := make([]*pb.Trip, len(trips))
	for i, t := range trips {
		out[i] = s.tripToProto(ctx, t)
	}
	return &pb.ListUserTripsResponse{Trips: out}, nil
}

// ListUserTripSummaries — лёгкая сводка для API Gateway при сборке статистики профиля.
// Возвращает все трипы, где user — участник, без пагинации; только id + counts.
func (s *TripService) ListUserTripSummaries(ctx context.Context, req *pb.ListUserTripSummariesRequest) (*pb.ListUserTripSummariesResponse, error) {
	authUserID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	userID := req.GetUserId()
	if userID == "" {
		userID = authUserID
	}
	summaries, err := s.tripRepo.ListSummariesByUserID(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list trip summaries")
	}
	out := make([]*pb.TripSummary, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, &pb.TripSummary{
			TripId: s.TripID,
			PinsCount: s.PinsCount,
			MediaCount: s.MediaCount,
		})
	}
	return &pb.ListUserTripSummariesResponse{Trips: out}, nil
}

// MaxSearchQueryLength ограничивает длину текстового поиска, чтобы защитить БД от очень больших ILIKE-паттернов.
const MaxSearchQueryLength = 128

// SearchPins — текстовый поиск пинов по name/description/тегам среди трипов, где user — участник.
func (s *TripService) SearchPins(ctx context.Context, req *pb.SearchPinsRequest) (*pb.SearchPinsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}
	if len(query) > MaxSearchQueryLength {
		query = query[:MaxSearchQueryLength]
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}
	pins, err := s.pinRepo.SearchByUserID(userID, query, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to search pins")
	}
	out := make([]*pb.TripPin, 0, len(pins))
	for _, pin := range pins {
		mediaList, err := s.mediaRepo.ListByPinID(pin.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list pin media")
		}
		tags, err := s.tagRepo.GetByPinID(pin.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list pin tags")
		}
		if tags == nil {
			tags = []string{}
		}
		out = append(out, s.pinWithMediaToProto(ctx, pin, mediaList, tags))
	}
	return &pb.SearchPinsResponse{Pins: out}, nil
}

func (s *TripService) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	// ТЗ 3.2: редактировать параметры путешествия может любой участник
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	// Merge optional fields
	if req.Name != nil {
		name := *req.Name
		if len(name) > MaxNameLength {
			return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
		}
		trip.Name = name
	}
	if req.Description != nil {
		if len(*req.Description) > MaxDescriptionLength {
			return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
		}
		trip.Description = *req.Description
	}
	if req.Category != nil {
		if !validateCategory(*req.Category) {
			return nil, status.Error(codes.InvalidArgument, "invalid category")
		}
		trip.Category = *req.Category
	}
	if req.Season != nil {
		if !validateSeason(*req.Season) {
			return nil, status.Error(codes.InvalidArgument, "invalid season")
		}
		trip.Season = *req.Season
	}
	if req.StartDateUnix != nil {
		t := time.Unix(*req.StartDateUnix, 0)
		trip.StartDate = &t
	}
	if req.EndDateUnix != nil {
		t := time.Unix(*req.EndDateUnix, 0)
		trip.EndDate = &t
	}
	if err := s.tripRepo.Update(trip); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to update trip")
	}
	updated, _ := s.tripRepo.GetByID(tripID)
	return &pb.UpdateTripResponse{Trip: s.tripToProto(ctx, updated)}, nil
}

// RequestTripCoverUpload выдаёт presigned PUT URL для загрузки обложки в S3 (step 1 двухшагового потока, аналог аватара пользователя).
// ТЗ 3.2: обложка — редактируемый параметр; доступ — любому участнику.
func (s *TripService) RequestTripCoverUpload(ctx context.Context, req *pb.RequestTripCoverUploadRequest) (*pb.RequestTripCoverUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	filename := req.GetFilename()
	if tripID == "" || filename == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and filename are required")
	}
	if _, err := s.tripRepo.GetByID(tripID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic":
	default:
		return nil, status.Error(codes.InvalidArgument, "cover must be .jpg, .jpeg, .png or .heic")
	}

	if s.mediaURLs == nil {
		return nil, status.Error(codes.Unavailable, "cover upload is not configured")
	}
	s3Key := fmt.Sprintf("trips/%s/cover/%s%s", tripID, uuid.NewString(), ext)
	uploadURL, err := s.mediaURLs.PresignedUploadURL(ctx, s3Key, req.GetContentType())
	if err != nil {
		slog.ErrorContext(ctx, "RequestTripCoverUpload: presign", "trip_id", tripID, "s3_key", s3Key, "error", err)
		return nil, status.Error(codes.Internal, "failed to generate upload URL")
	}
	return &pb.RequestTripCoverUploadResponse{UploadUrl: uploadURL, S3Key: s3Key}, nil
}

// ConfirmTripCoverUpload сохраняет новый cover_url после успешной загрузки в S3 (step 2).
// Старый объект в S3 удаляется best-effort, чтобы не оставлять мусор.
func (s *TripService) ConfirmTripCoverUpload(ctx context.Context, req *pb.ConfirmTripCoverUploadRequest) (*pb.ConfirmTripCoverUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	s3Key := req.GetS3Key()
	if tripID == "" || s3Key == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and s3_key are required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}

	if trip.CoverURL != "" && trip.CoverURL != s3Key && s.mediaURLs != nil {
		if err := s.mediaURLs.DeleteObject(ctx, trip.CoverURL); err != nil {
			slog.ErrorContext(ctx, "ConfirmTripCoverUpload: delete old cover (best-effort)", "trip_id", tripID, "key", trip.CoverURL, "error", err)
		}
	}
	if err := s.tripRepo.UpdateCoverURL(tripID, s3Key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		slog.ErrorContext(ctx, "ConfirmTripCoverUpload: update cover_url", "trip_id", tripID, "error", err)
		return nil, status.Error(codes.Internal, "failed to update cover")
	}
	updated, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload trip")
	}
	return &pb.ConfirmTripCoverUploadResponse{Trip: s.tripToProto(ctx, updated)}, nil
}

// DeleteTripCover удаляет обложку: best-effort чистит объект в S3 и очищает cover_url.
func (s *TripService) DeleteTripCover(ctx context.Context, req *pb.DeleteTripCoverRequest) (*pb.DeleteTripCoverResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}

	if trip.CoverURL != "" && s.mediaURLs != nil {
		if err := s.mediaURLs.DeleteObject(ctx, trip.CoverURL); err != nil {
			slog.ErrorContext(ctx, "DeleteTripCover: s3 delete (best-effort)", "trip_id", tripID, "key", trip.CoverURL, "error", err)
		}
	}
	if err := s.tripRepo.UpdateCoverURL(tripID, ""); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		slog.ErrorContext(ctx, "DeleteTripCover: clear cover_url", "trip_id", tripID, "error", err)
		return nil, status.Error(codes.Internal, "failed to delete cover")
	}
	updated, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload trip")
	}
	return &pb.DeleteTripCoverResponse{Trip: s.tripToProto(ctx, updated)}, nil
}

// DeleteTrip — только админ. (ТЗ 3.24.1/3.24.2): если трип в избранном у других — soft delete (удаление из списка участников); иначе — полное удаление.
func (s *TripService) DeleteTrip(ctx context.Context, req *pb.DeleteTripRequest) (*pb.DeleteTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	isAdmin, err := s.participantRepo.IsAdmin(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check admin")
	}
	if !isAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admin can delete trip")
	}
	// 3.24: если трип в избранном у других пользователей — soft delete
	inOthersFav, err := s.favouriteRepo.HasFavouritesByOtherUsers(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check favourites")
	}
	if inOthersFav {
		if err := s.participantRepo.RemoveAllByTripID(tripID); err != nil {
			return nil, status.Error(codes.Internal, "failed to remove participants")
		}
		if err := s.tripRepo.SetSoftDeleted(tripID); err != nil {
			return nil, status.Error(codes.Internal, "failed to soft delete trip")
		}
		return &pb.DeleteTripResponse{Success: true}, nil
	}
	// Полное удаление: выходим админа, если 0 участников — удаляем трип, иначе назначаем нового админа
	if err := s.participantRepo.Remove(tripID, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to leave trip")
	}
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list participants")
	}
	if len(participants) == 0 {
		if err := s.tripRepo.Delete(tripID); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete trip")
		}
		if s.eventRepo != nil {
			// TRIP_DELETED — для cleanup trip_locations_mirror в statistics-service.
			_ = s.eventRepo.PublishStatsEvent(ctx, "TRIP_DELETED", tripID, nil, nil)
			// Убираем per-trip WS-stream, чтобы не копить orphan-ключи в Redis.
			_ = s.eventRepo.DeleteTripEventStream(ctx, tripID)
		}
		return &pb.DeleteTripResponse{Success: true}, nil
	}
	if err := s.participantRepo.SetAdmin(tripID, participants[0].UserID); err != nil {
		return nil, status.Error(codes.Internal, "failed to assign new admin")
	}
	return &pb.DeleteTripResponse{Success: true}, nil
}

// GenerateInviteLink — только участники трипа могут генерировать ссылку (ТЗ 3.18).
func (s *TripService) GenerateInviteLink(ctx context.Context, req *pb.GenerateInviteLinkRequest) (*pb.GenerateInviteLinkResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "only participants can generate invite link")
	}
	expiresIn := req.GetExpiresInSeconds()
	if expiresIn <= 0 {
		expiresIn = defaultInviteExpiresInSec
	}
	if expiresIn > 30*24*3600 {
		expiresIn = 30 * 24 * 3600
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	token := uuid.New().String()
	link := &models.InvitationLink{
		ID: uuid.New().String(),
		TripID: tripID,
		Token: token,
		ExpiresAt: expiresAt,
	}
	if err := s.inviteRepo.Create(link); err != nil {
		return nil, status.Error(codes.Internal, "failed to create invite link")
	}
	return &pb.GenerateInviteLinkResponse{
		InviteLinkId: link.ID,
		Token: token,
		InviteUrl: "", // клиент/gateway собирает URL по token
		ExpiresAtUnix: expiresAt.Unix(),
	}, nil
}

// JoinTripByToken — добавление в trip_participants по токену инвайта, событие PARTICIPANT_JOINED.
func (s *TripService) JoinTripByToken(ctx context.Context, req *pb.JoinTripByTokenRequest) (*pb.JoinTripByTokenResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	token := req.GetToken()
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	link, err := s.inviteRepo.GetByToken(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "invite link not found or expired")
		}
		return nil, status.Error(codes.Internal, "failed to get invite link")
	}
	if time.Now().After(link.ExpiresAt) {
		return nil, status.Error(codes.FailedPrecondition, "invite link expired")
	}
	already, err := s.participantRepo.IsParticipant(link.TripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if already {
		return &pb.JoinTripByTokenResponse{TripId: link.TripID, AlreadyJoined: true}, nil
	}
	if _, err := s.tripRepo.GetByID(link.TripID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant := &models.TripParticipant{TripID: link.TripID, UserID: userID, IsAdmin: false}
	if err := s.participantRepo.Add(participant); err != nil {
		return nil, status.Error(codes.Internal, "failed to add participant")
	}
	_ = s.settingsRepo.EnsureDefaultSettings(link.TripID, userID)
	// дефолтный per-user уровень присоединившегося — Private (ТЗ 6.6). Пересчитываем
	// агрегат сразу: новый Private может опустить уровень всего трипа.
	if s.tripPrivacyRepo != nil {
		if err := s.tripPrivacyRepo.Upsert(ctx, link.TripID, userID, "Private"); err != nil {
			slog.WarnContext(ctx, "trip_service: default trip_privacy upsert failed on join", "trip_id", link.TripID, "user_id", userID, "err", err)
		} else {
			if entries, err := s.tripPrivacyRepo.GetByTripID(ctx, link.TripID); err == nil {
				if trip, err := s.tripRepo.GetByID(link.TripID); err == nil {
					effective := repositories.AggregatePrivacyLevel(trip.PrivacyLevel, entries)
					if effective != trip.PrivacyLevel {
						_ = s.tripRepo.SetPrivacyLevel(link.TripID, effective)
					}
				}
			}
			if s.eventRepo != nil {
				_ = s.eventRepo.PublishPrivacyEvent(ctx, "trip", link.TripID, link.TripID, userID, "Private")
			}
		}
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "PARTICIPANT_JOINED", link.TripID, userID)
	}
	return &pb.JoinTripByTokenResponse{TripId: link.TripID, AlreadyJoined: false}, nil
}

// RemoveParticipant — только админ может удалить участника (ТЗ 3.19), событие PARTICIPANT_REMOVED.
func (s *TripService) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*pb.RemoveParticipantResponse, error) {
	callerID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	targetUserID := req.GetUserId()
	if tripID == "" || targetUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and user_id are required")
	}
	if targetUserID == callerID {
		return nil, status.Error(codes.InvalidArgument, "use LeaveTrip to leave yourself")
	}
	isAdmin, err := s.participantRepo.IsAdmin(tripID, callerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check admin")
	}
	if !isAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admin can remove participant")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, targetUserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !isParticipant {
		return nil, status.Error(codes.NotFound, "user is not a participant")
	}
	if err := s.participantRepo.Remove(tripID, targetUserID); err != nil {
		return nil, status.Error(codes.Internal, "failed to remove participant")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "PARTICIPANT_REMOVED", tripID, targetUserID)
	}
	return &pb.RemoveParticipantResponse{Success: true}, nil
}

// LeaveTrip — любой участник выходит (ТЗ 3.20). Если единственный админ — удаляем трип (3.21);
// иначе назначаем нового админа (3.22), событие PARTICIPANT_LEFT или ADMIN_CHANGED.
func (s *TripService) LeaveTrip(ctx context.Context, req *pb.LeaveTripRequest) (*pb.LeaveTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	wasAdmin, err := s.participantRepo.IsAdmin(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check admin")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if err := s.participantRepo.Remove(tripID, userID); err != nil {
		return nil, status.Error(codes.Internal, "failed to leave trip")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "PARTICIPANT_LEFT", tripID, userID)
	}
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		return &pb.LeaveTripResponse{Success: true, TripDeleted: false}, nil
	}
	if len(participants) == 0 {
		_ = s.tripRepo.Delete(tripID)
		return &pb.LeaveTripResponse{Success: true, TripDeleted: true}, nil
	}
	if wasAdmin {
		if err := s.participantRepo.SetAdmin(tripID, participants[0].UserID); err != nil {
			return nil, status.Error(codes.Internal, "failed to assign new admin")
		}
		if s.eventRepo != nil {
			_ = s.eventRepo.PublishTripEvent(ctx, "ADMIN_CHANGED", tripID, participants[0].UserID)
		}
	}
	return &pb.LeaveTripResponse{Success: true, TripDeleted: false}, nil
}

// TransferAdmin — только текущий админ может передать права (ТЗ 3.22.1), событие ADMIN_CHANGED.
func (s *TripService) TransferAdmin(ctx context.Context, req *pb.TransferAdminRequest) (*pb.TransferAdminResponse, error) {
	callerID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	newAdminID := req.GetNewAdminUserId()
	if tripID == "" || newAdminID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and new_admin_user_id are required")
	}
	isAdmin, err := s.participantRepo.IsAdmin(tripID, callerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check admin")
	}
	if !isAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admin can transfer admin")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, newAdminID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.InvalidArgument, "new admin must be a participant")
	}
	if newAdminID == callerID {
		return &pb.TransferAdminResponse{Success: true}, nil
	}
	if err := s.participantRepo.SetAdmin(tripID, newAdminID); err != nil {
		return nil, status.Error(codes.Internal, "failed to transfer admin")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishTripEvent(ctx, "ADMIN_CHANGED", tripID, newAdminID)
	}
	return &pb.TransferAdminResponse{Success: true}, nil
}

func (s *TripService) ProcessMediaGrouping(ctx context.Context, req *pb.ProcessMediaGroupingRequest) (*pb.ProcessMediaGroupingResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if trip.Status != "UPLOADING" && trip.Status != "Created" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in UPLOADING or Created to process media grouping")
	}
	total, videos, _ := s.mediaRepo.CountByTripID(tripID)
	newVideos := 0
	for _, m := range req.GetMedia() {
		if m.GetMediaType() == "video" {
			newVideos++
		}
	}
	if total+len(req.GetMedia()) > MaxMediaPerTrip {
		return nil, status.Errorf(codes.InvalidArgument, "trip may have at most %d media", MaxMediaPerTrip)
	}
	if videos+newVideos > MaxVideosPerTrip {
		return nil, status.Errorf(codes.InvalidArgument, "trip may have at most %d videos", MaxVideosPerTrip)
	}
	// Save media to DB (pin_id = nil)
	for _, meta := range req.GetMedia() {
		media := &models.Media{
			TripID: tripID,
			S3Key: meta.GetS3Key(),
			MediaType: meta.GetMediaType(),
			BattleRating: 0,
			PrivacyLevel: trip.PrivacyLevel,
		}
		if meta.CapturedAtUnix != 0 {
			t := time.Unix(meta.GetCapturedAtUnix(), 0)
			media.CapturedAt = &t
		}
		if meta.Latitude != nil && meta.Longitude != nil {
			lat, lon := meta.GetLatitude(), meta.GetLongitude()
			media.Latitude = &lat
			media.Longitude = &lon
		}
		if err := s.mediaRepo.Create(media); err != nil {
			return nil, status.Error(codes.Internal, "failed to save media")
		}
	}
	// Cluster and build draft_pins (PostGIS + time grouping)
	draftPins := clusterMediaToDraftPins(s.mediaRepo, tripID)
	if err := s.tripRepo.SetStatus(tripID, "DRAFT_GROUPING_REVIEW"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	// Build response: draft_pins with media URLs (stub)
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	mediaByID := make(map[string]*models.Media)
	for _, m := range mediaList {
		mediaByID[m.ID] = m
	}
	respPins := make([]*pb.DraftPin, 0, len(draftPins))
	for i, group := range draftPins {
		draftPinID := fmt.Sprintf("cluster-%d", i)
		if i == len(draftPins)-1 && len(group) > 0 {
			if m := mediaByID[group[0]]; m != nil && m.Latitude == nil && m.CapturedAt == nil {
				draftPinID = "draft-unassigned"
			}
		}
		dp := &pb.DraftPin{DraftPinId: draftPinID}
		for _, mediaID := range group {
			m := mediaByID[mediaID]
			if m == nil {
				continue
			}
			dp.Media = append(dp.Media, &pb.DraftPinMedia{
				MediaId: m.ID,
				Url: s.presignedReadURL(ctx, m.S3Key),
				Type: m.MediaType,
			})
		}
		respPins = append(respPins, dp)
	}
	return &pb.ProcessMediaGroupingResponse{
		TripId: tripID,
		Status: "DRAFT_GROUPING_REVIEW",
		DraftPins: respPins,
	}, nil
}

func (s *TripService) ApplyGroupsAndProcess(ctx context.Context, req *pb.ApplyGroupsAndProcessRequest) (*pb.ApplyGroupsAndProcessResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != "DRAFT_GROUPING_REVIEW" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in DRAFT_GROUPING_REVIEW")
	}
	// Delete rejected media (DB then best-effort S3)
	if len(req.GetDeletedMediaIds()) > 0 {
		allowedIDs, s3Keys, err := s.resolveMediaDeletionsForTrip(tripID, req.GetDeletedMediaIds())
		if err != nil {
			return nil, err
		}
		if err := s.mediaRepo.DeleteByIDs(allowedIDs); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete media")
		}
		if s.mediaURLs != nil {
			for _, key := range s3Keys {
				_ = s.mediaURLs.DeleteObject(ctx, key)
			}
		}
	}
	// Create one pin per draft_pin and assign media
	for _, dp := range req.GetDraftPins() {
		if len(dp.GetMediaIds()) == 0 {
			continue
		}
		pin := &models.Pin{
			TripID: tripID,
			Name: "Pin",
			Description: "",
			Category: trip.Category,
			PrivacyLevel: trip.PrivacyLevel,
			MediaCount: int32(len(dp.GetMediaIds())),
		}
		if err := s.pinRepo.Create(pin); err != nil {
			return nil, status.Error(codes.Internal, "failed to create pin")
		}
		if err := s.mediaRepo.UpdatePinIDByIDs(dp.GetMediaIds(), pin.ID); err != nil {
			return nil, status.Error(codes.Internal, "failed to assign media to pin")
		}
		// Compute start_time/end_time from media (first/last by captured_at)
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	}
	// STUB: ML-пайплайн ещё не реализован, поэтому сразу переводим трип в
	// DRAFT_FINAL_REVIEW и публикуем TRIP_PROCESSING_COMPLETED.
	// TODO: вернуть AddMLTask, когда воркер pinz:trip:ml:tasks заработает в проде.
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusProcessing); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	s.finalizeProcessingStub(ctx, tripID, models.TripStatusDraftFinalReview)
	return &pb.ApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status: models.TripStatusProcessing,
	}, nil
}

// finalizeProcessingStub заменяет настоящий ML-воркер: синхронно переводит трип
// в targetStatus (DRAFT_FINAL_REVIEW или ADD_MEDIA_DRAFT_FINAL_REVIEW) и публикует
// TRIP_PROCESSING_COMPLETED подписчикам WS (канал pinz:trip:{id}:events + per-user
// каналы). Удалить вместе с TODO в ApplyGroupsAndProcess/AddMediaApplyGroupsAndProcess,
// когда воркер заработает.
func (s *TripService) finalizeProcessingStub(ctx context.Context, tripID, targetStatus string) {
	if err := s.tripRepo.SetStatus(tripID, targetStatus); err != nil {
		slog.ErrorContext(ctx, "finalizeProcessingStub: SetStatus failed", "trip_id", tripID, "error", err)
		return
	}
	if s.eventRepo == nil || s.participantRepo == nil {
		return
	}
	// Очищаем per-trip WS-stream от backfill предыдущих processing-сессий
	// (например, свежий add-media-flow на трипе, который ранее уже проходил
	// creation). Иначе подписчик на XREAD "0-0" получил бы чужое старое
	// TRIP_PROCESSING_COMPLETED ещё до того, как текущий publish произошёл.
	_ = s.eventRepo.DeleteTripEventStream(ctx, tripID)
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		slog.WarnContext(ctx, "finalizeProcessingStub: GetByTripID failed", "trip_id", tripID, "error", err)
		return
	}
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	if err := s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventTripProcessingCompleted, map[string]interface{}{
		"trip_id": tripID,
		"status": targetStatus,
	}); err != nil {
		slog.WarnContext(ctx, "finalizeProcessingStub: PublishTripEventWS failed", "trip_id", tripID, "error", err)
	}
}

func (s *TripService) GetTripReview(ctx context.Context, req *pb.GetTripReviewRequest) (*pb.GetTripReviewResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != "DRAFT_FINAL_REVIEW" && trip.Status != "PROCESSING" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in DRAFT_FINAL_REVIEW or PROCESSING for review")
	}
	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	tagsByPin, _ := s.tagRepo.GetByTripID(tripID)
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	// similar — двумерный массив: группы медиа с одинаковым similar_group_id по всему трипу (без привязки к пину).
	groupIDToIDs := make(map[string][]string)
	for _, m := range mediaList {
		if m.SimilarGroupID != nil {
			groupIDToIDs[*m.SimilarGroupID] = append(groupIDToIDs[*m.SimilarGroupID], m.ID)
		}
	}
	var similar []*pb.MediaSimilarGroup
	for _, ids := range groupIDToIDs {
		if len(ids) >= 2 {
			similar = append(similar, &pb.MediaSimilarGroup{MediaIds: ids})
		}
	}
	respPins := make([]*pb.ReviewPin, 0, len(pins))
	for _, pin := range pins {
		var pinMedia []*models.Media
		for _, m := range mediaList {
			if m.PinID != nil && *m.PinID == pin.ID {
				pinMedia = append(pinMedia, m)
			}
		}
		issues := []string{}
		if pin.Latitude == nil || pin.Longitude == nil {
			issues = append(issues, "MISSING_COORDINATES")
		}
		if pin.StartTime == nil || pin.EndTime == nil {
			issues = append(issues, "MISSING_DATES")
		}
		tags := tagsByPin[pin.ID]
		if tags == nil {
			tags = []string{}
		}
		reviewMedia := make([]*pb.ReviewPinMedia, 0, len(pinMedia))
		for _, m := range pinMedia {
			reviewMedia = append(reviewMedia, &pb.ReviewPinMedia{
				MediaId: m.ID,
				Url: s.presignedReadURL(ctx, m.S3Key),
				PrivacyLevel: m.PrivacyLevel,
			})
		}
		var startUnix, endUnix int64
		if pin.StartTime != nil {
			startUnix = pin.StartTime.Unix()
		}
		if pin.EndTime != nil {
			endUnix = pin.EndTime.Unix()
		}
		rp := &pb.ReviewPin{
			PinId: pin.ID,
			Name: pin.Name,
			Category: pin.Category,
			LocationName: pin.LocationName,
			StartTimeUnix: startUnix,
			EndTimeUnix: endUnix,
			Issues: issues,
			Tags: tags,
			Media: reviewMedia,
		}
		if pin.Latitude != nil {
			rp.Latitude = pin.Latitude
		}
		if pin.Longitude != nil {
			rp.Longitude = pin.Longitude
		}
		respPins = append(respPins, rp)
	}
	return &pb.GetTripReviewResponse{
		TripId: tripID,
		Status: trip.Status,
		Similar: similar,
		Pins: respPins,
	}, nil
}

func (s *TripService) FinalizeTrip(ctx context.Context, req *pb.FinalizeTripRequest) (*pb.FinalizeTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != "DRAFT_FINAL_REVIEW" && trip.Status != "PROCESSING" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in DRAFT_FINAL_REVIEW or PROCESSING to finalize")
	}
	if err := s.applyReviewEdits(ctx, trip, req.GetPinUpdates(), req.GetMediaToDelete()); err != nil {
		return nil, err
	}
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusReady); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	return &pb.FinalizeTripResponse{
		TripId:  tripID,
		Status:  models.TripStatusReady,
		Message: "Trip finalized",
	}, nil
}

// applyReviewEdits — общая логика финализации ревью (creation FinalizeTrip и
// add-media Confirm): применяет pin_updates (name, lat/lon), удаляет
// media_to_delete (ТЗ 3.15.1 — «похожие»), агрегирует cover_url + start/end
// dates по состоянию пинов/медиа. Reverse geocoding запускается асинхронно
// через PIN_LOCATIONS_REQUESTED для всех пинов с обновлёнными координатами
// (vkr.txt §2.5.4). Статус трипа меняет вызывающая сторона.
func (s *TripService) applyReviewEdits(ctx context.Context, trip *models.Trip, pinUpdates []*pb.PinUpdate, mediaToDelete []string) error {
	tripID := trip.ID
	geoPins := make([]repositories.GeoRequestPin, 0, len(pinUpdates))
	for _, pu := range pinUpdates {
		pin, err := s.pinRepo.GetByID(pu.GetPinId())
		if err != nil {
			continue
		}
		if pu.Name != nil {
			name := pu.GetName()
			if len(name) > MaxNameLength {
				return status.Errorf(codes.InvalidArgument, "pin name must be at most %d characters", MaxNameLength)
			}
			pin.Name = name
		}
		if pu.Latitude != nil && pu.Longitude != nil {
			pin.Latitude = pu.Latitude
			pin.Longitude = pu.Longitude
		}
		_ = s.pinRepo.Update(pin)
		if pu.Latitude != nil && pu.Longitude != nil {
			geoPins = append(geoPins, repositories.GeoRequestPin{
				PinID:     pin.ID,
				Latitude:  *pin.Latitude,
				Longitude: *pin.Longitude,
			})
		}
	}
	if len(geoPins) > 0 && s.eventRepo != nil {
		_ = s.eventRepo.PublishGeoRequest(ctx, tripID, geoPins)
	}
	// Удаление media из БД + best-effort S3 cleanup.
	if len(mediaToDelete) > 0 {
		allowedIDs, s3Keys, err := s.resolveMediaDeletionsForTrip(tripID, mediaToDelete)
		if err != nil {
			return err
		}
		if err := s.mediaRepo.DeleteByIDs(allowedIDs); err != nil {
			return status.Error(codes.Internal, "failed to delete media")
		}
		if s.mediaURLs != nil {
			for _, key := range s3Keys {
				_ = s.mediaURLs.DeleteObject(ctx, key)
			}
		}
	}
	// Агрегация: cover_url (первое image-медиа), start/end dates по пинам.
	// Presigned URL обложки резолвится в tripToProto по сохранённому S3 key.
	pins, _ := s.pinRepo.ListByTripID(tripID)
	var minStart, maxEnd *time.Time
	var coverURL string
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	for _, m := range mediaList {
		if m.PinID == nil {
			continue
		}
		if m.MediaType == "image" {
			coverURL = m.S3Key
		}
		break
	}
	for _, p := range pins {
		if p.StartTime != nil {
			if minStart == nil || p.StartTime.Before(*minStart) {
				minStart = p.StartTime
			}
		}
		if p.EndTime != nil {
			if maxEnd == nil || p.EndTime.After(*maxEnd) {
				maxEnd = p.EndTime
			}
		}
	}
	// Обложку в add-media не перезаписываем, если трип уже имел её. В creation-флоу
	// coverURL будет пустой до этого момента — поэтому условная запись ок для обоих.
	if coverURL != "" && trip.CoverURL == "" {
		trip.CoverURL = coverURL
	}
	trip.StartDate = minStart
	trip.EndDate = maxEnd
	_ = s.tripRepo.Update(trip)
	return nil
}

// AddMediaStart — ТЗ 5.3 → 3.8: старт сессии добавления медиа в готовый трип.
// идемпотентна — если сессия уже существует (race), возвращает её session_id
// и joined=true без upload_urls. Клиент в этом случае использует AddMediaRequestUploadUrls.
func (s *TripService) AddMediaStart(ctx context.Context, req *pb.AddMediaStartRequest) (*pb.AddMediaStartResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	// если сессия уже активна — мягкое присоединение (B1).
	if active, err := s.addMediaSessionRepo.GetActive(ctx, tripID); err == nil {
		return &pb.AddMediaStartResponse{
			SessionId: active.SessionID,
			Status: trip.Status,
			UploadUrls: nil,
			Joined: true,
		}, nil
	} else if !errors.Is(err, repositories.ErrNoActiveSession) {
		return nil, status.Error(codes.Internal, "failed to check active session")
	}
	if trip.Status != models.TripStatusReady {
		return nil, errWrongStatus(models.TripStatusReady, trip.Status)
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	for _, f := range files {
		if !validateContentType(f.GetContentType()) {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported content type: %s", f.GetContentType())
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
		return nil, errLimitExceeded("media", MaxMediaPerTrip, total+len(files))
	}
	if videos+newVideos > MaxVideosPerTrip {
		return nil, errLimitExceeded("video", MaxVideosPerTrip, videos+newVideos)
	}
	existing, err := s.mediaRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list existing media")
	}
	existingIDs := make([]string, 0, len(existing))
	for _, m := range existing {
		existingIDs = append(existingIDs, m.ID)
	}
	sessionID, err := s.addMediaSessionRepo.Create(ctx, tripID, existingIDs)
	if err != nil {
		// ON CONFLICT DO NOTHING → sql.ErrNoRows. Значит между нашими
		// GetActive и Create другой participant успел создать сессию — подхватываем её.
		if errors.Is(err, sql.ErrNoRows) {
			active, gerr := s.addMediaSessionRepo.GetActive(ctx, tripID)
			if gerr != nil {
				return nil, status.Error(codes.Internal, "failed to load race-created session")
			}
			return &pb.AddMediaStartResponse{
				SessionId: active.SessionID,
				Status: models.TripStatusAddMediaUploading,
				UploadUrls: nil,
				Joined: true,
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to create add-media session")
	}
	uploadUrls := make([]*pb.UploadUrl, 0, len(files))
	for _, f := range files {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + tripID + "/" + f.GetClientId() + ext
		url := ""
		if s.mediaURLs != nil {
			var perr error
			url, perr = s.mediaURLs.PresignedUploadURL(ctx, s3Key, f.GetContentType())
			if perr != nil {
				slog.Error("trip_service: S3 presign upload failed (add-media)", "trip_id", tripID, "client_id", f.GetClientId(), "s3_key", s3Key, "err", perr)
				return nil, status.Error(codes.Internal, "failed to presign upload url")
			}
		}
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key: s3Key,
			Url: url,
		})
	}
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusAddMediaUploading); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	s.publishTripStatusChanged(ctx, tripID, models.TripStatusAddMediaUploading, "add_media_started")
	return &pb.AddMediaStartResponse{
		SessionId: sessionID,
		Status: models.TripStatusAddMediaUploading,
		UploadUrls: uploadUrls,
		Joined: false,
	}, nil
}

// AddMediaRequestUploadUrls — выдаёт presigned URLs для догрузки файлов в уже
// активную сессию. Состояние не меняет; валидирует session_id и лимиты.
func (s *TripService) AddMediaRequestUploadUrls(ctx context.Context, req *pb.AddMediaRequestUploadUrlsRequest) (*pb.AddMediaRequestUploadUrlsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.validateActiveSession(ctx, tripID, sessionID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaUploading {
		return nil, errWrongStatus(models.TripStatusAddMediaUploading, trip.Status)
	}
	files := req.GetFilesToUpload()
	if len(files) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files_to_upload is required")
	}
	for _, f := range files {
		if !validateContentType(f.GetContentType()) {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported content type: %s", f.GetContentType())
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
		return nil, errLimitExceeded("media", MaxMediaPerTrip, total+len(files))
	}
	if videos+newVideos > MaxVideosPerTrip {
		return nil, errLimitExceeded("video", MaxVideosPerTrip, videos+newVideos)
	}
	uploadUrls := make([]*pb.UploadUrl, 0, len(files))
	for _, f := range files {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + tripID + "/" + f.GetClientId() + ext
		url := ""
		if s.mediaURLs != nil {
			var perr error
			url, perr = s.mediaURLs.PresignedUploadURL(ctx, s3Key, f.GetContentType())
			if perr != nil {
				slog.Error("trip_service: S3 presign upload failed (add-media request)", "trip_id", tripID, "client_id", f.GetClientId(), "s3_key", s3Key, "err", perr)
				return nil, status.Error(codes.Internal, "failed to presign upload url")
			}
		}
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key: s3Key,
			Url: url,
		})
	}
	return &pb.AddMediaRequestUploadUrlsResponse{UploadUrls: uploadUrls}, nil
}

// AddMediaCommitUpload — клиент зовёт после каждого успешного PUT в S3.
// Сервер создаёт media entry без pin_id, публикует WS ADD_MEDIA_PROGRESS.
func (s *TripService) AddMediaCommitUpload(ctx context.Context, req *pb.AddMediaCommitUploadRequest) (*pb.AddMediaCommitUploadResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.validateActiveSession(ctx, tripID, sessionID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaUploading {
		return nil, errWrongStatus(models.TripStatusAddMediaUploading, trip.Status)
	}
	s3Key := req.GetS3Key()
	mediaType := req.GetMediaType()
	if s3Key == "" || mediaType == "" {
		return nil, status.Error(codes.InvalidArgument, "s3_key and media_type are required")
	}
	m := &models.Media{
		TripID: tripID,
		S3Key: s3Key,
		MediaType: mediaType,
		PrivacyLevel: trip.PrivacyLevel,
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
	// Атомарная транзакция: advisory lock по session_id → COUNT → limit check →
	// INSERT → Touch сессии. Сериализует параллельные commit'ы на одной сессии,
	// без гонки между читаемым COUNT и вставкой. Лимит 500 строгий.
	totalAfter, _, err := s.mediaRepo.CommitInSession(ctx, m, sessionID, MaxMediaPerTrip, MaxVideosPerTrip)
	if err != nil {
		if errors.Is(err, repositories.ErrMediaLimitExceeded) {
			return nil, errLimitExceeded("media", MaxMediaPerTrip, totalAfter+1)
		}
		if errors.Is(err, repositories.ErrVideoLimitExceeded) {
			return nil, errLimitExceeded("video", MaxVideosPerTrip, totalAfter+1)
		}
		return nil, status.Error(codes.Internal, "failed to save media")
	}
	mediaCount := int32(totalAfter)
	remaining := int32(MaxMediaPerTrip) - mediaCount
	if remaining < 0 {
		remaining = 0
	}
	if s.eventRepo != nil {
		participants, _ := s.participantRepo.GetByTripID(tripID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		url := s.presignedReadURL(ctx, s3Key)
		_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventAddMediaProgress, map[string]interface{}{
			"action": "uploaded",
			"actor_user_id": userID,
			"media_count": mediaCount,
			"session_id": sessionID,
			"media": map[string]interface{}{
				"media_id": m.ID,
				"media_type": m.MediaType,
				"url": url,
			},
		})
	}
	return &pb.AddMediaCommitUploadResponse{
		MediaId: m.ID,
		MediaCountInSession: mediaCount - int32(len(existingIDsFromSession(ctx, s.addMediaSessionRepo, sessionID))),
		RemainingSlots: remaining,
	}, nil
}

// existingIDsFromSession — утилита для расчёта media_count_in_session в ответе commit-upload.
// Возвращает длину existing_media_ids (медиа, которые были до старта сессии), либо 0 при ошибке.
func existingIDsFromSession(ctx context.Context, repo repositories.AddMediaSessionRepositoryInterface, sessionID string) []string {
	ids, _, err := repo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		return nil
	}
	return ids
}

// AddMediaGetSessionMedia — снимок медиа сессии (свои и чужие).
// Используется participant'ом, который вошёл в сессию посередине — нужен для отрисовки
// уже загруженных медиа, прежде чем подпишется на WS ADD_MEDIA_PROGRESS.
func (s *TripService) AddMediaGetSessionMedia(ctx context.Context, req *pb.AddMediaGetSessionMediaRequest) (*pb.AddMediaGetSessionMediaResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	active, err := s.validateActiveSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	all, err := s.mediaRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list media")
	}
	existing := make(map[string]struct{}, len(active.ExistingMediaIDs))
	for _, id := range active.ExistingMediaIDs {
		existing[id] = struct{}{}
	}
	out := make([]*pb.SessionMedia, 0, len(all))
	for _, m := range all {
		if _, wasBefore := existing[m.ID]; wasBefore {
			continue
		}
		url := s.presignedReadURL(ctx, m.S3Key)
		actor := ""
		if m.UploadedBy != nil {
			actor = *m.UploadedBy
		}
		out = append(out, &pb.SessionMedia{
			MediaId: m.ID,
			Url: url,
			Type: m.MediaType,
			ActorUserId: actor,
			UploadedAtUnix: m.CreatedAt.Unix(),
		})
	}
	return &pb.AddMediaGetSessionMediaResponse{
		SessionId: sessionID,
		Media: out,
		MediaCountInSession: int32(len(out)),
	}, nil
}

// AddMediaProcessGrouping — кластеризация добавленных медиа
// с использованием существующих пинов как seed-групп (ТЗ 5.3.1-5.3.2).
//
// media[] больше не принимается — сервер берёт уже закоммиченные через
// AddMediaCommitUpload записи из БД. Флаг add_more=true на статусе GROUPING_REVIEW
// откатывает обратно в UPLOADING, чтобы participant докинул ещё файлов.
func (s *TripService) AddMediaProcessGrouping(ctx context.Context, req *pb.AddMediaProcessGroupingRequest) (*pb.AddMediaProcessGroupingResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.validateActiveSession(ctx, tripID, sessionID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	// откат GROUPING_REVIEW → UPLOADING.
	if req.GetAddMore() {
		if trip.Status != models.TripStatusAddMediaGroupingReview {
			return nil, errWrongStatus(models.TripStatusAddMediaGroupingReview, trip.Status)
		}
		if err := s.tripRepo.SetStatus(tripID, models.TripStatusAddMediaUploading); err != nil {
			return nil, status.Error(codes.Internal, "failed to update trip status")
		}
		if err := s.addMediaSessionRepo.Touch(ctx, sessionID, time.Now()); err != nil {
			slog.WarnContext(ctx, "addMediaProcessGrouping rollback: touch failed", "session_id", sessionID, "err", err)
		}
		if s.eventRepo != nil {
			participants, _ := s.participantRepo.GetByTripID(tripID)
			userIDs := make([]string, 0, len(participants))
			for _, p := range participants {
				userIDs = append(userIDs, p.UserID)
			}
			_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventAddMediaProgress, map[string]interface{}{
				"action": "rollback",
				"actor_user_id": userID,
				"session_id": sessionID,
			})
		}
		s.publishTripStatusChanged(ctx, tripID, models.TripStatusAddMediaUploading, "add_media_rollback")
		existingIDs, _, _ := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
		return &pb.AddMediaProcessGroupingResponse{
			TripId: tripID,
			SessionId: sessionID,
			Status: models.TripStatusAddMediaUploading,
			DraftPins: nil,
			ExistingMediaIds: existingIDs,
		}, nil
	}
	if trip.Status != models.TripStatusAddMediaUploading {
		return nil, errWrongStatus(models.TripStatusAddMediaUploading, trip.Status)
	}
	respPins, existingIDs, err := s.buildDraftPinsForSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusAddMediaGroupingReview); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	if err := s.addMediaSessionRepo.Touch(ctx, sessionID, time.Now()); err != nil {
		slog.WarnContext(ctx, "addMediaProcessGrouping: touch failed", "session_id", sessionID, "err", err)
	}
	s.publishTripStatusChanged(ctx, tripID, models.TripStatusAddMediaGroupingReview, "add_media_grouping")
	return &pb.AddMediaProcessGroupingResponse{
		TripId: tripID,
		SessionId: sessionID,
		Status: models.TripStatusAddMediaGroupingReview,
		DraftPins: respPins,
		ExistingMediaIds: existingIDs,
	}, nil
}

// AddMediaGetGrouping — снимок draft_pins для экрана GROUPING_REVIEW.
// Вычисляется той же функцией clusterMediaWithExistingPinsAsSeeds, что и в
// AddMediaProcessGrouping, но без изменения статуса. Используется participant'ом,
// который вошёл в сессию посередине.
func (s *TripService) AddMediaGetGrouping(ctx context.Context, req *pb.AddMediaGetGroupingRequest) (*pb.AddMediaGetGroupingResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.validateActiveSession(ctx, tripID, sessionID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaGroupingReview {
		return nil, errWrongStatus(models.TripStatusAddMediaGroupingReview, trip.Status)
	}
	respPins, existingIDs, err := s.buildDraftPinsForSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	return &pb.AddMediaGetGroupingResponse{
		TripId: tripID,
		SessionId: sessionID,
		DraftPins: respPins,
		ExistingMediaIds: existingIDs,
	}, nil
}

// buildDraftPinsForSession — общая для ProcessGrouping и GetGrouping сборка draft_pins.
// Вычисляется детерминированно через clusterMediaWithExistingPinsAsSeeds по тем же
// данным в БД, так что повторные вызовы дают один и тот же ответ.
func (s *TripService) buildDraftPinsForSession(ctx context.Context, tripID, sessionID string) ([]*pb.DraftPin, []string, error) {
	groups := clusterMediaWithExistingPinsAsSeeds(s.mediaRepo, s.pinRepo, tripID)
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	mediaByID := make(map[string]*models.Media, len(mediaList))
	for _, m := range mediaList {
		mediaByID[m.ID] = m
	}
	respPins := make([]*pb.DraftPin, 0, len(groups))
	for _, g := range groups {
		dp := &pb.DraftPin{DraftPinId: g.DraftPinID}
		for _, mediaID := range g.MediaIDs {
			m := mediaByID[mediaID]
			if m == nil {
				continue
			}
			dp.Media = append(dp.Media, &pb.DraftPinMedia{
				MediaId: m.ID,
				Url: s.presignedReadURL(ctx, m.S3Key),
				Type: m.MediaType,
			})
		}
		respPins = append(respPins, dp)
	}
	existingIDs, _, err := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "failed to load session")
	}
	return respPins, existingIDs, nil
}

// AddMediaApplyGroupsAndProcess — ТЗ 5.3.3-5.3.4: применение групп и запуск ML-обработки для добавленных медиа.
// Существующие медиа защищены от удаления/перемещения; ML worker получает flow="add_media" + new_pin_ids для пропуска тегов/категорий у существующих пинов.
func (s *TripService) AddMediaApplyGroupsAndProcess(ctx context.Context, req *pb.AddMediaApplyGroupsAndProcessRequest) (*pb.AddMediaApplyGroupsAndProcessResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if _, err := s.validateActiveSession(ctx, tripID, sessionID); err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaGroupingReview {
		return nil, errWrongStatus(models.TripStatusAddMediaGroupingReview, trip.Status)
	}
	existingIDs, _, err := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load session")
	}
	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}
	// ТЗ 5.3.3: запрещено удалять исходные медиа — фильтруем deleted_media_ids.
	if ids := req.GetDeletedMediaIds(); len(ids) > 0 {
		filtered := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, isExisting := existingSet[id]; !isExisting {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) > 0 {
			allowedIDs, s3Keys, err := s.resolveMediaDeletionsForTrip(tripID, filtered)
			if err != nil {
				return nil, err
			}
			if err := s.mediaRepo.DeleteByIDs(allowedIDs); err != nil {
				return nil, status.Error(codes.Internal, "failed to delete media")
			}
			if s.mediaURLs != nil {
				for _, key := range s3Keys {
					_ = s.mediaURLs.DeleteObject(ctx, key)
				}
			}
		}
	}
	// Применяем группы. Для "existing-{pin_id}" — только новые медиа добавляются к существующему пину (исходные не трогаем, ТЗ 5.3.3).
	// Для остальных (cluster-N / draft-unassigned) — создаём новый пин.
	newPinIDs := make([]string, 0)
	touchedPinIDs := make(map[string]struct{})
	for _, dp := range req.GetDraftPins() {
		draftID := dp.GetDraftPinId()
		mediaIDs := dp.GetMediaIds()
		if len(mediaIDs) == 0 {
			continue
		}
		if len(draftID) > len("existing-") && draftID[:len("existing-")] == "existing-" {
			existingPinID := draftID[len("existing-"):]
			pin, perr := s.pinRepo.GetByID(existingPinID)
			if perr != nil {
				continue
			}
			// Фильтруем исходные медиа из набора (они не должны переназначаться — ТЗ 5.3.3).
			newOnly := make([]string, 0, len(mediaIDs))
			for _, id := range mediaIDs {
				if _, isExisting := existingSet[id]; !isExisting {
					newOnly = append(newOnly, id)
				}
			}
			if len(newOnly) == 0 {
				continue
			}
			if err := s.mediaRepo.UpdatePinIDByIDs(newOnly, pin.ID); err != nil {
				return nil, status.Error(codes.Internal, "failed to assign media to existing pin")
			}
			touchedPinIDs[pin.ID] = struct{}{}
		} else {
			// Новый пин — пропускаем исходные медиа из mediaIDs (они должны остаться в своих пинах).
			newOnly := make([]string, 0, len(mediaIDs))
			for _, id := range mediaIDs {
				if _, isExisting := existingSet[id]; !isExisting {
					newOnly = append(newOnly, id)
				}
			}
			if len(newOnly) == 0 {
				continue
			}
			pin := &models.Pin{
				TripID: tripID,
				Name: "Pin",
				Description: "",
				Category: trip.Category,
				PrivacyLevel: trip.PrivacyLevel,
				MediaCount: int32(len(newOnly)),
			}
			if err := s.pinRepo.Create(pin); err != nil {
				return nil, status.Error(codes.Internal, "failed to create pin")
			}
			if err := s.mediaRepo.UpdatePinIDByIDs(newOnly, pin.ID); err != nil {
				return nil, status.Error(codes.Internal, "failed to assign media to new pin")
			}
			newPinIDs = append(newPinIDs, pin.ID)
			touchedPinIDs[pin.ID] = struct{}{}
		}
	}
	for pinID := range touchedPinIDs {
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pinID)
	}
	// переход в новый add-media-статус, назначение ведущего.
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusAddMediaProcessing); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	if err := s.addMediaSessionRepo.SetInitiator(ctx, sessionID, userID, time.Now()); err != nil {
		slog.ErrorContext(ctx, "AddMediaApplyGroupsAndProcess: SetInitiator failed", "session_id", sessionID, "err", err)
	}
	s.publishTripStatusChanged(ctx, tripID, models.TripStatusAddMediaProcessing, "add_media_processing")
	// уведомляем notification-service о добавленных пинах. userID —
	// автор добавления медиа; notification-service адресует пуш остальным
	// участникам трипа. Одно событие на весь add-media запрос — избегаем
	// флуда, если пользователь добавил сразу несколько pin'ов.
	if s.eventRepo != nil && len(newPinIDs) > 0 {
		_ = s.eventRepo.PublishTripEvent(ctx, "PIN_ADDED", tripID, userID)
	}
	// STUB: ML-пайплайн пока не реализован, поэтому SetMLContext/AddMLTaskWithFlow
	// не нужны — сразу двигаем трип в ADD_MEDIA_DRAFT_FINAL_REVIEW через общий стаб.
	// TODO: вернуть оригинальный enqueue, когда воркер ml:tasks заработает в проде.
	s.finalizeProcessingStub(ctx, tripID, models.TripStatusAddMediaDraftFinalReview)
	return &pb.AddMediaApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status: models.TripStatusAddMediaProcessing,
	}, nil
}

// AddMediaGetReview — снимок финального ревью add-media.
// Read-only для любого participant'а. Ведущий дополнительно получает can_edit=true
// (либо любой, если час бездействия истёк — но реальный перехват состоится только
// на следующем мутирующем действии через ensureInitiator).
func (s *TripService) AddMediaGetReview(ctx context.Context, req *pb.AddMediaGetReviewRequest) (*pb.AddMediaGetReviewResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	active, err := s.validateActiveSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaDraftFinalReview {
		return nil, errWrongStatus(models.TripStatusAddMediaDraftFinalReview, trip.Status)
	}
	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	tagsByPin, _ := s.tagRepo.GetByTripID(tripID)
	if tagsByPin == nil {
		tagsByPin = make(map[string][]string)
	}
	existingSet := make(map[string]struct{}, len(active.ExistingMediaIDs))
	for _, id := range active.ExistingMediaIDs {
		existingSet[id] = struct{}{}
	}
	outPins := make([]*pb.TripPin, 0, len(pins))
	newPinIDs := make([]string, 0)
	for _, pin := range pins {
		mediaList, err := s.mediaRepo.ListByPinID(pin.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list pin media")
		}
		tags := tagsByPin[pin.ID]
		if tags == nil {
			tags = []string{}
		}
		outPins = append(outPins, s.pinWithMediaToProto(ctx, pin, mediaList, tags))
		// new_pin: у пина нет ни одного media из existing_media_ids — значит пин
		// создан в этой add-media сессии. Для таких пинов клиент разрешает
		// полное редактирование; для остальных — только управление новыми медиа.
		isNew := true
		for _, m := range mediaList {
			if _, wasBefore := existingSet[m.ID]; wasBefore {
				isNew = false
				break
			}
		}
		if isNew {
			newPinIDs = append(newPinIDs, pin.ID)
		}
	}
	canEdit := false
	var takeoverAtUnix *int64
	var currentInitiator *string
	if active.CurrentInitiatorUserID != nil {
		v := *active.CurrentInitiatorUserID
		currentInitiator = &v
		if v == userID {
			canEdit = true
		}
	}
	if active.InitiatorAssignedAt != nil {
		takeover := active.InitiatorAssignedAt.Add(initiatorTakeoverTimeout).Unix()
		takeoverAtUnix = &takeover
		if !canEdit && time.Since(*active.InitiatorAssignedAt) >= initiatorTakeoverTimeout {
			// час прошёл — любой может перехватить следующим мутирующим действием.
			canEdit = true
		}
	}
	return &pb.AddMediaGetReviewResponse{
		TripId: tripID,
		SessionId: sessionID,
		Pins: outPins,
		NewPinIds: newPinIDs,
		ProtectedMediaIds: active.ExistingMediaIDs,
		CurrentInitiatorUserId: currentInitiator,
		TakeoverAvailableAtUnix: takeoverAtUnix,
		CanEdit: canEdit,
	}, nil
}

// AddMediaConfirm — финализация add-media. Только ведущий (или любой после
// часа бездействия через ensureInitiator). Закрывает сессию, трип → READY, публикует
// ADD_MEDIA_SESSION_COMPLETED в Redis Stream (notification-service) и TRIP_STATUS_CHANGED
// в WS. Идемпотентна: повторный вызов на уже закрытую сессию возвращает already_confirmed=true.
func (s *TripService) AddMediaConfirm(ctx context.Context, req *pb.AddMediaConfirmRequest) (*pb.AddMediaConfirmResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	// Идемпотентность: если активной сессии нет и трип в READY — считаем что кто-то
	// уже нажал Confirm, возвращаем already_confirmed=true.
	if _, err := s.addMediaSessionRepo.GetActive(ctx, tripID); errors.Is(err, repositories.ErrNoActiveSession) {
		trip, terr := s.tripRepo.GetByID(tripID)
		if terr == nil && trip.Status == models.TripStatusReady {
			return &pb.AddMediaConfirmResponse{
				Status: models.TripStatusReady,
				AlreadyConfirmed: true,
			}, nil
		}
	} else if err != nil {
		return nil, status.Error(codes.Internal, "failed to check active session")
	}
	// Сначала проверяем статус — иначе ensureInitiator вернул бы 403 NOT_INITIATOR
	// с пустым current_initiator_user_id для статусов, где ведущий ещё не назначен
	// (UPLOADING, GROUPING_REVIEW).
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaDraftFinalReview {
		return nil, errWrongStatus(models.TripStatusAddMediaDraftFinalReview, trip.Status)
	}
	if _, err := s.ensureInitiator(ctx, tripID, sessionID, userID); err != nil {
		return nil, err
	}
	// Применяем правки пинов и удаление «похожих» медиа атомарно с закрытием
	// сессии (ТЗ 3.14/3.15). Та же логика, что в FinalizeTrip для creation-флоу.
	if err := s.applyReviewEdits(ctx, trip, req.GetPinUpdates(), req.GetMediaToDelete()); err != nil {
		return nil, err
	}
	if _, err := s.addMediaSessionRepo.Close(ctx, sessionID, models.AddMediaSessionCloseReasonConfirmed, time.Now()); err != nil {
		if errors.Is(err, repositories.ErrNoActiveSession) {
			// race с другим ведущим — он уже закрыл.
			return &pb.AddMediaConfirmResponse{Status: models.TripStatusReady, AlreadyConfirmed: true}, nil
		}
		return nil, status.Error(codes.Internal, "failed to close session")
	}
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusReady); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	if s.eventRepo != nil {
		// Для notification-service: один пуш всем participant'ам, кроме confirmUserID.
		_ = s.eventRepo.PublishTripEvent(ctx, repositories.EventAddMediaSessionCompleted, tripID, userID)
		participants, _ := s.participantRepo.GetByTripID(tripID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventTripStatusChanged, map[string]interface{}{
			"trip_id": tripID,
			"new_status": models.TripStatusReady,
			"session_id": sessionID,
			"reason": "add_media_confirmed",
		})
	}
	return &pb.AddMediaConfirmResponse{
		Status: models.TripStatusReady,
		AlreadyConfirmed: false,
	}, nil
}

// AddMediaCancel — отмена add-media. Закрывает сессию (close_reason='cancelled'),
// удаляет медиа без pin_id из БД и S3 (F1), возвращает трип в READY.
// В ADD_MEDIA_DRAFT_FINAL_REVIEW вызывать может только ведущий (с правилом перехвата),
// на остальных этапах — любой participant (E2).
func (s *TripService) AddMediaCancel(ctx context.Context, req *pb.AddMediaCancelRequest) (*pb.AddMediaCancelResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	active, err := s.validateActiveSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	switch trip.Status {
	case models.TripStatusAddMediaUploading,
		models.TripStatusAddMediaGroupingReview,
		models.TripStatusAddMediaProcessing,
		models.TripStatusAddMediaDraftFinalReview:
		// ok — cancel разрешён
	default:
		return nil, errWrongStatus("ADD_MEDIA_*", trip.Status)
	}
	if trip.Status == models.TripStatusAddMediaDraftFinalReview {
		if _, err := s.ensureInitiator(ctx, tripID, sessionID, userID); err != nil {
			return nil, err
		}
	}
	// F1: удаляем медиа, которые были загружены в сессии, но не получили pin_id
	// (остались неприкреплёнными) — и из БД, и из S3.
	s3Keys, derr := s.mediaRepo.DeleteOrphanSessionMedia(tripID, active.ExistingMediaIDs)
	if derr != nil {
		slog.WarnContext(ctx, "AddMediaCancel: failed to delete orphan media", "trip_id", tripID, "err", derr)
	}
	if s.mediaURLs != nil {
		for _, key := range s3Keys {
			_ = s.mediaURLs.DeleteObject(ctx, key)
		}
	}
	if _, err := s.addMediaSessionRepo.Close(ctx, sessionID, models.AddMediaSessionCloseReasonCancelled, time.Now()); err != nil {
		return nil, status.Error(codes.Internal, "failed to close session")
	}
	if err := s.tripRepo.SetStatus(tripID, models.TripStatusReady); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	if s.eventRepo != nil && s.participantRepo != nil {
		participants, _ := s.participantRepo.GetByTripID(tripID)
		userIDs := make([]string, 0, len(participants))
		for _, p := range participants {
			userIDs = append(userIDs, p.UserID)
		}
		_ = s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, repositories.EventTripStatusChanged, map[string]interface{}{
			"trip_id": tripID,
			"new_status": models.TripStatusReady,
			"session_id": sessionID,
			"reason": "add_media_cancelled",
			"actor_user_id": userID,
		})
	}
	return &pb.AddMediaCancelResponse{Status: models.TripStatusReady}, nil
}

// AddMediaTakeover — явный перехват ведущего после истечения часа бездействия.
// Идемпотентен: если caller уже ведущий, вернёт 200 без mutation и без обновления
// таймера. Если час ещё не прошёл — 403 NOT_INITIATOR. На стрелка status check
// возвращает 412 (только в DRAFT_FINAL_REVIEW есть смысл перехвата).
func (s *TripService) AddMediaTakeover(ctx context.Context, req *pb.AddMediaTakeoverRequest) (*pb.AddMediaTakeoverResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	active, err := s.validateActiveSession(ctx, tripID, sessionID)
	if err != nil {
		return nil, err
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	if trip.Status != models.TripStatusAddMediaDraftFinalReview {
		return nil, errWrongStatus(models.TripStatusAddMediaDraftFinalReview, trip.Status)
	}
	// Caller — уже ведущий: idempotent no-op без обновления таймера.
	if active.CurrentInitiatorUserID != nil && *active.CurrentInitiatorUserID == userID {
		var takeoverAt int64
		if active.InitiatorAssignedAt != nil {
			takeoverAt = active.InitiatorAssignedAt.Add(initiatorTakeoverTimeout).Unix()
		}
		return &pb.AddMediaTakeoverResponse{
			CurrentInitiatorUserId:  userID,
			TakeoverAvailableAtUnix: takeoverAt,
			IsInitiator:             true,
		}, nil
	}
	// Час не прошёл — 403.
	currentInit := ""
	if active.CurrentInitiatorUserID != nil {
		currentInit = *active.CurrentInitiatorUserID
	}
	if active.InitiatorAssignedAt == nil || time.Since(*active.InitiatorAssignedAt) < initiatorTakeoverTimeout {
		var takeoverAt time.Time
		if active.InitiatorAssignedAt != nil {
			takeoverAt = active.InitiatorAssignedAt.Add(initiatorTakeoverTimeout)
		}
		return nil, errNotInitiator(currentInit, takeoverAt)
	}
	// Час прошёл — выполняем перехват.
	now, err := s.executeTakeover(ctx, tripID, sessionID, currentInit, userID)
	if err != nil {
		return nil, err
	}
	return &pb.AddMediaTakeoverResponse{
		CurrentInitiatorUserId:  userID,
		TakeoverAvailableAtUnix: now.Add(initiatorTakeoverTimeout).Unix(),
		IsInitiator:             true,
	}, nil
}

// PublishTrip — отдельный флоу публикации в общую ленту (ТЗ 3.3).
// publish_whole=true публикует всю поездку; иначе публикуются только выбранные пины.
// Для упрощения текущей реализации список опубликованных пинов отдельно не сохраняется —
// trip помечается как is_published=true, а выборка пинов для отображения выполняется на клиенте.
func (s *TripService) PublishTrip(ctx context.Context, req *pb.PublishTripRequest) (*pb.PublishTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}

	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}

	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}

	if trip.Status != "READY" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be READY to publish")
	}

	if trip.PrivacyLevel != "Public" {
		return nil, status.Errorf(codes.FailedPrecondition,
			"trip must have Public privacy level to publish, current level: %s", trip.PrivacyLevel)
	}

	publishWhole := req.GetPublishWhole()
	pinIDs := req.GetPinIds()

	if publishWhole && len(pinIDs) > 0 {
		return nil, status.Error(codes.InvalidArgument, "pin_ids must be empty when publish_whole is true")
	}
	if !publishWhole && len(pinIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "pin_ids must be provided when publish_whole is false")
	}

	// Валидация и установка флага публикации на пинах.
	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	pinIDSet := make(map[string]*models.Pin, len(pins))
	for _, p := range pins {
		pinIDSet[p.ID] = p
	}

	// Сбрасываем старое состояние публикации, чтобы можно было переопубликовать с другим набором пинов.
	for _, p := range pins {
		p.IsPublishedInFeed = false
		_ = s.pinRepo.Update(p)
	}

	if publishWhole {
		for _, p := range pins {
			p.IsPublishedInFeed = true
			_ = s.pinRepo.Update(p)
		}
	} else {
		for _, id := range pinIDs {
			p, ok := pinIDSet[id]
			if !ok {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("pin_id %s does not belong to trip", id))
			}
			p.IsPublishedInFeed = true
			_ = s.pinRepo.Update(p)
		}
	}

	wasPublished := trip.IsPublished
	if !trip.IsPublished {
		trip.IsPublished = true
		if err := s.tripRepo.Update(trip); err != nil {
			return nil, status.Error(codes.Internal, "failed to publish trip")
		}
	}

	if s.eventRepo != nil && !wasPublished {
		_ = s.eventRepo.PublishTripEvent(ctx, "TRIP_READY", tripID, userID)
	}

	updated, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload trip")
	}
	return &pb.PublishTripResponse{Trip: s.tripToProto(ctx, updated)}, nil
}

// UpsertTripPrivacy — ТЗ 6.4.1: участник выставляет свой уровень приватности на путешествии.
// Эффективный privacy_level пересчитывается синхронно по AggregatePrivacyLevel (ТЗ 6.6/6.7).
// Воркер processPrivacyEvents дублирует пересчёт асинхронно как fallback.
func (s *TripService) UpsertTripPrivacy(ctx context.Context, req *pb.UpsertTripPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	level := req.GetPrivacyLevel()
	if !validateUserPrivacyLevel(level) {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private")
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
	if trip.PrivacyLevel == "Restricted" {
		return nil, status.Error(codes.FailedPrecondition, "cannot change permanently private privacy level")
	}
	if err := s.tripPrivacyRepo.Upsert(ctx, tripID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to upsert trip privacy")
	}
	entries, err := s.tripPrivacyRepo.GetByTripID(ctx, tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read trip privacy entries")
	}
	effective := repositories.AggregatePrivacyLevel(trip.PrivacyLevel, entries)
	if err := s.tripRepo.SetPrivacyLevel(tripID, effective); err != nil {
		return nil, status.Error(codes.Internal, "failed to set trip privacy level")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "trip", tripID, tripID, userID, level)
	}
	return &pb.UpsertPrivacyResponse{EffectivePrivacyLevel: effective}, nil
}

// UpsertPinPrivacy — ТЗ 6.4.2: участник выставляет свой уровень приватности на пине.
func (s *TripService) UpsertPinPrivacy(ctx context.Context, req *pb.UpsertPinPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	pinID := req.GetPinId()
	if tripID == "" || pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and pin_id are required")
	}
	level := req.GetPrivacyLevel()
	if !validateUserPrivacyLevel(level) {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private")
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
	if pin.PrivacyLevel == "Restricted" {
		return nil, status.Error(codes.FailedPrecondition, "cannot change permanently private privacy level")
	}
	if err := s.pinPrivacyRepo.Upsert(ctx, pinID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to upsert pin privacy")
	}
	entries, err := s.pinPrivacyRepo.GetByPinID(ctx, pinID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read pin privacy entries")
	}
	effective := repositories.AggregatePrivacyLevel(pin.PrivacyLevel, entries)
	if err := s.pinRepo.SetPrivacyLevel(pinID, effective); err != nil {
		return nil, status.Error(codes.Internal, "failed to set pin privacy level")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "pin", pinID, tripID, userID, level)
	}
	return &pb.UpsertPrivacyResponse{EffectivePrivacyLevel: effective}, nil
}

// UpsertMediaPrivacy — ТЗ 6.4.3 / 5.2: участник выставляет свой уровень приватности на медиа.
func (s *TripService) UpsertMediaPrivacy(ctx context.Context, req *pb.UpsertMediaPrivacyRequest) (*pb.UpsertPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	mediaID := req.GetMediaId()
	if tripID == "" || mediaID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and media_id are required")
	}
	level := req.GetPrivacyLevel()
	if !validateUserPrivacyLevel(level) {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	media, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Error(codes.Internal, "failed to get media")
	}
	if media.TripID != tripID {
		return nil, status.Error(codes.NotFound, "media not found")
	}
	if media.PrivacyLevel == "Restricted" {
		return nil, status.Error(codes.FailedPrecondition, "cannot change permanently private privacy level")
	}
	if err := s.mediaPrivacyRepo.Upsert(ctx, mediaID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to upsert media privacy")
	}
	entries, err := s.mediaPrivacyRepo.GetByMediaID(ctx, mediaID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read media privacy entries")
	}
	effective := repositories.AggregatePrivacyLevel(media.PrivacyLevel, entries)
	if err := s.mediaRepo.SetPrivacyLevel(mediaID, effective); err != nil {
		return nil, status.Error(codes.Internal, "failed to set media privacy level")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "media", mediaID, tripID, userID, level)
	}
	return &pb.UpsertPrivacyResponse{EffectivePrivacyLevel: effective}, nil
}

// UpdateTripSettings — ТЗ 12.4.1: вкл/выкл уведомлений по трипу. Только участник, только свои настройки.
func (s *TripService) UpdateTripSettings(ctx context.Context, req *pb.UpdateTripSettingsRequest) (*pb.UpdateTripSettingsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	participant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !participant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	_ = s.settingsRepo.EnsureDefaultSettings(tripID, userID)
	if err := s.settingsRepo.UpdateNotifications(tripID, userID, req.GetNotificationsEnabled()); err != nil {
		return nil, status.Error(codes.Internal, "failed to update settings")
	}
	return &pb.UpdateTripSettingsResponse{Success: true}, nil
}

// ListFeed — лента опубликованных трипов . Пагинация 20, фильтры category/season/location, сортировка date|rating.
func (s *TripService) ListFeed(ctx context.Context, req *pb.ListFeedRequest) (*pb.ListFeedResponse, error) {
	_, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}
	sortBy := req.GetSortBy()
	if sortBy != "rating" && sortBy != "date" {
		sortBy = "date"
	}
	// ТЗ 7.9.3: фильтр по месту принимает имя города/страны строкой; резолвим в geo_registry.
	// Если регион не найден — лента возвращает пустой результат (не ошибку).
	locationIDs, ok, err := s.resolveFeedLocationIDs(ctx, req.GetCity(), req.GetCountry())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to resolve location filter")
	}
	if !ok {
		return &pb.ListFeedResponse{}, nil
	}
	trips, err := s.tripRepo.ListFeed(limit, offset, req.GetCategory(), req.GetSeason(), locationIDs, sortBy)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list feed")
	}
	if len(trips) == 0 {
		return &pb.ListFeedResponse{}, nil
	}

	tripIDs := make([]string, len(trips))
	for i, t := range trips {
		tripIDs[i] = t.ID
	}

	pinsByTrip, err := s.pinRepo.ListPublishedPinsByTripIDs(tripIDs)
	if err != nil {
		slog.WarnContext(ctx, "ListFeed: failed to fetch pins", "error", err)
		pinsByTrip = make(map[string][]*repositories.FeedPin)
	}

	mediaByTrip, err := s.mediaRepo.TopMediaByTripIDs(tripIDs, 8)
	if err != nil {
		slog.WarnContext(ctx, "ListFeed: failed to fetch media", "error", err)
		mediaByTrip = make(map[string][]*repositories.FeedMedia)
	}

	var pinIDs []string
	for _, fps := range pinsByTrip {
		for _, fp := range fps {
			pinIDs = append(pinIDs, fp.ID)
		}
	}
	mediaByPin, err := s.mediaRepo.TopMediaByPinIDs(pinIDs, 10)
	if err != nil {
		slog.WarnContext(ctx, "ListFeed: failed to fetch pin media", "error", err)
		mediaByPin = make(map[string][]*repositories.FeedMedia)
	}

	items := make([]*pb.FeedItem, len(trips))
	for i, t := range trips {
		feedPins := pinsByTrip[t.ID]
		protoPins := make([]*pb.FeedPin, len(feedPins))
		for j, fp := range feedPins {
			pinMedia := mediaByPin[fp.ID]
			protoPinMedia := make([]*pb.FeedMedia, len(pinMedia))
			for k, fm := range pinMedia {
				protoPinMedia[k] = &pb.FeedMedia{
					MediaId: fm.ID,
					Url: s.presignedReadURL(ctx, fm.S3Key),
					MediaType: fm.MediaType,
				}
			}
			protoPins[j] = &pb.FeedPin{
				Id: fp.ID,
				Latitude: fp.Latitude,
				Longitude: fp.Longitude,
				Media: protoPinMedia,
			}
		}

		feedMedia := mediaByTrip[t.ID]
		protoMedia := make([]*pb.FeedMedia, len(feedMedia))
		for j, fm := range feedMedia {
			protoMedia[j] = &pb.FeedMedia{
				MediaId: fm.ID,
				Url: s.presignedReadURL(ctx, fm.S3Key),
				MediaType: fm.MediaType,
			}
		}

		items[i] = &pb.FeedItem{
			Trip: s.tripToProto(ctx, t),
			Pins: protoPins,
			Media: protoMedia,
		}
	}
	return &pb.ListFeedResponse{Items: items}, nil
}

// resolveFeedLocationIDs — ТЗ 7.9.3: фильтр по строковому city/country.
// Возвращает (ids, ok, err): ok=false означает «регион указан, но не найден» —
// лента в этом случае возвращает пустой список (а не 500). Если оба параметра
// заданы, приоритет у города (он более узкий фильтр).
func (s *TripService) resolveFeedLocationIDs(ctx context.Context, city, country string) ([]int, bool, error) {
	city = strings.TrimSpace(city)
	country = strings.TrimSpace(country)
	if city == "" && country == "" {
		return nil, true, nil
	}
	if s.geoRepo == nil {
		return nil, true, nil
	}
	var (
		id int
		err error
	)
	if city != "" {
		id, err = s.geoRepo.FindCityByName(ctx, city)
	} else {
		id, err = s.geoRepo.FindCountryByName(ctx, country)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return []int{id}, true, nil
}

// LikeTrip — поставить лайк трипу в ленте .
func (s *TripService) LikeTrip(ctx context.Context, req *pb.LikeTripRequest) (*pb.LikeTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, status.Error(codes.NotFound, "trip not found")
	}
	if !trip.IsPublished {
		return nil, status.Error(codes.FailedPrecondition, "trip is not published")
	}
	old, err := s.socialRepo.SetReaction(userID, tripID, "Like")
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to set like")
	}
	if s.eventRepo != nil && old != "Like" {
		// Переход Dislike→Like — сбросить дизлайк в stats.
		if old == "Dislike" {
			_ = s.eventRepo.PublishStatsEvent(ctx, "DISLIKE_REMOVED", tripID, []string{userID}, nil)
		}
		_ = s.eventRepo.PublishStatsEvent(ctx, "LIKE_ADDED", tripID, []string{userID}, nil)
	}
	return &pb.LikeTripResponse{Success: true}, nil
}

// DislikeTrip — поставить дизлайк трипу в ленте .
func (s *TripService) DislikeTrip(ctx context.Context, req *pb.DislikeTripRequest) (*pb.DislikeTripResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, status.Error(codes.NotFound, "trip not found")
	}
	if !trip.IsPublished {
		return nil, status.Error(codes.FailedPrecondition, "trip is not published")
	}
	old, err := s.socialRepo.SetReaction(userID, tripID, "Dislike")
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to set dislike")
	}
	if s.eventRepo != nil && old != "Dislike" {
		if old == "Like" {
			_ = s.eventRepo.PublishStatsEvent(ctx, "LIKE_REMOVED", tripID, []string{userID}, nil)
		}
		_ = s.eventRepo.PublishStatsEvent(ctx, "DISLIKE_ADDED", tripID, []string{userID}, nil)
	}
	return &pb.DislikeTripResponse{Success: true}, nil
}

// AddToFavourites — добавить трип в избранное .
func (s *TripService) AddToFavourites(ctx context.Context, req *pb.AddToFavouritesRequest) (*pb.AddToFavouritesResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, status.Error(codes.NotFound, "trip not found")
	}
	if !trip.IsPublished {
		return nil, status.Error(codes.FailedPrecondition, "trip is not published")
	}
	if err := s.favouriteRepo.Add(userID, tripID); err != nil {
		return nil, status.Error(codes.Internal, "failed to add to favourites")
	}
	return &pb.AddToFavouritesResponse{Success: true}, nil
}

// RemoveFromFavourites — убрать трип из избранного .
func (s *TripService) RemoveFromFavourites(ctx context.Context, req *pb.RemoveFromFavouritesRequest) (*pb.RemoveFromFavouritesResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	if err := s.favouriteRepo.Remove(userID, tripID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "not in favourites")
		}
		return nil, status.Error(codes.Internal, "failed to remove from favourites")
	}
	return &pb.RemoveFromFavouritesResponse{Success: true}, nil
}

// ListFavourites returns trips that the current user has added to favourites . Excludes soft-deleted trips.
func (s *TripService) ListFavourites(ctx context.Context, req *pb.ListFavouritesRequest) (*pb.ListFavouritesResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}
	tripIDs, err := s.favouriteRepo.ListTripIDsByUserID(userID, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list favourites")
	}
	out := make([]*pb.Trip, 0, len(tripIDs))
	for _, id := range tripIDs {
		trip, err := s.tripRepo.GetByID(id)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, status.Error(codes.Internal, "failed to get trip")
		}
		if trip.IsSoftDeleted {
			continue
		}
		out = append(out, s.tripToProto(ctx, trip))
	}
	return &pb.ListFavouritesResponse{Trips: out}, nil
}

func (s *TripService) presignedReadURL(ctx context.Context, s3Key string) string {
	if s.mediaURLs == nil || s3Key == "" {
		return ""
	}
	u, err := s.mediaURLs.ReadURL(ctx, s3Key)
	if err != nil {
		return ""
	}
	return u
}

func (s *TripService) resolveMediaDeletionsForTrip(tripID string, ids []string) (allowedIDs []string, s3Keys []string, err error) {
	for _, id := range ids {
		m, err := s.mediaRepo.GetByID(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil, status.Error(codes.InvalidArgument, "unknown media id")
			}
			return nil, nil, status.Error(codes.Internal, "failed to get media")
		}
		if m.TripID != tripID {
			return nil, nil, status.Error(codes.PermissionDenied, "media does not belong to this trip")
		}
		allowedIDs = append(allowedIDs, id)
		if m.S3Key != "" {
			s3Keys = append(s3Keys, m.S3Key)
		}
	}
	return allowedIDs, s3Keys, nil
}

func (s *TripService) tripToProto(ctx context.Context, t *models.Trip) *pb.Trip {
	cover := ""
	if t.CoverURL != "" {
		cover = s.presignedReadURL(ctx, t.CoverURL)
	}
	out := &pb.Trip{
		Id: t.ID,
		OwnerUserId: t.OwnerUserID,
		Name: t.Name,
		Description: t.Description,
		Category: t.Category,
		Season: t.Season,
		Status: t.Status,
		PrivacyLevel: t.PrivacyLevel,
		LikesCount: t.LikesCount,
		DislikesCount: t.DislikesCount,
		CoverUrl: cover,
		IsPublished: t.IsPublished,
		IsGenerated: t.IsGenerated,
		CreatedAtUnix: t.CreatedAt.Unix(),
		UpdatedAtUnix: t.UpdatedAt.Unix(),
		MediaCount: t.MediaCount,
		ParticipantsCount: t.ParticipantsCount,
		PinsCount: t.PinsCount,
	}
	if t.StartDate != nil {
		out.StartDateUnix = t.StartDate.Unix()
	}
	if t.EndDate != nil {
		out.EndDateUnix = t.EndDate.Unix()
	}
	return out
}

// GetNotificationSettings — . Используется notification-service
// worker'ом/scheduler'ом чтобы отфильтровать адресатов пуша по их настройкам.
// Не требует быть участником (мониторинг делается на стороне клиента).
func (s *TripService) GetNotificationSettings(ctx context.Context, req *pb.GetNotificationSettingsRequest) (*pb.GetNotificationSettingsResponse, error) {
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	userIDs := req.GetUserIds()
	if len(userIDs) == 0 {
		return &pb.GetNotificationSettingsResponse{NotificationsEnabled: map[string]bool{}}, nil
	}
	settings, err := s.settingsRepo.GetByTripAndUsers(tripID, userIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get settings: %v", err)
	}
	return &pb.GetNotificationSettingsResponse{NotificationsEnabled: settings}, nil
}

// ListAnniversaryTrips — . Возвращает трипы, у которых created_at
// пришёлся ровно на today-1y. Используется scheduler'ом notification-service.
func (s *TripService) ListAnniversaryTrips(ctx context.Context, req *pb.ListAnniversaryTripsRequest) (*pb.ListAnniversaryTripsResponse, error) {
	candidates, err := s.tripRepo.ListAnniversaryCandidates(req.GetTodayUnix())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list anniversary: %v", err)
	}
	return &pb.ListAnniversaryTripsResponse{Trips: toNotificationTrips(candidates)}, nil
}

// ListEndedMonthAgoTrips — . Возвращает трипы, у которых end_date
// пришёлся ровно на today-1m.
func (s *TripService) ListEndedMonthAgoTrips(ctx context.Context, req *pb.ListEndedMonthAgoTripsRequest) (*pb.ListEndedMonthAgoTripsResponse, error) {
	candidates, err := s.tripRepo.ListEndedMonthAgoCandidates(req.GetTodayUnix())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ended month ago: %v", err)
	}
	return &pb.ListEndedMonthAgoTripsResponse{Trips: toNotificationTrips(candidates)}, nil
}

// ListTripParticipants — . Список user_id участников трипа для
// notification-service (рассылка пушей по ТЗ 11.1).
func (s *TripService) ListTripParticipants(ctx context.Context, req *pb.ListTripParticipantsRequest) (*pb.ListTripParticipantsResponse, error) {
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	parts, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.UserID)
	}
	return &pb.ListTripParticipantsResponse{UserIds: out}, nil
}

func toNotificationTrips(candidates []*repositories.NotificationTripCandidate) []*pb.NotificationTrip {
	out := make([]*pb.NotificationTrip, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, &pb.NotificationTrip{
			TripId: c.TripID,
			Name: c.Name,
			ParticipantUserIds: c.Participants,
			StartDateUnix: c.StartDateUnix,
			EndDateUnix: c.EndDateUnix,
			YearsElapsed: c.YearsElapsed,
		})
	}
	return out
}
