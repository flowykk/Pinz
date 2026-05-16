package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// buildService — сахар для тестов. Инжектит необходимые моки, остальные — nil.
func buildService(tripRepo repositories.TripRepositoryInterface,
	participantRepo repositories.TripParticipantRepositoryInterface,
	sessionRepo repositories.AddMediaSessionRepositoryInterface,
	mediaRepo repositories.MediaRepositoryInterface,
	pinRepo repositories.PinRepositoryInterface,
	tagRepo repositories.TagRepositoryInterface,
	eventRepo repositories.TripEventPublisher) *TripService {
	return NewTripService(
		tripRepo, participantRepo, nil, nil,
		eventRepo, mediaRepo, nil, pinRepo, tagRepo,
		nil, nil, nil, sessionRepo, nil,
		nil, nil, nil, nil, nil, nil,
	)
}

// B1: второй AddMediaStart в момент race видит существующую сессию
// через GetActive и возвращает joined=true без создания новой.
func TestAddMediaStart_Race_ReturnsJoined(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-2"
	ctrl := gomock.NewController(t)

	trip := &models.Trip{ID: tripID, Status: models.TripStatusReady, PrivacyLevel: "private"}
	activeSession := &models.AddMediaSession{SessionID: "sess-1", TripID: tripID}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(activeSession, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	resp, err := svc.AddMediaStart(ctxWithUser(userID), &pb.AddMediaStartRequest{TripId: tripID})
	require.NoError(t, err)
	require.True(t, resp.GetJoined(), "second Start must return joined=true")
	require.Equal(t, "sess-1", resp.GetSessionId())
	require.Empty(t, resp.GetUploadUrls(), "joined=true response must have no upload_urls")
}

// Новый Start из READY без активной сессии создаёт её и переводит трип в UPLOADING.
func TestAddMediaStart_NewSession_Success(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	trip := &models.Trip{ID: tripID, Status: models.TripStatusReady, PrivacyLevel: "private"}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(nil, repositories.ErrNoActiveSession)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	mediaRepo.EXPECT().ListByTripID(tripID).Return(nil, nil)
	sessionRepo.EXPECT().Create(gomock.Any(), tripID, gomock.Any()).Return("sess-new", nil)
	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusAddMediaUploading).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, mediaRepo, nil, nil, nil)
	resp, err := svc.AddMediaStart(ctxWithUser(userID), &pb.AddMediaStartRequest{
		TripId: tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.NoError(t, err)
	require.False(t, resp.GetJoined())
	require.Equal(t, "sess-new", resp.GetSessionId())
	require.Equal(t, models.TripStatusAddMediaUploading, resp.GetStatus())
}

// ProcessGrouping с флагом add_more=true откатывает GROUPING_REVIEW → UPLOADING,
// не удаляя ранее загруженные медиа (они живут в media по session_id).
func TestAddMediaProcessGrouping_AddMore_Rollback(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	const sessionID = "sess-1"
	ctrl := gomock.NewController(t)

	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaGroupingReview}
	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusAddMediaUploading).Return(nil)
	sessionRepo.EXPECT().Touch(gomock.Any(), sessionID, gomock.Any()).Return(nil)
	sessionRepo.EXPECT().GetExistingMediaIDs(gomock.Any(), sessionID).Return([]string{"old-1"}, tripID, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	resp, err := svc.AddMediaProcessGrouping(ctxWithUser(userID), &pb.AddMediaProcessGroupingRequest{
		TripId: tripID,
		SessionId: sessionID,
		AddMore: true,
	})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusAddMediaUploading, resp.GetStatus())
	require.Empty(t, resp.GetDraftPins())
	require.Equal(t, []string{"old-1"}, resp.GetExistingMediaIds())
}

// validateActiveSession возвращает FailedPrecondition, если session_id клиента
// устарел (api-gateway маппит в 409/FailedPrecondition — маркер для перечтения /trips).
func TestValidateActiveSession_Mismatch(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	active := &models.AddMediaSession{SessionID: "sess-current", TripID: tripID}

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)

	svc := buildService(nil, participantRepo, sessionRepo, nil, nil, nil, nil)
	_, err := svc.AddMediaRequestUploadUrls(ctxWithUser(userID), &pb.AddMediaRequestUploadUrlsRequest{
		TripId: tripID,
		SessionId: "sess-old",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "session_id does not match")
}

