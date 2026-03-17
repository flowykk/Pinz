package services

import (
	"context"
	"database/sql"
	"fmt"
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

type TripService struct {
	pb.UnimplementedTripServiceServer
	tripRepo            *repositories.TripRepository
	participantRepo     *repositories.TripParticipantRepository
	inviteRepo          *repositories.InvitationLinkRepository
	settingsRepo        *repositories.TripSettingsRepository
	eventRepo           *repositories.RedisRepository
	mediaRepo           *repositories.MediaRepository
	pinRepo             *repositories.PinRepository
	tagRepo             *repositories.TagRepository
	socialRepo          *repositories.SocialRepository
	favouriteRepo       *repositories.FavouriteRepository
	geoRepo             *repositories.GeoRegistryRepository
	geocodingClient     *GeocodingClient
	mediaURLs           MediaURLResolver
	addMediaSessionRepo *repositories.AddMediaSessionRepository
	tripPrivacyRepo     *repositories.TripPrivacyRepository
	pinPrivacyRepo      *repositories.PinPrivacyRepository
	mediaPrivacyRepo    *repositories.MediaPrivacyRepository
	pinHiddenRepo       *repositories.PinHiddenRepository
}

func NewTripService(
	tripRepo *repositories.TripRepository,
	participantRepo *repositories.TripParticipantRepository,
	inviteRepo *repositories.InvitationLinkRepository,
	settingsRepo *repositories.TripSettingsRepository,
	eventRepo *repositories.RedisRepository,
	mediaRepo *repositories.MediaRepository,
	pinRepo *repositories.PinRepository,
	tagRepo *repositories.TagRepository,
	socialRepo *repositories.SocialRepository,
	favouriteRepo *repositories.FavouriteRepository,
	geoRepo *repositories.GeoRegistryRepository,
	geocodingClient *GeocodingClient,
	addMediaSessionRepo *repositories.AddMediaSessionRepository,
	tripPrivacyRepo *repositories.TripPrivacyRepository,
	pinPrivacyRepo *repositories.PinPrivacyRepository,
	mediaPrivacyRepo *repositories.MediaPrivacyRepository,
	pinHiddenRepo *repositories.PinHiddenRepository,
) *TripService {
	var resolver MediaURLResolver = stubMediaURLResolver{}
	return &TripService{
		tripRepo:            tripRepo,
		participantRepo:     participantRepo,
		inviteRepo:          inviteRepo,
		settingsRepo:        settingsRepo,
		eventRepo:           eventRepo,
		mediaRepo:           mediaRepo,
		pinRepo:             pinRepo,
		tagRepo:             tagRepo,
		socialRepo:          socialRepo,
		favouriteRepo:       favouriteRepo,
		geoRepo:             geoRepo,
		geocodingClient:     geocodingClient,
		mediaURLs:           resolver,
		addMediaSessionRepo: addMediaSessionRepo,
		tripPrivacyRepo:     tripPrivacyRepo,
		pinPrivacyRepo:      pinPrivacyRepo,
		mediaPrivacyRepo:    mediaPrivacyRepo,
		pinHiddenRepo:       pinHiddenRepo,
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
	// Privacy level is set by default; participants change it via UpdateTripPrivacy, worker aggregates.
	privacyLevel := "Private"

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
	for _, f := range req.GetFilesToUpload() {
		if !validateContentType(f.GetContentType()) {
			return nil, status.Error(codes.InvalidArgument, "content_type must be one of: image/jpeg, image/png, image/heic, video/mp4, video/quicktime")
		}
	}
	uploadUrls := make([]*pb.UploadUrl, 0, len(req.GetFilesToUpload()))
	for _, f := range req.GetFilesToUpload() {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + trip.ID + "/" + f.GetClientId() + ext
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key:    s3Key,
			Url:      s.mediaURLs.PresignedUploadURL(s3Key),
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
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check participant")
	}
	if !isParticipant {
		if trip.PrivacyLevel != "Public" {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
	}
	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	if s.pinHiddenRepo != nil {
		hiddenIDs, _ := s.pinHiddenRepo.ListHiddenPinIDsForUser(tripID, userID)
		hiddenSet := make(map[string]struct{}, len(hiddenIDs))
		for _, id := range hiddenIDs {
			hiddenSet[id] = struct{}{}
		}
		filtered := pins[:0]
		for _, p := range pins {
			if _, ok := hiddenSet[p.ID]; !ok {
				filtered = append(filtered, p)
			}
		}
		pins = filtered
	}
	mediaList, err := s.mediaRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list media")
	}
	if !isParticipant {
		pins = filterPinsByPrivacy(pins, "Public")
		mediaList = filterMediaByPrivacy(mediaList, "Public")
	}
	publicPinIDs := make(map[string]struct{})
	for _, p := range pins {
		publicPinIDs[p.ID] = struct{}{}
	}
	mediaByPin := make(map[string][]*models.Media)
	for _, m := range mediaList {
		if m.PinID == nil {
			continue
		}
		if isParticipant {
			mediaByPin[*m.PinID] = append(mediaByPin[*m.PinID], m)
		} else if _, ok := publicPinIDs[*m.PinID]; ok {
			mediaByPin[*m.PinID] = append(mediaByPin[*m.PinID], m)
		}
	}
	outPins := make([]*pb.TripDetailPin, 0, len(pins))
	for _, p := range pins {
		pinMedia := mediaByPin[p.ID]
		pbMedia := make([]*pb.TripDetailMedia, 0, len(pinMedia))
		for _, m := range pinMedia {
			pbMedia = append(pbMedia, &pb.TripDetailMedia{
				Id:           m.ID,
				S3Key:        m.S3Key,
				MediaType:    m.MediaType,
				PrivacyLevel: m.PrivacyLevel,
			})
		}
		dp := &pb.TripDetailPin{
			Id:           p.ID,
			Name:         p.Name,
			Description:  p.Description,
			Category:     p.Category,
			PrivacyLevel: p.PrivacyLevel,
			Media:        pbMedia,
		}
		if p.Latitude != nil {
			dp.Latitude = p.Latitude
		}
		if p.Longitude != nil {
			dp.Longitude = p.Longitude
		}
		if p.StartTime != nil {
			dp.StartTimeUnix = p.StartTime.Unix()
		}
		if p.EndTime != nil {
			dp.EndTimeUnix = p.EndTime.Unix()
		}
		outPins = append(outPins, dp)
	}
	participants, err := s.participantRepo.GetByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list participants")
	}
	participantRefs := make([]*pb.TripParticipantRef, 0, len(participants))
	for _, p := range participants {
		participantRefs = append(participantRefs, &pb.TripParticipantRef{UserId: p.UserID, IsAdmin: p.IsAdmin})
	}
	return &pb.GetTripResponse{Trip: tripToProto(trip), Pins: outPins, Participants: participantRefs}, nil
}

func filterPinsByPrivacy(pins []*models.Pin, level string) []*models.Pin {
	out := make([]*models.Pin, 0, len(pins))
	for _, p := range pins {
		if p.PrivacyLevel == level {
			out = append(out, p)
		}
	}
	return out
}

func filterMediaByPrivacy(media []*models.Media, level string) []*models.Media {
	out := make([]*models.Media, 0, len(media))
	for _, m := range media {
		if m.PrivacyLevel == level {
			out = append(out, m)
		}
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
		InviteUrl:     "",
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
	isAdmin, err := s.participantRepo.IsAdmin(tripID, callerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check admin")
	}
	if !isAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admin can remove participant")
	}
	if targetUserID == callerID {
		return nil, status.Error(codes.InvalidArgument, "use LeaveTrip to leave yourself")
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
		if err := validateMediaMeta(m); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
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
		if meta.ContentHash != nil {
			media.ContentHash = meta.ContentHash
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
				Url:     s.mediaURLs.ReadURL(m.S3Key),
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

// AddMediaStart возвращает s3_key и (в будущем) presigned URL для загрузки новых медиа в существующий трип.
func (s *TripService) AddMediaStart(ctx context.Context, req *pb.AddMediaStartRequest) (*pb.AddMediaStartResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	if _, err := s.tripRepo.GetByID(tripID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}

	existingMediaList, _ := s.mediaRepo.ListByTripID(tripID)
	existingIDs := make([]string, 0, len(existingMediaList))
	for _, m := range existingMediaList {
		existingIDs = append(existingIDs, m.ID)
	}
	sessionID := ""
	if s.addMediaSessionRepo != nil {
		if sid, err := s.addMediaSessionRepo.Create(ctx, tripID, existingIDs); err == nil {
			sessionID = sid
		}
	}

	for _, f := range req.GetFilesToUpload() {
		if !validateContentType(f.GetContentType()) {
			return nil, status.Error(codes.InvalidArgument, "content_type must be one of: image/jpeg, image/png, image/heic, video/mp4, video/quicktime")
		}
	}
	uploadUrls := make([]*pb.UploadUrl, 0, len(req.GetFilesToUpload()))
	for _, f := range req.GetFilesToUpload() {
		ext := contentTypeToExt(f.GetContentType())
		s3Key := "trips/" + tripID + "/" + f.GetClientId() + ext
		uploadUrls = append(uploadUrls, &pb.UploadUrl{
			ClientId: f.GetClientId(),
			S3Key:    s3Key,
			Url:      s.mediaURLs.PresignedUploadURL(s3Key),
		})
	}
	return &pb.AddMediaStartResponse{
		TripId:     tripID,
		SessionId:  sessionID,
		UploadUrls: uploadUrls,
	}, nil
}

// AddMediaProcessGrouping сохраняет метаданные новых медиа и группирует их относительно существующих пинов.
func (s *TripService) AddMediaProcessGrouping(ctx context.Context, req *pb.AddMediaProcessGroupingRequest) (*pb.AddMediaProcessGroupingResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	sessionID := req.GetSessionId()
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
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

	total, videos, _ := s.mediaRepo.CountByTripID(tripID)
	newVideos := 0
	for _, m := range req.GetMedia() {
		if err := validateMediaMeta(m); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
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

	existingList, sidTripID, err := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "add-media session not found")
		}
		return nil, status.Error(codes.Internal, "failed to load add-media session")
	}
	if sidTripID != tripID {
		return nil, status.Error(codes.InvalidArgument, "session_id does not match trip_id")
	}
	existingIDs := make(map[string]struct{}, len(existingList))
	for _, id := range existingList {
		existingIDs[id] = struct{}{}
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
		if meta.ContentHash != nil {
			media.ContentHash = meta.ContentHash
		}
		if err := s.mediaRepo.Create(media); err != nil {
			return nil, status.Error(codes.Internal, "failed to save media")
		}
	}

	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	mediaList, _ := s.mediaRepo.ListByTripID(tripID)
	mediaByID := make(map[string]*models.Media, len(mediaList))
	for _, m := range mediaList {
		mediaByID[m.ID] = m
	}
	var newMedia []*models.Media
	for _, m := range mediaList {
		if _, ok := existingIDs[m.ID]; !ok {
			newMedia = append(newMedia, m)
		}
	}

	attached, remaining, unassigned := attachNewMediaToExistingPins(newMedia, pins, ClusterRadiusMeters, TimeClusterMinutes)
	newPinGroups := clusterNewMediaStandalone(remaining, ClusterRadiusMeters, TimeClusterMinutes)

	// 1) existing pins groups: include existing media (read_only=true) + attached new media (read_only=false)
	respPins := make([]*pb.GroupedPin, 0, len(pins)+len(newPinGroups)+1)
	for _, p := range pins {
		gp := &pb.GroupedPin{PinId: p.ID, ReadOnly: true}
		// existing media for this pin
		for _, m := range mediaList {
			if m.PinID != nil && *m.PinID == p.ID {
				gp.Media = append(gp.Media, &pb.GroupedMedia{
					MediaId:  m.ID,
					ReadOnly: true,
					Url:      s.mediaURLs.ReadURL(m.S3Key),
					Type:     m.MediaType,
				})
			}
		}
		// attached new media
		for _, mid := range attached[p.ID] {
			if m := mediaByID[mid]; m != nil {
				gp.Media = append(gp.Media, &pb.GroupedMedia{
					MediaId:  m.ID,
					ReadOnly: false,
					Url:      s.mediaURLs.ReadURL(m.S3Key),
					Type:     m.MediaType,
				})
			}
		}
		respPins = append(respPins, gp)
	}

	// 2) new pins groups
	for i, group := range newPinGroups {
		gp := &pb.GroupedPin{PinId: fmt.Sprintf("new-%d", i), ReadOnly: false}
		for _, mid := range group {
			if m := mediaByID[mid]; m != nil {
				gp.Media = append(gp.Media, &pb.GroupedMedia{
					MediaId:  m.ID,
					ReadOnly: false,
					Url:      s.mediaURLs.ReadURL(m.S3Key),
					Type:     m.MediaType,
				})
			}
		}
		respPins = append(respPins, gp)
	}

	// 3) unassigned
	if len(unassigned) > 0 {
		gp := &pb.GroupedPin{PinId: "unassigned", ReadOnly: false}
		for _, mid := range unassigned {
			if m := mediaByID[mid]; m != nil {
				gp.Media = append(gp.Media, &pb.GroupedMedia{
					MediaId:  m.ID,
					ReadOnly: false,
					Url:      s.mediaURLs.ReadURL(m.S3Key),
					Type:     m.MediaType,
				})
			}
		}
		respPins = append(respPins, gp)
	}

	return &pb.AddMediaProcessGroupingResponse{
		TripId:    tripID,
		SessionId: sessionID,
		Pins:      respPins,
	}, nil
}

// AddMediaApplyGroupsAndProcess применяет группировку новых медиа и запускает фоновую обработку (ML).
func (s *TripService) AddMediaApplyGroupsAndProcess(ctx context.Context, req *pb.AddMediaApplyGroupsAndProcessRequest) (*pb.AddMediaApplyGroupsAndProcessResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	sessionID := req.GetSessionId()
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
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
	if trip.Status != "READY" {
		return nil, status.Error(codes.FailedPrecondition, "trip must be READY to apply added media")
	}

	if s.addMediaSessionRepo == nil {
		return nil, status.Error(codes.FailedPrecondition, "add-media sessions not configured")
	}
	existingIDsList, sidTripID, err := s.addMediaSessionRepo.GetExistingMediaIDs(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "add-media session not found")
		}
		return nil, status.Error(codes.Internal, "failed to load add-media session")
	}
	if sidTripID != tripID {
		return nil, status.Error(codes.InvalidArgument, "session_id does not match trip_id")
	}
	existingIDs := make(map[string]struct{}, len(existingIDsList))
	for _, id := range existingIDsList {
		existingIDs[id] = struct{}{}
	}
	for _, mid := range req.GetDeletedMediaIds() {
		if _, ok := existingIDs[mid]; ok {
			return nil, status.Error(codes.InvalidArgument, "cannot delete existing media in add-media flow")
		}
	}
	for _, dp := range req.GetDraftPins() {
		for _, mid := range dp.GetMediaIds() {
			if _, ok := existingIDs[mid]; ok {
				return nil, status.Error(codes.InvalidArgument, "cannot move existing media in add-media flow")
			}
		}
	}

	tripMedia, err := s.mediaRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list trip media")
	}
	hashToIDs := make(map[string][]string)
	for _, m := range tripMedia {
		if m.ContentHash != nil && *m.ContentHash != "" {
			hashToIDs[*m.ContentHash] = append(hashToIDs[*m.ContentHash], m.ID)
		}
	}
	duplicateMediaIDs := make(map[string]struct{})
	for _, ids := range hashToIDs {
		if len(ids) <= 1 {
			continue
		}
		for i := 1; i < len(ids); i++ {
			duplicateMediaIDs[ids[i]] = struct{}{}
		}
	}
	toDelete := make([]string, 0, len(req.GetDeletedMediaIds())+len(duplicateMediaIDs))
	toDelete = append(toDelete, req.GetDeletedMediaIds()...)
	for mid := range duplicateMediaIDs {
		toDelete = append(toDelete, mid)
	}
	if len(toDelete) > 0 {
		if err := s.mediaRepo.DeleteByIDs(toDelete); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete media")
		}
	}

	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	pinByID := make(map[string]*models.Pin, len(pins))
	for _, p := range pins {
		pinByID[p.ID] = p
	}

	newPinIDs := make([]string, 0)
	for _, dp := range req.GetDraftPins() {
		mediaIDs := make([]string, 0, len(dp.GetMediaIds()))
		for _, mid := range dp.GetMediaIds() {
			if _, isDup := duplicateMediaIDs[mid]; !isDup {
				mediaIDs = append(mediaIDs, mid)
			}
		}
		if len(mediaIDs) == 0 {
			continue
		}
		draftID := dp.GetDraftPinId()
		if existingPin, ok := pinByID[draftID]; ok {
			if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, existingPin.ID); err != nil {
				return nil, status.Error(codes.Internal, "failed to assign media to existing pin")
			}
			existingPin.MediaCount += int32(len(mediaIDs))
			if err := s.pinRepo.Update(existingPin); err != nil {
				return nil, status.Error(codes.Internal, "failed to update existing pin")
			}
			updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, existingPin.ID)
			continue
		}

		pin := &models.Pin{
			TripID:       tripID,
			Name:         "Pin",
			Description:  "",
			Category:     trip.Category,
			PrivacyLevel: trip.PrivacyLevel,
			MediaCount:   int32(len(mediaIDs)),
		}
		if err := s.pinRepo.Create(pin); err != nil {
			return nil, status.Error(codes.Internal, "failed to create pin")
		}
		newPinIDs = append(newPinIDs, pin.ID)
		if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, pin.ID); err != nil {
			return nil, status.Error(codes.Internal, "failed to assign media to pin")
		}
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	}

	if err := s.tripRepo.SetStatus(tripID, "PROCESSING"); err != nil {
		return nil, status.Error(codes.Internal, "failed to update status")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.SetMLContext(ctx, tripID, "add_media", newPinIDs, 2*time.Hour)
		_ = s.eventRepo.AddMLTaskWithFlow(ctx, tripID, "add_media", newPinIDs)
	}

	return &pb.AddMediaApplyGroupsAndProcessResponse{
		Message: "Processing started",
		Status:  "PROCESSING",
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
	tripMedia, err := s.mediaRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list trip media")
	}
	tripMediaIDs := make(map[string]struct{}, len(tripMedia))
	for _, m := range tripMedia {
		tripMediaIDs[m.ID] = struct{}{}
	}
	hashToIDs := make(map[string][]string)
	for _, m := range tripMedia {
		if m.ContentHash != nil && *m.ContentHash != "" {
			hashToIDs[*m.ContentHash] = append(hashToIDs[*m.ContentHash], m.ID)
		}
	}
	duplicateMediaIDs := make(map[string]struct{})
	for _, ids := range hashToIDs {
		if len(ids) <= 1 {
			continue
		}
		for i := 1; i < len(ids); i++ {
			duplicateMediaIDs[ids[i]] = struct{}{}
		}
	}
	for _, mid := range req.GetDeletedMediaIds() {
		if _, ok := tripMediaIDs[mid]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "media_id %q does not belong to this trip", mid)
		}
	}
	seenInDraftPins := make(map[string]struct{})
	for _, dp := range req.GetDraftPins() {
		for _, mid := range dp.GetMediaIds() {
			if _, ok := tripMediaIDs[mid]; !ok {
				return nil, status.Errorf(codes.InvalidArgument, "media_id %q does not belong to this trip", mid)
			}
			if _, dup := seenInDraftPins[mid]; dup {
				return nil, status.Errorf(codes.InvalidArgument, "media_id %q must not appear in more than one pin", mid)
			}
			seenInDraftPins[mid] = struct{}{}
		}
	}
	toDelete := make([]string, 0, len(req.GetDeletedMediaIds())+len(duplicateMediaIDs))
	toDelete = append(toDelete, req.GetDeletedMediaIds()...)
	for mid := range duplicateMediaIDs {
		toDelete = append(toDelete, mid)
	}
	if len(toDelete) > 0 {
		if err := s.mediaRepo.DeleteByIDs(toDelete); err != nil {
			return nil, status.Error(codes.Internal, "failed to delete media")
		}
	}
	for _, dp := range req.GetDraftPins() {
		mediaIDs := make([]string, 0, len(dp.GetMediaIds()))
		for _, mid := range dp.GetMediaIds() {
			if _, isDup := duplicateMediaIDs[mid]; !isDup {
				mediaIDs = append(mediaIDs, mid)
			}
		}
		if len(mediaIDs) == 0 {
			continue
		}
		pin := &models.Pin{
			TripID:       tripID,
			Name:         "Pin",
			Description:  "",
			Category:     trip.Category,
			PrivacyLevel: trip.PrivacyLevel,
			MediaCount:   int32(len(mediaIDs)),
		}
		if err := s.pinRepo.Create(pin); err != nil {
			return nil, status.Error(codes.Internal, "failed to create pin")
		}
		if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, pin.ID); err != nil {
			return nil, status.Error(codes.Internal, "failed to assign media to pin")
		}
		updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	}
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
	if s.pinHiddenRepo != nil {
		hiddenIDs, _ := s.pinHiddenRepo.ListHiddenPinIDsForUser(tripID, userID)
		hiddenSet := make(map[string]struct{}, len(hiddenIDs))
		for _, id := range hiddenIDs {
			hiddenSet[id] = struct{}{}
		}
		filtered := pins[:0]
		for _, p := range pins {
			if _, ok := hiddenSet[p.ID]; !ok {
				filtered = append(filtered, p)
			}
		}
		pins = filtered
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
				Url:          s.mediaURLs.ReadURL(m.S3Key),
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
	for _, pu := range req.GetPinUpdates() {
		pin, err := s.pinRepo.GetByID(pu.GetPinId())
		if err != nil {
			continue
		}
		if pin.TripID != tripID {
			continue
		}
		if pu.Name != nil {
			name := pu.GetName()
			if len(name) > MaxNameLength {
				return nil, status.Errorf(codes.InvalidArgument, "pin name must be at most %d characters", MaxNameLength)
			}
			pin.Name = name
		}
		if pu.Description != nil {
			desc := pu.GetDescription()
			if len(desc) > MaxDescriptionLength {
				return nil, status.Errorf(codes.InvalidArgument, "pin description must be at most %d characters", MaxDescriptionLength)
			}
			pin.Description = desc
		}
		if pu.Category != nil {
			if !validatePinCategory(pu.GetCategory()) {
				return nil, status.Error(codes.InvalidArgument, "invalid pin category")
			}
			pin.Category = pu.GetCategory()
		}
		if pu.PrivacyLevel != nil {
			if !validatePrivacyLevel(pu.GetPrivacyLevel()) {
				return nil, status.Error(codes.InvalidArgument, "invalid pin privacy_level")
			}
			pin.PrivacyLevel = pu.GetPrivacyLevel()
		}
		if pu.Latitude != nil && pu.Longitude != nil {
			pin.Latitude = pu.Latitude
			pin.Longitude = pu.Longitude
		}
		if err := s.pinRepo.Update(pin); err != nil {
			return nil, status.Error(codes.Internal, "failed to update pin")
		}
		if len(pu.GetTags()) > 0 {
			if err := s.tagRepo.SetForPin(tripID, pin.ID, pu.GetTags()); err != nil {
				return nil, status.Error(codes.Internal, "failed to update pin tags")
			}
		}
	}
	// Delete media (DB only; S3 via Media Service later)
	if len(req.GetMediaToDelete()) > 0 {
		_ = s.mediaRepo.DeleteByIDs(req.GetMediaToDelete())
	}
	pins, _ := s.pinRepo.ListByTripID(tripID)
	var minStart, maxEnd *time.Time
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
		return nil, status.Error(codes.FailedPrecondition, "cannot publish private trip; set privacy to Public first")
	}

	publishWhole := req.GetPublishWhole()
	pinIDs := req.GetPinIds()

	if publishWhole && len(pinIDs) > 0 {
		return nil, status.Error(codes.InvalidArgument, "pin_ids must be empty when publish_whole is true")
	}
	if !publishWhole && len(pinIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "pin_ids must be provided when publish_whole is false")
	}

	pins, err := s.pinRepo.ListByTripID(tripID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pins")
	}
	pinIDSet := make(map[string]*models.Pin, len(pins))
	for _, p := range pins {
		pinIDSet[p.ID] = p
	}

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
	var locationIDs []int
	if req.GetLocationName() != "" {
		ids, err := s.geoRepo.FindLocationIDsByName(ctx, req.GetLocationName())
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to resolve location")
		}
		locationIDs = ids
	} else if req.GetLocationId() != 0 {
		locationIDs = []int{int(req.GetLocationId())}
	}
	trips, err := s.tripRepo.ListFeed(limit, offset, req.GetCategory(), req.GetSeason(), locationIDs, sortBy)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list feed")
	}
	tripIDs := make([]string, len(trips))
	for i, t := range trips {
		tripIDs[i] = t.ID
	}
	pinsByTrip, _ := s.pinRepo.ListPublishedPinsByTripIDs(tripIDs)
	mediaByTrip, _ := s.mediaRepo.TopMediaByTripIDs(tripIDs, 8)
	out := make([]*pb.Trip, len(trips))
	cards := make([]*pb.FeedCard, len(trips))
	for i, t := range trips {
		out[i] = tripToProto(t)
		card := &pb.FeedCard{Trip: out[i]}
		for _, fp := range pinsByTrip[t.ID] {
			card.Pins = append(card.Pins, &pb.FeedCardPin{Id: fp.ID, Latitude: fp.Latitude, Longitude: fp.Longitude})
		}
		for _, fm := range mediaByTrip[t.ID] {
			card.Media = append(card.Media, &pb.FeedCardMedia{Id: fm.ID, S3Key: fm.S3Key, MediaType: fm.MediaType})
		}
		cards[i] = card
	}
	return &pb.ListFeedResponse{Trips: out, Cards: cards}, nil
}

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

