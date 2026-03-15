package services

import (
	"time"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

// clusterMediaToDraftPins returns groups of media IDs: first by PostGIS clustering (with location),
// then media with time only assigned to nearest cluster (<10 min), rest in last group.
func clusterMediaToDraftPins(mediaRepo *repositories.MediaRepository, tripID string) [][]string {
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
		// Fallback: each with-loc media its own cluster
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

func updatePinTimesAndLocation(pinRepo *repositories.PinRepository, mediaRepo *repositories.MediaRepository, pinID string) {
	pinMedia, err := mediaRepo.ListByPinID(pinID)
	if err != nil || len(pinMedia) == 0 {
		return
	}
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
	pin, err := pinRepo.GetByID(pinID)
	if err != nil {
		return
	}
	pin.StartTime = startTime
	pin.EndTime = endTime
	pin.Latitude = lat
	pin.Longitude = lon
	_ = pinRepo.Update(pin)
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
