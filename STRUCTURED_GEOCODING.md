# Structured Geocoding API

The `/v1/geocode_structured` endpoint allows you to geocode addresses using structured components instead of a single address string.

## Endpoint

```
GET /v1/geocode_structured
```

## Query Parameters

- `key` (required): Your API key
- `address_line_1` (optional): Primary street address
- `address_line_2` (optional): Secondary address line (apartment, suite, etc.)
- `city` (optional): City name
- `state` (optional): State/province
- `postal_code` (optional): ZIP/postal code
- `country` (optional): Country name

**Note**: At least one address field must be provided, but all fields are optional.

## Response

Returns the same response format as the standard geocoding endpoint:

```json
{
  "lat": 37.7749,
  "lng": -122.4194,
  "formatted_address": "123 Main St, Apt 4B, San Francisco, CA 94102, USA",
  "country_name": "United States",
  "country_code": "US",
  "backend": "google_maps_platform_geocoding",
  "raw_backend_response": {...}
}
```

## Examples

### Full Address
```bash
curl "https://your-domain.com/v1/geocode_structured?address_line_1=1600+Amphitheatre+Parkway&city=Mountain+View&state=CA&postal_code=94043&country=USA&key=your_api_key"
```

### Partial Address
```bash
curl "https://your-domain.com/v1/geocode_structured?city=Paris&country=France&key=your_api_key"
```

### JavaScript Example
```javascript
const params = new URLSearchParams({
  address_line_1: '221B Baker Street',
  city: 'London',
  country: 'United Kingdom',
  key: 'your_api_key'
});

const response = await fetch(`/v1/geocode_structured?${params}`);
const result = await response.json();
console.log(`Coordinates: ${result.lat}, ${result.lng}`);
```

## Benefits

- **Better Data Quality**: Structured input provides more precise geocoding
- **Easier Integration**: No need to format addresses manually  
- **Same Features**: Uses the same caching, rate limiting, and cost tracking as the standard endpoint
- **Flexible**: Accept partial addresses with missing components

## Error Responses

- `400 Bad Request`: All address fields empty
- `401 Unauthorized`: Missing or invalid API key  
- `503 Service Unavailable`: Geocoding service not configured
- `502 Bad Gateway`: External geocoding API error