func (s *TripService) UpdateTripPrivacy(ctx context.Context, req *pb.UpdateTripPrivacyRequest) (*pb.UpdateTripPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	level := req.GetPrivacyLevel()
	if level != "Public" && level != "Private" {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be Public or Private (Restricted is set only by system)")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if s.tripPrivacyRepo == nil {
		return nil, status.Error(codes.Unavailable, "privacy not configured")
	}
	if err := s.tripPrivacyRepo.Upsert(ctx, tripID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to save privacy choice")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "trip", tripID, tripID, userID, level)
	}
	return &pb.UpdateTripPrivacyResponse{Success: true}, nil
}

// UpdatePinPrivacy: участник меняет приватность пина; Restricted запрещён.
func (s *TripService) UpdatePinPrivacy(ctx context.Context, req *pb.UpdatePinPrivacyRequest) (*pb.UpdatePinPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	pinID := req.GetPinId()
	if pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "pin_id is required")
	}
	level := req.GetPrivacyLevel()
	if level != "Public" && level != "Private" {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be Public or Private")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	isParticipant, err := s.participantRepo.IsParticipant(pin.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if s.pinPrivacyRepo == nil {
		return nil, status.Error(codes.Unavailable, "privacy not configured")
	}
	if err := s.pinPrivacyRepo.Upsert(ctx, pinID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to save privacy choice")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "pin", pinID, pin.TripID, userID, level)
	}
	return &pb.UpdatePinPrivacyResponse{Success: true}, nil
}

