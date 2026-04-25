package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	"pinz/backend/trip-service/internal/testinfra"
	pb "pinz/backend/trip-service/pkg/proto"
)

func callAsUser(t *testing.T, userID string, fullMethod string, fn func(ctx context.Context) error) error {
	t.Helper()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(server.MetadataUserIDKey, userID))
	_, err := server.AuthUnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: fullMethod}, func(ctx context.Context, req any) (any, error) {
		return nil, fn(ctx)
	})
	return err
}

func TestTripService_Integration(t *testing.T) {
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

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, nil, mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo, nil, nil, nil, nil)
	ownerID := uuid.New().String()
	user2ID := uuid.New().String()
	user3ID := uuid.New().String()

	env := &struct {
		tripID string
		tripID2 string
		inviteToken string
	}{}

	cases := map[string]func(t *testing.T){
		"CreateTrip": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
				resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
					Name: "Trip",
					Category: "Отпуск",
					Season: "Лето",
					PrivacyLevel: "Private",
					FilesToUpload: []*pb.FileToUpload{
						{ClientId: "c1", ContentType: "image/jpeg"},
					},
				})
				if err != nil {
					return err
				}
				require.Equal(t, "UPLOADING", resp.GetStatus())
				require.NotEmpty(t, resp.GetTripId())
				env.tripID = resp.GetTripId()
				return nil
			})
			require.NoError(t, err)
			isParticipant, err := participantRepo.IsParticipant(env.tripID, ownerID)
			require.NoError(t, err)
			require.True(t, isParticipant)
		},
		"ListUserTrips_returns_created_trip": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/ListUserTrips", func(ctx context.Context) error {
				resp, err := svc.ListUserTrips(ctx, &pb.ListUserTripsRequest{})
				if err != nil {
					return err
				}
				require.Len(t, resp.GetTrips(), 1)
				require.Equal(t, env.tripID, resp.GetTrips()[0].GetId())
				return nil
			})
			require.NoError(t, err)
		},
		"GetTrip_returns_trip": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/GetTrip", func(ctx context.Context) error {
				resp, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: env.tripID})
				if err != nil {
					return err
				}
				require.Equal(t, env.tripID, resp.GetTrip().GetId())
				require.Equal(t, "Trip", resp.GetTrip().GetName())
				return nil
			})
			require.NoError(t, err)
		},
		"GetTrip_non_participant_forbidden": func(t *testing.T) {
			err := callAsUser(t, user3ID, "/pinz.TripService/GetTrip", func(ctx context.Context) error {
				_, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: env.tripID})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"PublishTrip_not_ready_fails": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/PublishTrip", func(ctx context.Context) error {
				_, err := svc.PublishTrip(ctx, &pb.PublishTripRequest{TripId: env.tripID, PublishWhole: true})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.FailedPrecondition, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"LikeTrip_unpublished_fails": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/LikeTrip", func(ctx context.Context) error {
				_, err := svc.LikeTrip(ctx, &pb.LikeTripRequest{TripId: env.tripID})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.FailedPrecondition, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"GenerateInviteLink": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/GenerateInviteLink", func(ctx context.Context) error {
				resp, err := svc.GenerateInviteLink(ctx, &pb.GenerateInviteLinkRequest{TripId: env.tripID})
				if err != nil {
					return err
				}
				require.NotEmpty(t, resp.GetToken())
				env.inviteToken = resp.GetToken()
				return nil
			})
			require.NoError(t, err)
		},
		"GenerateInviteLink_non_participant_forbidden": func(t *testing.T) {
			err := callAsUser(t, user3ID, "/pinz.TripService/GenerateInviteLink", func(ctx context.Context) error {
				_, err := svc.GenerateInviteLink(ctx, &pb.GenerateInviteLinkRequest{TripId: env.tripID})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"JoinTripByToken_invalid_token_not_found": func(t *testing.T) {
			err := callAsUser(t, user2ID, "/pinz.TripService/JoinTripByToken", func(ctx context.Context) error {
				_, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: uuid.New().String()})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.NotFound, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"JoinTripByToken": func(t *testing.T) {
			err := callAsUser(t, user2ID, "/pinz.TripService/JoinTripByToken", func(ctx context.Context) error {
				resp, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: env.inviteToken})
				if err != nil {
					return err
				}
				require.False(t, resp.GetAlreadyJoined())
				require.Equal(t, env.tripID, resp.GetTripId())
				return nil
			})
			require.NoError(t, err)
		},
		"JoinTripByToken_already_joined": func(t *testing.T) {
			err := callAsUser(t, user2ID, "/pinz.TripService/JoinTripByToken", func(ctx context.Context) error {
				resp, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: env.inviteToken})
				if err != nil {
					return err
				}
				require.True(t, resp.GetAlreadyJoined())
				require.Equal(t, env.tripID, resp.GetTripId())
				return nil
			})
			require.NoError(t, err)
		},
		"DeleteTrip_non_admin_forbidden": func(t *testing.T) {
			err := callAsUser(t, user2ID, "/pinz.TripService/DeleteTrip", func(ctx context.Context) error {
				_, err := svc.DeleteTrip(ctx, &pb.DeleteTripRequest{TripId: env.tripID})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
		"TransferAdmin": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/TransferAdmin", func(ctx context.Context) error {
				resp, err := svc.TransferAdmin(ctx, &pb.TransferAdminRequest{TripId: env.tripID, NewAdminUserId: user2ID})
				if err != nil {
					return err
				}
				require.True(t, resp.GetSuccess())
				return nil
			})
			require.NoError(t, err)
		},
		"LeaveTrip_owner_leaves_trip_stays": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/LeaveTrip", func(ctx context.Context) error {
				resp, err := svc.LeaveTrip(ctx, &pb.LeaveTripRequest{TripId: env.tripID})
				if err != nil {
					return err
				}
				require.False(t, resp.GetTripDeleted())
				require.True(t, resp.GetSuccess())
				return nil
			})
			require.NoError(t, err)
		},
		"LeaveTrip_last_admin_leaves_trip_deleted": func(t *testing.T) {
			err := callAsUser(t, user2ID, "/pinz.TripService/LeaveTrip", func(ctx context.Context) error {
				resp, err := svc.LeaveTrip(ctx, &pb.LeaveTripRequest{TripId: env.tripID})
				if err != nil {
					return err
				}
				require.True(t, resp.GetTripDeleted())
				require.True(t, resp.GetSuccess())
				return nil
			})
			require.NoError(t, err)
		},
		"CreateTrip_second_for_delete": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
				resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
					Name: "TripToDelete",
					Category: "Отпуск",
					Season: "Лето",
					PrivacyLevel: "Private",
					FilesToUpload: []*pb.FileToUpload{
						{ClientId: "c2", ContentType: "image/jpeg"},
					},
				})
				if err != nil {
					return err
				}
				env.tripID2 = resp.GetTripId()
				return nil
			})
			require.NoError(t, err)
		},
		"DeleteTrip_admin_removes_trip": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/DeleteTrip", func(ctx context.Context) error {
				resp, err := svc.DeleteTrip(ctx, &pb.DeleteTripRequest{TripId: env.tripID2})
				if err != nil {
					return err
				}
				require.True(t, resp.GetSuccess())
				return nil
			})
			require.NoError(t, err)
		},
		"GetTrip_after_delete_returns_not_found": func(t *testing.T) {
			err := callAsUser(t, ownerID, "/pinz.TripService/GetTrip", func(ctx context.Context) error {
				_, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: env.tripID2})
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.NotFound, st.Code())
				return nil
			})
			require.NoError(t, err)
		},
	}

	order := []string{
		"CreateTrip", "ListUserTrips_returns_created_trip", "GetTrip_returns_trip", "GetTrip_non_participant_forbidden",
		"PublishTrip_not_ready_fails", "LikeTrip_unpublished_fails",
		"GenerateInviteLink", "GenerateInviteLink_non_participant_forbidden", "JoinTripByToken_invalid_token_not_found", "JoinTripByToken", "JoinTripByToken_already_joined",
		"DeleteTrip_non_admin_forbidden", "TransferAdmin",
		"LeaveTrip_owner_leaves_trip_stays", "LeaveTrip_last_admin_leaves_trip_deleted",
		"CreateTrip_second_for_delete", "DeleteTrip_admin_removes_trip", "GetTrip_after_delete_returns_not_found",
	}
	for _, name := range order {
		t.Run(name, cases[name])
	}
}

