package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestE2E_Health(t *testing.T) {
	st := startStack(t)
	t.Run("GET_health_returns_200", func(t *testing.T) {
		resp, err := http.Get(st.baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestE2E_Auth_DevLogin(t *testing.T) {
	st := startStack(t)
	userID := uuid.New().String()
	email := "user@example.com"
	st.seedUser(t, userID, email, "user")

	t.Run("returns_tokens", func(t *testing.T) {
		resp, body := st.doJSON(t, http.MethodPost, "/api/v1/auth/dev-login", "", `{"email":"`+email+`"}`)
		require.Equal(t, http.StatusOK, resp.StatusCode, body)
		var login struct {
			AccessToken string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &login))
		require.NotEmpty(t, login.AccessToken)
		require.NotEmpty(t, login.RefreshToken)
	})
}

func TestE2E_Trip_CreationStart(t *testing.T) {
	st := startStack(t)
	userID := uuid.New().String()
	email := "user@example.com"
	st.seedUser(t, userID, email, "user")

	resp, body := st.doJSON(t, http.MethodPost, "/api/v1/auth/dev-login", "", `{"email":"`+email+`"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var login struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &login))
	require.NotEmpty(t, login.AccessToken)

	t.Run("POST_creation_start_returns_201_and_upload_urls", func(t *testing.T) {
		startBody := `{
	 "name":"Trip",
	 "category":"vacation",
	 "season":"summer",
	 "description":"d",
	 "privacy_level":"private",
	 "files_to_upload":[{"client_id":"c1","content_type":"image/jpeg"}]
	}`
		resp, body := st.doJSON(t, http.MethodPost, "/api/v1/trips/creation/start", login.AccessToken, startBody)
		require.Equal(t, http.StatusCreated, resp.StatusCode, body)

		var created struct {
			TripID string `json:"trip_id"`
			Status string `json:"status"`
			UploadUrls []struct {
				ClientID string `json:"client_id"`
				S3Key string `json:"s3_key"`
				URL string `json:"url"`
			} `json:"upload_urls"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &created))
		require.NotEmpty(t, created.TripID)
		require.Equal(t, "UPLOADING", created.Status)
		require.Len(t, created.UploadUrls, 1)
		require.Contains(t, created.UploadUrls[0].S3Key, created.TripID)
	})
}

func TestE2E_Trip_InviteJoin_GetTrip(t *testing.T) {
	st := startStack(t)

	ownerID := uuid.New().String()
	ownerEmail := "owner@example.com"
	st.seedUser(t, ownerID, ownerEmail, "owner")

	user2ID := uuid.New().String()
	user2Email := "user2@example.com"
	st.seedUser(t, user2ID, user2Email, "user2")

	resp, body := st.doJSON(t, http.MethodPost, "/api/v1/auth/dev-login", "", `{"email":"`+ownerEmail+`"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var ownerLogin struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &ownerLogin))
	require.NotEmpty(t, ownerLogin.AccessToken)

	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips", ownerLogin.AccessToken, `{
	 "name":"Trip",
	 "category":"vacation",
	 "season":"summer",
	 "description":"d",
	 "privacy_level":"private",
	 "files_to_upload":[]
	}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode, body)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(body), &created)
	var created2 struct {
		TripID string `json:"trip_id"`
	}
	_ = json.Unmarshal([]byte(body), &created2)
	tripID := created.ID
	if tripID == "" {
		tripID = created2.TripID
	}
	require.NotEmpty(t, tripID)

	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/"+tripID+"/invite", ownerLogin.AccessToken, `{}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode, body)
	var invite struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &invite))
	require.NotEmpty(t, invite.Token)

	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/auth/dev-login", "", `{"email":"`+user2Email+`"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var user2Login struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &user2Login))
	require.NotEmpty(t, user2Login.AccessToken)

	t.Run("POST_join_returns_200", func(t *testing.T) {
		resp, body := st.doJSON(t, http.MethodPost, "/api/v1/trips/join", user2Login.AccessToken, `{"token":"`+invite.Token+`"}`)
		require.Equal(t, http.StatusOK, resp.StatusCode, body)
	})

	t.Run("GET_trip_as_member_returns_200", func(t *testing.T) {
		resp, body := st.doJSON(t, http.MethodGet, "/api/v1/trips/"+tripID, user2Login.AccessToken, "")
		require.Equal(t, http.StatusOK, resp.StatusCode, body)
		_ = body
	})
}