func (s *TripService) UpdatePin(ctx context.Context, req *pb.UpdatePinRequest) (*pb.UpdatePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	pinID := req.GetPinId()
	if pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "pin_id is required")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	isParticipant, err := s.participantRepo.IsParticipant(pin.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len(name) > MaxNameLength {
			return nil, status.Errorf(codes.InvalidArgument, "name must be at most %d characters", MaxNameLength)
		}
		pin.Name = name
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > MaxDescriptionLength {
			return nil, status.Errorf(codes.InvalidArgument, "description must be at most %d characters", MaxDescriptionLength)
		}
		pin.Description = desc
	}
	if req.Category != nil {
		c := strings.TrimSpace(*req.Category)
		if !validatePinCategory(c) {
			return nil, status.Error(codes.InvalidArgument, "invalid pin category")
		}
		pin.Category = c
	}
	if req.PrivacyLevel != nil {
		level := *req.PrivacyLevel
		if level != "Public" && level != "Private" {
			return nil, status.Error(codes.InvalidArgument, "privacy_level must be Public or Private")
		}
		pin.PrivacyLevel = level
	}
	if req.Latitude != nil && req.Longitude != nil {
		lat, lon := *req.Latitude, *req.Longitude
		pin.Latitude = &lat
		pin.Longitude = &lon
	} else if req.Latitude != nil || req.Longitude != nil {
		return nil, status.Error(codes.InvalidArgument, "both latitude and longitude must be set together")
	}
	if req.StartTimeUnix != nil {
		t := time.Unix(*req.StartTimeUnix, 0)
		pin.StartTime = &t
	}
	if req.EndTimeUnix != nil {
		t := time.Unix(*req.EndTimeUnix, 0)
		pin.EndTime = &t
	}
	if err := s.pinRepo.Update(pin); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to update pin")
	}
	return &pb.UpdatePinResponse{Success: true}, nil
}

