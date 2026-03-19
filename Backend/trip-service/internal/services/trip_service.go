package services

import (
	"context"
	"database/sql"
	"fmt"
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

type TripService struct {
	pb.UnimplementedTripServiceServer
	tripRepo        repositories.TripRepositoryInterface
	participantRepo repositories.TripParticipantRepositoryInterface
	inviteRepo      repositories.InvitationLinkRepositoryInterface
	settingsRepo    repositories.TripSettingsRepositoryInterface
	eventRepo       repositories.TripEventPublisher
	mediaRepo       repositories.MediaRepositoryInterface
	pinRepo         repositories.PinRepositoryInterface
	tagRepo         repositories.TagRepositoryInterface
	socialRepo      repositories.SocialRepositoryInterface
	favouriteRepo   repositories.FavouriteRepositoryInterface
}

func NewTripService(
	tripRepo repositories.TripRepositoryInterface,
	participantRepo repositories.TripParticipantRepositoryInterface,
	inviteRepo repositories.InvitationLinkRepositoryInterface,
	settingsRepo repositories.TripSettingsRepositoryInterface,
	eventRepo repositories.TripEventPublisher,
	mediaRepo repositories.MediaRepositoryInterface,
	pinRepo repositories.PinRepositoryInterface,
	tagRepo repositories.TagRepositoryInterface,
	socialRepo repositories.SocialRepositoryInterface,
	favouriteRepo repositories.FavouriteRepositoryInterface,
) *TripService {
	return &TripService{
		tripRepo:        tripRepo,
		participantRepo: participantRepo,
		inviteRepo:      inviteRepo,
		settingsRepo:    settingsRepo,
		eventRepo:       eventRepo,
		mediaRepo:       mediaRepo,
		pinRepo:         pinRepo,
		tagRepo:         tagRepo,
		socialRepo:      socialRepo,
		favouriteRepo:   favouriteRepo,
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
	if !validatePrivacyLevel(privacyLevel) {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be one of: Public, Private, Restricted")
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
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key:    s3Key,
			Url:      "", // Media Service will provide presigned URL; stub for now
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
		resp, err := s.getTripResponseWithPins(trip)
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
		resp, err := s.getTripResponseWithPins(trip)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	return nil, status.Error(codes.PermissionDenied, "not a participant")
}

// getTripResponseWithPins builds GetTripResponse with trip and pins (each pin with its media).
func (s *TripService) getTripResponseWithPins(trip *models.Trip) (*pb.GetTripResponse, error) {
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
		outPins = append(outPins, pinWithMediaToProto(pin, mediaList, tags))
	}
	return &pb.GetTripResponse{
		Trip: tripToProto(trip),
		Pins: outPins,
	}, nil
}

func pinWithMediaToProto(pin *models.Pin, mediaList []*models.Media, tags []string) *pb.TripPin {
	out := &pb.TripPin{
		Id:           pin.ID,
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
			Url:          "", // Media service or gateway may resolve presigned URL
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
		out[i] = tripToProto(t)
	}
	return &pb.ListUserTripsResponse{Trips: out}, nil
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
		if !validatePrivacyLevel(*req.PrivacyLevel) {
			return nil, status.Error(codes.InvalidArgument, "invalid privacy_level")
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
	if req.CoverUrl != nil {
		trip.CoverURL = *req.CoverUrl
	}
	if err := s.tripRepo.Update(trip); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to update trip")
	}
	updated, _ := s.tripRepo.GetByID(tripID)
	return &pb.UpdateTripResponse{Trip: tripToProto(updated)}, nil
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
				Url:     "", // Media Service read URL; stub
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
	// Delete rejected media
	if len(req.GetDeletedMediaIds()) > 0 {
		if err := s.mediaRepo.DeleteByIDs(req.GetDeletedMediaIds()); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete media")
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
	// Status -> PROCESSING, push to Redis Streams for worker
	if err := s.tripRepo.SetStatus(tripID, "PROCESSING"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.AddMLTask(ctx, tripID)
	}
	return &pb.ApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status:  "PROCESSING",
	}, nil
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
				Url:          "",
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
			LocationName:  "",
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
	}
	// Delete media (DB only; S3 via Media Service later)
	if len(req.GetMediaToDelete()) > 0 {
		_ = s.mediaRepo.DeleteByIDs(req.GetMediaToDelete())
	}
	// Aggregate trip: cover_url (first media), start_date, end_date from pins
	pins, _ := s.pinRepo.ListByTripID(tripID)
	var minStart, maxEnd *time.Time
	var coverURL string
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	for _, m := range mediaList {
		if m.PinID == nil {
			continue
		}
		if coverURL == "" && m.MediaType == "image" {
			coverURL = "" // would be Media Service URL from s3_key
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
	return &pb.PublishTripResponse{Trip: tripToProto(updated)}, nil
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
	var locationID *int
	if req.GetLocationId() != 0 {
		id := int(req.GetLocationId())
		locationID = &id
	}
	trips, err := s.tripRepo.ListFeed(limit, offset, req.GetCategory(), req.GetSeason(), locationID, sortBy)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list feed")
	}
	out := make([]*pb.Trip, len(trips))
	for i, t := range trips {
		out[i] = tripToProto(t)
	}
	return &pb.ListFeedResponse{Trips: out}, nil
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
	if err := s.socialRepo.SetReaction(userID, tripID, "Like"); err != nil {
		return nil, status.Error(codes.Internal, "failed to set like")
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
	if err := s.socialRepo.SetReaction(userID, tripID, "Dislike"); err != nil {
		return nil, status.Error(codes.Internal, "failed to set dislike")
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
		out = append(out, tripToProto(trip))
	}
	return &pb.ListFavouritesResponse{Trips: out}, nil
}

func tripToProto(t *models.Trip) *pb.Trip {
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
		CoverUrl:      t.CoverURL,
		IsPublished:   t.IsPublished,
		IsGenerated:   t.IsGenerated,
		CreatedAtUnix: t.CreatedAt.Unix(),
		UpdatedAtUnix: t.UpdatedAt.Unix(),
	}
	if t.StartDate != nil {
		out.StartDateUnix = t.StartDate.Unix()
	}
	if t.EndDate != nil {
		out.EndDateUnix = t.EndDate.Unix()
	}
	return out
}
