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

// LocationResolver выполняет reverse geocoding по координатам.
type LocationResolver interface {
	ResolveLocation(ctx context.Context, lat, lon float64) (countryName, cityName, displayName string, err error)
}

type TripService struct {
	pb.UnimplementedTripServiceServer
	tripRepo            repositories.TripRepositoryInterface
	participantRepo     repositories.TripParticipantRepositoryInterface
	inviteRepo          repositories.InvitationLinkRepositoryInterface
	settingsRepo        repositories.TripSettingsRepositoryInterface
	eventRepo           repositories.TripEventPublisher
	mediaRepo           repositories.MediaRepositoryInterface
	mediaURLs           MediaURLResolver
	pinRepo             repositories.PinRepositoryInterface
	tagRepo             repositories.TagRepositoryInterface
	socialRepo          repositories.SocialRepositoryInterface
	favouriteRepo       repositories.FavouriteRepositoryInterface
	geocoder            LocationResolver
	geoRepo             repositories.GeoRegistryRepositoryInterface
	addMediaSessionRepo *repositories.AddMediaSessionRepository
	battleRepo          repositories.MediaBattleRepositoryInterface
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
	geocoder LocationResolver,
	geoRepo repositories.GeoRegistryRepositoryInterface,
	addMediaSessionRepo *repositories.AddMediaSessionRepository,
	battleRepo repositories.MediaBattleRepositoryInterface,
) *TripService {
	return &TripService{
		tripRepo:            tripRepo,
		participantRepo:     participantRepo,
		inviteRepo:          inviteRepo,
		settingsRepo:        settingsRepo,
		eventRepo:           eventRepo,
		mediaRepo:           mediaRepo,
		mediaURLs:           mediaURLs,
		pinRepo:             pinRepo,
		tagRepo:             tagRepo,
		socialRepo:          socialRepo,
		favouriteRepo:       favouriteRepo,
		geocoder:            geocoder,
		geoRepo:             geoRepo,
		addMediaSessionRepo: addMediaSessionRepo,
		battleRepo:          battleRepo,
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
	privacyLevel := req.GetPrivacyLevel()
	if privacyLevel == "" {
		privacyLevel = "Private"
	}
	if !validateUserPrivacyLevel(privacyLevel) {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private")
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
	trip := &models.Trip{
		OwnerUserID:   userID,
		Name:          name,
		Description:   req.GetDescription(),
		Category:      category,
		Season:        season,
		Status:        tripStatus,
		PrivacyLevel:  privacyLevel,
		LikesCount:    0,
		DislikesCount: 0,
		IsPublished:   false,
		IsGenerated:   false,
	}
	if err := s.tripRepo.Create(trip); err != nil {
		return nil, status.Error(codes.Internal, "failed to create trip")
	}
	participant := &models.TripParticipant{TripID: trip.ID, UserID: userID, IsAdmin: true}
	if err := s.participantRepo.Add(participant); err != nil {
		return nil, status.Error(codes.Internal, "failed to add owner as participant")
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
			S3Key:    s3Key,
			Url:      url,
		})
	}
	return &pb.CreateTripResponse{
		TripId:     trip.ID,
		Status:     trip.Status,
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
		resp, err := s.getTripResponseWithPins(ctx, trip)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	// PINZ-98: allow access if user has trip in favourites (e.g. after soft delete)
	hasFav, err := s.favouriteRepo.HasFavourite(userID, tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check favourite")
	}
	if hasFav {
		resp, err := s.getTripResponseWithPins(ctx, trip)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	return nil, status.Error(codes.PermissionDenied, "not a participant")
}

// getTripResponseWithPins builds GetTripResponse with trip and pins (each pin with its media).
func (s *TripService) getTripResponseWithPins(ctx context.Context, trip *models.Trip) (*pb.GetTripResponse, error) {
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
	return &pb.GetTripResponse{
		Trip: s.tripToProto(ctx, trip),
		Pins: outPins,
	}, nil
}

func (s *TripService) pinWithMediaToProto(ctx context.Context, pin *models.Pin, mediaList []*models.Media, tags []string) *pb.TripPin {
	out := &pb.TripPin{
		Id:           pin.ID,
		TripId:       pin.TripID,
		Name:         pin.Name,
		Description:  pin.Description,
		Category:     pin.Category,
		PrivacyLevel: pin.PrivacyLevel,
		Tags:         tags,
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
			MediaId:      m.ID,
			Url:          s.presignedReadURL(ctx, m.S3Key),
			MediaType:    m.MediaType,
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
			TripId:     s.TripID,
			PinsCount:  s.PinsCount,
			MediaCount: s.MediaCount,
		})
	}
	return &pb.ListUserTripSummariesResponse{Trips: out}, nil
}

