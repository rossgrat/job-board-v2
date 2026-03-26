.PHONY: build run-worker docker-clean docker-up docker-down generate migrate-diff migrate-apply


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

generate:
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
	set -a && . ./.env && atlas migrate diff $(NAME) --env $(ENV)

migrate-apply: ENV ?= local
migrate-apply:
	set -a && . ./.env && atlas migrate apply --env $(ENV)
