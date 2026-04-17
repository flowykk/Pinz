package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"pinz/backend/api-gateway-service/internal/middleware"
	"pinz/backend/api-gateway-service/pkg/proto"
)

type fakeWSTripClient struct {
	resp *proto.GetTripResponse
	err  error
}

func (f *fakeWSTripClient) GetTrip(ctx context.Context, req *proto.GetTripRequest) (*proto.GetTripResponse, error) {
	return f.resp, f.err
}

func tripIDCtxRequest(path, userID, tripID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if userID != "" {
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
		req = req.WithContext(ctx)
	}
	if tripID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", tripID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return req
}

func TestServeTripCreationReviewWS_MissingTripIDReturns400(t *testing.T) {
	h := NewWSHandler(nil, nil)
	req := tripIDCtxRequest("/api/v1/trips/creation//review/ws", "user-1", "")
	rr := httptest.NewRecorder()

	h.ServeTripCreationReviewWS(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestServeTripAddMediaReviewWS_MissingTripIDReturns400(t *testing.T) {
	h := NewWSHandler(nil, nil)
	req := tripIDCtxRequest("/api/v1/trips//media/add/review/ws", "user-1", "")
	rr := httptest.NewRecorder()

	h.ServeTripAddMediaReviewWS(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestServeTripCreationReviewWS_NoRedisReturns503(t *testing.T) {
	h := NewWSHandler(nil, &fakeWSTripClient{resp: &proto.GetTripResponse{}})
	req := tripIDCtxRequest("/api/v1/trips/creation/abc/review/ws", "user-1", "abc")
	rr := httptest.NewRecorder()

	h.ServeTripCreationReviewWS(rr, req)

	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestTripAccessError_Mapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"permission denied → 403", grpcstatus.Error(codes.PermissionDenied, "x"), http.StatusForbidden},
		{"not found → 404", grpcstatus.Error(codes.NotFound, "x"), http.StatusNotFound},
		{"unauthenticated → 401", grpcstatus.Error(codes.Unauthenticated, "x"), http.StatusUnauthorized},
		{"invalid argument → 400", grpcstatus.Error(codes.InvalidArgument, "bad"), http.StatusBadRequest},
		{"unknown gRPC code → 502", grpcstatus.Error(codes.Internal, "x"), http.StatusBadGateway},
		{"non-grpc error → 500", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := tripAccessError(tc.err)
			require.Equal(t, tc.code, code)
		})
	}
}