// TestTripService_Integration_CreationFlow проверяет полный флоу создания путешествия по ТЗ (п. 3.7–3.16):
// CreateTrip → ProcessMediaGrouping → ApplyGroupsAndProcess → GetTripReview → FinalizeTrip.
func TestTripService_Integration_CreationFlow(t *testing.T) {
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

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, nil, mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo, nil, nil, nil, nil)
	ownerID := uuid.New().String()

	var tripID string
	var draftPinIDs []string
	var mediaIDsByDraftPin [][]string

	t.Run("CreateTrip", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
			resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
				Name: "Алтай 2026",
				Category: "Отпуск",
				Season: "Лето",
				Description: "Поход с друзьями",
				PrivacyLevel: "Private",
				FilesToUpload: []*pb.FileToUpload{
					{ClientId: "file-1", ContentType: "image/jpeg"},
					{ClientId: "file-2", ContentType: "image/jpeg"},
				},
			})
			if err != nil {
				return err
			}
			require.Equal(t, "UPLOADING", resp.GetStatus())
			require.Len(t, resp.GetUploadUrls(), 2)
			tripID = resp.GetTripId()
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("ProcessMediaGrouping", func(t *testing.T) {
		require.NotEmpty(t, tripID)
		s3Key1 := "trips/" + tripID + "/file-1.jpg"
		s3Key2 := "trips/" + tripID + "/file-2.jpg"
		lat, lon := 50.123, 87.456
		err := callAsUser(t, ownerID, "/pinz.TripService/ProcessMediaGrouping", func(ctx context.Context) error {
			resp, err := svc.ProcessMediaGrouping(ctx, &pb.ProcessMediaGroupingRequest{
				TripId: tripID,
				Media: []*pb.MediaMeta{
					{S3Key: s3Key1, MediaType: "image", CapturedAtUnix: 1714550400, Latitude: &lat, Longitude: &lon},
					{S3Key: s3Key2, MediaType: "image", CapturedAtUnix: 1714550700, Latitude: &lat, Longitude: &lon},
				},
			})
			if err != nil {
				return err
			}
			require.Equal(t, "DRAFT_GROUPING_REVIEW", resp.GetStatus())
			require.NotEmpty(t, resp.GetDraftPins())
			for _, dp := range resp.GetDraftPins() {
				draftPinIDs = append(draftPinIDs, dp.GetDraftPinId())
				var ids []string
				for _, m := range dp.GetMedia() {
					ids = append(ids, m.GetMediaId())
				}
				mediaIDsByDraftPin = append(mediaIDsByDraftPin, ids)
			}
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("ApplyGroupsAndProcess", func(t *testing.T) {
		require.NotEmpty(t, tripID)
		require.NotEmpty(t, draftPinIDs)
		draftPins := make([]*pb.DraftPinInput, 0, len(draftPinIDs))
		for i, id := range draftPinIDs {
			draftPins = append(draftPins, &pb.DraftPinInput{
				DraftPinId: id,
				MediaIds: mediaIDsByDraftPin[i],
			})
		}
		err := callAsUser(t, ownerID, "/pinz.TripService/ApplyGroupsAndProcess", func(ctx context.Context) error {
			resp, err := svc.ApplyGroupsAndProcess(ctx, &pb.ApplyGroupsAndProcessRequest{
				TripId: tripID,
				DraftPins: draftPins,
			})
			if err != nil {
				return err
			}
			require.Equal(t, "PROCESSING", resp.GetStatus())
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("GetTripReview", func(t *testing.T) {
		require.NotEmpty(t, tripID)
		err := callAsUser(t, ownerID, "/pinz.TripService/GetTripReview", func(ctx context.Context) error {
			resp, err := svc.GetTripReview(ctx, &pb.GetTripReviewRequest{TripId: tripID})
			if err != nil {
				return err
			}
			require.Contains(t, []string{"PROCESSING", "DRAFT_FINAL_REVIEW"}, resp.GetStatus())
			require.NotEmpty(t, resp.GetPins())
			return nil
		})
		require.NoError(t, err)
	})

	// FinalizeTrip допускает статус PROCESSING (воркер в тестах не запущен).
	t.Run("FinalizeTrip", func(t *testing.T) {
		require.NotEmpty(t, tripID)
		err := callAsUser(t, ownerID, "/pinz.TripService/FinalizeTrip", func(ctx context.Context) error {
			resp, err := svc.FinalizeTrip(ctx, &pb.FinalizeTripRequest{TripId: tripID})
			if err != nil {
				return err
			}
			require.Equal(t, "READY", resp.GetStatus())
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("GetTrip_after_finalize_returns_READY", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/GetTrip", func(ctx context.Context) error {
			resp, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: tripID})
			if err != nil {
				return err
			}
			require.Equal(t, "READY", resp.GetTrip().GetStatus())
			return nil
		})
		require.NoError(t, err)
	})
}

// TestTripService_Integration_SoftDelete проверяет ТЗ 3.24: при удалении трипа админом,
// если трип в избранном у других пользователей — выполняется soft delete (SetSoftDeleted), не hard delete.
func TestTripService_Integration_SoftDelete(t *testing.T) {
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

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, nil, mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo, nil, nil, nil, nil)
	ownerID := uuid.New().String()
	user2ID := uuid.New().String()

	var tripID string
	var draftPinIDs []string
	var mediaIDsByDraftPin [][]string

	t.Run("CreateTrip", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/CreateTrip", func(ctx context.Context) error {
			resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
				Name: "SoftDeleteTrip",
				Category: "Отпуск",
				Season: "Лето",
				PrivacyLevel: "Public",
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
	})

	t.Run("ProcessMediaGrouping", func(t *testing.T) {
		require.NotEmpty(t, tripID)
		lat, lon := 50.0, 87.0
		err := callAsUser(t, ownerID, "/pinz.TripService/ProcessMediaGrouping", func(ctx context.Context) error {
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
				draftPinIDs = append(draftPinIDs, dp.GetDraftPinId())
				var ids []string
				for _, m := range dp.GetMedia() {
					ids = append(ids, m.GetMediaId())
				}
				mediaIDsByDraftPin = append(mediaIDsByDraftPin, ids)
			}
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("ApplyGroupsAndProcess", func(t *testing.T) {
		require.NotEmpty(t, draftPinIDs)
		draftPins := make([]*pb.DraftPinInput, 0, len(draftPinIDs))
		for i, id := range draftPinIDs {
			draftPins = append(draftPins, &pb.DraftPinInput{DraftPinId: id, MediaIds: mediaIDsByDraftPin[i]})
		}
		err := callAsUser(t, ownerID, "/pinz.TripService/ApplyGroupsAndProcess", func(ctx context.Context) error {
			_, err := svc.ApplyGroupsAndProcess(ctx, &pb.ApplyGroupsAndProcessRequest{TripId: tripID, DraftPins: draftPins})
			return err
		})
		require.NoError(t, err)
	})

	t.Run("FinalizeTrip", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/FinalizeTrip", func(ctx context.Context) error {
			_, err := svc.FinalizeTrip(ctx, &pb.FinalizeTripRequest{TripId: tripID})
			return err
		})
		require.NoError(t, err)
	})

	t.Run("PublishTrip", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/PublishTrip", func(ctx context.Context) error {
			_, err := svc.PublishTrip(ctx, &pb.PublishTripRequest{TripId: tripID, PublishWhole: true})
			return err
		})
		require.NoError(t, err)
	})

	t.Run("User2_adds_to_favourites", func(t *testing.T) {
		err := callAsUser(t, user2ID, "/pinz.TripService/AddToFavourites", func(ctx context.Context) error {
			_, err := svc.AddToFavourites(ctx, &pb.AddToFavouritesRequest{TripId: tripID})
			return err
		})
		require.NoError(t, err)
	})

	t.Run("Owner_deletes_soft_delete_path", func(t *testing.T) {
		err := callAsUser(t, ownerID, "/pinz.TripService/DeleteTrip", func(ctx context.Context) error {
			resp, err := svc.DeleteTrip(ctx, &pb.DeleteTripRequest{TripId: tripID})
			if err != nil {
				return err
			}
			require.True(t, resp.GetSuccess())
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("Trip_is_soft_deleted_not_hard_deleted", func(t *testing.T) {
		trip, err := tripRepo.GetByID(tripID)
		require.NoError(t, err)
		require.True(t, trip.IsSoftDeleted, "trip should be soft-deleted when in others' favourites")
	})
}
