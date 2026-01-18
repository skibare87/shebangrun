.PHONY: help build run stop clean logs test

help:
	@echo "shebang.run - Makefile commands"
	@echo ""
	@echo "  make build    - Build Docker images"
	@echo "  make run      - Start all services"
	@echo "  make stop     - Stop all services"
	@echo "  make restart  - Restart all services"
	@echo "  make logs     - View application logs"
	@echo "  make clean    - Remove all containers and volumes"
	@echo "  make db       - Access database shell"
	@echo "  make test     - Run tests"
	@echo "  make dev      - Run locally without Docker"

build:
	docker-compose build

run:
	docker-compose up -d
	@echo "Application starting at http://localhost"
	@echo "MinIO console at http://localhost:9001"

stop:
	docker-compose stop

restart:
	docker-compose restart

logs:
	docker-compose logs -f app

clean:
	docker-compose down -v
	@echo "All containers and volumes removed"

db:
	docker-compose exec mariadb mysql -u root -prootpassword shebang

test:
	go test ./...

dev:
	go run cmd/server/main.go
