.PHONY: build run-worker docker-clean docker-up docker-down generate migrate-diff migrate-apply deploy deploy-stop deploy-logs

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

run-api:
	./.bin/job-board api

gen:
	sqlc generate
	mise exec -- templ generate

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
# Deploy
#
IMAGE = ghcr.io/rossgrat/job-board-v2
SERVER = potatoserver
REMOTE_DIR = ~/services/job-board-v2

deploy:
	scp deploy/docker-compose.yml $(SERVER):$(REMOTE_DIR)/docker-compose.yml
	ssh $(SERVER) "cd $(REMOTE_DIR) && docker pull $(IMAGE):latest && docker compose up -d"

ATLAS_IMAGE = arigaio/atlas:1.1.6-community
DOCKER_NETWORK = job-board-v2_job-board-internal

deploy-migrate:
	rsync -av --delete database/migrations/ $(SERVER):$(REMOTE_DIR)/migrations/
	ssh $(SERVER) 'cd $(REMOTE_DIR) && source .env && \
		docker run --rm \
			--network $(DOCKER_NETWORK) \
			-v ./migrations:/migrations \
			$(ATLAS_IMAGE) \
			migrate apply \
				--dir file://migrations \
				--url postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable'

deploy-stop:
	ssh $(SERVER) "cd $(REMOTE_DIR) && docker compose down"

deploy-logs:
	ssh $(SERVER) "cd $(REMOTE_DIR) && docker compose logs -f"

#
# Develop
#
dev-db-connect:
	docker exec -it job-board-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)