// UpdateMediaPrivacy: участник меняет приватность медиа; Restricted запрещён.
func (s *TripService) UpdateMediaPrivacy(ctx context.Context, req *pb.UpdateMediaPrivacyRequest) (*pb.UpdateMediaPrivacyResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	mediaID := req.GetMediaId()
	if mediaID == "" {
		return nil, status.Error(codes.InvalidArgument, "media_id is required")
	}
	level := req.GetPrivacyLevel()
	if level != "Public" && level != "Private" {
		return nil, status.Error(codes.InvalidArgument, "privacy_level must be Public or Private")
	}
	media, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Error(codes.Internal, "failed to get media")
	}
	isParticipant, err := s.participantRepo.IsParticipant(media.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if s.mediaPrivacyRepo == nil {
		return nil, status.Error(codes.Unavailable, "privacy not configured")
	}
	if err := s.mediaPrivacyRepo.Upsert(ctx, mediaID, userID, level); err != nil {
		return nil, status.Error(codes.Internal, "failed to save privacy choice")
	}
	if s.eventRepo != nil {
		_ = s.eventRepo.PublishPrivacyEvent(ctx, "media", mediaID, media.TripID, userID, level)
	}
	return &pb.UpdateMediaPrivacyResponse{Success: true}, nil
}

