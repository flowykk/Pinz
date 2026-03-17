package services

import (
	"math"
	"time"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

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

func attachNewMediaToExistingPins(newMedia []*models.Media, existingPins []*models.Pin, radiusMeters float64, timeWindowMinutes int) (attached map[string][]string, remaining []*models.Media, unassigned []string) {
	attached = make(map[string][]string)
	if len(newMedia) == 0 {
		return attached, nil, nil
	}
	// Precompute pins with geo/time
	type pinCtx struct {
		id        string
		lat, lon  *float64
		startTime *time.Time
		endTime   *time.Time
	}
	pins := make([]pinCtx, 0, len(existingPins))
	for _, p := range existingPins {
		pins = append(pins, pinCtx{
			id:        p.ID,
			lat:       p.Latitude,
			lon:       p.Longitude,
			startTime: p.StartTime,
			endTime:   p.EndTime,
		})
	}

	timeWindow := time.Duration(timeWindowMinutes) * time.Minute
	for _, m := range newMedia {
		// 1) Geo attach
		if m.Latitude != nil && m.Longitude != nil {
			bestPin := ""
			bestDist := radiusMeters + 1
			for _, p := range pins {
				if p.lat == nil || p.lon == nil {
					continue
				}
				d := haversineMeters(*m.Latitude, *m.Longitude, *p.lat, *p.lon)
				if d <= radiusMeters && d < bestDist {
					bestDist = d
					bestPin = p.id
				}
			}
			if bestPin != "" {
				attached[bestPin] = append(attached[bestPin], m.ID)
				continue
			}
		}

		// 2) Time attach (only if has capturedAt)
		if m.CapturedAt != nil {
			bestPin := ""
			var bestDiff time.Duration
			for _, p := range pins {
				if p.startTime == nil && p.endTime == nil {
					continue
				}
				diff := durationToPinWindow(*m.CapturedAt, p.startTime, p.endTime)
				if bestPin == "" || diff < bestDiff {
					bestDiff = diff
					bestPin = p.id
				}
			}
			if bestPin != "" && bestDiff <= timeWindow {
				attached[bestPin] = append(attached[bestPin], m.ID)
				continue
			}
			remaining = append(remaining, m)
			continue
		}

		// 3) No geo/time
		unassigned = append(unassigned, m.ID)
	}
	return attached, remaining, unassigned
}

func clusterNewMediaStandalone(media []*models.Media, radiusMeters float64, timeWindowMinutes int) [][]string {
	if len(media) == 0 {
		return nil
	}
	withLoc := make([]*models.Media, 0)
	noLocWithTime := make([]*models.Media, 0)
	for _, m := range media {
		if m.Latitude != nil && m.Longitude != nil {
			withLoc = append(withLoc, m)
		} else if m.CapturedAt != nil {
			noLocWithTime = append(noLocWithTime, m)
		}
	}
	// Geo clustering (naive union-find)
	parent := make([]int, len(withLoc))
	for i := range parent {
		parent[i] = i
	}
	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for i := 0; i < len(withLoc); i++ {
		for j := i + 1; j < len(withLoc); j++ {
			di := haversineMeters(*withLoc[i].Latitude, *withLoc[i].Longitude, *withLoc[j].Latitude, *withLoc[j].Longitude)
			if di <= radiusMeters {
				union(i, j)
			}
		}
	}
	geoGroups := make(map[int][]*models.Media)
	for i, m := range withLoc {
		geoGroups[find(i)] = append(geoGroups[find(i)], m)
	}
	// Create deterministic list of geo groups (by earliest captured_at, then id)
	type geoGroup struct {
		key   int
		media []*models.Media
	}
	gg := make([]geoGroup, 0, len(geoGroups))
	for k, g := range geoGroups {
		gg = append(gg, geoGroup{key: k, media: g})
	}
	// simple stable ordering by min id (good enough for deterministic output)
	for i := 0; i < len(gg); i++ {
		for j := i + 1; j < len(gg); j++ {
			if minMediaID(gg[j].media) < minMediaID(gg[i].media) {
				gg[i], gg[j] = gg[j], gg[i]
			}
		}
	}

	// Precompute time centroid for each geo group (first captured_at found)
	type timeCentroid struct {
		ids []string
		t   *time.Time
	}
	out := make([]timeCentroid, 0, len(gg))
	for _, g := range gg {
		ids := make([]string, 0, len(g.media))
		var t *time.Time
		for _, m := range g.media {
			ids = append(ids, m.ID)
			if t == nil && m.CapturedAt != nil {
				tt := *m.CapturedAt
				t = &tt
			}
		}
		out = append(out, timeCentroid{ids: ids, t: t})
	}

	timeWindow := time.Duration(timeWindowMinutes) * time.Minute
	// Attach time-only media to nearest geo cluster by time window
	for _, m := range noLocWithTime {
		bestIdx := -1
		var bestDiff time.Duration
		for i := range out {
			if out[i].t == nil {
				continue
			}
			d := *m.CapturedAt
			diff := d.Sub(*out[i].t)
			if diff < 0 {
				diff = -diff
			}
			if bestIdx == -1 || diff < bestDiff {
				bestIdx = i
				bestDiff = diff
			}
		}
		if bestIdx != -1 && bestDiff <= timeWindow {
			out[bestIdx].ids = append(out[bestIdx].ids, m.ID)
		} else {
			// new group of its own
			out = append(out, timeCentroid{ids: []string{m.ID}})
		}
	}

	res := make([][]string, 0, len(out))
	for _, g := range out {
		if len(g.ids) > 0 {
			res = append(res, g.ids)
		}
	}
	return res
}

func durationToPinWindow(t time.Time, start, end *time.Time) time.Duration {
	if start != nil && end != nil {
		if !t.Before(*start) && !t.After(*end) {
			return 0
		}
		if t.Before(*start) {
			return start.Sub(t)
		}
		return t.Sub(*end)
	}
	if start != nil {
		d := t.Sub(*start)
		if d < 0 {
			d = -d
		}
		return d
	}
	d := t.Sub(*end)
	if d < 0 {
		d = -d
	}
	return d
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	phi1, phi2 := toRad(lat1), toRad(lat2)
	dphi := toRad(lat2 - lat1)
	dlam := toRad(lon2 - lon1)
	a := math.Sin(dphi/2)*math.Sin(dphi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dlam/2)*math.Sin(dlam/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func minMediaID(list []*models.Media) string {
	min := ""
	for _, m := range list {
		if min == "" || m.ID < min {
			min = m.ID
		}
	}
	return min
}
