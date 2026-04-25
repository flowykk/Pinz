package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDecodeJSONBody(t *testing.T) {
	cases := map[string]struct {
		contentType string
		body string
		wantErr bool
		wantSubstr string
		wantDst map[string]any
	}{
		"rejects_non_application_json": {
			contentType: "text/plain",
			body: `{"a":1}`,
			wantErr: true,
			wantSubstr: "application/json",
		},
		"accepts_application_json": {
			contentType: "application/json",
			body: `{"a":1}`,
			wantDst: map[string]any{"a": 1.0},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var dst map[string]any
			req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			err := decodeJSONBody(req, &dst)
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantSubstr != "" {
					require.Contains(t, err.Error(), tc.wantSubstr)
				}
				return
			}
			require.NoError(t, err)
			for k, v := range tc.wantDst {
				require.Equal(t, v, dst[k])
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	cases := map[string]struct {
		s string
		wantVal int
		wantErr bool
	}{
		"zero": {"0", 0, false},
		"positive": {"42", 42, false},
		"negative": {"-5", -5, false},
		"invalid": {"abc", 0, true},
		"empty": {"", 0, true},
		"with_space": {" 1", 0, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseInt(tc.s)
			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, 0, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantVal, got)
		})
	}
}

func TestHandleServiceError(t *testing.T) {
	cases := map[string]struct {
		code codes.Code
		wantHTTP int
		wantBody string
	}{
		"invalid_argument": {codes.InvalidArgument, http.StatusBadRequest, "boom"},
		"unauthenticated": {codes.Unauthenticated, http.StatusUnauthorized, "boom"},
		"permission_denied": {codes.PermissionDenied, http.StatusForbidden, "boom"},
		"not_found": {codes.NotFound, http.StatusNotFound, "boom"},
		"already_exists": {codes.AlreadyExists, http.StatusConflict, "boom"},
		"unavailable": {codes.Unavailable, http.StatusServiceUnavailable, "service unavailable"},
		"deadline_exceeded": {codes.DeadlineExceeded, http.StatusGatewayTimeout, "boom"},
		"unimplemented": {codes.Unimplemented, http.StatusNotImplemented, "boom"},
		"failed_precondition": {codes.FailedPrecondition, http.StatusConflict, "boom"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			handleServiceError(rr, req, status.Error(tc.code, "boom"), "x")
			require.Equal(t, tc.wantHTTP, rr.Code)
			require.Contains(t, rr.Body.String(), tc.wantBody)
		})
	}
}

func TestHandleServiceError_NonGRPC(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"plain_error_returns_500_internal_server_error": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			handleServiceError(rr, req, fmt.Errorf("plain error"), "x")
			require.Equal(t, http.StatusInternalServerError, rr.Code)
			require.Contains(t, rr.Body.String(), "internal server error")
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}

func TestHealthCheck(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"status_200": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			HealthCheck(rr, req)
			require.Equal(t, http.StatusOK, rr.Code)
		},
		"content_type_json": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			HealthCheck(rr, req)
			require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		},
		"body_has_status_service_timestamp": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			HealthCheck(rr, req)
			var body map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			require.Equal(t, "healthy", body["status"])
			require.Equal(t, "api-gateway", body["service"])
			require.NotEmpty(t, body["timestamp"])
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}

func TestAppleAppSiteAssociation(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"status_200": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/.well-known/apple-app-site-association", nil)
			AppleAppSiteAssociation(rr, req)
			require.Equal(t, http.StatusOK, rr.Code)
		},
		"content_type_json": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/.well-known/apple-app-site-association", nil)
			AppleAppSiteAssociation(rr, req)
			require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		},
		"body_contains_applinks_and_webcredentials": func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/.well-known/apple-app-site-association", nil)
			AppleAppSiteAssociation(rr, req)
			body := rr.Body.String()
			require.Contains(t, body, "applinks")
			require.Contains(t, body, "webcredentials")
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}
