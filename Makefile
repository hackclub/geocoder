.PHONY: build test run dev migrate keygen docker-build docker-up docker-down clean

# Development
dev:
	go run cmd/server/main.go

run:
	go run cmd/server/main.go

# Build
build:
	go build -o bin/geocoder cmd/server/main.go

# Testing
test:
	go test ./...

test-verbose:
	go test -v ./...

test-integration:
	go test -tags=integration ./tests/

test-integration-verbose:
	go test -tags=integration -v ./tests/

test-all:
	go test ./...
	go test -tags=integration ./tests/

# Database
migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down

# Key generation
keygen:
	go run cmd/keygen/main.go --name "dev-key" --prefix "dev"

# Docker Production
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Docker Development
docker-dev-build:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml build

docker-dev-up:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

docker-dev-down:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

docker-dev-logs:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

# Development with Docker (live reload)
docker-dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build

# Code quality
fmt:
	go fmt ./...

lint:
	golangci-lint run

# Clean
clean:
	rm -rf bin/
	docker-compose down -v

# Quick setup for development
setup:
	cp .env.example .env
	docker-compose up -d postgres
	sleep 5
	go run cmd/keygen/main.go --name "dev-key" --prefix "dev"

# Help
help:
	@echo "Available commands:"
	@echo "  dev         - Run development server (auto-runs migrations)"
	@echo "  build       - Build the application"
	@echo "  test        - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-all    - Run all tests (unit + integration)"
	@echo "  migrate-up  - Run database migrations manually"
	@echo "  keygen      - Generate an API key"
	@echo "  docker-up   - Start production with Docker Compose"
	@echo "  docker-down - Stop Docker Compose"
	@echo "  docker-dev  - Start development with Docker (live reload)"
	@echo "  setup       - Quick setup for development"
	@echo "  clean       - Clean up build artifacts and Docker volumes"
