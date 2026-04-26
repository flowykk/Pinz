package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/testinfra"
	pb "pinz/backend/trip-service/pkg/proto"
)

// TestRecommendations_Integration — ТЗ §9: проверяем фильтр по региону и 2-летнему окну,
// а также применение квот ТЗ 9.2.4–9.2.5 на реальной кластеризации PostGIS.
//
// Сидирование:
// - Москва: 2 опубликованных трипа (score=10 и score=2).
// - Москва: 1 трип старше 2 лет (должен отфильтроваться).
// - Париж: 1 трип (не должен попадать в выборку по Москве).
// Пины: разные категории, разные media_count.
func TestRecommendations_Integration(t *testing.T) {
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
	geoRepo := repositories.NewGeoRegistryRepository(sqlDB)

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, nil, mediaRepo, nil, pinRepo, tagRepo, socialRepo, favRepo, nil, geoRepo, nil, nil, nil, nil, nil, nil, nil, nil)

	ownerID := uuid.New().String()

	// geo_registry seed
	moscowID := upsertCityForTest(t, sqlDB, "Москва", nil)
	parisID := upsertCityForTest(t, sqlDB, "Париж", nil)

	// Москва: top trip (high score), много пинов разных категорий.
	topTripID := insertPublishedTripForTest(t, sqlDB, ownerID, "Москва-топ", 20, 0,
		time.Now().AddDate(0, -1, 0), time.Now().AddDate(0, 0, -10))
	insertTripLocationForTest(t, sqlDB, topTripID, moscowID)
	// 6 sightseeing-кластеров (разные координаты), пины с разным media_count.
	insertPinForTest(t, sqlDB, topTripID, "Достопримечательность", 55.7558, 37.6173, 5, "")
	insertPinForTest(t, sqlDB, topTripID, "Достопримечательность", 55.7600, 37.6200, 3, "")
	insertPinForTest(t, sqlDB, topTripID, "Природа", 55.7700, 37.6300, 2, "")
	insertPinForTest(t, sqlDB, topTripID, "Природа", 55.7800, 37.6400, 1, "")
	insertPinForTest(t, sqlDB, topTripID, "Развлечение", 55.7900, 37.6500, 1, "")
	insertPinForTest(t, sqlDB, topTripID, "Развлечение", 55.8000, 37.6600, 1, "")
	// еда — 4 пина в разных кластерах.
	insertPinForTest(t, sqlDB, topTripID, "Еда и напитки", 55.8100, 37.6700, 4, "")
	insertPinForTest(t, sqlDB, topTripID, "Еда и напитки", 55.8200, 37.6800, 3, "")
	insertPinForTest(t, sqlDB, topTripID, "Еда и напитки", 55.8300, 37.6900, 2, "")
	insertPinForTest(t, sqlDB, topTripID, "Еда и напитки", 55.8400, 37.7000, 1, "")
	// дополнительно — Шопинг для проверки добора.
	insertPinForTest(t, sqlDB, topTripID, "Шопинг", 55.8500, 37.7100, 1, "")

	// Москва: второй трип (low score) — должен быть учтён, но ниже по приоритету.
	secondTripID := insertPublishedTripForTest(t, sqlDB, ownerID, "Москва-2", 2, 0,
		time.Now().AddDate(0, -2, 0), time.Now().AddDate(0, -1, 0))
	insertTripLocationForTest(t, sqlDB, secondTripID, moscowID)
	insertPinForTest(t, sqlDB, secondTripID, "Достопримечательность", 55.9000, 37.7200, 2, "")

	// Москва: устаревший трип (>2 лет) — не должен попасть.
	oldTripID := insertPublishedTripForTest(t, sqlDB, ownerID, "Москва-старая", 100, 0,
		time.Now().AddDate(-3, 0, 0), time.Now().AddDate(-3, 1, 0))
	insertTripLocationForTest(t, sqlDB, oldTripID, moscowID)
	insertPinForTest(t, sqlDB, oldTripID, "Достопримечательность", 56.0000, 38.0000, 9, "should-not-appear")

	// Париж — не должен попадать в выдачу по Москве.
	parisTripID := insertPublishedTripForTest(t, sqlDB, ownerID, "Париж-1", 50, 0,
		time.Now().AddDate(0, -3, 0), time.Now().AddDate(0, -2, 0))
	insertTripLocationForTest(t, sqlDB, parisTripID, parisID)
	insertPinForTest(t, sqlDB, parisTripID, "Достопримечательность", 48.8566, 2.3522, 9, "")

	// сам запрос
	var resp *pb.GetRecommendationsResponse
	err = callAsUser(t, ownerID, "/pinz.TripService/GetRecommendations", func(ctx context.Context) error {
		var err error
		resp, err = svc.GetRecommendations(ctx, &pb.GetRecommendationsRequest{City: "Москва"})
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetMap())
	require.Equal(t, "City", resp.GetMap().GetRegionType())
	require.Equal(t, "Москва", resp.GetMap().GetRegionName())

	pins := resp.GetMap().GetPins()
	require.NotEmpty(t, pins)

	// устаревший пин не должен присутствовать
	for _, p := range pins {
		require.NotEqual(t, "should-not-appear", p.GetDescription(), "old trip's pin must be filtered out")
	}

	// все пины — из московского региона (по координатам близко к Москве, не к Парижу).
	for _, p := range pins {
		require.InDelta(t, 55.8, p.GetLatitude(), 1.0, "pin %s is not in Moscow area", p.GetName())
	}

	// проверяем квоты
	var sightseeing, food int
	for _, p := range pins {
		if recommendationSightseeingCategories[p.GetCategory()] {
			sightseeing++
		}
		if p.GetCategory() == recommendationFoodCategory {
			food++
		}
	}
	require.GreaterOrEqual(t, sightseeing, recommendationMinSightseeing, "expected >=6 sightseeing pins")
	require.GreaterOrEqual(t, food, recommendationMinFood, "expected >=3 food pins")
}

// upsertCityForTest вставляет город и возвращает его id.
func upsertCityForTest(t *testing.T, db *sql.DB, name string, parentID *int) int {
	t.Helper()
	var id int
	row := db.QueryRow(`INSERT INTO geo_registry (name, type, parent_id) VALUES ($1, 'City', $2) RETURNING id`, name, parentID)
	require.NoError(t, row.Scan(&id))
	return id
}

func insertPublishedTripForTest(t *testing.T, db *sql.DB, ownerID, name string, likes, dislikes int, startDate, endDate time.Time) string {
	t.Helper()
	var id string
	row := db.QueryRow(`
INSERT INTO trips (owner_user_id, name, category, season, status, privacy_level, start_date, end_date, likes_count, dislikes_count, is_published)
VALUES ($1, $2, 'Отпуск', 'Лето', 'READY', 'Public', $3, $4, $5, $6, true)
RETURNING id`, ownerID, name, startDate, endDate, likes, dislikes)
	require.NoError(t, row.Scan(&id))
	return id
}

func insertTripLocationForTest(t *testing.T, db *sql.DB, tripID string, locationID int) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO trip_locations (trip_id, location_id) VALUES ($1, $2)`, tripID, locationID)
	require.NoError(t, err)
}

func insertPinForTest(t *testing.T, db *sql.DB, tripID, category string, lat, lon float64, mediaCount int, description string) string {
	t.Helper()
	var id string
	row := db.QueryRow(`
INSERT INTO pins (trip_id, name, description, category, location, media_count, is_published_in_feed, location_name)
VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, true, '')
RETURNING id`, tripID, category, description, category, lon, lat, mediaCount)
	require.NoError(t, row.Scan(&id))
	return id
}

