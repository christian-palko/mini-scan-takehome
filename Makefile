.PHONY: up down psql test

up:
	docker compose up --build -d
	@until docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; do \
		echo "Waiting for Postgres..."; \
		sleep 1; \
	done
	docker compose exec -T postgres psql -U postgres -d scans \
		< db/migrations/0001_init.sql
	docker compose logs -f processor scanner

down:
	docker compose down

psql:
	docker compose exec postgres psql -U postgres -d scans

test:
	go test ./...