// MaxSearchQueryLength ограничивает длину текстового поиска, чтобы защитить БД от очень больших ILIKE-паттернов.
const MaxSearchQueryLength = 128

// SearchPins — PINZ-135: текстовый поиск пинов по name/description/тегам среди трипов, где user — участник.
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
	if req.PrivacyLevel != nil {
		if !validateUserPrivacyLevel(*req.PrivacyLevel) {
			return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private")
		}
		if trip.PrivacyLevel == "Restricted" {
			return nil, status.Error(codes.FailedPrecondition, "cannot change permanently private privacy level")
		}
		trip.PrivacyLevel = *req.PrivacyLevel
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

// DeleteTrip — только админ. PINZ-98 (ТЗ 3.24.1/3.24.2): если трип в избранном у других — soft delete (удаление из списка участников); иначе — полное удаление.
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
		ID:        uuid.New().String(),
		TripID:    tripID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	if err := s.inviteRepo.Create(link); err != nil {
		return nil, status.Error(codes.Internal, "failed to create invite link")
	}
	return &pb.GenerateInviteLinkResponse{
		InviteLinkId:  link.ID,
		Token:         token,
		InviteUrl:     "", // клиент/gateway собирает URL по token
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
			TripID:       tripID,
			S3Key:        meta.GetS3Key(),
			MediaType:    meta.GetMediaType(),
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
				Url:     s.presignedReadURL(ctx, m.S3Key),
				Type:    m.MediaType,
			})
		}
		respPins = append(respPins, dp)
	}
	return &pb.ProcessMediaGroupingResponse{
		TripId:    tripID,
		Status:    "DRAFT_GROUPING_REVIEW",
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
			TripID:       tripID,
			Name:         "Pin",
			Description:  "",
			Category:     trip.Category,
			PrivacyLevel: trip.PrivacyLevel,
			MediaCount:   int32(len(dp.GetMediaIds())),
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
	if err := s.tripRepo.SetStatus(tripID, "PROCESSING"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	s.finalizeProcessingStub(ctx, tripID)
	return &pb.ApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status:  "PROCESSING",
	}, nil
}

