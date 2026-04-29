package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// GeocodingClient — клиент BigDataCloud Reverse Geocoding API.
// Statistics-service единственный владелец интеграции (vkr.txt §2.4.6.3, §2.5.4):
// trip-service отправляет события PIN_LOCATIONS_REQUESTED, statistics резолвит
// координаты и возвращает результат событием PIN_LOCATIONS_RESOLVED.
type GeocodingClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	language   string
}

type geocodingResponse struct {
	CountryName          string `json:"countryName"`
	CountryCode          string `json:"countryCode"`
	PrincipalSubdivision string `json:"principalSubdivision"`
	City                 string `json:"city"`
	Locality             string `json:"locality"`
	Postcode             string `json:"postcode"`
	Continent            string `json:"continent"`
	ContinentCode        string `json:"continentCode"`
	LocalityLanguageReq  string `json:"localityLanguageRequested"`
}

func NewGeocodingClientFromEnv() *GeocodingClient {
	baseURL := os.Getenv("GEOCODING_BASE_URL")
	apiKey := os.Getenv("GEOCODING_API_KEY")
	if baseURL == "" {
		// Client-side endpoint бесплатный без API key (free tier).
		// Server-side /reverse-geocode требует ключ; используется только если
		// явно задан GEOCODING_BASE_URL + GEOCODING_API_KEY.
		baseURL = "https://api.bigdatacloud.net/data/reverse-geocode-client"
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
	return &GeocodingClient{
		httpClient: client,
		baseURL:    baseURL,
		apiKey:     apiKey,
		language:   "ru",
	}
}

// ResolveLocation возвращает страну, город и display name по координатам.
// Ошибки некритичны: вызывающий код логирует и идёт дальше.
func (c *GeocodingClient) ResolveLocation(ctx context.Context, lat, lon float64) (countryName, cityName, displayName string, err error) {
	if c == nil || c.httpClient == nil {
		return "", "", "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return "", "", "", err
	}

	q := req.URL.Query()
	q.Set("latitude", fmt.Sprintf("%f", lat))
	q.Set("longitude", fmt.Sprintf("%f", lon))
	q.Set("localityLanguage", c.language)
	if c.apiKey != "" {
		q.Set("key", c.apiKey)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("geocoding: status %d", resp.StatusCode)
	}

	var body geocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", "", err
	}

	country := body.CountryName
	city := body.City
	if city == "" {
		city = body.Locality
	}
	if city == "" {
		city = body.PrincipalSubdivision
	}

	var name string
	switch {
	case country != "" && city != "":
		name = country + ", " + city
	case city != "":
		name = city
	default:
		name = country
	}

	return country, city, name, nil
}

// LocationResolver описывает интерфейс geocoder для подстановки моков в тестах.
type LocationResolver interface {
	ResolveLocation(ctx context.Context, lat, lon float64) (countryName, cityName, displayName string, err error)
}
