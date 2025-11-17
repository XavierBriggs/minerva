.PHONY: build run test clean docker-build docker-run

# Build the binary
build:
	go build -o bin/minerva ./cmd/minerva

# Run the service locally
run:
	go run ./cmd/minerva

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Build Docker image
docker-build:
	docker build -t fortuna-minerva:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 -p 8081:8081 \
		-e GRINGOTTS_DSN="postgres://fortuna:fortuna_pw@host.docker.internal:5434/gringotts?sslmode=disable" \
		-e REDIS_URL="redis://host.docker.internal:6379" \
		fortuna-minerva:latest

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run migrations
migrate-up:
	migrate -path infra/atlas/migrations -database "${ATLAS_DSN}" up

# Rollback migrations
migrate-down:
	migrate -path infra/atlas/migrations -database "${ATLAS_DSN}" down 1

