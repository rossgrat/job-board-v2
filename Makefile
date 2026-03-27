.PHONY: build run-worker docker-clean docker-up docker-down generate migrate-diff migrate-apply

include .env
export

#
# Setup
#

install:
	mise install

#
# Build and run
#
build:
	go build -o .bin/job-board .

run-worker:
	./.bin/job-board worker

gen:
	sqlc generate

#
# Docker
#
docker-clean:
	docker compose down -v

docker-up:
	docker compose up -d

docker-down:
	docker compose down

#
# DB
#

migrate-diff: ENV ?= local
migrate-diff: NAME ?= $(shell git rev-parse --abbrev-ref HEAD)
migrate-diff:
	atlas migrate diff $(NAME) --env $(ENV)

migrate-apply: ENV ?= local
migrate-apply:
	atlas migrate apply --env $(ENV)


#
# Develop
#
dev-db-connect:
	docker exec -it job-board-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)
