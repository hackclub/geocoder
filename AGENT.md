# Hack Club Geocoder - Agent Instructions

## Overview
This is a Go-based geocoding API service that provides address geocoding, IP geolocation, caching, and admin management features.

## Development Commands

### Build & Test
```bash
go build cmd/server/main.go          # Build the main server
go run cmd/server/main.go             # Run development server (auto-runs migrations)

# Unit Tests (fast, no external dependencies)
go test ./...                        # Run all unit tests
go test ./internal/...               # Run only internal package unit tests
go test -short ./...                 # Run unit tests with -short flag
go test -v ./internal/api            # Run specific package tests with verbose output

# Integration Tests (comprehensive, with database simulation)
go test -tags=integration .          # Run integration tests in root directory
./scripts/run_integration_tests.sh   # Run full integration test suite with real database

# Build Verification (must pass before deployment)
go build ./cmd/server                # Verify server builds
go build ./cmd/migrate               # Verify migrate builds 
go build ./cmd/keygen                # Verify keygen builds
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

## Testing Strategy

### Test Structure
The project has a comprehensive testing structure with multiple types of tests:

#### Unit Tests (Fast, Isolated)
- **Location**: `internal/*/` directories
- **Purpose**: Test individual components in isolation
- **Run Command**: `go test ./internal/...` or `go test -short ./...`
- **Key Files**:
  - `internal/api/handlers_test.go` - HTTP handler tests
  - `internal/database/database_test.go` - Database utility tests  
  - `internal/cache/cache_test.go` - Cache service tests
  - `internal/middleware/auth_test.go` - Authentication tests
  - `internal/geocoding/client_test.go` - Geocoding client tests
  - `internal/geoip/client_test.go` - GeoIP client tests

#### Integration Tests (Comprehensive, System-level)
- **Location**: `integration_test.go` (root directory)
- **Purpose**: Test complete system functionality with mock or real databases
- **Run Command**: `go test -tags=integration .` or `./scripts/run_integration_tests.sh`
- **Coverage**: All 11 tests cover:
  - Health endpoint functionality
  - API authentication and rate limiting
  - Geocoding and IP geolocation endpoints
  - Admin interface and API key management
  - **Cache eviction when hitting max database size limits**
  - Error handling and CORS functionality

### Testing After Code Changes

#### Required Test Sequence (Always Run This)
```bash
# 1. Unit tests (fast verification)
go test ./internal/...

# 2. Build verification (ensures code compiles)
go build ./cmd/server
go build ./cmd/migrate  
go build ./cmd/keygen

# 3. Integration tests (comprehensive system testing)
go test -tags=integration .

# 4. Full integration suite with real database (optional but recommended)
./scripts/run_integration_tests.sh
```

#### Quick Development Testing
```bash
go test -short ./...                 # Fast unit tests only
go test -v ./internal/api            # Test specific component
go test -tags=integration . -v       # Integration tests with output
```

#### Pre-deployment Testing
```bash
./scripts/run_integration_tests.sh   # Full test suite with real database
```

### Mock Database Pattern
- All tests use comprehensive mock database implementations
- Mocks implement `database.DatabaseInterface` 
- **Cache eviction testing**: Simulates FIFO eviction when max sizes reached
- Realistic cache behavior with configurable limits

### Build Tags
- Unit tests: No special tags required
- Integration tests: Use `// +build integration` tag
- Run with `-tags=integration` flag

### Test Coverage Areas
- **Authentication**: API key validation, Basic Auth for admin
- **Rate Limiting**: Per-API-key token bucket implementation  
- **Caching**: Cache hits/misses, FIFO eviction, max size limits
- **API Endpoints**: All v1 and admin endpoints
- **Error Handling**: All error codes and response formats
- **Security**: CORS, input validation, SQL injection prevention

### Critical Cache Testing
The integration tests specifically verify **max database size and cache clearing**:
- Address cache: 10k entries max (configurable via `MAX_ADDRESS_CACHE_SIZE`)
- IP cache: 5k entries max (configurable via `MAX_IP_CACHE_SIZE`)  
- FIFO eviction: Deletes oldest 10% when limit reached
- Verified in `TestIntegration_CacheEviction`

## Test Environments

### Development Testing (Mock Database)
```bash
go test -tags=integration .          # Uses comprehensive mock database
```
- **Fast execution** (no Docker required)
- **Comprehensive coverage** of all functionality
- **Mock database** simulates real behavior including cache eviction
- **Ideal for**: Development, CI/CD, quick verification

### Production-like Testing (Real Database)  
```bash
./scripts/run_integration_tests.sh   # Uses real PostgreSQL in Docker
```
- **Real database** connections and operations
- **Docker PostgreSQL** container (postgres:16-alpine)
- **Database migrations** and actual SQL execution
- **Ideal for**: Pre-deployment verification, debugging database issues

### Test Selection Guidelines

#### When to Run Unit Tests (`go test ./internal/...`)
- ✅ After changing individual components
- ✅ During development for quick feedback
- ✅ Before committing code
- ✅ In CI/CD pipelines (fast)

#### When to Run Integration Tests (`go test -tags=integration .`)
- ✅ After API changes
- ✅ After database schema changes  
- ✅ Before creating pull requests
- ✅ When testing system interactions

#### When to Run Full Integration Suite (`./scripts/run_integration_tests.sh`)
- ✅ Before deployment to production
- ✅ After major feature changes
- ✅ When debugging database-related issues
- ✅ For comprehensive system validation

### Makefile Integration
```bash
make test                            # Unit tests only
make test-integration               # Integration tests (Note: ./tests/ directory is outdated)
make test-all                       # Both unit and integration (outdated)
```

**⚠️ Important**: The Makefile's `test-integration` target points to outdated tests in `./tests/` directory. Use the commands above instead.
