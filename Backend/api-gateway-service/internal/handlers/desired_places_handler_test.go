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
	"pinz/backend/api-gateway-service/pkg/proto"
)

func authedReq(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(r.Context(), middleware.UserIDContextKey, "user-1")
	return r.WithContext(ctx)
}

func TestAuthHandler_ListDesiredPlaces_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().ListDesiredPlaces(gomock.Any(), "user-1").Return(&proto.ListDesiredPlacesResponse{
		Places: []*proto.DesiredPlace{
			{Id: "p1", Name: "Eiffel", Description: "tower", ImageUrl: "https://s3/eiffel", CreatedAtUnix: 100},
			{Id: "p2", Name: "Colosseum", Description: "arena", CreatedAtUnix: 200},
		},
	}, nil)

	h := NewAuthHandler(authSvc)
	rr := httptest.NewRecorder()
	h.ListDesiredPlaces(rr, authedReq(http.MethodGet, "/api/v1/profile/desired-places", ""))

	require.Equal(t, http.StatusOK, rr.Code)
	var dst struct {
		Places []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			ImageURL string `json:"image_url"`
		} `json:"places"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &dst))
	require.Len(t, dst.Places, 2)
	require.Equal(t, "Eiffel", dst.Places[0].Name)
	require.Equal(t, "https://s3/eiffel", dst.Places[0].ImageURL)
	require.Empty(t, dst.Places[1].ImageURL)
}

func TestAuthHandler_CreateDesiredPlace_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().CreateDesiredPlace(gomock.Any(), "user-1", "Eiffel", "Want to visit", "key.jpg").Return(
		&proto.CreateDesiredPlaceResponse{
			Place: &proto.DesiredPlace{Id: "p1", Name: "Eiffel", Description: "Want to visit", ImageUrl: "https://s3/g", CreatedAtUnix: 1},
		}, nil,
	)

	h := NewAuthHandler(authSvc)
	rr := httptest.NewRecorder()
	body := `{"name":"Eiffel","description":"Want to visit","s3_key":"key.jpg"}`
	h.CreateDesiredPlace(rr, authedReq(http.MethodPost, "/api/v1/profile/desired-places", body))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"id":"p1"`)
}

func TestAuthHandler_UpdateDesiredPlace_ImageChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().UpdateDesiredPlace(
		gomock.Any(), "user-1", "p1", "new", "new",
		true, "newkey.jpg",
	).Return(
		&proto.UpdateDesiredPlaceResponse{
			Place: &proto.DesiredPlace{Id: "p1", Name: "new", Description: "new"},
		}, nil,
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("place_id", "p1")
	body := `{"name":"new","description":"new","image_s3_key":"newkey.jpg"}`
	r := authedReq(http.MethodPatch, "/api/v1/profile/desired-places/p1", body)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.UpdateDesiredPlace(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthHandler_UpdateDesiredPlace_NoImageField_DoesNotTouchImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	// setImageKey=false, s3Key=""
	authSvc.EXPECT().UpdateDesiredPlace(
		gomock.Any(), "user-1", "p1", "new", "new",
		false, "",
	).Return(
		&proto.UpdateDesiredPlaceResponse{Place: &proto.DesiredPlace{Id: "p1"}}, nil,
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("place_id", "p1")
	body := `{"name":"new","description":"new"}`
	r := authedReq(http.MethodPatch, "/api/v1/profile/desired-places/p1", body)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.UpdateDesiredPlace(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthHandler_DeleteDesiredPlace(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().DeleteDesiredPlace(gomock.Any(), "user-1", "p1").Return(
		&proto.DeleteDesiredPlaceResponse{Success: true}, nil,
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("place_id", "p1")
	r := authedReq(http.MethodDelete, "/api/v1/profile/desired-places/p1", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.DeleteDesiredPlace(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"success":true`)
}

func TestAuthHandler_DeleteDesiredPlace_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().DeleteDesiredPlace(gomock.Any(), "user-1", "p1").Return(
		nil, status.Error(codes.NotFound, "desired place not found"),
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("place_id", "p1")
	r := authedReq(http.MethodDelete, "/api/v1/profile/desired-places/p1", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.DeleteDesiredPlace(rr, r)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAuthHandler_RequestDesiredPlaceImageUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().RequestDesiredPlaceImageUpload(gomock.Any(), "user-1", "x.jpg", "image/jpeg").Return(
		&proto.RequestDesiredPlaceImageUploadResponse{
			UploadUrl: "https://s3/put",
			S3Key:     "desired-places/user-1/abc.jpg",
		}, nil,
	)

	h := NewAuthHandler(authSvc)
	body := `{"filename":"x.jpg","content_type":"image/jpeg"}`
	rr := httptest.NewRecorder()
	h.RequestDesiredPlaceImageUpload(rr, authedReq(http.MethodPost, "/api/v1/profile/desired-places/upload-url", body))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"upload_url":"https://s3/put"`)
	require.Contains(t, rr.Body.String(), `"s3_key":"desired-places/user-1/abc.jpg"`)
}

func TestAuthHandler_GetPublicUserProfile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().GetPublicUserProfile(gomock.Any(), "other-user").Return(
		&proto.GetPublicUserProfileResponse{
			Profile: &proto.PublicUserProfile{
				UserId:        "other-user",
				Username:      "alice",
				AvatarUrl:     "https://s3/avatar",
				CreatedAtUnix: 42,
			},
			DesiredPlaces: []*proto.DesiredPlace{
				{Id: "p1", Name: "Eiffel", Description: "want", CreatedAtUnix: 100},
			},
		}, nil,
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "other-user")
	r := authedReq(http.MethodGet, "/api/v1/profile/other-user", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetPublicUserProfile(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `"id":"other-user"`)
	require.Contains(t, body, `"username":"alice"`)
	require.Contains(t, body, `"avatar_url":"https://s3/avatar"`)
	require.Contains(t, body, `"created_at":42`)
	require.Contains(t, body, `"desired_places"`)
	require.NotContains(t, body, "email", "email must not leak in public profile")
}

func TestAuthHandler_GetPublicUserProfile_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	authSvc.EXPECT().GetPublicUserProfile(gomock.Any(), "x").Return(
		nil, status.Error(codes.NotFound, "user not found"),
	)

	h := NewAuthHandler(authSvc)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x")
	r := authedReq(http.MethodGet, "/api/v1/profile/x", "")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetPublicUserProfile(rr, r)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAuthHandler_GetPublicUserProfile_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := mocks.NewMockAuthServiceInterface(ctrl)
	// no expectations — service must not be called
	_ = authSvc

	h := NewAuthHandler(authSvc)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/profile/x", nil)
	rr := httptest.NewRecorder()
	h.GetPublicUserProfile(rr, r)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
