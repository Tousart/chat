.PHONY: build up down rebuild restart

build:
	docker compose build

up:
	docker compose up

down:
	docker compose down -v

rebuild: down
	docker compose build

restart: rebuild
	docker compose up