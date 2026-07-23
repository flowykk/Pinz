package services

import (
	"fmt"
	"log/slog"
	"time"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

// clusterMediaToDraftPins returns groups of media IDs: first by PostGIS clustering (with location),
// then media with time only assigned to nearest cluster (<10 min), rest in last group.
func clusterMediaToDraftPins(mediaRepo repositories.MediaRepositoryInterface, tripID string) [][]string {
	mediaList, err := mediaRepo.ListByTripID(tripID)
	if err != nil || len(mediaList) == 0 {
		return nil
	}
	// 1) Cluster by location (PostGIS) - get cluster index per media ID
	withLoc := make([]*models.Media, 0)
	noLocWithTime := make([]*models.Media, 0)
	noLocNoTime := make([]*models.Media, 0)
	for _, m := range mediaList {
		if m.Latitude != nil && m.Longitude != nil {
			withLoc = append(withLoc, m)
		} else if m.CapturedAt != nil {
			noLocWithTime = append(noLocWithTime, m)
		} else {
			noLocNoTime = append(noLocNoTime, m)
		}
	}
	// Run clustering in DB for withLoc
	clusterIDs, err := mediaRepo.ClusterIDsByLocation(tripID, float64(ClusterRadiusMeters))
	if err != nil {
		slog.Warn("grouping: PostGIS clustering failed, falling back to per-media clusters",
			"trip_id", tripID, "err", err)
		clusterIDs = make(map[string]int)
		for i, m := range withLoc {
			clusterIDs[m.ID] = i
		}
	}
	// Build groups: cluster index -> []mediaID
	groups := make(map[int][]string)
	for _, m := range withLoc {
		cid := clusterIDs[m.ID]
		groups[cid] = append(groups[cid], m.ID)
	}
	// 2) Assign noLocWithTime to nearest cluster by time (< TimeClusterMinutes)
	clusterTimes := make(map[int]time.Time) // cluster centroid time (first media)
	for cid, ids := range groups {
		if len(ids) == 0 {
			continue
		}
		for _, m := range withLoc {
			if m.ID == ids[0] && m.CapturedAt != nil {
				clusterTimes[cid] = *m.CapturedAt
				break
			}
		}
	}
	for _, m := range noLocWithTime {
		t := *m.CapturedAt
		assigned := false
		for cid, ct := range clusterTimes {
			diff := t.Sub(ct)
			if diff < 0 {
				diff = -diff
			}
			if diff < TimeClusterMinutes*time.Minute {
				groups[cid] = append(groups[cid], m.ID)
				assigned = true
				break
			}
		}
		if !assigned {
			noLocNoTime = append(noLocNoTime, m)
		}
	}
	// 3) noLocNoTime -> one group "draft-unassigned"
	var unassigned []string
	for _, m := range noLocNoTime {
		unassigned = append(unassigned, m.ID)
	}
	// Order groups into [][]string (deterministic: by first media ID or cluster index)
	result := make([][]string, 0, len(groups)+1)
	seen := make(map[int]bool)
	for _, m := range withLoc {
		cid := clusterIDs[m.ID]
		if !seen[cid] {
			seen[cid] = true
			result = append(result, groups[cid])
		}
	}
	if len(unassigned) > 0 {
		result = append(result, unassigned)
	}
	return result
}

// DraftPinGroup is a media group with an assigned draft_pin_id identifier.
// "existing-{pin_id}" — original pin seed-group.
// "cluster-{N}" — new cluster from just-added media.
// "draft-unassigned" — media without coordinates and time.
type DraftPinGroup struct {
	DraftPinID string
	MediaIDs   []string
}

// clusterMediaWithExistingPinsAsSeeds groups trip media using existing pins as seed clusters.
// It returns groups in deterministic order: existing pins first (by pin ID), then new clusters, then unassigned.
func clusterMediaWithExistingPinsAsSeeds(mediaRepo repositories.MediaRepositoryInterface, pinRepo repositories.PinRepositoryInterface, tripID string) []DraftPinGroup {
	mediaList, err := mediaRepo.ListByTripID(tripID)
	if err != nil || len(mediaList) == 0 {
		return nil
	}
	pins, _ := pinRepo.ListByTripID(tripID)

	// 1) Media с pin_id — seed-группы по существующим пинам.
	existingGroups := make(map[string][]string)
	mediaByID := make(map[string]*models.Media, len(mediaList))
	existingMediaSet := make(map[string]struct{})
	for _, m := range mediaList {
		mediaByID[m.ID] = m
		if m.PinID != nil {
			existingGroups[*m.PinID] = append(existingGroups[*m.PinID], m.ID)
			existingMediaSet[m.ID] = struct{}{}
		}
	}

	// 2) Кластеризация по геолокации среди всех медиа с координатами.
	// Новое медиа с координатами, попавшее в кластер с existing-медиа, присоединяется к соответствующему пину.
	clusterIDs, clusterErr := mediaRepo.ClusterIDsByLocation(tripID, float64(ClusterRadiusMeters))
	if clusterErr != nil {
		slog.Warn("grouping: PostGIS clustering failed during add-media",
			"trip_id", tripID, "err", clusterErr)
	}
	clusterToExistingPin := make(map[int]string)
	for mediaID, cid := range clusterIDs {
		if m, ok := mediaByID[mediaID]; ok && m.PinID != nil {
			clusterToExistingPin[cid] = *m.PinID
		}
	}

	// Новые медиа с координатами: либо приклеиваются к существующему пину, либо формируют новые кластеры.
	newClusters := make(map[int][]string) // cluster_id -> []media_id
	newWithLocNoCluster := make([]*models.Media, 0)
	noLocWithTime := make([]*models.Media, 0)
	noLocNoTime := make([]*models.Media, 0)
	for _, m := range mediaList {
		if _, isExisting := existingMediaSet[m.ID]; isExisting {
			continue
		}
		if m.Latitude != nil && m.Longitude != nil {
			cid, ok := clusterIDs[m.ID]
			if !ok {
				newWithLocNoCluster = append(newWithLocNoCluster, m)
				continue
			}
			if pinID, ok := clusterToExistingPin[cid]; ok {
				existingGroups[pinID] = append(existingGroups[pinID], m.ID)
			} else {
				newClusters[cid] = append(newClusters[cid], m.ID)
			}
		} else if m.CapturedAt != nil {
			noLocWithTime = append(noLocWithTime, m)
		} else {
			noLocNoTime = append(noLocNoTime, m)
		}
	}
	// Фоллбэк: если ClusterIDsByLocation не сработал — каждое оставшееся медиа в отдельный кластер.
	for i, m := range newWithLocNoCluster {
		cid := -(i + 1)
		newClusters[cid] = append(newClusters[cid], m.ID)
	}

	// 3) Медиа без координат, но со временем — приклеить к ближайшему кластеру по времени (< TimeClusterMinutes).
	// Центроид времени кластера = время первого медиа в группе.
	existingClusterTimes := make(map[string]time.Time)
	for pinID, ids := range existingGroups {
		for _, id := range ids {
			if m := mediaByID[id]; m != nil && m.CapturedAt != nil {
				existingClusterTimes[pinID] = *m.CapturedAt
				break
			}
		}
	}
	newClusterTimes := make(map[int]time.Time)
	for cid, ids := range newClusters {
		for _, id := range ids {
			if m := mediaByID[id]; m != nil && m.CapturedAt != nil {
				newClusterTimes[cid] = *m.CapturedAt
				break
			}
		}
	}
	for _, m := range noLocWithTime {
		t := *m.CapturedAt
		bestKind := "" // "existing" / "new" / ""
		bestID := ""   // pin_id для existing
		bestCID := 0   // cluster id для new
		bestDiff := time.Duration(TimeClusterMinutes) * time.Minute
		for pinID, ct := range existingClusterTimes {
			diff := t.Sub(ct)
			if diff < 0 {
				diff = -diff
			}
			if diff < bestDiff {
				bestDiff = diff
				bestKind = "existing"
				bestID = pinID
			}
		}
		for cid, ct := range newClusterTimes {
			diff := t.Sub(ct)
			if diff < 0 {
				diff = -diff
			}
			if diff < bestDiff {
				bestDiff = diff
				bestKind = "new"
				bestCID = cid
			}
		}
		switch bestKind {
		case "existing":
			existingGroups[bestID] = append(existingGroups[bestID], m.ID)
		case "new":
			newClusters[bestCID] = append(newClusters[bestCID], m.ID)
		default:
			noLocNoTime = append(noLocNoTime, m)
		}
	}

	// 4) Собрать результат: существующие пины (в порядке pinRepo.ListByTripID), затем новые кластеры, затем unassigned.
	result := make([]DraftPinGroup, 0, len(existingGroups)+len(newClusters)+1)
	seenPins := make(map[string]bool)
	for _, p := range pins {
		if ids, ok := existingGroups[p.ID]; ok {
			result = append(result, DraftPinGroup{DraftPinID: "existing-" + p.ID, MediaIDs: ids})
			seenPins[p.ID] = true
		}
	}
	// Дополнительно добавить группы, чей пин не попал в pinRepo.ListByTripID (на всякий случай).
	for pinID, ids := range existingGroups {
		if !seenPins[pinID] {
			result = append(result, DraftPinGroup{DraftPinID: "existing-" + pinID, MediaIDs: ids})
		}
	}
	clusterIdx := 0
	for _, ids := range newClusters {
		if len(ids) > 0 {
			result = append(result, DraftPinGroup{DraftPinID: fmt.Sprintf("cluster-%d", clusterIdx), MediaIDs: ids})
			clusterIdx++
		}
	}
	if len(noLocNoTime) > 0 {
		unassigned := make([]string, 0, len(noLocNoTime))
		for _, m := range noLocNoTime {
			unassigned = append(unassigned, m.ID)
		}
		result = append(result, DraftPinGroup{DraftPinID: "draft-unassigned", MediaIDs: unassigned})
	}
	return result
}

func updatePinTimesAndLocation(pinRepo repositories.PinRepositoryInterface, mediaRepo repositories.MediaRepositoryInterface, pinID string) *models.Pin {
	pinMedia, err := mediaRepo.ListByPinID(pinID)
	if err != nil || len(pinMedia) == 0 {
		return nil
	}
	pin, err := pinRepo.GetByID(pinID)
	if err != nil {
		return nil
	}
	applyPinTimesAndLocationFromMedia(pin, pinMedia)
	if err := pinRepo.Update(pin); err != nil {
		return nil
	}
	return pin
}

// updatePinTimesAndLocationFor — вариант для случаев, когда pin только что
// создан/прочитан и лишний GetByID не нужен (PINZ-223).
func updatePinTimesAndLocationFor(pinRepo repositories.PinRepositoryInterface, mediaRepo repositories.MediaRepositoryInterface, pin *models.Pin) *models.Pin {
	if pin == nil {
		return nil
	}
	pinMedia, err := mediaRepo.ListByPinID(pin.ID)
	if err != nil || len(pinMedia) == 0 {
		return nil
	}
	applyPinTimesAndLocationFromMedia(pin, pinMedia)
	if err := pinRepo.Update(pin); err != nil {
		return nil
	}
	return pin
}

func applyPinTimesAndLocationFromMedia(pin *models.Pin, pinMedia []*models.Media) {
	var startTime, endTime *time.Time
	var lat, lon *float64
	for _, m := range pinMedia {
		if m.CapturedAt != nil {
			if startTime == nil || m.CapturedAt.Before(*startTime) {
				startTime = m.CapturedAt
			}
			if endTime == nil || m.CapturedAt.After(*endTime) {
				endTime = m.CapturedAt
			}
		}
		if m.Latitude != nil && m.Longitude != nil && lat == nil {
			lat = m.Latitude
			lon = m.Longitude
		}
	}
	// Ненулевые значения пина (в т.ч. выставленные вручную) не трогаем;
	// из медиа заполняем только пустые поля.
	if pin.StartTime == nil {
		pin.StartTime = startTime
	}
	if pin.EndTime == nil {
		pin.EndTime = endTime
	}
	if pin.Latitude == nil {
		pin.Latitude = lat
		pin.Longitude = lon
	}
}

func contentTypeToExt(ct string) string {
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/heic":
		return ".heic"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	default:
		return ".bin"
	}
}
