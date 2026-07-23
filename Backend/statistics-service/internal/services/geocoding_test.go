package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelFromErr(t *testing.T) {
	cases := map[string]struct {
		err  error
		name string
		want string
	}{
		"err_present": {errors.New("boom"), "moscow", "error"},
		"empty_name":  {nil, "", "empty"},
		"resolved":    {nil, "moscow", "resolved"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, labelFromErr(tc.err, tc.name))
		})
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *GeocodingClient {
	t.Helper()
	return &GeocodingClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		language:   "en",
	}
}

func TestResolveLocation_OkFullResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "en", r.URL.Query().Get("localityLanguage"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"countryName":"Russia","city":"Moscow"}`))
	}))
	defer srv.Close()

	country, city, name, err := newTestClient(t, srv).ResolveLocation(context.Background(), 55.75, 37.62)
	require.NoError(t, err)
	require.Equal(t, "Russia", country)
	require.Equal(t, "Moscow", city)
	require.Equal(t, "Russia, Moscow", name)
}

func TestResolveLocation_FallsBackToLocalityThenSubdivision(t *testing.T) {
	t.Run("locality used when city empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"countryName":"X","locality":"Loc"}`))
		}))
		defer srv.Close()
		_, city, name, err := newTestClient(t, srv).ResolveLocation(context.Background(), 0, 0)
		require.NoError(t, err)
		require.Equal(t, "Loc", city)
		require.Equal(t, "X, Loc", name)
	})
	t.Run("subdivision used when city/locality empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"principalSubdivision":"Region"}`))
		}))
		defer srv.Close()
		_, city, name, err := newTestClient(t, srv).ResolveLocation(context.Background(), 0, 0)
		require.NoError(t, err)
		require.Equal(t, "Region", city)
		require.Equal(t, "Region", name)
	})
	t.Run("only country", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"countryName":"OnlyCountry"}`))
		}))
		defer srv.Close()
		_, _, name, err := newTestClient(t, srv).ResolveLocation(context.Background(), 0, 0)
		require.NoError(t, err)
		require.Equal(t, "OnlyCountry", name)
	})
}

func TestResolveLocation_ErrorPaths(t *testing.T) {
	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()
		_, _, _, err := newTestClient(t, srv).ResolveLocation(context.Background(), 0, 0)
		require.Error(t, err)
	})
	t.Run("invalid json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer srv.Close()
		_, _, _, err := newTestClient(t, srv).ResolveLocation(context.Background(), 0, 0)
		require.Error(t, err)
	})
	t.Run("nil receiver short-circuits", func(t *testing.T) {
		var c *GeocodingClient
		country, city, name, err := c.ResolveLocation(context.Background(), 0, 0)
		require.NoError(t, err)
		require.Empty(t, country)
		require.Empty(t, city)
		require.Empty(t, name)
	})
}

func TestResolveLocation_ApiKeyForwarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret", r.URL.Query().Get("key"))
		_, _ = w.Write([]byte(`{"countryName":"X","city":"Y"}`))
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	c.apiKey = "secret"
	_, _, _, err := c.ResolveLocation(context.Background(), 0, 0)
	require.NoError(t, err)
}

func TestNewGeocodingClientFromEnv_DefaultsAndOverride(t *testing.T) {
	t.Run("default base url when env unset", func(t *testing.T) {
		t.Setenv("GEOCODING_BASE_URL", "")
		t.Setenv("GEOCODING_API_KEY", "")
		c := NewGeocodingClientFromEnv()
		require.NotNil(t, c)
		require.Contains(t, c.baseURL, "bigdatacloud")
		require.Empty(t, c.apiKey)
	})
	t.Run("respects env override", func(t *testing.T) {
		t.Setenv("GEOCODING_BASE_URL", "https://example.test/api")
		t.Setenv("GEOCODING_API_KEY", "xxx")
		c := NewGeocodingClientFromEnv()
		require.Equal(t, "https://example.test/api", c.baseURL)
		require.Equal(t, "xxx", c.apiKey)
	})
}
