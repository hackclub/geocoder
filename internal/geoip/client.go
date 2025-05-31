package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type IPInfoResponse struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"` // "lat,lng" format
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetIPInfo(ip string) (*IPInfoResponse, error) {
	var url string
	if c.apiKey != "" {
		url = fmt.Sprintf("https://ipinfo.io/%s?token=%s", ip, c.apiKey)
	} else {
		// Use free tier without API key (limited to 50k/month)
		url = fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to IPinfo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPinfo API returned status %d", resp.StatusCode)
	}

	var ipInfoResp IPInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipInfoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ipInfoResp, nil
}

func (c *Client) IsConfigured() bool {
	return true // IPinfo works without API key (with limits)
}
