package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// helper для краткого создания кандидатов в тестах.
func makeCand(id, category string, cluster int32, mediaCount int32, descLen int, score int32) *repositories.RecommendationPinCandidate {
	desc := ""
	for i := 0; i < descLen; i++ {
		desc += "x"
	}
	return &repositories.RecommendationPinCandidate{
		ID: id,
		TripID: "trip-" + id,
		Name: id,
		Description: desc,
		Category: category,
		MediaCount: mediaCount,
		ClusterID: cluster,
		TripScore: score,
	}
}

func ids(picks []*repositories.RecommendationPinCandidate) []string {
	out := make([]string, len(picks))
	for i, p := range picks {
		out[i] = p.ID
	}
	return out
}

func TestPickRecommendationPins_PicksTopFromCluster_ByMediaCount(t *testing.T) {
	cands := []*repositories.RecommendationPinCandidate{
		makeCand("a", "Достопримечательность", 1, 2, 5, 10),
		makeCand("b", "Достопримечательность", 1, 5, 5, 10), // best media_count → выбран
		makeCand("c", "Достопримечательность", 1, 1, 50, 10),
	}
	got := pickRecommendationPins(cands)
	require.Equal(t, []string{"b"}, ids(got))
}

func TestPickRecommendationPins_TieBreakByLongerDescription(t *testing.T) {
	cands := []*repositories.RecommendationPinCandidate{
		makeCand("a", "Природа", 1, 3, 10, 5),
		makeCand("b", "Природа", 1, 3, 80, 5), // тот же media_count, длиннее description
	}
	got := pickRecommendationPins(cands)
	require.Equal(t, []string{"b"}, ids(got))
}

func TestPickRecommendationPins_PartitionsByCategoryAndCluster(t *testing.T) {
	// две категории, у каждой свой cluster_id=1 (PARTITION BY category) — должны быть оба пина.
	cands := []*repositories.RecommendationPinCandidate{
		makeCand("sight", "Достопримечательность", 1, 1, 1, 10),
		makeCand("food", "Еда и напитки", 1, 1, 1, 10),
	}
	got := pickRecommendationPins(cands)
	require.ElementsMatch(t, []string{"sight", "food"}, ids(got))
}

func TestPickRecommendationPins_FillsQuotasFirst(t *testing.T) {
	// Меньше 6 sightseeing и 3 food — берём всё, что есть, без падения.
	cands := []*repositories.RecommendationPinCandidate{
		makeCand("s1", "Достопримечательность", 1, 1, 1, 10),
		makeCand("s2", "Природа", 2, 1, 1, 9),
		makeCand("f1", "Еда и напитки", 1, 1, 1, 8),
		makeCand("o1", "Шопинг", 1, 1, 1, 7),
	}
	got := pickRecommendationPins(cands)
	require.ElementsMatch(t, []string{"s1", "s2", "f1", "o1"}, ids(got))
}

func TestPickRecommendationPins_AddsExtraUpToCap(t *testing.T) {
	var cands []*repositories.RecommendationPinCandidate
	// 8 sightseeing, 5 food, 20 other — итог должен быть = recommendationMaxPins (25),
	// и стартовать с минимумов.
	for i := 0; i < 8; i++ {
		cands = append(cands, makeCand(uniqueID("s", i), "Природа", int32(i), int32(10-i), 5, int32(100-i)))
	}
	for i := 0; i < 5; i++ {
		cands = append(cands, makeCand(uniqueID("f", i), "Еда и напитки", int32(i), int32(10-i), 5, int32(50-i)))
	}
	for i := 0; i < 20; i++ {
		cands = append(cands, makeCand(uniqueID("o", i), "Шопинг", int32(i), int32(10-i), 5, int32(30-i)))
	}
	got := pickRecommendationPins(cands)
	require.Len(t, got, recommendationMaxPins)
	// первые 6 — sightseeing, следующие 3 — food (квоты), остальное — добор.
	for i := 0; i < recommendationMinSightseeing; i++ {
		require.True(t, recommendationSightseeingCategories[got[i].Category],
			"position %d expected sightseeing, got %s", i, got[i].Category)
	}
	for i := recommendationMinSightseeing; i < recommendationMinSightseeing+recommendationMinFood; i++ {
		require.Equal(t, recommendationFoodCategory, got[i].Category)
	}
}