// finalizeProcessingStub заменяет настоящий ML-воркер: синхронно переводит трип
// в DRAFT_FINAL_REVIEW и публикует TRIP_PROCESSING_COMPLETED подписчикам WS
// (канал pinz:trip:{id}:events + per-user каналы). Удалить вместе с TODO в
// ApplyGroupsAndProcess/AddMediaApplyGroupsAndProcess, когда воркер заработает.
func (s *TripService) finalizeProcessingStub(ctx context.Context, tripID string) {
	if err := s.tripRepo.SetStatus(tripID, "DRAFT_FINAL_REVIEW"); err != nil {
		slog.ErrorContext(ctx, "finalizeProcessingStub: SetStatus failed", "trip_id", tripID, "error", err)
		return
	}
	if s.eventRepo == nil || s.participantRepo == nil {
		return
	}
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		slog.WarnContext(ctx, "finalizeProcessingStub: GetByTripID failed", "trip_id", tripID, "error", err)
		return
	}
	userIDs := make([]string, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	if err := s.eventRepo.PublishTripEventWS(ctx, tripID, userIDs, "TRIP_PROCESSING_COMPLETED", map[string]interface{}{
		"trip_id": tripID,
		"status":  "DRAFT_FINAL_REVIEW",
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
				MediaId:      m.ID,
				Url:          s.presignedReadURL(ctx, m.S3Key),
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
			PinId:         pin.ID,
			Name:          pin.Name,
			Category:      pin.Category,
			LocationName:  pin.LocationName,
			StartTimeUnix: startUnix,
			EndTimeUnix:   endUnix,
			Issues:        issues,
			Tags:          tags,
			Media:         reviewMedia,
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
		TripId:  tripID,
		Status:  trip.Status,
		Similar: similar,
		Pins:    respPins,
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
	// Apply pin_updates (name, manual lat/lon) — task 4.1.2: название пина не более 100 символов
	for _, pu := range req.GetPinUpdates() {
		pin, err := s.pinRepo.GetByID(pu.GetPinId())
		if err != nil {
			continue
		}
		if pu.Name != nil {
			name := pu.GetName()
			if len(name) > MaxNameLength {
				return nil, status.Errorf(codes.InvalidArgument, "pin name must be at most %d characters", MaxNameLength)
			}
			pin.Name = name
		}
		if pu.Latitude != nil && pu.Longitude != nil {
			pin.Latitude = pu.Latitude
			pin.Longitude = pu.Longitude
		}
		_ = s.pinRepo.Update(pin)

		// Geocode pins where coordinates were manually set by user.
		if pu.Latitude != nil && pu.Longitude != nil && s.geocoder != nil {
			country, city, displayName, geoErr := s.geocoder.ResolveLocation(ctx, *pin.Latitude, *pin.Longitude)
			if geoErr != nil {
				slog.WarnContext(ctx, "geocoding failed for manually set pin", "pin_id", pin.ID, "error", geoErr)
			} else if displayName != "" {
				pin.LocationName = displayName
				_ = s.pinRepo.Update(pin)
				if s.geoRepo != nil {
					countryID, cityID, _, ensureErr := s.geoRepo.EnsureLocationByName(ctx, country, city)
					if ensureErr != nil {
						slog.WarnContext(ctx, "geo registry ensure failed", "pin_id", pin.ID, "error", ensureErr)
					} else {
						var locIDs []int
						if countryID != nil {
							locIDs = append(locIDs, *countryID)
						}
						if cityID != nil {
							locIDs = append(locIDs, *cityID)
						}
						_ = s.geoRepo.UpsertTripLocations(ctx, tripID, locIDs)
					}
				}
			}
		}
	}
	// Delete media from DB, then best-effort remove from object storage
	if len(req.GetMediaToDelete()) > 0 {
		allowedIDs, s3Keys, err := s.resolveMediaDeletionsForTrip(tripID, req.GetMediaToDelete())
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
	// Aggregate trip: cover_url (S3 key of first image media), start_date, end_date from pins.
	// Presigned URL resolves in tripToProto, so the column stores only the key.
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
	trip.CoverURL = coverURL
	trip.StartDate = minStart
	trip.EndDate = maxEnd
	_ = s.tripRepo.Update(trip)
	if err := s.tripRepo.SetStatus(tripID, "READY"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	return &pb.FinalizeTripResponse{
		TripId:  tripID,
		Status:  "READY",
		Message: "Trip finalized",
	}, nil
}

// AddMediaStart — PINZ-131, ТЗ 5.3 → 3.8: старт сессии добавления медиа в готовый трип.
// Трип должен быть READY. Генерируются presigned URL и session_id.
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
	if trip.Status != "READY" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be READY to add media")
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
		return nil, status.Errorf(codes.InvalidArgument, "trip may have at most %d media", MaxMediaPerTrip)
	}
	if videos+newVideos > MaxVideosPerTrip {
		return nil, status.Errorf(codes.InvalidArgument, "trip may have at most %d videos", MaxVideosPerTrip)
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
			S3Key:    s3Key,
			Url:      url,
		})
	}
	if err := s.tripRepo.SetStatus(tripID, "ADD_MEDIA_UPLOADING"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	return &pb.AddMediaStartResponse{
		SessionId:  sessionID,
		Status:     "ADD_MEDIA_UPLOADING",
		UploadUrls: uploadUrls,
	}, nil
}

// AddMediaProcessGrouping — PINZ-131, ТЗ 5.3.1-5.3.2: кластеризация добавленных медиа с использованием существующих пинов как seed-групп.
func (s *TripService) AddMediaProcessGrouping(ctx context.Context, req *pb.AddMediaProcessGroupingRequest) (*pb.AddMediaProcessGroupingResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	exists, err := s.addMediaSessionRepo.Exists(ctx, tripID, sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to validate session")
	}
	if !exists {
		return nil, status.Error(codes.NotFound, "add-media session not found")
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
	if trip.Status != "ADD_MEDIA_UPLOADING" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in ADD_MEDIA_UPLOADING")
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
	for _, meta := range req.GetMedia() {
		media := &models.Media{
			TripID:       tripID,
			S3Key:        meta.GetS3Key(),
			MediaType:    meta.GetMediaType(),
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
				Url:     s.presignedReadURL(ctx, m.S3Key),
				Type:    m.MediaType,
			})
		}
		respPins = append(respPins, dp)
	}
	existingIDs, _, err := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load session")
	}
	if err := s.tripRepo.SetStatus(tripID, "ADD_MEDIA_GROUPING_REVIEW"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update trip status")
	}
	return &pb.AddMediaProcessGroupingResponse{
		TripId:           tripID,
		SessionId:        sessionID,
		Status:           "ADD_MEDIA_GROUPING_REVIEW",
		DraftPins:        respPins,
		ExistingMediaIds: existingIDs,
	}, nil
}

