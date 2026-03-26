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

type GeocodingClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

type geocodingResponse struct {
	CountryName string `json:"countryName"`
	City        string `json:"city"`
	// Fallback поля на случай другого формата ответа
	Locality    string `json:"locality"`
	Principal   string `json:"principalSubdivision"`
	DisplayName string `json:"displayName"`
}

func NewGeocodingClientFromEnv() *GeocodingClient {
	baseURL := os.Getenv("GEOCODING_BASE_URL")
	apiKey := os.Getenv("GEOCODING_API_KEY")
	if baseURL == "" {
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
	}
}

// ResolveLocation выполняет reverse geocoding. Ошибки считаются некритичными: при неуспехе возвращаем пустые значения.
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
	// BigDataCloud free эндпоинт не требует ключа, но оставляем на будущее.
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
	name := body.DisplayName
	if name == "" {
		switch {
		case country != "" && city != "":
			name = country + ", " + city
		case city != "":
			name = city
		default:
			name = country
		}
	}

	return country, city, name, nil
}
