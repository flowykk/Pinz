package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sort"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

// Параметры рекомендательной системы из ТЗ §9.
const (
	// 9.2.3.a/b: адаптивный радиус кластеризации.
	recommendationCityEpsMeters = 50.0
	recommendationCountryEpsMeters = 500.0
	// 9.2.4 / 9.2.5: минимумы по тематикам.
	recommendationMinSightseeing = 6
	recommendationMinFood = 3
	// сколько медиа-превью отдавать на пин в карте популярных мест.
	recommendationMediaPerPin = 8
	// мягкий потолок размера итоговой карты, чтобы не возвращать на клиент сотни пинов.
	recommendationMaxPins = 25
	// сколько медиа отдавать в общей карусели карточки ленты (как в ListFeed).
	recommendationFeedMediaLimit = 8
)

// Категории из ТЗ 2.2.4, на которые опираются квоты ТЗ 9.2.4–9.2.5.
var (
	recommendationSightseeingCategories = map[string]bool{
		"Достопримечательность": true,
		"Природа": true,
		"Развлечение": true,
	}
	recommendationFoodCategory = "Еда и напитки"
)

// GetRecommendations — ТЗ 9.1–9.3. Принимает либо city, либо country (XOR),
// возвращает карту популярных мест: один пин на кластер из топ-трипов региона
// за последние 2 года.
func (s *TripService) GetRecommendations(ctx context.Context, req *pb.GetRecommendationsRequest) (*pb.GetRecommendationsResponse, error) {
	if _, ok := server.UserIDFromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	region, err := s.resolveRecommendationRegion(ctx, req.GetCity(), req.GetCountry())
	if err != nil {
		return nil, err
	}
	pins, err := s.buildRecommendationPins(ctx, region)
	if err != nil {
		return nil, err
	}
	media := topMediaFromRecommendedPins(pins, recommendationFeedMediaLimit)
	return &pb.GetRecommendationsResponse{
		Map: &pb.RecommendedMap{
			RegionName: region.name,
			RegionType: region.kind,
			Pins: pins,
			Trip: virtualRecommendationTrip(region, pins),
			Media: media,
		},
	}, nil
}

// SaveRecommendation — ТЗ 9.4. Переисполняет алгоритм и материализует карту
// как сгенерированный приватный трип пользователя, добавляя его в favourites.
func (s *TripService) SaveRecommendation(ctx context.Context, req *pb.SaveRecommendationRequest) (*pb.SaveRecommendationResponse, error) {
	userID, ok := server.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user_id required")
	}
	region, err := s.resolveRecommendationRegion(ctx, req.GetCity(), req.GetCountry())
	if err != nil {
		return nil, err
	}
	pins, err := s.buildRecommendationPins(ctx, region)
	if err != nil {
		return nil, err
	}
	if len(pins) == 0 {
		return nil, status.Error(codes.NotFound, "no recommendations available for region")
	}

	trip := &models.Trip{
		OwnerUserID: userID,
		Name: "Популярные места: " + region.name,
		Description: "Автоматически собранная карта популярных мест по " + region.name,
		Category: "Другое",
		Season: currentSeason(time.Now()),
		Status: "READY",
		PrivacyLevel: "Private",
		IsGenerated: true,
	}
	if err := s.tripRepo.Create(trip); err != nil {
		slog.ErrorContext(ctx, "SaveRecommendation: create trip failed", "error", err)
		return nil, status.Error(codes.Internal, "failed to create trip")
	}
	if err := s.participantRepo.Add(&models.TripParticipant{
		TripID: trip.ID,
		UserID: userID,
		IsAdmin: true,
	}); err != nil {
		slog.ErrorContext(ctx, "SaveRecommendation: add participant failed", "error", err, "trip_id", trip.ID)
		return nil, status.Error(codes.Internal, "failed to add participant")
	}
	for _, p := range pins {
		lat := p.GetLatitude()
		lon := p.GetLongitude()
		copyPin := &models.Pin{
			TripID: trip.ID,
			Name: p.GetName(),
			Description: p.GetDescription(),
			Category: p.GetCategory(),
			LocationName: p.GetLocationName(),
			PrivacyLevel: "Private",
			MediaCount: 0,
			IsPublishedInFeed: false,
			Latitude: &lat,
			Longitude: &lon,
		}
		if err := s.pinRepo.Create(copyPin); err != nil {
			slog.WarnContext(ctx, "SaveRecommendation: create pin failed", "error", err, "trip_id", trip.ID)
		}
	}
	if err := s.geoRepo.UpsertTripLocations(ctx, trip.ID, []int{region.id}); err != nil {
		slog.WarnContext(ctx, "SaveRecommendation: upsert trip_locations failed", "error", err, "trip_id", trip.ID)
	}
	if err := s.favouriteRepo.Add(userID, trip.ID); err != nil {
		slog.WarnContext(ctx, "SaveRecommendation: add favourite failed", "error", err, "trip_id", trip.ID)
	}

	saved, err := s.tripRepo.GetByID(trip.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reload trip")
	}
	return &pb.SaveRecommendationResponse{Trip: s.tripToProto(ctx, saved)}, nil
}

