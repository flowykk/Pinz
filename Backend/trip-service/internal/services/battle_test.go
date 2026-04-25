package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	pb "pinz/backend/trip-service/pkg/proto"
)

// buildBattleMedia собирает 8 media с заданным trip_id для теста StartBattle.
func buildBattleMedia(tripID string) []*models.Media {
	out := make([]*models.Media, battleSize)
	for i := 0; i < battleSize; i++ {
		id := "m-" + string(rune('0'+i))
		out[i] = &models.Media{ID: id, TripID: tripID, S3Key: "k/" + id, MediaType: "photo"}
	}
	return out
}

func TestStartBattle_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)

	picked := buildBattleMedia("trip-1")
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("trip-1").Return(10, 0, nil)
	mediaRepo.EXPECT().PickRandomForBattle("trip-1", battleSize).Return(picked, nil)
	battleRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(b *models.MediaBattle) error {
		b.ID = "battle-1"
		require.Equal(t, "trip-1", b.TripID)
		require.Equal(t, "u1", b.UserID)
		require.Len(t, b.MediaIDs, battleSize)
		return nil
	})

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
	resp, err := svc.StartBattle(ctxWithUser("u1"), &pb.StartBattleRequest{TripId: "trip-1"})
	require.NoError(t, err)
	require.Equal(t, "battle-1", resp.GetBattleId())
	require.Len(t, resp.GetMedia(), battleSize)
	require.Equal(t, "m-0", resp.GetMedia()[0].GetMediaId())
}

func TestStartBattle_ValidationAndAccess(t *testing.T) {
	t.Run("missing_user", func(t *testing.T) {
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		_, err := svc.StartBattle(t.Context(), &pb.StartBattleRequest{TripId: "trip-1"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.Unauthenticated, st.Code())
	})
	t.Run("empty_trip_id", func(t *testing.T) {
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		_, err := svc.StartBattle(ctxWithUser("u1"), &pb.StartBattleRequest{TripId: ""})
		st, _ := status.FromError(err)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})
	t.Run("not_participant", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
		participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(false, nil)
		svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		_, err := svc.StartBattle(ctxWithUser("u1"), &pb.StartBattleRequest{TripId: "trip-1"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})
}

func TestStartBattle_NotEnoughMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("trip-1").Return(7, 0, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.StartBattle(ctxWithUser("u1"), &pb.StartBattleRequest{TripId: "trip-1"})
	st, _ := status.FromError(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestStartBattle_NotEnoughAvailableAfterRestrictedFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("trip-1").Return(10, 0, nil)
	// Половина Restricted — PickRandomForBattle вернёт меньше 8.
	mediaRepo.EXPECT().PickRandomForBattle("trip-1", battleSize).Return(buildBattleMedia("trip-1")[:5], nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.StartBattle(ctxWithUser("u1"), &pb.StartBattleRequest{TripId: "trip-1"})
	st, _ := status.FromError(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestSubmitBattleResult_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)

	battleRepo.EXPECT().GetByID("battle-1").Return(&models.MediaBattle{
		ID: "battle-1",
		TripID: "trip-1",
		UserID: "u1",
		MediaIDs: []string{"m-0", "m-1", "m-2", "m-3", "m-4", "m-5", "m-6", "m-7"},
	}, nil)
	battleRepo.EXPECT().SetWinner("battle-1", "m-3").Return(nil)
	mediaRepo.EXPECT().IncrementBattleRating("m-3").Return(int32(1), nil)

	svc := NewTripService(nil, nil, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
	resp, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{
		BattleId: "battle-1",
		WinnerMediaId: "m-3",
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.GetNewBattleRating())
}

func TestSubmitBattleResult_Errors(t *testing.T) {
	t.Run("unknown_battle", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)
		battleRepo.EXPECT().GetByID("battle-x").Return(nil, sql.ErrNoRows)
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{BattleId: "battle-x", WinnerMediaId: "m-0"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.NotFound, st.Code())
	})
	t.Run("wrong_owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)
		battleRepo.EXPECT().GetByID("battle-1").Return(&models.MediaBattle{
			ID: "battle-1", UserID: "other", MediaIDs: []string{"m-0"},
		}, nil)
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{BattleId: "battle-1", WinnerMediaId: "m-0"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})
	t.Run("already_finished", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)
		finished := time.Now()
		battleRepo.EXPECT().GetByID("battle-1").Return(&models.MediaBattle{
			ID: "battle-1", UserID: "u1", MediaIDs: []string{"m-0"}, FinishedAt: &finished,
		}, nil)
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{BattleId: "battle-1", WinnerMediaId: "m-0"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.FailedPrecondition, st.Code())
	})
	t.Run("winner_not_in_battle", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)
		battleRepo.EXPECT().GetByID("battle-1").Return(&models.MediaBattle{
			ID: "battle-1", UserID: "u1", MediaIDs: []string{"m-0", "m-1"},
		}, nil)
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{BattleId: "battle-1", WinnerMediaId: "intruder"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})
	t.Run("set_winner_race_lost", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		battleRepo := mocks.NewMockMediaBattleRepositoryInterface(ctrl)
		battleRepo.EXPECT().GetByID("battle-1").Return(&models.MediaBattle{
			ID: "battle-1", UserID: "u1", MediaIDs: []string{"m-0"},
		}, nil)
		// Параллельный запрос уже закрыл батл — SetWinner возвращает ErrNoRows; инкремент не выполняется.
		battleRepo.EXPECT().SetWinner("battle-1", "m-0").Return(sql.ErrNoRows)
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, battleRepo)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{BattleId: "battle-1", WinnerMediaId: "m-0"})
		st, _ := status.FromError(err)
		require.Equal(t, codes.FailedPrecondition, st.Code())
	})
	t.Run("missing_fields", func(t *testing.T) {
		svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		_, err := svc.SubmitBattleResult(ctxWithUser("u1"), &pb.SubmitBattleResultRequest{})
		st, _ := status.FromError(err)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGetBestMemories_OnlyPositiveRating(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	captured := time.Unix(1700000000, 0)
	mediaRepo.EXPECT().ListWithPositiveBattleRating("trip-1").Return([]*models.Media{
		{ID: "m-1", TripID: "trip-1", S3Key: "k1", MediaType: "photo", BattleRating: 3, CapturedAt: &captured},
		{ID: "m-2", TripID: "trip-1", S3Key: "k2", MediaType: "video", BattleRating: 1},
	}, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.GetBestMemories(ctxWithUser("u1"), &pb.GetBestMemoriesRequest{TripId: "trip-1"})
	require.NoError(t, err)
	require.Len(t, resp.GetMedia(), 2)
	require.Equal(t, int32(3), resp.GetMedia()[0].GetBattleRating())
	require.Equal(t, captured.Unix(), resp.GetMedia()[0].GetCapturedAtUnix())
	require.Equal(t, int64(0), resp.GetMedia()[1].GetCapturedAtUnix())
}

func TestGetBestMemories_EmptyOk(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	mediaRepo.EXPECT().ListWithPositiveBattleRating("trip-1").Return(nil, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.GetBestMemories(ctxWithUser("u1"), &pb.GetBestMemoriesRequest{TripId: "trip-1"})
	require.NoError(t, err)
	require.Empty(t, resp.GetMedia())
}