// AddMediaApplyGroupsAndProcess — PINZ-131, ТЗ 5.3.3-5.3.4: применение групп и запуск ML-обработки для добавленных медиа.
// Существующие медиа защищены от удаления/перемещения; ML worker получает flow="add_media" + new_pin_ids для пропуска тегов/категорий у существующих пинов.
func (s *TripService) AddMediaApplyGroupsAndProcess(ctx context.Context, req *pb.AddMediaApplyGroupsAndProcessRequest) (*pb.AddMediaApplyGroupsAndProcessResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	sessionID := req.GetSessionId()
	if tripID == "" || sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and session_id are required")
	}
	exists, err := s.addMediaSessionRepo.Exists(ctx, tripID, sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to validate session")
	}
	if !exists {
		return nil, status.Error(codes.NotFound, "add-media session not found")
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
	if trip.Status != "ADD_MEDIA_GROUPING_REVIEW" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be in ADD_MEDIA_GROUPING_REVIEW")
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
				TripID:       tripID,
				Name:         "Pin",
				Description:  "",
				Category:     trip.Category,
				PrivacyLevel: trip.PrivacyLevel,
				MediaCount:   int32(len(newOnly)),
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
	if err := s.tripRepo.SetStatus(tripID, "PROCESSING"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	// PINZ-134: уведомляем notification-service о добавленных пинах. userID —
	// автор добавления медиа; notification-service адресует пуш остальным
	// участникам трипа. Одно событие на весь add-media запрос — избегаем
	// флуда, если пользователь добавил сразу несколько pin'ов.
	if s.eventRepo != nil && len(newPinIDs) > 0 {
		_ = s.eventRepo.PublishTripEvent(ctx, "PIN_ADDED", tripID, userID)
	}
	// STUB: ML-пайплайн пока не реализован, поэтому SetMLContext/AddMLTaskWithFlow
	// не нужны — сразу двигаем трип в DRAFT_FINAL_REVIEW через общий стаб.
	// TODO: вернуть оригинальный enqueue, когда воркер ml:tasks заработает в проде.
	s.finalizeProcessingStub(ctx, tripID)
	return &pb.AddMediaApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status:  "PROCESSING",
	}, nil
}

// PublishTrip — отдельный флоу публикации в общую ленту (PINZ-105, ТЗ 3.3).
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