func (s *TripService) SearchPins(ctx context.Context, req *pb.SearchPinsRequest) (*pb.SearchPinsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	query := req.GetQuery()
	if query == "" {
		return &pb.SearchPinsResponse{Pins: nil}, nil
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
		return nil, status.Error(codes.Internal, "search failed")
	}
	items := make([]*pb.SearchPinItem, 0, len(pins))
	for _, p := range pins {
		tags, _ := s.tagRepo.GetByPinID(p.ID)
		items = append(items, &pb.SearchPinItem{
			PinId:       p.ID,
			TripId:      p.TripID,
			Name:        p.Name,
			Description: p.Description,
			Category:    p.Category,
			Tags:        tags,
		})
	}
	return &pb.SearchPinsResponse{Pins: items}, nil
}

func (s *TripService) CreatePin(ctx context.Context, req *pb.CreatePinRequest) (*pb.CreatePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	mediaIDs := req.GetMediaIds()
	if len(mediaIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one media_id is required")
	}
	isParticipant, err := s.participantRepo.IsParticipant(tripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	trip, err := s.tripRepo.GetByID(tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "trip not found")
		}
		return nil, status.Error(codes.Internal, "failed to get trip")
	}
	for _, mid := range mediaIDs {
		media, err := s.mediaRepo.GetByID(mid)
		if err != nil || media == nil || media.TripID != tripID {
			return nil, status.Error(codes.InvalidArgument, "media must belong to the trip")
		}
	}
	pin := &models.Pin{
		TripID:       tripID,
		Name:         "Pin",
		Category:     trip.Category,
		PrivacyLevel: trip.PrivacyLevel,
		MediaCount:   int32(len(mediaIDs)),
	}
	if err := s.pinRepo.Create(pin); err != nil {
		return nil, status.Error(codes.Internal, "failed to create pin")
	}
	if err := s.mediaRepo.UpdatePinIDByIDs(mediaIDs, pin.ID); err != nil {
		_ = s.pinRepo.Delete(pin.ID)
		return nil, status.Error(codes.Internal, "failed to assign media to pin")
	}
	updatePinTimesAndLocation(s.pinRepo, s.mediaRepo, pin.ID)
	return &pb.CreatePinResponse{PinId: pin.ID}, nil
}