// Confirm от не-ведущего ДО истечения часа — 403 PermissionDenied.
func TestAddMediaConfirm_NotInitiator_Denied(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const initiatorUserID = "user-initiator"
	const otherUserID = "user-other"
	ctrl := gomock.NewController(t)

	// Ведущий активен (30 минут назад), ещё не истёк.
	assignedAt := time.Now().Add(-30 * time.Minute)
	initiator := initiatorUserID
	active := &models.AddMediaSession{
		SessionID: sessionID,
		TripID: tripID,
		CurrentInitiatorUserID: &initiator,
		InitiatorAssignedAt: &assignedAt,
	}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, otherUserID).Return(true, nil)
	// Для идемпотентного short-circuit сначала смотрится active session; в нашем случае активна.
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	// Статус проверяется ДО ensureInitiator (иначе 403 с пустым initiator).
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}, nil)
	// ensureInitiator → validateActiveSession → второй GetActive.
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	_, err := svc.AddMediaConfirm(ctxWithUser(otherUserID), &pb.AddMediaConfirmRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not the current initiator")
}

// Confirm в неправильном статусе (UPLOADING) должен возвращать WRONG_STATUS (412),
// а не NOT_INITIATOR (403) — даже если сессия активна, но initiator ещё не назначен.
func TestAddMediaConfirm_WrongStatus_Returns412(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	// Сессия активна, но ведущий ещё не назначен (добавления до Apply).
	active := &models.AddMediaSession{
		SessionID: sessionID,
		TripID: tripID,
		CurrentInitiatorUserID: nil,
	}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	_, err := svc.AddMediaConfirm(ctxWithUser(userID), &pb.AddMediaConfirmRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	require.Error(t, err)
	// Ошибка должна быть WRONG_STATUS (FailedPrecondition), не NOT_INITIATOR (PermissionDenied).
	require.Contains(t, err.Error(), "trip status does not allow this operation")
	require.NotContains(t, err.Error(), "not the current initiator")
}

// Confirm от ведущего успешно закрывает сессию, трип → READY, идёт ADD_MEDIA_SESSION_COMPLETED.
func TestAddMediaConfirm_Initiator_Success(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const initiatorUserID = "user-initiator"
	ctrl := gomock.NewController(t)

	assignedAt := time.Now().Add(-5 * time.Minute)
	initiator := initiatorUserID
	active := &models.AddMediaSession{
		SessionID: sessionID,
		TripID: tripID,
		CurrentInitiatorUserID: &initiator,
		InitiatorAssignedAt: &assignedAt,
	}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, initiatorUserID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil) // идемпотентность-check
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil) // ensureInitiator
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	// applyReviewEdits без правок — делает только агрегацию cover/dates.
	pinRepo.EXPECT().ListByTripIDIncludingDrafts(tripID).Return(nil, nil)
	mediaRepo.EXPECT().ListByTripID(tripID).Return(nil, nil)
	tripRepo.EXPECT().Update(gomock.Any()).Return(nil)
	pinRepo.EXPECT().ClearAddMediaSessionByID(sessionID).Return(nil)
	sessionRepo.EXPECT().Close(gomock.Any(), sessionID, models.AddMediaSessionCloseReasonConfirmed, gomock.Any()).Return(tripID, nil)
	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusReady).Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "ADD_MEDIA_SESSION_COMPLETED", tripID, initiatorUserID).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "TRIP_STATUS_CHANGED", gomock.Any()).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, mediaRepo, pinRepo, nil, eventRepo)
	resp, err := svc.AddMediaConfirm(ctxWithUser(initiatorUserID), &pb.AddMediaConfirmRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusReady, resp.GetStatus())
	require.False(t, resp.GetAlreadyConfirmed())
}