func uniqueID(prefix string, i int) string {
	return prefix + string(rune('a'+i))
}

func makeServiceForSaveRecommendation(
	t *testing.T,
	ctrl *gomock.Controller,
) (
	*TripService,
	*mocks.MockTripRepositoryInterface,
	*mocks.MockTripParticipantRepositoryInterface,
	*mocks.MockPinRepositoryInterface,
	*mocks.MockFavouriteRepositoryInterface,
	*mocks.MockGeoRegistryRepositoryInterface,
	*mocks.MockRecommendationSnapshotRepositoryInterface,
) {
	t.Helper()
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	geoRepo := mocks.NewMockGeoRegistryRepositoryInterface(ctrl)
	snapRepo := mocks.NewMockRecommendationSnapshotRepositoryInterface(ctrl)
	svc := NewTripService(
		tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil,
		favRepo, geoRepo, nil, nil, nil, nil, nil, nil, nil, snapRepo,
	)
	return svc, tripRepo, participantRepo, pinRepo, favRepo, geoRepo, snapRepo
}

func expectMaterialize(
	tripRepo *mocks.MockTripRepositoryInterface,
	participantRepo *mocks.MockTripParticipantRepositoryInterface,
	pinRepo *mocks.MockPinRepositoryInterface,
	favRepo *mocks.MockFavouriteRepositoryInterface,
	geoRepo *mocks.MockGeoRegistryRepositoryInterface,
	regionID int,
	pins []*models.Pin,
	tripID string,
) {
	tripRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(t *models.Trip) error {
		t.ID = tripID
		return nil
	})
	participantRepo.EXPECT().Add(gomock.Any()).Return(nil)
	pinRepo.EXPECT().GetByIDs(gomock.Any()).Return(pins, nil)
	pinRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(len(pins))
	geoRepo.EXPECT().UpsertTripLocations(gomock.Any(), tripID, []int{regionID}).Return(nil)
	favRepo.EXPECT().Add("user-1", tripID).Return(nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, OwnerUserID: "user-1", Name: "Популярные места: Москва", IsGenerated: true}, nil)
}

func TestSaveRecommendation_TokenFastPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, tripRepo, participantRepo, pinRepo, favRepo, geoRepo, snapRepo := makeServiceForSaveRecommendation(t, ctrl)
	const token = "tok-1"
	snap := &repositories.RecommendationSnapshot{
		UserID:     "user-1",
		RegionID:   42,
		RegionName: "Москва",
		PinIDs:     []string{"p1"},
	}
	snapRepo.EXPECT().Get(gomock.Any(), token).Return(snap, true, nil)

	lat, lon := 55.75, 37.62
	expectMaterialize(tripRepo, participantRepo, pinRepo, favRepo, geoRepo, 42, []*models.Pin{
		{ID: "p1", TripID: "tsrc", Name: "Кремль", Category: "Достопримечательность", Latitude: &lat, Longitude: &lon, IsPublishedInFeed: true},
	}, "new-trip-1")
	snapRepo.EXPECT().Delete(gomock.Any(), token).Return(nil)

	resp, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{SnapshotToken: token})
	require.NoError(t, err)
	require.Equal(t, "new-trip-1", resp.GetTrip().GetId())
}

func TestSaveRecommendation_TokenForeignUser_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _, _, _, _, snapRepo := makeServiceForSaveRecommendation(t, ctrl)
	snap := &repositories.RecommendationSnapshot{UserID: "other"}
	snapRepo.EXPECT().Get(gomock.Any(), "tok").Return(snap, true, nil)

	_, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{SnapshotToken: "tok"})
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestSaveRecommendation_FallbackByPinIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, tripRepo, participantRepo, pinRepo, favRepo, geoRepo, snapRepo := makeServiceForSaveRecommendation(t, ctrl)
	geoRepo.EXPECT().FindCityByName(gomock.Any(), "Москва").Return(42, nil)
	geoRepo.EXPECT().TripIDsAtLocation(gomock.Any(), 42, gomock.Any()).Return(map[string]struct{}{"tsrc": {}}, nil)
	_ = snapRepo

	lat, lon := 55.75, 37.62
	expectMaterialize(tripRepo, participantRepo, pinRepo, favRepo, geoRepo, 42, []*models.Pin{
		{ID: "p1", TripID: "tsrc", Name: "Кремль", Category: "Достопримечательность", Latitude: &lat, Longitude: &lon, IsPublishedInFeed: true},
	}, "new-trip-2")

	resp, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{
		City:   "Москва",
		PinIds: []string{"p1"},
	})
	require.NoError(t, err)
	require.Equal(t, "new-trip-2", resp.GetTrip().GetId())
}

