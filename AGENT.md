# Hack Club Geocoder - Agent Instructions

## Overview
This is a Go-based geocoding API service that provides address geocoding, IP geolocation, caching, and admin management features.

## Development Commands

### Build & Test
```bash
go build cmd/server/main.go          # Build the main server
go test ./...                        # Run all tests
go run cmd/server/main.go             # Run development server (auto-runs migrations)
```

### Database
```bash
go run cmd/migrate/main.go up         # Run migrations manually (optional)
go run cmd/migrate/main.go down       # Rollback latest migration
```

### API Key Management
```bash
go run cmd/keygen/main.go --name "test-key" --prefix "test"  # Generate API key
```

### Docker Development
```bash
make docker-dev                      # Start development with live reload
make docker-dev-up                   # Start development containers
make docker-dev-down                 # Stop development containers
make docker-up                       # Start production containers
make docker-down                     # Stop Docker Compose
make setup                           # Quick development setup
```

### Quick Start
```bash
make setup                           # Sets up database and creates dev key
go run cmd/server/main.go             # Start server on port 8080
```

### Docker Development Workflow
```bash
# Option 1: Local development with live reload
make docker-dev                      # Starts PostgreSQL + Go server with live reload

# Option 2: Just database, run Go locally  
make docker-dev-up                   # Start just PostgreSQL
go run cmd/server/main.go             # Run server locally

# Option 3: Production-like build
make docker-up                       # Full production Docker setup
```

## Project Structure
- `cmd/` - Command-line applications (server, migrate, keygen)
- `internal/api/` - HTTP handlers and routes
- `internal/cache/` - Caching logic using PostgreSQL
- `internal/config/` - Configuration management
- `internal/database/` - Database operations and interface
- `internal/geocoding/` - Google Geocoding API client
- `internal/geoip/` - IPinfo API client
- `internal/middleware/` - Authentication and rate limiting
- `internal/models/` - Data structures
- `migrations/` - Database migration files
- `web/` - Admin dashboard frontend

## API Endpoints
- `GET /` - API documentation (no auth required)
- `GET /v1/geocode?address={address}&key={api_key}` - Geocode address
- `GET /v1/geoip?ip={ip}&key={api_key}` - IP geolocation
- `GET /health` - Health check
- `GET /admin/dashboard` - Admin web interface (Basic Auth)
- Admin API endpoints under `/admin/` for key management

## Environment Variables
```bash
GOOGLE_GEOCODING_API_KEY=your_api_key
IPINFO_API_KEY=your_api_key          # Optional
DATABASE_URL=postgres://...
ADMIN_USERNAME=admin
ADMIN_PASSWORD=secure_password
```

## Database
Uses PostgreSQL with the following main tables:
- `address_cache` - Cached geocoding results
- `ip_cache` - Cached IP geolocation results  
- `api_keys` - API key management
- `usage_logs` - Request tracking
- `cost_tracking` - Daily cost aggregates

## External APIs
- Google Geocoding API - For address to coordinates conversion
- IPinfo API - For IP to location conversion (works without API key)

## Features
- Intelligent caching with FIFO eviction
- Per-API-key rate limiting
- Real-time WebSocket updates for admin dashboard
- Live statistics and analytics with instant updates
- Cost tracking and monitoring
- Admin web interface with live map visualization

## Code Style
- Use interfaces for database operations (see `database.DatabaseInterface`)
- Error handling with structured error responses
- Middleware pattern for authentication and rate limiting
- WebSocket for real-time updates

## Testing
- Mock database interface for unit tests
- Health endpoint testing
- Build verification before deployment
