package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/repositories"
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