func TestSaveRecommendation_FallbackForeignRegion_InvalidArgument(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _, pinRepo, _, geoRepo, _ := makeServiceForSaveRecommendation(t, ctrl)
	geoRepo.EXPECT().FindCityByName(gomock.Any(), "Москва").Return(42, nil)
	lat, lon := 55.75, 37.62
	pinRepo.EXPECT().GetByIDs([]string{"p1"}).Return([]*models.Pin{
		{ID: "p1", TripID: "tsrc", Name: "Кремль", Latitude: &lat, Longitude: &lon, IsPublishedInFeed: true},
	}, nil)
	geoRepo.EXPECT().TripIDsAtLocation(gomock.Any(), 42, gomock.Any()).Return(map[string]struct{}{}, nil)

	_, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{
		City:   "Москва",
		PinIds: []string{"p1"},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSaveRecommendation_NoTokenNoPinIDs_InvalidArgument(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _, _, _, _, _ := makeServiceForSaveRecommendation(t, ctrl)
	_, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{City: "Москва"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSaveRecommendation_TokenMissed_FallsBackToPinIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, tripRepo, participantRepo, pinRepo, favRepo, geoRepo, snapRepo := makeServiceForSaveRecommendation(t, ctrl)
	snapRepo.EXPECT().Get(gomock.Any(), "stale").Return(nil, false, nil)
	geoRepo.EXPECT().FindCityByName(gomock.Any(), "Москва").Return(42, nil)
	geoRepo.EXPECT().TripIDsAtLocation(gomock.Any(), 42, gomock.Any()).Return(map[string]struct{}{"tsrc": {}}, nil)
	lat, lon := 55.75, 37.62
	expectMaterialize(tripRepo, participantRepo, pinRepo, favRepo, geoRepo, 42, []*models.Pin{
		{ID: "p1", TripID: "tsrc", Name: "Кремль", Latitude: &lat, Longitude: &lon, IsPublishedInFeed: true},
	}, "new-trip-3")

	resp, err := svc.SaveRecommendation(ctxWithUser("user-1"), &pb.SaveRecommendationRequest{
		SnapshotToken: "stale",
		City:          "Москва",
		PinIds:        []string{"p1"},
	})
	require.NoError(t, err)
	require.Equal(t, "new-trip-3", resp.GetTrip().GetId())
}

var _ = context.Background

func TestPublishTrip_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{
		ID: "t1", OwnerUserID: "user-1", Status: "READY", PrivacyLevel: "Private", IsGenerated: true,
	}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "user-1").Return(true, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.PublishTrip(ctxWithUser("user-1"), &pb.PublishTripRequest{TripId: "t1", PublishWhole: true})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpsertTripPrivacy_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "user-1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{
		ID: "t1", OwnerUserID: "user-1", PrivacyLevel: "Private", IsGenerated: true,
	}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertTripPrivacy(ctxWithUser("user-1"), &pb.UpsertTripPrivacyRequest{TripId: "t1", PrivacyLevel: "Public"})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpdateTrip_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{
		ID: "t1", OwnerUserID: "user-1", IsGenerated: true,
	}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "user-1").Return(true, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	newName := "x"
	_, err := svc.UpdateTrip(ctxWithUser("user-1"), &pb.UpdateTripRequest{TripId: "t1", Name: &newName})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCurrentSeason(t *testing.T) {
	cases := map[string]struct {
		month time.Month
		want string
	}{
		"jan_winter": {time.January, "Зима"},
		"feb_winter": {time.February, "Зима"},
		"dec_winter": {time.December, "Зима"},
		"mar_spring": {time.March, "Весна"},
		"may_spring": {time.May, "Весна"},
		"jun_summer": {time.June, "Лето"},
		"aug_summer": {time.August, "Лето"},
		"sep_autumn": {time.September, "Осень"},
		"nov_autumn": {time.November, "Осень"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := currentSeason(time.Date(2026, tc.month, 15, 0, 0, 0, 0, time.UTC))
			require.Equal(t, tc.want, got)
		})
	}
}
