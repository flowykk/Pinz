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

// TestPinCRUD_Integration_HappyPath покрывает GetPin → UpdatePin → DeletePin (full)
// поверх готового READY-трипа. Add-media-сессия и soft-delete покрыты unit-тестами;
// здесь — реальный Postgres путь для проверки SQL и каскадов.
func TestPinCRUD_Integration_HappyPath(t *testing.T) {
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
	var tripID, pinID string

	// 1. Через creation pipeline получаем READY-трип с одним пином и двумя медиа.
	t.Run("setup_create_trip_to_READY", func(t *testing.T) {
		var draftIDs []string
		var mediaByDraft [][]string
		err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
			resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
				Name: "Pin CRUD",
				Category: "Отпуск",
				Season: "Лето",
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
		// получаем pin_id из БД
		pins, err := pinRepo.ListByTripID(tripID)
		require.NoError(t, err)
		require.NotEmpty(t, pins)
		pinID = pins[0].ID
	})

	t.Run("GetPin_returns_full", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/GetPin", func(ctx context.Context) error {
			resp, err := svc.GetPin(ctx, &pb.GetPinRequest{TripId: tripID, PinId: pinID})
			if err != nil {
				return err
			}
			require.Equal(t, pinID, resp.GetPin().GetId())
			require.Len(t, resp.GetPin().GetMedia(), 2)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("UpdatePin_replaces_tags_and_name", func(t *testing.T) {
		newName := "Кафе"
		newDesc := "Уютное место"
		err := callAsUser(t, ownerID, "/pinz.TripService/UpdatePin", func(ctx context.Context) error {
			resp, err := svc.UpdatePin(ctx, &pb.UpdatePinRequest{
				TripId: tripID, PinId: pinID,
				Name: &newName, Description: &newDesc,
				Tags: []string{"food", "coffee"}, TagsSet: true,
			})
			if err != nil {
				return err
			}
			require.Equal(t, "Кафе", resp.GetPin().GetName())
			require.ElementsMatch(t, []string{"food", "coffee"}, resp.GetPin().GetTags())
			return nil
		})
		require.NoError(t, err)
		// Replace-all: ещё один UpdatePin со списком из 1 тега должен заменить.
		err = callAsUser(t, ownerID, "/pinz.TripService/UpdatePin", func(ctx context.Context) error {
			_, err := svc.UpdatePin(ctx, &pb.UpdatePinRequest{
				TripId: tripID, PinId: pinID,
				Tags: []string{"only"}, TagsSet: true,
			})
			return err
		})
		require.NoError(t, err)
		got, err := tagRepo.GetByPinID(pinID)
		require.NoError(t, err)
		require.Equal(t, []string{"only"}, got)
	})

	t.Run("DeletePin_full_when_no_others_in_favourites", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/DeletePin", func(ctx context.Context) error {
			resp, err := svc.DeletePin(ctx, &pb.DeletePinRequest{TripId: tripID, PinId: pinID})
			if err != nil {
				return err
			}
			require.Equal(t, "full", resp.GetDeletionMode())
			return nil
		})
		require.NoError(t, err)
		// pin удалён из БД
		_, err = pinRepo.GetByID(pinID)
		require.Error(t, err, "pin must be removed from DB")
		// media пина удалены каскадом сервиса
		mediaList, err := mediaRepo.ListByPinID(pinID)
		require.NoError(t, err)
		require.Empty(t, mediaList)
	})
}
