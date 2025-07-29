package geocoding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hackclub/geocoder/internal/models"
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

// GeocodeToStandardFormat converts a Google Geocoding API response to our standard format
func (c *Client) GeocodeToStandardFormat(address string) (*models.GeocodeAPIResponse, error) {
	googleResp, err := c.Geocode(address)
	if err != nil {
		return nil, err
	}

	if len(googleResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for address: %s", address)
	}

	// Use the first result
	result := googleResp.Results[0]

	// Extract country and state information from address components
	var countryName, countryCode, stateName, stateCode string
	for _, component := range result.AddressComponents {
		for _, componentType := range component.Types {
			if componentType == "country" {
				countryName = component.LongName
				countryCode = component.ShortName
			} else if componentType == "administrative_area_level_1" {
				stateName = component.LongName
				stateCode = component.ShortName
			}
		}
	}

	response := &models.GeocodeAPIResponse{
		Lat:                result.Geometry.Location.Lat,
		Lng:                result.Geometry.Location.Lng,
		FormattedAddress:   result.FormattedAddress,
		StateName:          stateName,
		StateCode:          stateCode,
		CountryName:        countryName,
		CountryCode:        countryCode,
		Backend:            "google_maps_platform_geocoding",
		RawBackendResponse: googleResp,
	}

	return response, nil
}

func (c *Client) ReverseGeocode(lat, lng float64) (*GeocodeResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Google Geocoding API key not configured")
	}

	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
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

// ReverseGeocodeToStandardFormat converts a Google Reverse Geocoding API response to our standard format
func (c *Client) ReverseGeocodeToStandardFormat(lat, lng float64) (*models.ReverseGeocodeAPIResponse, error) {
	googleResp, err := c.ReverseGeocode(lat, lng)
	if err != nil {
		return nil, err
	}

	if len(googleResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for coordinates: %f, %f", lat, lng)
	}

	// Use the first result
	result := googleResp.Results[0]

	// Extract address components
	var (
		countryName, countryCode string
		addressLine1, city, state, stateFull, postalCode string
		streetNumber, route string
	)

	for _, component := range result.AddressComponents {
		for _, componentType := range component.Types {
			switch componentType {
			case "street_number":
				streetNumber = component.LongName
			case "route":
				route = component.LongName
			case "locality":
				city = component.LongName
			case "administrative_area_level_1":
				state = component.ShortName      // e.g., "CA"
				stateFull = component.LongName   // e.g., "California"
			case "postal_code":
				postalCode = component.LongName
			case "country":
				countryName = component.LongName
				countryCode = component.ShortName
			}
		}
	}

	// Construct address line 1 from street number and route
	if streetNumber != "" && route != "" {
		addressLine1 = streetNumber + " " + route
	} else if route != "" {
		addressLine1 = route
	} else if streetNumber != "" {
		addressLine1 = streetNumber
	}

	response := &models.ReverseGeocodeAPIResponse{
		Lat:                lat,
		Lng:                lng,
		FormattedAddress:   result.FormattedAddress,
		AddressLine1:       addressLine1,
		City:               city,
		State:              state,
		StateFull:          stateFull,
		PostalCode:         postalCode,
		CountryName:        countryName,
		CountryCode:        countryCode,
		Backend:            "google_maps_platform_geocoding",
		RawBackendResponse: googleResp,
	}

	return response, nil
}
