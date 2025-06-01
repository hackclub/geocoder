# Hack Club Geocoder

A lightweight geocoding API service that centralizes and caches geocoding requests across Hack Club services.

## Overview

Geocoder provides a unified interface for geocoding operations while intelligently caching results to minimize external API costs and improve performance. It supports both address-to-coordinates geocoding and IP geolocation.

## Features

### Core Functionality
- **Address Geocoding**: Convert addresses to coordinates using Google Geocoding API with standardized response format
- **Address Normalization**: Extract structured components (country name/code, formatted address) from raw addresses
- **IP Geolocation**: Convert IP addresses to location data using IPinfo API with separated lat/lng coordinates
- **Standardized Responses**: Consistent JSON format across both geocoding and IP geolocation with country name expansion
- **Intelligent Caching**: Postgres-backed cache to reduce external API calls
- **Cost Tracking**: Monitor API usage and associated costs
- **API Key Management**: Issue and manage access keys for different services

### Admin Interface
- **Dashboard**: Overview of API usage, costs, and cache hit rates
- **API Key Management**: Create, view, and deactivate API keys
- **Usage Analytics**: Charts showing request patterns and trends
- **Cost Monitoring**: Track spending on Google Geocoding and IPinfo APIs
- **Real-time Map**: Live visualization of geocoding requests using WebSocket + Leaflet.js
- **Protected Access**: HTTP Basic Auth protection

## API Endpoints

### Address Geocoding & Normalization
```
GET /v1/geocode?address={address}&key={api_key}
```

**Input Parameters:**
- `address` (required): Raw, unstructured address string (e.g., "1600 Amphitheatre Parkway, Mountain View, CA")
- `key` (required): Your API key for authentication

**Response Format:**
Returns standardized JSON with extracted coordinates and country information:

```json
{
  "lat": 37.4223,
  "lng": -122.0844,
  "formatted_address": "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
  "country_name": "United States",
  "country_code": "US",
  "backend": "google_maps_platform_geocoding",
  "raw_backend_response": {
    "results": [...],
    "status": "OK"
  }
}
```

### IP Geolocation
```
GET /v1/geoip?ip={ip_address}&key={api_key}
```

**Input Parameters:**
- `ip` (required): IPv4 or IPv6 address (e.g., "8.8.8.8" or "2001:4860:4860::8888")
- `key` (required): Your API key for authentication

**Response Format:**
Returns standardized JSON with separated coordinates and expanded country information:

```json
{
  "lat": 37.4056,
  "lng": -122.0775,
  "ip": "8.8.8.8",
  "city": "Mountain View",
  "region": "California",
  "country_name": "United States",
  "country_code": "US",
  "postal_code": "94043",
  "timezone": "America/Los_Angeles",
  "org": "AS15169 Google LLC",
  "backend": "ipinfo_api",
  "raw_backend_response": {
    "ip": "8.8.8.8",
    "city": "Mountain View",
    "region": "California",
    "country": "US",
    "loc": "37.4056,-122.0775",
    "org": "AS15169 Google LLC",
    "postal": "94043",
    "timezone": "America/Los_Angeles"
  }
}
```

**Rate Limit Headers:**
All API responses include: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

### Health Check
```
GET /health
```
Returns service status for both monitoring and rolling deployments.

**Status Codes:**
- `healthy` (200): All systems operational
- `degraded` (200): One external API failing but still serving traffic  
- `unhealthy` (503): Critical failure - remove from load balancer

### Admin API Endpoints
```
GET /admin/keys                    - List all API keys
POST /admin/keys                   - Create new API key  
PUT /admin/keys/{key_id}/rate-limit - Update API key rate limit
DELETE /admin/keys/{key_id}        - Deactivate API key
GET /admin/stats                   - Usage statistics
GET /admin/costs                   - Cost breakdown
GET /admin/dashboard               - Admin web interface
```

### System Endpoints (No Versioning)
```
GET /health                        - Service health check & readiness
GET /metrics                       - Prometheus metrics (requires auth)
```

**Create API Key Request:**
```json
{
  "name": "hackclub-site",
  "prefix": "hcs",
  "rate_limit_per_second": 10
}
```

**Create API Key Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "hackclub-site", 
  "key": "hcs_live_sk_1a2b3c4d5e6f7g8h9i0j",
  "rate_limit_per_second": 10,
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Update API Key Rate Limit:**
```
PUT /admin/keys/{key_id}/rate-limit
```
```json
{
  "rate_limit_per_second": 25
}
```

### Error Responses
All endpoints return standard error format:

