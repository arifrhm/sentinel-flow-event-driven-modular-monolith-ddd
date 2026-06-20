.PHONY: run run-prod simulate test test-coverage build clean help db-up db-down docker-build docker-run

help:
	@echo "Sentinel-Flow Modular Monolith Makefile commands:"
	@echo "  make run           - Launch the modular monolith locally in memory mode"
	@echo "  make run-prod      - Launch the modular monolith locally in production mode (requires Postgres & Redis)"
	@echo "  make db-up         - Start PostgreSQL and Redis Docker containers"
	@echo "  make db-down       - Stop and remove PostgreSQL and Redis Docker containers"
	@echo "  make simulate      - Execute the traffic simulator client suite"
	@echo "  make test          - Run all project unit tests"
	@echo "  make test-coverage - Run all tests and generate a code coverage report"
	@echo "  make build         - Compile the monolith and simulator into bin/"
	@echo "  make clean         - Clean up build artifacts"
	@echo "  make docker-build  - Build the production monolith Docker image"
	@echo "  make docker-run    - Run the modular monolith container locally"

run:
	DATABASE_TYPE=memory BROKER_TYPE=memory go run cmd/monolith/*.go

run-prod:
	DATABASE_TYPE=postgres \
	DATABASE_URL="postgres://sentinel_flow_admin:sentinel_flow_secure_password@localhost:5432/sentinel_flow_production?sslmode=disable" \
	BROKER_TYPE=redis \
	REDIS_URL="redis://localhost:6379" \
	go run cmd/monolith/*.go

db-up:
	docker-compose up -d

db-down:
	docker-compose down

simulate:
	go run scripts/simulator/*.go

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "Coverage HTML report generated at coverage.html"
	go tool cover -html=coverage.out -o coverage.html

build:
	mkdir -p bin
	go build -o bin/monolith cmd/monolith/*.go
	go build -o bin/simulator scripts/simulator/*.go

clean:
	rm -rf bin coverage.out coverage.html

docker-build:
	docker build -t sentinel-flow-monolith .

docker-run:
	docker run -p 8081:8081 -p 8082:8082 -p 8083:8083 -p 8084:8084 sentinel-flow-monolith
