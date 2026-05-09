package services

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// =============================================================================
// GetPin
// =============================================================================

func TestGetPin_HiddenForUser_ReturnsNotFound(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	hiddenRepo.EXPECT().IsHidden(pinID, userID).Return(true, nil)

	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, hiddenRepo, nil)
	_, err := svc.GetPin(ctxWithUser(userID), &pb.GetPinRequest{TripId: tripID, PinId: pinID})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetPin_NotParticipant_NotInFavourites_PermissionDenied(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)

	hiddenRepo.EXPECT().IsHidden(pinID, userID).Return(false, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(false, nil)
	favRepo.EXPECT().HasFavourite(userID, tripID).Return(false, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, hiddenRepo, nil)
	_, err := svc.GetPin(ctxWithUser(userID), &pb.GetPinRequest{TripId: tripID, PinId: pinID})
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetPin_PinFromAnotherTrip_NotFound(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	hiddenRepo.EXPECT().IsHidden(pinID, userID).Return(false, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: "trip-OTHER"}, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, hiddenRepo, nil)
	_, err := svc.GetPin(ctxWithUser(userID), &pb.GetPinRequest{TripId: tripID, PinId: pinID})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetPin_HappyPath(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)

	hiddenRepo.EXPECT().IsHidden(pinID, userID).Return(false, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{
		ID: pinID, TripID: tripID, Name: "Кафе", Category: "Еда и напитки", PrivacyLevel: "Private",
	}, nil)
	mediaRepo.EXPECT().ListByPinID(pinID).Return(nil, nil)
	tagRepo.EXPECT().GetByPinID(pinID).Return([]string{"food", "coffee"}, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, hiddenRepo, nil)
	resp, err := svc.GetPin(ctxWithUser(userID), &pb.GetPinRequest{TripId: tripID, PinId: pinID})
	require.NoError(t, err)
	require.Equal(t, pinID, resp.GetPin().GetId())
	require.Equal(t, "Кафе", resp.GetPin().GetName())
	require.Equal(t, []string{"food", "coffee"}, resp.GetPin().GetTags())
}

// =============================================================================
// UpdatePin
// =============================================================================

func TestUpdatePin_ValidationErrors(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"

	mkCtrl := func(t *testing.T) (*mocks.MockTripParticipantRepositoryInterface, *mocks.MockTripRepositoryInterface, *mocks.MockPinRepositoryInterface) {
		ctrl := gomock.NewController(t)
		participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
		tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
		pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
		participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil).AnyTimes()
		tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil).AnyTimes()
		pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil).AnyTimes()
		return participantRepo, tripRepo, pinRepo
	}

	mkSvc := func(participantRepo *mocks.MockTripParticipantRepositoryInterface, tripRepo *mocks.MockTripRepositoryInterface, pinRepo *mocks.MockPinRepositoryInterface) *TripService {
		return NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}

	t.Run("name_too_long", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		long := strings.Repeat("a", MaxNameLength+1)
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Name: &long})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("empty_name", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		empty := ""
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Name: &empty})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("description_too_long", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		long := strings.Repeat("a", MaxDescriptionLength+1)
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Description: &long})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("lat_only_no_lon", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		lat := 55.0
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Latitude: &lat})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("lat_out_of_range", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		lat := 91.0
		lon := 30.0
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Latitude: &lat, Longitude: &lon})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("end_before_start", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		start := int64(2000)
		end := int64(1000)
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, StartTimeUnix: &start, EndTimeUnix: &end})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("too_many_tags", func(t *testing.T) {
		participantRepo, tripRepo, pinRepo := mkCtrl(t)
		svc := mkSvc(participantRepo, tripRepo, pinRepo)
		tags := make([]string, MaxTagsPerPin+1)
		for i := range tags {
			tags[i] = "x"
		}
		_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID, Tags: tags, TagsSet: true})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestUpdatePin_NotParticipant_PermissionDenied(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(false, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestUpdatePin_TripNotReady_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpdatePin_HappyPath_TagsReplace(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID, Name: "Old"}, nil)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)
	tagRepo.EXPECT().SetForPin(tripID, pinID, []string{"a", "b"}).Return(nil)
	mediaRepo.EXPECT().ListByPinID(pinID).Return(nil, nil)
	tagRepo.EXPECT().GetByPinID(pinID).Return([]string{"a", "b"}, nil)

	newName := "Кафе"
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{
		TripId: tripID, PinId: pinID, Name: &newName, Tags: []string{"a", "b"}, TagsSet: true,
	})
	require.NoError(t, err)
	require.Equal(t, "Кафе", resp.GetPin().GetName())
	require.Equal(t, []string{"a", "b"}, resp.GetPin().GetTags())
}

func TestUpdatePin_TagsSetFalse_NotTouched(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)
	// SetForPin не должен вызываться при tags_set=false.
	mediaRepo.EXPECT().ListByPinID(pinID).Return(nil, nil)
	tagRepo.EXPECT().GetByPinID(pinID).Return([]string{"existing"}, nil)

	desc := "new"
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{
		TripId: tripID, PinId: pinID, Description: &desc,
		Tags: []string{"ignored"}, TagsSet: false,
	})
	require.NoError(t, err)
}

