.PHONY: dev build test lint clean help

dev:
	@echo "Starting development..."
	cd backend && go run ./cmd/server

build:
	@echo "Building backend..."
	cd backend && go build -o bin/server ./cmd/server
	@echo "Building frontend..."
	cd frontend && npm install && npm run build

test:
	@echo "Running backend tests..."
	cd backend && go test ./...
	@echo "Running frontend tests..."
	cd frontend && npm test -- --run

lint:
	@echo "Linting backend..."
	cd backend && gofmt -d .
	@echo "Note: Frontend linting disabled (npm permission issues). Use 'npm run build' for TypeScript checking."

clean:
	@echo "Cleaning..."
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf node_modules

run-docker:
	docker compose up --build

run-docker-detached:
	docker compose up -d --build

stop-docker:
	docker compose down

install-backend:
	cd backend && go mod download

install-frontend:
	cd frontend && npm install

help:
	@echo "Available targets:"
	@echo "  dev              - Run backend in development mode"
	@echo "  build            - Build backend and frontend"
	@echo "  test             - Run tests"
	@echo "  lint             - Run linters"
	@echo "  clean            - Clean build artifacts"
	@echo "  run-docker       - Run with Docker Compose (build)"
	@echo "  run-docker-detached - Run with Docker Compose (detached)"
	@echo "  stop-docker      - Stop Docker Compose"
	@echo "  install-backend  - Install backend dependencies"
	@echo "  install-frontend - Install frontend dependencies"