// Идемпотентность Confirm: повторный вызов, когда сессия уже закрыта и трип READY.
func TestAddMediaConfirm_AlreadyConfirmed_Idempotent(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	trip := &models.Trip{ID: tripID, Status: models.TripStatusReady}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(nil, repositories.ErrNoActiveSession)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	resp, err := svc.AddMediaConfirm(ctxWithUser(userID), &pb.AddMediaConfirmRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.True(t, resp.GetAlreadyConfirmed())
	require.Equal(t, models.TripStatusReady, resp.GetStatus())
}

// Неявный перехват ведущего после истечения часа. Следующий мутирующий запрос
// от другого participant'а должен пройти; ensureInitiator переназначает ведущего
// и публикует ADD_MEDIA_INITIATOR_CHANGED.
func TestEnsureInitiator_TakeoverAfterHour(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const initiatorUserID = "user-initiator"
	const newUserID = "user-new"
	ctrl := gomock.NewController(t)

	// час + 5 минут назад — timeout истёк.
	assignedAt := time.Now().Add(-(time.Hour + 5*time.Minute))
	initiator := initiatorUserID
	active := &models.AddMediaSession{
		SessionID: sessionID,
		TripID: tripID,
		CurrentInitiatorUserID: &initiator,
		InitiatorAssignedAt: &assignedAt,
	}

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	sessionRepo.EXPECT().SetInitiator(gomock.Any(), sessionID, newUserID, gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "ADD_MEDIA_INITIATOR_CHANGED", gomock.Any()).Return(nil)

	svc := buildService(nil, participantRepo, sessionRepo, nil, nil, nil, eventRepo)
	got, err := svc.ensureInitiator(context.Background(), tripID, sessionID, newUserID)
	require.NoError(t, err)
	require.NotNil(t, got.CurrentInitiatorUserID)
	require.Equal(t, newUserID, *got.CurrentInitiatorUserID)
}

// Cancel на UPLOADING любым participant'ом: удаляет orphan медиа, статус → READY.
func TestAddMediaCancel_UploadingByAnyone_DeletesOrphans(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-other"
	ctrl := gomock.NewController(t)

	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID, ExistingMediaIDs: []string{"old-1"}}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	sessionRepo.EXPECT().GetPendingExistingAttachments(gomock.Any(), sessionID).Return(nil, nil)
	pinRepo.EXPECT().DeleteByAddMediaSessionID(sessionID).Return(nil, nil)
	mediaRepo.EXPECT().DeleteOrphanSessionMedia(tripID, []string{"old-1"}).Return([]string{"s3/new-1"}, nil)
	sessionRepo.EXPECT().Close(gomock.Any(), sessionID, models.AddMediaSessionCloseReasonCancelled, gomock.Any()).Return(tripID, nil)
	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusReady).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "TRIP_STATUS_CHANGED", gomock.Any()).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, mediaRepo, pinRepo, nil, eventRepo)
	resp, err := svc.AddMediaCancel(ctxWithUser(userID), &pb.AddMediaCancelRequest{
		TripId: tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusReady, resp.GetStatus())
}