// =============================================================================
// DeletePin
// =============================================================================

func TestDeletePin_ActiveSession_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	pinUploadSessionRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	pinUploadSessionRepo.EXPECT().GetActiveAdditionForPin(gomock.Any(), pinID).Return(&models.PinUploadSession{SessionID: "sess-1"}, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, pinUploadSessionRepo)
	_, err := svc.DeletePin(ctxWithUser(userID), &pb.DeletePinRequest{TripId: tripID, PinId: pinID})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestDeletePin_FullDelete(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	pinUploadSessionRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	pinUploadSessionRepo.EXPECT().GetActiveAdditionForPin(gomock.Any(), pinID).Return(nil, repositories.ErrPinUploadSessionNotFound)
	favRepo.EXPECT().HasFavouritesByOtherUsers(tripID, userID).Return(false, nil)
	mediaRepo.EXPECT().DeleteByPinID(pinID).Return([]string{"k1", "k2"}, nil)
	tagRepo.EXPECT().DeleteForPin(pinID).Return(nil)
	pinRepo.EXPECT().Delete(pinID).Return(nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, pinUploadSessionRepo)
	resp, err := svc.DeletePin(ctxWithUser(userID), &pb.DeletePinRequest{TripId: tripID, PinId: pinID})
	require.NoError(t, err)
	require.Equal(t, "full", resp.GetDeletionMode())
}

func TestDeletePin_SoftDeleteForSelf(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	pinUploadSessionRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	pinUploadSessionRepo.EXPECT().GetActiveAdditionForPin(gomock.Any(), pinID).Return(nil, repositories.ErrPinUploadSessionNotFound)
	favRepo.EXPECT().HasFavouritesByOtherUsers(tripID, userID).Return(true, nil)
	hiddenRepo.EXPECT().HidePinForUser(pinID, userID).Return(nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, hiddenRepo, pinUploadSessionRepo)
	resp, err := svc.DeletePin(ctxWithUser(userID), &pb.DeletePinRequest{TripId: tripID, PinId: pinID})
	require.NoError(t, err)
	require.Equal(t, "soft_for_user", resp.GetDeletionMode())
}

// =============================================================================
// AddMediaToPin* — sessioned
// =============================================================================

func TestRemoveMediaFromPin_LastMedia_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const mediaID = "m1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	pinIDStr := pinID
	mediaRepo.EXPECT().GetByID(mediaID).Return(&models.Media{
		ID: mediaID, TripID: tripID, PinID: &pinIDStr, S3Key: "k1",
	}, nil)
	mediaRepo.EXPECT().ListByPinID(pinID).Return([]*models.Media{{ID: mediaID}}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RemoveMediaFromPin(ctxWithUser(userID), &pb.RemoveMediaFromPinRequest{
		TripId: tripID, PinId: pinID, MediaId: mediaID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRemoveMediaFromPin_HappyPath(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const mediaID = "m1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)

	pinIDStr := pinID
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	// pinRepo.GetByID вызывается 3 раза: assertParticipantAndPinReady, updatePinTimesAndLocation, финальный сбор.
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil).Times(3)
	mediaRepo.EXPECT().GetByID(mediaID).Return(&models.Media{
		ID: mediaID, TripID: tripID, PinID: &pinIDStr, S3Key: "k1",
	}, nil)
	// ListByPinID вызывается 3 раза: проверка count>=2, updatePinTimesAndLocation, финальный сбор.
	mediaRepo.EXPECT().ListByPinID(pinID).Return([]*models.Media{
		{ID: mediaID, PinID: &pinIDStr},
		{ID: "m2", PinID: &pinIDStr},
	}, nil).Times(3)
	mediaRepo.EXPECT().DeleteByIDs([]string{mediaID}).Return(nil)
	pinRepo.EXPECT().IncMediaCount(pinID, -1).Return(nil)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)
	tagRepo.EXPECT().GetByPinID(pinID).Return(nil, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.RemoveMediaFromPin(ctxWithUser(userID), &pb.RemoveMediaFromPinRequest{
		TripId: tripID, PinId: pinID, MediaId: mediaID,
	})
	require.NoError(t, err)
	require.Equal(t, pinID, resp.GetPin().GetId())
}

// =============================================================================
// pin_id из другого trip — для всех мутирующих ручек гарантируем NotFound
// =============================================================================

func TestUpdatePin_PinFromAnotherTrip_NotFound(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: "trip-OTHER"}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdatePin(ctxWithUser(userID), &pb.UpdatePinRequest{TripId: tripID, PinId: pinID})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// быстрая проверка что sql.ErrNoRows маппится в NotFound внутри GetByID-цепочек
func TestGetPin_PinNotInDB_NotFound(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-missing"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	hiddenRepo := mocks.NewMockPinHiddenRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	hiddenRepo.EXPECT().IsHidden(pinID, userID).Return(false, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(nil, sql.ErrNoRows)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, hiddenRepo, nil)
	_, err := svc.GetPin(ctxWithUser(userID), &pb.GetPinRequest{TripId: tripID, PinId: pinID})
	require.Equal(t, codes.NotFound, status.Code(err))
}