func (s *TripService) DeletePin(ctx context.Context, req *pb.DeletePinRequest) (*pb.DeletePinResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	pinID := req.GetPinId()
	if pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "pin_id is required")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	isParticipant, err := s.participantRepo.IsParticipant(pin.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	if s.pinHiddenRepo != nil {
		inOthersFav, err := s.favouriteRepo.HasFavouritesByOtherUsers(pin.TripID, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check favourites")
		}
		if inOthersFav {
			if err := s.pinHiddenRepo.HidePinForUser(pinID, userID); err != nil {
				return nil, status.Error(codes.Internal, "failed to hide pin")
			}
			return &pb.DeletePinResponse{Success: true}, nil
		}
	}
	if err := s.pinRepo.Delete(pinID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete pin")
	}
	return &pb.DeletePinResponse{Success: true}, nil
}

func (s *TripService) AddPinTags(ctx context.Context, req *pb.AddPinTagsRequest) (*pb.AddPinTagsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	pinID := req.GetPinId()
	if pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "pin_id is required")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	isParticipant, err := s.participantRepo.IsParticipant(pin.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	current, _ := s.tagRepo.GetByPinID(pinID)
	seen := make(map[string]struct{}, len(current))
	for _, t := range current {
		seen[t] = struct{}{}
	}
	for _, t := range req.GetTags() {
		if t != "" && len(t) <= MaxTagLength {
			seen[t] = struct{}{}
		}
	}
	merged := make([]string, 0, len(seen))
	for t := range seen {
		merged = append(merged, t)
	}
	if len(merged) > MaxTagsPerPin {
		merged = merged[:MaxTagsPerPin]
	}
	if err := s.tagRepo.SetForPin(pin.TripID, pinID, merged); err != nil {
		return nil, status.Error(codes.Internal, "failed to update tags")
	}
	return &pb.AddPinTagsResponse{Success: true}, nil
}

func (s *TripService) RemovePinTags(ctx context.Context, req *pb.RemovePinTagsRequest) (*pb.RemovePinTagsResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	pinID := req.GetPinId()
	if pinID == "" {
		return nil, status.Error(codes.InvalidArgument, "pin_id is required")
	}
	pin, err := s.pinRepo.GetByID(pinID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "pin not found")
		}
		return nil, status.Error(codes.Internal, "failed to get pin")
	}
	isParticipant, err := s.participantRepo.IsParticipant(pin.TripID, userID)
	if err != nil || !isParticipant {
		return nil, status.Error(codes.PermissionDenied, "not a participant")
	}
	toRemove := make(map[string]struct{}, len(req.GetTags()))
	for _, t := range req.GetTags() {
		toRemove[t] = struct{}{}
	}
	current, _ := s.tagRepo.GetByPinID(pinID)
	var merged []string
	for _, t := range current {
		if _, ok := toRemove[t]; !ok {
			merged = append(merged, t)
		}
	}
	if err := s.tagRepo.SetForPin(pin.TripID, pinID, merged); err != nil {
		return nil, status.Error(codes.Internal, "failed to update tags")
	}
	return &pb.RemovePinTagsResponse{Success: true}, nil
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
