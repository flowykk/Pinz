package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/internal/mocks"
	"pinz/backend/api-gateway-service/internal/responses"
	"pinz/backend/api-gateway-service/pkg/proto"
)

// reqWithTripID attaches chi route context with {id} = tripID so handlers using chi.URLParam work under httptest.
func reqWithTripID(r *http.Request, tripID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", tripID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDContextKey, userID)
}

func TestTripHandler_TripProtoToResponse_ShareURL(t *testing.T) {
	t.Run("with_base_appends_id", func(t *testing.T) {
		h := NewTripHandler(nil, nil, "https://pinz.website/trips")
		out := h.tripProtoToResponse(&proto.Trip{Id: "abc-123", Name: "X"})
		require.Equal(t, "https://pinz.website/trips/abc-123", out.ShareURL)
	})
	t.Run("empty_base_no_share_url", func(t *testing.T) {
		h := NewTripHandler(nil, nil, "")
		out := h.tripProtoToResponse(&proto.Trip{Id: "abc-123", Name: "X"})
		require.Empty(t, out.ShareURL)
	})
	t.Run("empty_id_no_share_url", func(t *testing.T) {
		h := NewTripHandler(nil, nil, "https://pinz.website/trips")
		out := h.tripProtoToResponse(&proto.Trip{Id: ""})
		require.Empty(t, out.ShareURL)
	})
}

func TestTripHandler_ListTrips_WithoutJWT(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	// No ListUserTrips call expected — handler returns 401 before calling client

	h := NewTripHandler(tripClient, nil, "")
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
			Limit: 20,
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

	h := NewTripHandler(tripClient, nil, "")
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

	h := NewTripHandler(tripClient, nil, "")
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

	h := NewTripHandler(tripClient, nil, "")
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

	h := NewTripHandler(tripClient, nil, "")
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

	h := NewTripHandler(tripClient, nil, "")
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
			OwnerUserId: "user-1",
			Name: "New Trip",
			Description: "desc",
			Category: "Отпуск",
			Season: "Лето",
			FilesToUpload: []*proto.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
		}).
		Return(&proto.CreateTripResponse{
			TripId: "trip-new",
			Status: "UPLOADING",
			UploadUrls: []*proto.UploadUrl{
				{ClientId: "c1", S3Key: "key1", Url: "https://s3.example.com/presigned"},
			},
		}, nil)

	h := NewTripHandler(tripClient, nil, "")
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

func TestTripHandler_RequestTripCoverUpload_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		RequestTripCoverUpload(gomock.Any(), &proto.RequestTripCoverUploadRequest{
			TripId: "trip-1",
			UserId: "user-1",
			Filename: "cover.jpg",
			ContentType: "image/jpeg",
		}).
		Return(&proto.RequestTripCoverUploadResponse{
			UploadUrl: "https://s3.example.com/put?sig=1",
			S3Key: "trips/trip-1/cover/abc.jpg",
		}, nil)

	h := NewTripHandler(tripClient, nil, "")
	body := `{"filename":"cover.jpg","content_type":"image/jpeg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/cover/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithTripID(req.WithContext(ctxWithUserID("user-1")), "trip-1")
	rr := httptest.NewRecorder()

	h.RequestTripCoverUpload(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp responses.TripCoverUploadResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "https://s3.example.com/put?sig=1", resp.UploadURL)
	require.Equal(t, "trips/trip-1/cover/abc.jpg", resp.S3Key)
}

func TestTripHandler_RequestTripCoverUpload_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		RequestTripCoverUpload(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.PermissionDenied, "not a participant"))

	h := NewTripHandler(tripClient, nil, "")
	body := `{"filename":"cover.jpg","content_type":"image/jpeg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/cover/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithTripID(req.WithContext(ctxWithUserID("user-1")), "trip-1")
	rr := httptest.NewRecorder()

	h.RequestTripCoverUpload(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTripHandler_ConfirmTripCoverUpload_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		ConfirmTripCoverUpload(gomock.Any(), &proto.ConfirmTripCoverUploadRequest{
			TripId: "trip-1",
			UserId: "user-1",
			S3Key: "trips/trip-1/cover/abc.jpg",
		}).
		Return(&proto.ConfirmTripCoverUploadResponse{
			Trip: &proto.Trip{Id: "trip-1", CoverUrl: "https://s3/get?sig=1"},
		}, nil)

	h := NewTripHandler(tripClient, nil, "")
	body := `{"s3_key":"trips/trip-1/cover/abc.jpg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/cover/confirm", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithTripID(req.WithContext(ctxWithUserID("user-1")), "trip-1")
	rr := httptest.NewRecorder()

	h.ConfirmTripCoverUpload(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp responses.Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "trip-1", resp.ID)
	require.Equal(t, "https://s3/get?sig=1", resp.CoverURL)
}

func TestTripHandler_DeleteTripCover_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		DeleteTripCover(gomock.Any(), &proto.DeleteTripCoverRequest{TripId: "trip-1", UserId: "user-1"}).
		Return(&proto.DeleteTripCoverResponse{Trip: &proto.Trip{Id: "trip-1"}}, nil)

	h := NewTripHandler(tripClient, nil, "")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/trips/trip-1/cover", nil)
	req = reqWithTripID(req.WithContext(ctxWithUserID("user-1")), "trip-1")
	rr := httptest.NewRecorder()

	h.DeleteTripCover(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp responses.Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "trip-1", resp.ID)
	require.Empty(t, resp.CoverURL)
}

func TestTripHandler_SearchPins_WithoutJWT(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)

	h := NewTripHandler(tripClient, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pins/search?q=cafe", nil)
	rr := httptest.NewRecorder()

	h.SearchPins(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestTripHandler_SearchPins_MissingQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)

	h := NewTripHandler(tripClient, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pins/search", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.SearchPins(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "q is required")
}

func TestTripHandler_SearchPins_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	lat, lon := 55.75, 37.62
	tripClient.EXPECT().
		SearchPins(gomock.Any(), &proto.SearchPinsRequest{Query: "cafe", Limit: 50, Offset: 10}).
		Return(&proto.SearchPinsResponse{
			Pins: []*proto.TripPin{
				{
					Id: "pin-1", TripId: "trip-1", Name: "Cafe Central",
					Description: "best cafe", Category: "Food",
					Latitude: &lat, Longitude: &lon,
					PrivacyLevel: "Public", Tags: []string{"cafe", "coffee"},
					Media: []*proto.TripPinMedia{
						{MediaId: "m1", Url: "https://s3/m1", MediaType: "photo", PrivacyLevel: "Public"},
					},
				},
			},
		}, nil)

	h := NewTripHandler(tripClient, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pins/search?q=cafe&limit=50&offset=10", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.SearchPins(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var pins []responses.TripPin
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &pins))
	require.Len(t, pins, 1)
	require.Equal(t, "pin-1", pins[0].ID)
	require.Equal(t, "trip-1", pins[0].TripID)
	require.Equal(t, "Cafe Central", pins[0].Name)
	require.Equal(t, []string{"cafe", "coffee"}, pins[0].Tags)
	require.Len(t, pins[0].Media, 1)
	require.Equal(t, "m1", pins[0].Media[0].MediaID)
}

func TestTripHandler_SearchPins_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripClient := mocks.NewMockTripClient(ctrl)
	tripClient.EXPECT().
		SearchPins(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "boom"))

	h := NewTripHandler(tripClient, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pins/search?q=cafe", nil)
	req = req.WithContext(ctxWithUserID("user-1"))
	rr := httptest.NewRecorder()

	h.SearchPins(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}
