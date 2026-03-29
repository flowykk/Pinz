package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/mocks"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDContextKey, userID)
}

func TestTripHandler_ListTrips_WithoutJWT(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	// No ListUserTrips call expected — handler returns 401 before calling client

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	rr := httptest.NewRecorder()

	h.ListTrips(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Contains(t, rr.Body.String(), "unauthorized")
}

func TestTripHandler_ListTrips_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		ListUserTrips(gomock.Any(), &proto.ListUserTripsRequest{
			UserId: "user-1",
			Limit:  20,
			Offset: 0,
		}).
		Return(&proto.ListUserTripsResponse{
			Trips: []*proto.Trip{
				{
					Id: "trip-1", OwnerUserId: "user-1", Name: "My Trip",
					Category: "Отпуск", Season: "Лето", Status: "READY",
					PrivacyLevel: "Private", LikesCount: 0, DislikesCount: 0,
					CreatedAtUnix: 1000, UpdatedAtUnix: 1000,
				},
			},
		}, nil)

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.ListTrips(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var trips []responses.Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trips))
	require.Len(t, trips, 1)
	require.Equal(t, "trip-1", trips[0].ID)
	require.Equal(t, "My Trip", trips[0].Name)
}

func TestTripHandler_ListTrips_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		ListUserTrips(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "db error"))

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.ListTrips(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), "db error")
}

func TestTripHandler_ListFavourites_WithoutJWT(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/favourites", nil)
	rr := httptest.NewRecorder()

	h.ListFavourites(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Contains(t, rr.Body.String(), "unauthorized")
}

func TestTripHandler_ListFavourites_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		ListFavourites(gomock.Any(), &proto.ListFavouritesRequest{Limit: 20, Offset: 0}).
		Return(&proto.ListFavouritesResponse{
			Trips: []*proto.Trip{
				{
					Id: "trip-fav", OwnerUserId: "other", Name: "Favourite Trip",
					Category: "Отпуск", Season: "Лето", Status: "READY",
					PrivacyLevel: "Private", LikesCount: 0, DislikesCount: 0,
					CreatedAtUnix: 1000, UpdatedAtUnix: 1000,
				},
			},
		}, nil)

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/favourites", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.ListFavourites(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var trips []responses.Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trips))
	require.Len(t, trips, 1)
	require.Equal(t, "trip-fav", trips[0].ID)
	require.Equal(t, "Favourite Trip", trips[0].Name)
}

func TestTripHandler_ListFavourites_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		ListFavourites(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "db error"))

	h := NewTripHandler(tripClient)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/favourites", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.ListFavourites(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), "db error")
}

func TestTripHandler_CreateTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		CreateTrip(gomock.Any(), &proto.CreateTripRequest{
			OwnerUserId:   "user-1",
			Name:          "New Trip",
			Description:   "desc",
			Category:      "Отпуск",
			Season:        "Лето",
			FilesToUpload: []*proto.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
		}).
		Return(&proto.CreateTripResponse{
			TripId: "trip-new",
			Status: "UPLOADING",
			UploadUrls: []*proto.UploadUrl{
				{ClientId: "c1", S3Key: "key1", Url: "https://s3.example.com/presigned"},
			},
		}, nil)

	h := NewTripHandler(tripClient)
	body := `{"name":"New Trip","description":"desc","category":"Отпуск","season":"Лето","privacy_level":"Private","files_to_upload":[{"client_id":"c1","content_type":"image/jpeg"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/creation/start", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.CreateTrip(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp responses.CreateTripResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "trip-new", resp.TripID)
	require.Equal(t, "UPLOADING", resp.Status)
	require.Len(t, resp.UploadURLs, 1)
	require.Equal(t, "c1", resp.UploadURLs[0].ClientID)
	require.Equal(t, "https://s3.example.com/presigned", resp.UploadURLs[0].URL)
}
