package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hackclub/geocoder/internal/models"
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

// GetCountryName converts country code to full country name
func (c *Client) GetCountryName(countryCode string) string {
	countryMap := map[string]string{
		"US": "United States",
		"CA": "Canada",
		"GB": "United Kingdom",
		"DE": "Germany",
		"FR": "France",
		"JP": "Japan",
		"AU": "Australia",
		"CN": "China",
		"IN": "India",
		"BR": "Brazil",
		"RU": "Russia",
		"IT": "Italy",
		"ES": "Spain",
		"KR": "South Korea",
		"NL": "Netherlands",
		"SE": "Sweden",
		"NO": "Norway",
		"DK": "Denmark",
		"FI": "Finland",
		"CH": "Switzerland",
		"AT": "Austria",
		"BE": "Belgium",
		"IE": "Ireland",
		"PT": "Portugal",
		"GR": "Greece",
		"PL": "Poland",
		"CZ": "Czech Republic",
		"HU": "Hungary",
		"SK": "Slovakia",
		"SI": "Slovenia",
		"HR": "Croatia",
		"BG": "Bulgaria",
		"RO": "Romania",
		"LT": "Lithuania",
		"LV": "Latvia",
		"EE": "Estonia",
		"LU": "Luxembourg",
		"MT": "Malta",
		"CY": "Cyprus",
		"IS": "Iceland",
		"LI": "Liechtenstein",
		"MC": "Monaco",
		"SM": "San Marino",
		"VA": "Vatican City",
		"AD": "Andorra",
		"MX": "Mexico",
		"AR": "Argentina",
		"CL": "Chile",
		"CO": "Colombia",
		"PE": "Peru",
		"VE": "Venezuela",
		"UY": "Uruguay",
		"PY": "Paraguay",
		"BO": "Bolivia",
		"EC": "Ecuador",
		"GY": "Guyana",
		"SR": "Suriname",
		"FK": "Falkland Islands",
		"ZA": "South Africa",
		"EG": "Egypt",
		"NG": "Nigeria",
		"KE": "Kenya",
		"MA": "Morocco",
		"GH": "Ghana",
		"TN": "Tunisia",
		"DZ": "Algeria",
		"LY": "Libya",
		"SD": "Sudan",
		"ET": "Ethiopia",
		"UG": "Uganda",
		"TZ": "Tanzania",
		"MZ": "Mozambique",
		"MG": "Madagascar",
		"ZW": "Zimbabwe",
		"BW": "Botswana",
		"NA": "Namibia",
		"ZM": "Zambia",
		"MW": "Malawi",
		"SZ": "Eswatini",
		"LS": "Lesotho",
		"ID": "Indonesia",
		"MY": "Malaysia",
		"TH": "Thailand",
		"VN": "Vietnam",
		"PH": "Philippines",
		"SG": "Singapore",
		"MM": "Myanmar",
		"KH": "Cambodia",
		"LA": "Laos",
		"BN": "Brunei",
		"TL": "East Timor",
		"IL": "Israel",
		"TR": "Turkey",
		"SA": "Saudi Arabia",
		"AE": "United Arab Emirates",
		"QA": "Qatar",
		"KW": "Kuwait",
		"BH": "Bahrain",
		"OM": "Oman",
		"JO": "Jordan",
		"LB": "Lebanon",
		"SY": "Syria",
		"IQ": "Iraq",
		"IR": "Iran",
		"AF": "Afghanistan",
		"PK": "Pakistan",
		"BD": "Bangladesh",
		"LK": "Sri Lanka",
		"MV": "Maldives",
		"NP": "Nepal",
		"BT": "Bhutan",
		"NZ": "New Zealand",
		"FJ": "Fiji",
		"PG": "Papua New Guinea",
		"SB": "Solomon Islands",
		"VU": "Vanuatu",
		"NC": "New Caledonia",
		"PF": "French Polynesia",
		"WS": "Samoa",
		"KI": "Kiribati",
		"TO": "Tonga",
		"FM": "Micronesia",
		"MH": "Marshall Islands",
		"PW": "Palau",
		"NR": "Nauru",
		"TV": "Tuvalu",
	}
	
	if name, exists := countryMap[countryCode]; exists {
		return name
	}
	return countryCode // Fallback to country code if not found
}

// GetIPInfoToStandardFormat converts IPinfo response to our standard format
func (c *Client) GetIPInfoToStandardFormat(ip string) (*models.GeoIPAPIResponse, error) {
	ipinfoResp, err := c.GetIPInfo(ip)
	if err != nil {
		return nil, err
	}

	// Parse lat,lng from the "loc" field
	var lat, lng float64
	if ipinfoResp.Loc != "" {
		coords := strings.Split(ipinfoResp.Loc, ",")
		if len(coords) == 2 {
			lat, _ = strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
			lng, _ = strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)
		}
	}

	response := &models.GeoIPAPIResponse{
		Lat:                lat,
		Lng:                lng,
		IP:                 ipinfoResp.IP,
		City:               ipinfoResp.City,
		Region:             ipinfoResp.Region,
		CountryName:        c.GetCountryName(ipinfoResp.Country),
		CountryCode:        ipinfoResp.Country,
		PostalCode:         ipinfoResp.Postal,
		Timezone:           ipinfoResp.Timezone,
		Org:                ipinfoResp.Org,
		Backend:            "ipinfo_api",
		RawBackendResponse: ipinfoResp,
	}

	return response, nil
}