// recommendationRegion — резолвленный регион (geo_registry id + тип + имя).
type recommendationRegion struct {
	id int
	kind string // "City" | "Country"
	name string
	eps float64
}

func (s *TripService) resolveRecommendationRegion(ctx context.Context, city, country string) (*recommendationRegion, error) {
	city = trimToNonEmpty(city)
	country = trimToNonEmpty(country)
	if (city == "") == (country == "") {
		return nil, status.Error(codes.InvalidArgument, "exactly one of city or country must be provided")
	}
	var (
		id int
		err error
		kind, name string
		eps float64
	)
	if city != "" {
		id, err = s.geoRepo.FindCityByName(ctx, city)
		kind, name, eps = "City", city, recommendationCityEpsMeters
	} else {
		id, err = s.geoRepo.FindCountryByName(ctx, country)
		kind, name, eps = "Country", country, recommendationCountryEpsMeters
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "region not found")
		}
		return nil, status.Error(codes.Internal, "failed to resolve region")
	}
	return &recommendationRegion{id: id, kind: kind, name: name, eps: eps}, nil
}

// buildRecommendationPins — общий пайплайн ТЗ 9.2–9.3, используется и для GET, и для save.
func (s *TripService) buildRecommendationPins(ctx context.Context, region *recommendationRegion) ([]*pb.RecommendedPin, error) {
	candidates, err := s.pinRepo.ListRecommendationCandidates(region.id, region.eps)
	if err != nil {
		slog.ErrorContext(ctx, "GetRecommendations: list candidates failed", "error", err, "region_id", region.id)
		return nil, status.Error(codes.Internal, "failed to list candidates")
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	selected := pickRecommendationPins(candidates)
	if len(selected) == 0 {
		return nil, nil
	}
	pinIDs := make([]string, len(selected))
	for i, c := range selected {
		pinIDs[i] = c.ID
	}
	mediaByPin, err := s.mediaRepo.TopMediaByPinIDs(pinIDs, recommendationMediaPerPin)
	if err != nil {
		slog.WarnContext(ctx, "GetRecommendations: fetch media failed", "error", err)
		mediaByPin = make(map[string][]*repositories.FeedMedia)
	}
	out := make([]*pb.RecommendedPin, len(selected))
	for i, c := range selected {
		media := mediaByPin[c.ID]
		protoMedia := make([]*pb.FeedMedia, len(media))
		for j, m := range media {
			protoMedia[j] = &pb.FeedMedia{
				MediaId: m.ID,
				Url: s.presignedReadURL(ctx, m.S3Key),
				MediaType: m.MediaType,
			}
		}
		out[i] = &pb.RecommendedPin{
			Id: c.ID,
			TripId: c.TripID,
			Latitude: c.Latitude,
			Longitude: c.Longitude,
			Name: c.Name,
			Description: c.Description,
			Category: c.Category,
			LocationName: c.LocationName,
			MediaCount: c.MediaCount,
			Media: protoMedia,
		}
	}
	return out, nil
}

// pickRecommendationPins — выбор по правилам ТЗ 9.3.1, 9.2.4, 9.2.5:
//   - по одному пину из каждого (category, cluster_id);
//   - в кластере выбираем по media_count DESC, при равенстве — по длиннее description;
//   - сначала добираем минимумы для жёстких категорий (6 sightseeing + 3 food),
//     затем заполняем оставшееся другими категориями по убыванию trip_score кандидатов
//     до recommendationMaxPins.
func pickRecommendationPins(candidates []*repositories.RecommendationPinCandidate) []*repositories.RecommendationPinCandidate {
	type clusterKey struct {
		category string
		clusterID int32
	}
	clusters := make(map[clusterKey]*repositories.RecommendationPinCandidate)
	for _, c := range candidates {
		key := clusterKey{c.Category, c.ClusterID}
		best, ok := clusters[key]
		if !ok || isBetterCandidate(c, best) {
			clusters[key] = c
		}
	}
	picks := make([]*repositories.RecommendationPinCandidate, 0, len(clusters))
	for _, c := range clusters {
		picks = append(picks, c)
	}

	var sightseeing, food, other []*repositories.RecommendationPinCandidate
	for _, p := range picks {
		switch {
		case recommendationSightseeingCategories[p.Category]:
			sightseeing = append(sightseeing, p)
		case p.Category == recommendationFoodCategory:
			food = append(food, p)
		default:
			other = append(other, p)
		}
	}
	for _, slice := range [][]*repositories.RecommendationPinCandidate{sightseeing, food, other} {
		sort.SliceStable(slice, func(i, j int) bool {
			return slice[i].TripScore > slice[j].TripScore
		})
	}

	final := make([]*repositories.RecommendationPinCandidate, 0, recommendationMaxPins)
	final = append(final, takeFirst(sightseeing, recommendationMinSightseeing)...)
	final = append(final, takeFirst(food, recommendationMinFood)...)
	if len(final) < recommendationMaxPins {
		seen := make(map[string]bool, len(final))
		for _, p := range final {
			seen[p.ID] = true
		}
		extra := make([]*repositories.RecommendationPinCandidate, 0, len(picks))
		for _, slice := range [][]*repositories.RecommendationPinCandidate{sightseeing, food, other} {
			for _, p := range slice {
				if !seen[p.ID] {
					extra = append(extra, p)
				}
			}
		}
		sort.SliceStable(extra, func(i, j int) bool {
			return extra[i].TripScore > extra[j].TripScore
		})
		for _, p := range extra {
			if len(final) >= recommendationMaxPins {
				break
			}
			final = append(final, p)
		}
	}
	return final
}

// isBetterCandidate — реализует правило ТЗ 9.3.1:
// первичный ключ — media_count DESC, тай-брейк — длиннее description.
func isBetterCandidate(a, b *repositories.RecommendationPinCandidate) bool {
	if a.MediaCount != b.MediaCount {
		return a.MediaCount > b.MediaCount
	}
	return len(a.Description) > len(b.Description)
}

func takeFirst(s []*repositories.RecommendationPinCandidate, n int) []*repositories.RecommendationPinCandidate {
	if n > len(s) {
		n = len(s)
	}
	out := make([]*repositories.RecommendationPinCandidate, n)
	copy(out, s[:n])
	return out
}

func trimToNonEmpty(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// virtualRecommendationTrip собирает Trip-обёртку, чтобы карта рекомендаций
// могла встраиваться в общую ленту как обычный FeedItem. id оставляется пустым:
// материализуется только при SaveRecommendation. Счётчики лайков/дизлайков —
// 0, т.к. виртуальный объект ещё нигде не опубликован.
func virtualRecommendationTrip(region *recommendationRegion, pins []*pb.RecommendedPin) *pb.Trip {
	now := time.Now().Unix()
	return &pb.Trip{
		Name: "Популярные места: " + region.name,
		Description: "Автоматически собранная карта популярных мест по " + region.name,
		Category: "Другое",
		Season: currentSeason(time.Now()),
		Status: "READY",
		PrivacyLevel: "Public",
		StartDateUnix: now,
		EndDateUnix: now,
		IsPublished: true,
		IsGenerated: true,
		CreatedAtUnix: now,
		UpdatedAtUnix: now,
		PinsCount: int32(len(pins)),
	}
}

// topMediaFromRecommendedPins собирает топ-N медиа для карусели карточки ленты.
// Внутри каждого пина media уже отсортирован по battle_rating DESC; берём
// поочерёдно первое медиа из каждого пина (round-robin), что обеспечивает
// разнообразие пинов в карусели и сохраняет приоритет лучших медиа.
func topMediaFromRecommendedPins(pins []*pb.RecommendedPin, limit int) []*pb.FeedMedia {
	if limit <= 0 {
		return nil
	}
	out := make([]*pb.FeedMedia, 0, limit)
	indexes := make([]int, len(pins))
	for len(out) < limit {
		appended := false
		for i, p := range pins {
			if indexes[i] >= len(p.GetMedia()) {
				continue
			}
			out = append(out, p.GetMedia()[indexes[i]])
			indexes[i]++
			appended = true
			if len(out) >= limit {
				break
			}
		}
		if !appended {
			break
		}
	}
	return out
}

// currentSeason — ТЗ 2.3.5 список (Зима/Весна/Лето/Осень) по месяцу.
func currentSeason(t time.Time) string {
	switch t.Month() {
	case time.December, time.January, time.February:
		return "Зима"
	case time.March, time.April, time.May:
		return "Весна"
	case time.June, time.July, time.August:
		return "Лето"
	default:
		return "Осень"
	}
}
