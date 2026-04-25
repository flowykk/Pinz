package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/testinfra"
	pb "pinz/backend/trip-service/pkg/proto"
)

// TestPinCreate_Integration_HappyPath покрывает полный sessioned флоу создания
// одиночного пина в READY-трипе (ТЗ 4.1, 4.6-4.11):
// CreatePinStart → CommitPinCreationUpload×2 → ProcessPinCreation →
// FinalizePinCreation. Проверяет, что pin создан, media привязаны, агрегаты
// (start/end/lat/lon/media_count) посчитаны.
func TestPinCreate_Integration_HappyPath(t *testing.T) {
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
	pinAddSessionRepo := repositories.NewPinMediaAdditionSessionRepository(sqlDB)
	pinCreationSessionRepo := repositories.NewPinCreationSessionRepository(sqlDB)

	svc := NewTripService(
		tripRepo, participantRepo, inviteRepo, settingsRepo, nil,
		mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo,
		nil, nil, nil, nil, nil, nil, nil,
		pinHiddenRepo, pinAddSessionRepo, pinCreationSessionRepo,
	)

	ownerID := uuid.New().String()
	var tripID, sessionID, pinID string
	var createdMediaIDs []string

	// 1. Создать READY-трип через creation pipeline (1 пин с 2 медиа — нужен
	// для покрытия лимита, не пересекается с pin_creation_sessions).
	t.Run("setup_create_trip_to_READY", func(t *testing.T) {
		var draftIDs []string
		var mediaByDraft [][]string
		err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
			resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
				Name: "PinCreate", Category: "Отпуск", Season: "Лето",
				FilesToUpload: []*pb.FileToUpload{
					{ClientId: "f1", ContentType: "image/jpeg"},
					{ClientId: "f2", ContentType: "image/jpeg"},
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
					{S3Key: "trips/" + tripID + "/f2.jpg", MediaType: "image", CapturedAtUnix: 1714550700, Latitude: &lat, Longitude: &lon},
				},
			})
			if err != nil {
				return err
			}
			for _, dp := range resp.GetDraftPins() {
				draftIDs = append(draftIDs, dp.GetDraftPinId())
				var ids []string
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

	// 2. CreatePinStart.
	t.Run("CreatePinStart", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CreatePinStart", func(ctx context.Context) error {
			resp, err := svc.CreatePinStart(ctx, &pb.CreatePinStartRequest{
				TripId: tripID,
				FilesToUpload: []*pb.FileToUpload{
					{ClientId: "p1", ContentType: "image/jpeg"},
					{ClientId: "p2", ContentType: "image/jpeg"},
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

	// 3. CreatePinStart дважды → второй вызов получает FailedPrecondition (UNIQUE).
	t.Run("CreatePinStart_conflict", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CreatePinStart", func(ctx context.Context) error {
			_, err := svc.CreatePinStart(ctx, &pb.CreatePinStartRequest{
				TripId: tripID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "x", ContentType: "image/jpeg"}},
			})
			require.Error(t, err)
			return nil
		})
		require.NoError(t, err)
	})

	// 4. Commit двух медиа.
	t.Run("CommitPinCreationUpload_x2", func(t *testing.T) {
		lat, lon := 56.0, 38.0
		captured1 := int64(1714560000)
		captured2 := int64(1714560300)
		err := callAsUser(t, ownerID, "/pinz.TripService/CommitPinCreationUpload", func(ctx context.Context) error {
			resp, err := svc.CommitPinCreationUpload(ctx, &pb.CommitPinCreationUploadRequest{
				TripId: tripID, SessionId: sessionID,
				S3Key: "trips/" + tripID + "/pins/p1.jpg", MediaType: "image",
				CapturedAtUnix: &captured1, Latitude: &lat, Longitude: &lon,
			})
			if err != nil {
				return err
			}
			require.Equal(t, int32(1), resp.GetMediaCountInSession())
			createdMediaIDs = append(createdMediaIDs, resp.GetMediaId())
			return nil
		})
		require.NoError(t, err)
		err = callAsUser(t, ownerID, "/pinz.TripService/CommitPinCreationUpload", func(ctx context.Context) error {
			resp, err := svc.CommitPinCreationUpload(ctx, &pb.CommitPinCreationUploadRequest{
				TripId: tripID, SessionId: sessionID,
				S3Key: "trips/" + tripID + "/pins/p2.jpg", MediaType: "image",
				CapturedAtUnix: &captured2,
			})
			if err != nil {
				return err
			}
			require.Equal(t, int32(2), resp.GetMediaCountInSession())
			createdMediaIDs = append(createdMediaIDs, resp.GetMediaId())
			return nil
		})
		require.NoError(t, err)
	})

	// 5. ProcessPinCreation: snapshot заполнен, suggested поля установлены.
	t.Run("ProcessPinCreation", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/ProcessPinCreation", func(ctx context.Context) error {
			resp, err := svc.ProcessPinCreation(ctx, &pb.ProcessPinCreationRequest{
				TripId: tripID, SessionId: sessionID,
			})
			if err != nil {
				return err
			}
			require.NotNil(t, resp.GetDraft())
			require.Equal(t, PinCategoryDefault, resp.GetDraft().GetSuggestedCategory())
			require.NotNil(t, resp.GetDraft().SuggestedLatitude)
			require.NotNil(t, resp.GetDraft().SuggestedStartTimeUnix)
			require.NotNil(t, resp.GetDraft().SuggestedEndTimeUnix)
			// Оба captured_at известны → MISSING_DATES не должно быть.
			require.NotContains(t, resp.GetDraft().GetPinIssues(), pinIssueMissingDates)
			require.Len(t, resp.GetDraft().GetMedia(), 2)
			return nil
		})
		require.NoError(t, err)
	})

	// 6. FinalizePinCreation с пользовательскими правками.
	t.Run("FinalizePinCreation_with_user_edits", func(t *testing.T) {
		name := "Любимое кафе"
		category := "Еда и напитки"
		err := callAsUser(t, ownerID, "/pinz.TripService/FinalizePinCreation", func(ctx context.Context) error {
			resp, err := svc.FinalizePinCreation(ctx, &pb.FinalizePinCreationRequest{
				TripId: tripID, SessionId: sessionID,
				Name: &name, Category: &category,
				Tags: []string{"кофе", "ужин"}, TagsSet: true,
			})
			if err != nil {
				return err
			}
			require.NotEmpty(t, resp.GetPin().GetId())
			require.Equal(t, "Любимое кафе", resp.GetPin().GetName())
			require.Equal(t, "Еда и напитки", resp.GetPin().GetCategory())
			require.Equal(t, []string{"кофе", "ужин"}, resp.GetPin().GetTags())
			require.Len(t, resp.GetPin().GetMedia(), 2)
			pinID = resp.GetPin().GetId()
			return nil
		})
		require.NoError(t, err)
	})

	// 7. Проверка БД: pin создан, media привязаны, агрегаты согласованы.
	t.Run("DB_state_after_finalize", func(t *testing.T) {
		pin, err := pinRepo.GetByID(pinID)
		require.NoError(t, err)
		require.Equal(t, "Любимое кафе", pin.Name)
		require.Equal(t, "Еда и напитки", pin.Category)
		require.Equal(t, int32(2), pin.MediaCount)
		// updatePinTimesAndLocation не вызывался (req не задавал координаты/start/end явно
		// → NO такие поля в req → НЕ вызывался — но запросные значения брались из snapshot
		// → они и попали в Create. Лат/start/end должны быть заполнены).
		require.NotNil(t, pin.Latitude)
		require.NotNil(t, pin.StartTime)
		require.NotNil(t, pin.EndTime)
		// Media привязаны к pin_id.
		mediaList, err := mediaRepo.ListByPinID(pinID)
		require.NoError(t, err)
		require.Len(t, mediaList, 2)
	})

	// 8. Повторный CreatePinStart после finalize должен работать (сессия закрыта).
	t.Run("CreatePinStart_after_finalize_works", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CreatePinStart", func(ctx context.Context) error {
			resp, err := svc.CreatePinStart(ctx, &pb.CreatePinStartRequest{
				TripId: tripID,
				FilesToUpload: []*pb.FileToUpload{{ClientId: "z", ContentType: "image/jpeg"}},
			})
			if err != nil {
				return err
			}
			require.NotEmpty(t, resp.GetSessionId())
			sessionID = resp.GetSessionId()
			return nil
		})
		require.NoError(t, err)
		// Сразу cancel — orphan media пустой, сессия закрывается.
		err = callAsUser(t, ownerID, "/pinz.TripService/CancelPinCreation", func(ctx context.Context) error {
			_, err := svc.CancelPinCreation(ctx, &pb.CancelPinCreationRequest{
				TripId: tripID, SessionId: sessionID,
			})
			return err
		})
		require.NoError(t, err)
	})
}
