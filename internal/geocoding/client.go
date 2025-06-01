package geocoding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type GeocodeResponse struct {
	Results []GeocodeResult `json:"results"`
	Status  string          `json:"status"`
}

type GeocodeResult struct {
	FormattedAddress  string             `json:"formatted_address"`
	Geometry          GeocodeGeometry    `json:"geometry"`
	PlaceID           string             `json:"place_id"`
	Types             []string           `json:"types"`
	AddressComponents []AddressComponent `json:"address_components"`
}

type GeocodeGeometry struct {
	Location     GeocodeLocation `json:"location"`
	LocationType string          `json:"location_type"`
	Viewport     GeocodeBounds   `json:"viewport"`
	Bounds       *GeocodeBounds  `json:"bounds,omitempty"`
}

type GeocodeLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type GeocodeBounds struct {
	Northeast GeocodeLocation `json:"northeast"`
	Southwest GeocodeLocation `json:"southwest"`
}

type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Geocode(address string) (*GeocodeResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Google Geocoding API key not configured")
	}

	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Set("address", address)
	params.Set("key", c.apiKey)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Google Geocoding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Geocoding API returned status %d", resp.StatusCode)
	}

	var geocodeResp GeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geocodeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if geocodeResp.Status != "OK" && geocodeResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google Geocoding API returned status: %s", geocodeResp.Status)
	}

	return &geocodeResp, nil
}

func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