func TestE2E_Social_LikeDislikeFavourite(t *testing.T) {
	st := startStack(t)

	ownerID := uuid.New().String()
	ownerEmail := "social@example.com"
	st.seedUser(t, ownerID, ownerEmail, "social")

	resp, body := st.doJSON(t, http.MethodPost, "/api/v1/auth/dev-login", "", `{"email":"`+ownerEmail+`"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var login struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &login))
	require.NotEmpty(t, login.AccessToken)

	// Create trip via creation flow (with file) to get UPLOADING
	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/creation/start", login.AccessToken, `{
	 "name":"Trip",
	 "category":"vacation",
	 "season":"summer",
	 "description":"d",
	 "privacy_level":"public",
	 "files_to_upload":[{"client_id":"c1","content_type":"image/jpeg"}]
	}`)
	require.Equal(t, http.StatusCreated, resp.StatusCode, body)
	var created struct {
		ID string `json:"id"`
		TripID string `json:"trip_id"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	tripID := created.ID
	if tripID == "" {
		tripID = created.TripID
	}
	require.NotEmpty(t, tripID)

	// ProcessMediaGrouping: save media metadata (S3 key stub; no actual upload in e2e)
	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/creation/"+tripID+"/media/process-grouping", login.AccessToken, `{
	 "media":[{"s3_key":"trips/`+tripID+`/c1.jpg","media_type":"image","captured_at":"2024-05-01T12:00:00Z","latitude":50.0,"longitude":87.0}]
	}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var groupingResp struct {
		DraftPins []struct {
			DraftPinID string `json:"draft_pin_id"`
			Media []struct {
				MediaID string `json:"media_id"`
			} `json:"media"`
		} `json:"draft_pins"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &groupingResp))
	require.NotEmpty(t, groupingResp.DraftPins)

	// ApplyGroupsAndProcess
	draftPinsJSON := `[{"draft_pin_id":"` + groupingResp.DraftPins[0].DraftPinID + `","media_ids":[`
	for i, m := range groupingResp.DraftPins[0].Media {
		if i > 0 {
			draftPinsJSON += ","
		}
		draftPinsJSON += `"` + m.MediaID + `"`
	}
	draftPinsJSON += `]}]`
	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/creation/"+tripID+"/apply-groups-and-process", login.AccessToken, `{"draft_pins":`+draftPinsJSON+`}`)
	require.Equal(t, http.StatusAccepted, resp.StatusCode, body)

	// FinalizeTrip (worker not running in e2e; FinalizeTrip accepts PROCESSING status)
	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/creation/"+tripID+"/finalize", login.AccessToken, `{}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)

	resp, body = st.doJSON(t, http.MethodPut, "/api/v1/trips/"+tripID+"/privacy", login.AccessToken, `{"privacy_level":"public"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)

	// PublishTrip: trip must be READY and published for like/dislike/favourite
	resp, body = st.doJSON(t, http.MethodPost, "/api/v1/trips/"+tripID+"/publish", login.AccessToken, `{"publish_whole":true}`)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)

	t.Run("like_returns_200", func(t *testing.T) {
		resp, _ := st.doJSON(t, http.MethodPost, "/api/v1/trips/"+tripID+"/like", login.AccessToken, `{}`)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("dislike_returns_200", func(t *testing.T) {
		resp, _ := st.doJSON(t, http.MethodPost, "/api/v1/trips/"+tripID+"/dislike", login.AccessToken, `{}`)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("favourite_add_returns_200", func(t *testing.T) {
		resp, _ := st.doJSON(t, http.MethodPost, "/api/v1/trips/"+tripID+"/favourite", login.AccessToken, `{}`)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("favourite_remove_returns_204", func(t *testing.T) {
		resp, _ := st.doJSON(t, http.MethodDelete, "/api/v1/trips/"+tripID+"/favourite", login.AccessToken, `{}`)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

// Тест /v1/ws удалён вместе с самим endpoint'ом — все WS теперь per-resource
// (creation/review/ws, media/add/review/ws, pins/creation/sessions/{sid}/ws,
// pins/{pin_id}/media/sessions/{sid}/ws).