// ListFeed — лента опубликованных трипов (PINZ-98). Пагинация 20, фильтры category/season/location, сортировка date|rating.
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
	locationProtoIDs := req.GetLocationIds()
	locationIDs := make([]int, 0, len(locationProtoIDs))
	for _, id := range locationProtoIDs {
		if id > 0 {
			locationIDs = append(locationIDs, int(id))
		}
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

	items := make([]*pb.FeedItem, len(trips))
	for i, t := range trips {
		feedPins := pinsByTrip[t.ID]
		protoPins := make([]*pb.FeedPin, len(feedPins))
		for j, fp := range feedPins {
			protoPins[j] = &pb.FeedPin{Id: fp.ID, Latitude: fp.Latitude, Longitude: fp.Longitude}
		}

		feedMedia := mediaByTrip[t.ID]
		protoMedia := make([]*pb.FeedMedia, len(feedMedia))
		for j, fm := range feedMedia {
			protoMedia[j] = &pb.FeedMedia{
				MediaId:   fm.ID,
				Url:       s.presignedReadURL(ctx, fm.S3Key),
				MediaType: fm.MediaType,
			}
		}

		items[i] = &pb.FeedItem{
			Trip:  s.tripToProto(ctx, t),
			Pins:  protoPins,
			Media: protoMedia,
		}
	}
	return &pb.ListFeedResponse{Items: items}, nil
}

// LikeTrip — поставить лайк трипу в ленте (PINZ-98).
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

// DislikeTrip — поставить дизлайк трипу в ленте (PINZ-98).
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

// AddToFavourites — добавить трип в избранное (PINZ-98).
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

// RemoveFromFavourites — убрать трип из избранного (PINZ-98).
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

// ListFavourites returns trips that the current user has added to favourites (PINZ-98). Excludes soft-deleted trips.
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
		Id:            t.ID,
		OwnerUserId:   t.OwnerUserID,
		Name:          t.Name,
		Description:   t.Description,
		Category:      t.Category,
		Season:        t.Season,
		Status:        t.Status,
		PrivacyLevel:  t.PrivacyLevel,
		LikesCount:    t.LikesCount,
		DislikesCount: t.DislikesCount,
		CoverUrl:      cover,
		IsPublished:   t.IsPublished,
		IsGenerated:   t.IsGenerated,
		CreatedAtUnix:     t.CreatedAt.Unix(),
		UpdatedAtUnix:     t.UpdatedAt.Unix(),
		MediaCount:        t.MediaCount,
		ParticipantsCount: t.ParticipantsCount,
		PinsCount:         t.PinsCount,
	}
	if t.StartDate != nil {
		out.StartDateUnix = t.StartDate.Unix()
	}
	if t.EndDate != nil {
		out.EndDateUnix = t.EndDate.Unix()
	}
	return out
}

// GetNotificationSettings — PINZ-134. Используется notification-service
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

// ListAnniversaryTrips — PINZ-134. Возвращает трипы, у которых created_at
// пришёлся ровно на today-1y. Используется scheduler'ом notification-service.
func (s *TripService) ListAnniversaryTrips(ctx context.Context, req *pb.ListAnniversaryTripsRequest) (*pb.ListAnniversaryTripsResponse, error) {
	candidates, err := s.tripRepo.ListAnniversaryCandidates(req.GetTodayUnix())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list anniversary: %v", err)
	}
	return &pb.ListAnniversaryTripsResponse{Trips: toNotificationTrips(candidates)}, nil
}

// ListEndedMonthAgoTrips — PINZ-134. Возвращает трипы, у которых end_date
// пришёлся ровно на today-1m.
func (s *TripService) ListEndedMonthAgoTrips(ctx context.Context, req *pb.ListEndedMonthAgoTripsRequest) (*pb.ListEndedMonthAgoTripsResponse, error) {
	candidates, err := s.tripRepo.ListEndedMonthAgoCandidates(req.GetTodayUnix())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ended month ago: %v", err)
	}
	return &pb.ListEndedMonthAgoTripsResponse{Trips: toNotificationTrips(candidates)}, nil
}

// ListTripParticipants — PINZ-134. Список user_id участников трипа для
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
			TripId:             c.TripID,
			Name:               c.Name,
			ParticipantUserIds: c.Participants,
			StartDateUnix:      c.StartDateUnix,
			EndDateUnix:        c.EndDateUnix,
			YearsElapsed:       c.YearsElapsed,
		})
	}
	return out
}