// AddMediaTakeover: caller — уже ведущий, idempotent no-op без mutation
// и без публикации INITIATOR_CHANGED.
func TestAddMediaTakeover_Idempotent_AlreadyLeader(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-leader"
	ctrl := gomock.NewController(t)

	assignedAt := time.Now().Add(-30 * time.Minute)
	leader := userID
	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID, CurrentInitiatorUserID: &leader, InitiatorAssignedAt: &assignedAt}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	// Никаких SetInitiator / WS publish — это idempotent no-op.

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	resp, err := svc.AddMediaTakeover(ctxWithUser(userID), &pb.AddMediaTakeoverRequest{
		TripId:    tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.True(t, resp.GetIsInitiator())
	require.Equal(t, userID, resp.GetCurrentInitiatorUserId())
	require.Equal(t, assignedAt.Add(initiatorTakeoverTimeout).Unix(), resp.GetTakeoverAvailableAtUnix())
}

// AddMediaTakeover: рано (час не прошёл) → 403 NOT_INITIATOR + payload.
func TestAddMediaTakeover_TooEarly_403(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const leaderID = "user-leader"
	const callerID = "user-other"
	ctrl := gomock.NewController(t)

	assignedAt := time.Now().Add(-30 * time.Minute)
	leader := leaderID
	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID, CurrentInitiatorUserID: &leader, InitiatorAssignedAt: &assignedAt}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, callerID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	_, err := svc.AddMediaTakeover(ctxWithUser(callerID), &pb.AddMediaTakeoverRequest{
		TripId:    tripID,
		SessionId: sessionID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not the current initiator")
}

// AddMediaTakeover: час прошёл → SetInitiator(caller, now) + WS INITIATOR_CHANGED.
func TestAddMediaTakeover_AfterHour_Success(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const leaderID = "user-leader"
	const callerID = "user-other"
	ctrl := gomock.NewController(t)

	assignedAt := time.Now().Add(-2 * time.Hour)
	leader := leaderID
	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID, CurrentInitiatorUserID: &leader, InitiatorAssignedAt: &assignedAt}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, callerID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	sessionRepo.EXPECT().SetInitiator(gomock.Any(), sessionID, callerID, gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "ADD_MEDIA_INITIATOR_CHANGED", gomock.Any()).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, eventRepo)
	resp, err := svc.AddMediaTakeover(ctxWithUser(callerID), &pb.AddMediaTakeoverRequest{
		TripId:    tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.True(t, resp.GetIsInitiator())
	require.Equal(t, callerID, resp.GetCurrentInitiatorUserId())
	require.Greater(t, resp.GetTakeoverAvailableAtUnix(), time.Now().Add(50*time.Minute).Unix())
}

// AddMediaTakeover: не на DRAFT_FINAL_REVIEW → 412 WRONG_STATUS.
func TestAddMediaTakeover_WrongStatus_412(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, nil, nil, nil, nil)
	_, err := svc.AddMediaTakeover(ctxWithUser(userID), &pb.AddMediaTakeoverRequest{
		TripId:    tripID,
		SessionId: sessionID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "trip status does not allow this operation")
}

func TestAddMediaApply_MarksPinDraftAndJournalsExistingAttachments(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	const existingPinID = "pin-existing"
	ctrl := gomock.NewController(t)

	active := &models.AddMediaSession{SessionID: sessionID, TripID: tripID, ExistingMediaIDs: []string{"old-1"}}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaGroupingReview, Category: "Другое", PrivacyLevel: "Private"}
	existingPin := &models.Pin{ID: existingPinID, TripID: tripID}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	sessionRepo.EXPECT().GetExistingMediaIDs(gomock.Any(), sessionID).Return([]string{"old-1"}, tripID, nil)

	mediaRepo.EXPECT().ListByTripID(tripID).Return(nil, nil)
	pinRepo.EXPECT().GetByID(existingPinID).Return(existingPin, nil)
	mediaRepo.EXPECT().UpdatePinIDByIDs([]string{"new-existing"}, existingPinID).Return(nil)
	sid := sessionID
	pinRepo.EXPECT().Create(gomock.AssignableToTypeOf(&models.Pin{})).DoAndReturn(func(p *models.Pin) error {
		require.NotNil(t, p.AddMediaSessionID)
		require.Equal(t, sid, *p.AddMediaSessionID)
		p.ID = "pin-new"
		return nil
	})
	mediaRepo.EXPECT().UpdatePinIDByIDs([]string{"new-cluster"}, "pin-new").Return(nil)
	sessionRepo.EXPECT().AppendPendingExistingAttachments(gomock.Any(), sessionID, []string{"new-existing"}).Return(nil)

	mediaRepo.EXPECT().ListByPinID(gomock.Any()).Return(nil, nil).AnyTimes()
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil).AnyTimes()

	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusAddMediaProcessing).Return(nil)
	sessionRepo.EXPECT().SetInitiator(gomock.Any(), sessionID, userID, gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "TRIP_STATUS_CHANGED", gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "PIN_ADDED", tripID, userID).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, mediaRepo, pinRepo, nil, eventRepo)
	resp, err := svc.AddMediaApplyGroupsAndProcess(ctxWithUser(userID), &pb.AddMediaApplyGroupsAndProcessRequest{
		TripId:    tripID,
		SessionId: sessionID,
		DraftPins: []*pb.DraftPinInput{
			{DraftPinId: "existing-" + existingPinID, MediaIds: []string{"new-existing"}},
			{DraftPinId: "cluster-1", MediaIds: []string{"new-cluster"}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusAddMediaProcessing, resp.GetStatus())
}

func TestAddMediaCancel_AfterApply_DeletesDraftPinsAndRollsBackExisting(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-leader"
	ctrl := gomock.NewController(t)

	assignedAt := time.Now().Add(-5 * time.Minute)
	leader := userID
	active := &models.AddMediaSession{
		SessionID:              sessionID,
		TripID:                 tripID,
		ExistingMediaIDs:       []string{"old-1"},
		CurrentInitiatorUserID: &leader,
		InitiatorAssignedAt:    &assignedAt,
	}
	trip := &models.Trip{ID: tripID, Status: models.TripStatusAddMediaDraftFinalReview}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockAddMediaSessionRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	sessionRepo.EXPECT().GetActive(gomock.Any(), tripID).Return(active, nil)

	sessionRepo.EXPECT().GetPendingExistingAttachments(gomock.Any(), sessionID).Return([]string{"attached-1"}, nil)
	mediaRepo.EXPECT().ClearPinIDByIDs([]string{"attached-1"}).Return(nil)
	pinRepo.EXPECT().DeleteByAddMediaSessionID(sessionID).Return([]string{"draft-pin-1"}, nil)
	mediaRepo.EXPECT().DeleteOrphanSessionMedia(tripID, []string{"old-1"}).Return(nil, nil)
	sessionRepo.EXPECT().Close(gomock.Any(), sessionID, models.AddMediaSessionCloseReasonCancelled, gomock.Any()).Return(tripID, nil)
	tripRepo.EXPECT().SetStatus(tripID, models.TripStatusReady).Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), tripID, "TRIP_STATUS_CHANGED", gomock.Any()).Return(nil)

	svc := buildService(tripRepo, participantRepo, sessionRepo, mediaRepo, pinRepo, nil, eventRepo)
	resp, err := svc.AddMediaCancel(ctxWithUser(userID), &pb.AddMediaCancelRequest{
		TripId:    tripID,
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusReady, resp.GetStatus())
}

// compile-guard: sql.ErrNoRows импортируется для явной семантики race в Create,
// оставляем импорт явным (тест-файл ссылается на него).
var _ = sql.ErrNoRows
