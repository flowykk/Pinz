package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/testinfra"
	pb "pinz/backend/trip-service/pkg/proto"
)

// TestPinUpload_Integration_HappyPath покрывает оба сценария на реальном Postgres:
// creation (target_pin_id=nil) → finalize создаёт новый пин;
// addition (target_pin_id=новый pin) → finalize прибавляет медиа.
// ML-обработка инжектится inline через RunPinUploadProcessing вместо запуска полного worker-loop.
func TestPinUpload_Integration_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	participantRepo := repositories.NewTripParticipantRepository(sqlDB)
	inviteRepo := repositories.NewInvitationLinkRepository(sqlDB)
	settingsRepo := repositories.NewTripSettingsRepository(sqlDB)
	mediaRepo := repositories.NewMediaRepository(sqlDB)
	pinRepo := repositories.NewPinRepository(sqlDB)
	tagRepo := repositories.NewTagRepository(sqlDB)
	socialRepo := repositories.NewSocialRepository(sqlDB)
	favRepo := repositories.NewFavouriteRepository(sqlDB)
	pinHiddenRepo := repositories.NewPinHiddenRepository(sqlDB)
	pinUploadSessionRepo := repositories.NewPinUploadSessionRepository(sqlDB)

	svc := NewTripService(
		tripRepo, participantRepo, inviteRepo, settingsRepo, nil,
		mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo,
		nil, nil, nil, nil, nil, nil,
		pinHiddenRepo, pinUploadSessionRepo, nil,
	)

	ownerID := uuid.New().String()
	var tripID, sessionID, newPinID string

	t.Run("setup_ready_trip", func(t *testing.T) {
		var draftIDs []string
		var mediaByDraft [][]string
		err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
			resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
				Name: "PinUpload", Category: "vacation", Season: "summer",
				FilesToUpload: []*pb.FileToUpload{
					{ClientId: "f1", ContentType: "image/jpeg"},
				},
			})
			if err != nil {
				return err
			}
			tripID = resp.GetTripId()
			return nil
		})
		require.NoError(t, err)
		lat, lon := 55.0, 37.0
		err = callAsUser(t, ownerID, "/pinz.TripService/ProcessMediaGrouping", func(ctx context.Context) error {
			resp, err := svc.ProcessMediaGrouping(ctx, &pb.ProcessMediaGroupingRequest{
				TripId: tripID,
				Media: []*pb.MediaMeta{
					{S3Key: "trips/" + tripID + "/f1.jpg", MediaType: "image", CapturedAtUnix: 1714550400, Latitude: &lat, Longitude: &lon},
				},
			})
			if err != nil {
				return err
			}
			for _, dp := range resp.GetDraftPins() {
				draftIDs = append(draftIDs, dp.GetDraftPinId())
				ids := []string{}
				for _, m := range dp.GetMedia() {
					ids = append(ids, m.GetMediaId())
				}
				mediaByDraft = append(mediaByDraft, ids)
			}
			return nil
		})
		require.NoError(t, err)
		draftPins := make([]*pb.DraftPinInput, 0, len(draftIDs))
		for i, id := range draftIDs {
			draftPins = append(draftPins, &pb.DraftPinInput{DraftPinId: id, MediaIds: mediaByDraft[i]})
		}
		err = callAsUser(t, ownerID, "/pinz.TripService/ApplyGroupsAndProcess", func(ctx context.Context) error {
			_, err := svc.ApplyGroupsAndProcess(ctx, &pb.ApplyGroupsAndProcessRequest{TripId: tripID, DraftPins: draftPins})
			return err
		})
		require.NoError(t, err)
		err = callAsUser(t, ownerID, "/pinz.TripService/FinalizeTrip", func(ctx context.Context) error {
			_, err := svc.FinalizeTrip(ctx, &pb.FinalizeTripRequest{TripId: tripID})
			return err
		})
		require.NoError(t, err)
	})

	// =============================================================================
	// Сценарий A: creation (target_pin_id == nil)
	// =============================================================================

	t.Run("creation_start", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/PinUploadStart", func(ctx context.Context) error {
			resp, err := svc.PinUploadStart(ctx, &pb.PinUploadStartRequest{
				TripId: tripID,
				FilesToUpload: []*pb.FileToUpload{
					{ClientId: "p1", ContentType: "image/jpeg"},
				},
			})
			if err != nil {
				return err
			}
			require.NotEmpty(t, resp.GetSessionId())
			sessionID = resp.GetSessionId()
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("creation_duplicate_start_409", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/PinUploadStart", func(ctx context.Context) error {
			_, err := svc.PinUploadStart(ctx, &pb.PinUploadStartRequest{
				TripId: tripID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "x", ContentType: "image/jpeg"}},
			})
			require.Error(t, err)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("creation_commit", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CommitPinUpload", func(ctx context.Context) error {
			lat, lon := 56.0, 38.0
			captured := int64(1714650000)
			resp, err := svc.CommitPinUpload(ctx, &pb.CommitPinUploadRequest{
				TripId: tripID, SessionId: sessionID,
				S3Key: "trips/" + tripID + "/pins/p1.jpg", MediaType: "image",
				Latitude: &lat, Longitude: &lon, CapturedAtUnix: &captured,
			})
			if err != nil {
				return err
			}
			require.Equal(t, int32(1), resp.GetMediaCountInSession())
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("creation_process_async_then_review", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/ProcessPinUpload", func(ctx context.Context) error {
			resp, err := svc.ProcessPinUpload(ctx, &pb.ProcessPinUploadRequest{TripId: tripID, SessionId: sessionID})
			if err != nil {
				return err
			}
			require.Equal(t, "PROCESSING", resp.GetProcessingStatus())
			return nil
		})
		require.NoError(t, err)

		// Эмулируем worker.
		_, err = RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
			SessionRepo: pinUploadSessionRepo,
			MediaRepo:   mediaRepo,
		})
		require.NoError(t, err)

		err = callAsUser(t, ownerID, "/pinz.TripService/GetPinUploadReview", func(ctx context.Context) error {
			resp, err := svc.GetPinUploadReview(ctx, &pb.GetPinUploadReviewRequest{TripId: tripID, SessionId: sessionID})
			if err != nil {
				return err
			}
			require.Equal(t, "READY_FOR_REVIEW", resp.GetProcessingStatus())
			require.NotNil(t, resp.GetDraft())
			require.NotNil(t, resp.GetDraft().GetSuggested(), "creation: suggested обязан быть")
			require.NotNil(t, resp.GetDraft().GetSuggested().Latitude)
			require.Len(t, resp.GetDraft().GetMedia(), 1)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("creation_finalize", func(t *testing.T) {
		name := "PinFromUpload"
		err := callAsUser(t, ownerID, "/pinz.TripService/FinalizePinUpload", func(ctx context.Context) error {
			resp, err := svc.FinalizePinUpload(ctx, &pb.FinalizePinUploadRequest{
				TripId: tripID, SessionId: sessionID,
				Name: &name, Tags: []string{"food"}, TagsSet: true,
			})
			if err != nil {
				return err
			}
			require.NotEmpty(t, resp.GetPin().GetId())
			require.Equal(t, "PinFromUpload", resp.GetPin().GetName())
			newPinID = resp.GetPin().GetId()
			return nil
		})
		require.NoError(t, err)
	})

	// =============================================================================
	// Сценарий B: addition в только что созданный пин (target_pin_id != nil)
	// =============================================================================

	var sessionID2 string

	t.Run("addition_start", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/PinUploadStart", func(ctx context.Context) error {
			resp, err := svc.PinUploadStart(ctx, &pb.PinUploadStartRequest{
				TripId: tripID, TargetPinId: &newPinID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "a1", ContentType: "image/jpeg"}},
			})
			if err != nil {
				return err
			}
			require.NotEmpty(t, resp.GetSessionId())
			sessionID2 = resp.GetSessionId()
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("addition_commit_process_review_finalize", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CommitPinUpload", func(ctx context.Context) error {
			lat, lon := 56.5, 38.5
			captured := int64(1714660000)
			_, err := svc.CommitPinUpload(ctx, &pb.CommitPinUploadRequest{
				TripId: tripID, SessionId: sessionID2,
				S3Key: "trips/" + tripID + "/pins/a1.jpg", MediaType: "image",
				Latitude: &lat, Longitude: &lon, CapturedAtUnix: &captured,
			})
			return err
		})
		require.NoError(t, err)

		err = callAsUser(t, ownerID, "/pinz.TripService/ProcessPinUpload", func(ctx context.Context) error {
			resp, err := svc.ProcessPinUpload(ctx, &pb.ProcessPinUploadRequest{TripId: tripID, SessionId: sessionID2})
			if err != nil {
				return err
			}
			require.Equal(t, "PROCESSING", resp.GetProcessingStatus())
			return nil
		})
		require.NoError(t, err)

		_, err = RunPinUploadProcessing(context.Background(), sessionID2, PinUploadProcessorDeps{
			SessionRepo: pinUploadSessionRepo,
			MediaRepo:   mediaRepo,
		})
		require.NoError(t, err)

		err = callAsUser(t, ownerID, "/pinz.TripService/GetPinUploadReview", func(ctx context.Context) error {
			resp, err := svc.GetPinUploadReview(ctx, &pb.GetPinUploadReviewRequest{TripId: tripID, SessionId: sessionID2})
			if err != nil {
				return err
			}
			require.Equal(t, "READY_FOR_REVIEW", resp.GetProcessingStatus())
			require.NotNil(t, resp.GetDraft())
			require.Nil(t, resp.GetDraft().GetSuggested(), "addition: suggested обязан быть nil")
			return nil
		})
		require.NoError(t, err)

		err = callAsUser(t, ownerID, "/pinz.TripService/FinalizePinUpload", func(ctx context.Context) error {
			resp, err := svc.FinalizePinUpload(ctx, &pb.FinalizePinUploadRequest{
				TripId: tripID, SessionId: sessionID2,
			})
			if err != nil {
				return err
			}
			require.Equal(t, newPinID, resp.GetPin().GetId())
			require.Len(t, resp.GetPin().GetMedia(), 2, "после addition в пине должно быть 2 медиа")
			return nil
		})
		require.NoError(t, err)
	})

	// =============================================================================
	// Параллельные addition в разные пины — обе сессии успешны (partial UNIQUE).
	// =============================================================================

	t.Run("partial_unique_two_additions_in_different_pins_both_succeed", func(t *testing.T) {
		// Создаём ещё один пин напрямую через репозиторий — полный creation-flow здесь не нужен.
		another := &models.Pin{
			TripID:       tripID,
			Name:         "Second",
			Category:     "custom",
			PrivacyLevel: "private",
			MediaCount:   0,
		}
		require.NoError(t, pinRepo.Create(another))
		anotherPinID := another.ID

		// Две одновременные addition-сессии в разные пины.
		var sidA, sidB string
		err := callAsUser(t, ownerID, "/pinz.TripService/PinUploadStart", func(ctx context.Context) error {
			resp, err := svc.PinUploadStart(ctx, &pb.PinUploadStartRequest{
				TripId: tripID, TargetPinId: &newPinID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "addA", ContentType: "image/jpeg"}},
			})
			if err != nil {
				return err
			}
			sidA = resp.GetSessionId()
			return nil
		})
		require.NoError(t, err)
		err = callAsUser(t, ownerID, "/pinz.TripService/PinUploadStart", func(ctx context.Context) error {
			resp, err := svc.PinUploadStart(ctx, &pb.PinUploadStartRequest{
				TripId: tripID, TargetPinId: &anotherPinID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "addB", ContentType: "image/jpeg"}},
			})
			if err != nil {
				return err
			}
			sidB = resp.GetSessionId()
			return nil
		})
		require.NoError(t, err)
		require.NotEqual(t, sidA, sidB)
		// cancel обе.
		_ = callAsUser(t, ownerID, "/pinz.TripService/CancelPinUpload", func(ctx context.Context) error {
			_, err := svc.CancelPinUpload(ctx, &pb.CancelPinUploadRequest{TripId: tripID, SessionId: sidA})
			return err
		})
		_ = callAsUser(t, ownerID, "/pinz.TripService/CancelPinUpload", func(ctx context.Context) error {
			_, err := svc.CancelPinUpload(ctx, &pb.CancelPinUploadRequest{TripId: tripID, SessionId: sidB})
			return err
		})
	})
}