```json
{
  "error": {
    "code": "INVALID_API_KEY",
    "message": "The provided API key is invalid or expired",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

**Error Codes & HTTP Status:**
- `INVALID_API_KEY` (401): API key missing, invalid, or expired
- `RATE_LIMIT_EXCEEDED` (429): Too many requests (configurable per API key, sliding window)
- `INVALID_ADDRESS` (400): Address parameter missing or malformed
- `INVALID_IP` (400): IP parameter missing or malformed
- `EXTERNAL_API_ERROR` (502): Upstream API (Google/IPinfo) error
- `CACHE_ERROR` (500): Database/cache system error
- `UNSUPPORTED_VERSION` (404): API version not supported
- `EXTERNAL_RATE_LIMIT` (503): IPinfo free tier limit exceeded (50k/month)

## API Versioning

All public API endpoints are versioned with a `/v1/` prefix to ensure backward compatibility:

- **Current Version**: `v1` (stable)
- **Versioning Strategy**: URL path versioning for future compatibility
- **Version Detection**: Automatic routing based on URL path
- **Default Behavior**: Requests to unversioned endpoints (e.g., `/geocode`) return `UNSUPPORTED_VERSION` error
- **Internal Endpoints**: Admin and system endpoints (`/admin/*`, `/health`, `/metrics`) are not versioned

## Tech Stack

- **Language**: Go
- **Database**: PostgreSQL (for caching and analytics)
- **Deployment**: Docker Compose
- **External APIs**: 
  - Google Geocoding API
  - IPinfo API

## Development Setup

### Prerequisites
- Docker and Docker Compose
- Go 1.21+

### Quick Start
```bash
# Clone the repository
git clone https://github.com/hackclub/geocoder.git
cd geocoder

# Start the development environment
docker-compose up -d

# Run the application (migrations run automatically)
go run cmd/server/main.go
```

### Development Commands
```bash
# Run tests
go test ./...

# Run with live reload
air

# Build for production
go build -o bin/geocoder cmd/server/main.go

# Run database migrations
go run cmd/migrate/main.go up

# Generate new API key
go run cmd/keygen/main.go --name "test-key"

# Lint code
golangci-lint run

# Format code
go fmt ./...
```

### Environment Variables
```env
GOOGLE_GEOCODING_API_KEY=your_google_api_key
IPINFO_API_KEY=your_ipinfo_api_key
DATABASE_URL=postgres://user:pass@localhost:5432/geocoder
PORT=8080
ADMIN_USERNAME=admin
ADMIN_PASSWORD=secure_password

# Configuration
MAX_ADDRESS_CACHE_SIZE=10000
MAX_IP_CACHE_SIZE=5000
DEFAULT_RATE_LIMIT_PER_SECOND=10
LOG_LEVEL=info
```

## Configuration

### Database Schema

**Cache Tables:**
```sql
-- Address geocoding cache
CREATE TABLE address_cache (
  id SERIAL PRIMARY KEY,
  query_hash VARCHAR(64) UNIQUE NOT NULL,     -- SHA-256 of normalized query
  query_text TEXT NOT NULL,                   -- Original query for debugging
  response_data JSONB NOT NULL,               -- Standardized response with raw_backend_response
  created_at TIMESTAMP DEFAULT NOW(),
  INDEX(query_hash), INDEX(created_at)        -- FIFO ordering
);

-- IP geolocation cache  
CREATE TABLE ip_cache (
  id SERIAL PRIMARY KEY,
  ip_address INET UNIQUE NOT NULL,            -- Native IP type for efficiency
  response_data JSONB NOT NULL,               -- Standardized response with raw_backend_response
  created_at TIMESTAMP DEFAULT NOW(),
  INDEX(ip_address), INDEX(created_at)        -- FIFO ordering
);
```

**Management Tables:**
```sql
-- API key management
CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key_hash VARCHAR(64) UNIQUE NOT NULL,       -- SHA-256 of actual key
  name VARCHAR(255) NOT NULL,                 -- Human-readable name
  is_active BOOLEAN DEFAULT true,
  rate_limit_per_second INTEGER DEFAULT 10,   -- Configurable rate limit
  created_at TIMESTAMP DEFAULT NOW(),
  last_used_at TIMESTAMP,
  request_count INTEGER DEFAULT 0
);

-- Usage tracking (minimal for storage efficiency)
CREATE TABLE usage_logs (
  id BIGSERIAL PRIMARY KEY,
  api_key_id UUID REFERENCES api_keys(id),
  endpoint VARCHAR(20) NOT NULL,              -- 'v1/geocode' or 'v1/geoip'
  cache_hit BOOLEAN NOT NULL,
  response_time_ms INTEGER,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Cost tracking aggregates
CREATE TABLE cost_tracking (
  date DATE PRIMARY KEY,
  geocode_requests INTEGER DEFAULT 0,
  geocode_cache_hits INTEGER DEFAULT 0,
  geoip_requests INTEGER DEFAULT 0,
  geoip_cache_hits INTEGER DEFAULT 0,
  estimated_cost_usd DECIMAL(10,4) DEFAULT 0
);
```

**Query Normalization:**
- Lowercase, trim whitespace, collapse multiple spaces
- Store SHA-256 hash for deduplication

### Cache Strategy
- **FIFO Eviction**: When cache hits size limit, delete oldest 10% of entries based on created_at timestamp
- **Configurable Limits**: Separate maximum cache sizes for address (10k) and IP (5k) geocoding
- **Standardized Format**: Caches the transformed standardized responses (not raw external API responses)
- **Exact Query Matching**: Results cached by SHA-256 hash of normalized query string
- **Cache Hit Logic**: Query hash exists in cache table
- **Cache Miss Logic**: Query hash not found, requires external API call and transformation
- **Eviction Trigger**: Synchronous eviction on INSERT when count >= max_size

## Admin Dashboard

Access the admin interface at `/admin` with HTTP Basic Auth. Features include:

- **API Keys**: View active keys, usage statistics, issue new keys with custom prefixes, and configure rate limits
- **Usage Charts**: Request volume, cache hit rates, and cost trends
- **Real-time Map**: Live WebSocket-powered map showing geocoding requests with different colors for cache hits/misses
- **Cost Analysis**: Breakdown of external API costs vs cache savings

### Real-time Map Implementation

The admin dashboard includes a live map visualization using:

- **WebSocket Connection**: `/admin/ws` endpoint for real-time updates (uses HTTP Basic Auth upgrade)
- **Leaflet.js**: Lightweight, open-source mapping library
- **OpenStreetMap**: Free map tiles (no API key required)
- **Marker System**: 
  - 🟢 Green markers for cache hits
  - 🔴 Red markers for cache misses (external API calls)
  - 🔵 Blue markers for IP geolocation requests
- **Animation**: Markers fade out after 30 seconds
- **Clustering**: Marker clustering for high-density areas

**WebSocket Message Format:**
```json
{
  "type": "geocode_request",
  "lat": 37.4224764,
  "lng": -122.0842499,
  "cache_hit": true,
  "endpoint": "v1/geocode",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Deployment

### Production Dockerfile

Create this `Dockerfile` in your project root for Coolify deployment:

```dockerfile
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o geocoder cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /root/
COPY --from=builder /app/geocoder .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/web ./web
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -q --spider http://localhost:8080/health || exit 1
CMD ["./geocoder"]
```

Coolify will automatically detect and build this Dockerfile. Configure environment variables in the Coolify dashboard.

### Scaling Considerations
- Stateless application design for horizontal scaling
- Database connection pooling
- Redis integration for distributed caching (future enhancement)

## API Cost Management

The service tracks costs for:
- **Google Geocoding API**: $0.005 per request
- **IPinfo API**: $0.001 per request (estimated)

Cache hit rates typically achieve 70-80% cost savings on repeated queries.

## Project Structure

```
geocoder/
├── cmd/
│   ├── server/main.go              # Main application entry point
│   ├── migrate/main.go             # Database migration tool
│   └── keygen/main.go              # API key generator
├── internal/
│   ├── api/                        # HTTP handlers and routes
│   ├── cache/                      # Cache management logic
│   ├── config/                     # Configuration management
│   ├── database/                   # Database connection and queries
│   ├── geocoding/                  # Google Geocoding API client
│   ├── geoip/                      # IPinfo API client
│   ├── middleware/                 # HTTP middleware (auth, rate limiting)
│   └── models/                     # Data structures
├── migrations/                     # SQL migration files
├── web/                            # Admin dashboard frontend
├── docker-compose.yml              # Development environment
├── Dockerfile                      # Production multi-stage build
└── .env.example                    # Environment variable template
```

## Security Considerations

- **API Key Storage**: Keys stored as SHA-256 hashes, never in plaintext
- **Rate Limiting**: Per-key rate limiting to prevent abuse
- **Input Validation**: Strict validation of all input parameters
- **SQL Injection**: All queries use parameterized statements
- **CORS**: Configurable CORS policy for web client access
- **Admin Access**: HTTP Basic Auth for admin interface
- **Request Size Limits**: Prevent large request DoS attacks
- **Timeout Protection**: Request timeouts to prevent resource exhaustion

## Monitoring

- **Health Check**: `/health` endpoint
- **Metrics**: `/metrics` endpoint (requires auth) 
- **Logging**: Structured JSON logs

## Implementation Details

### API Key Generation
```go
// Format: {prefix}_live_sk_{random}
// Example: hcs_live_sk_1a2b3c4d5e6f7g8h9i0j
key := fmt.Sprintf("%s_live_sk_%s", prefix, generateRandomString(20))
```

### Key Implementation Notes
- **IPinfo Limit**: 50k requests/month free tier
- **Migrations**: Run automatically on startup
- **Health Check**: Returns 503 when not ready for traffic
- **Graceful Shutdown**: 30-second timeout on SIGTERM

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Run linting: `golangci-lint run`
6. Submit a pull request

## License

MIT License - see LICENSE file for details.
