.PHONY: build run-worker docker-clean docker-up docker-down

#
# Build and run
#
build:
	go build -o .bin/job-board .

run-worker:
	./.bin/job-board worker

#
# Docker
#
docker-clean:
	docker compose down -v

docker-up:
	docker compose up -d

docker-down:
	docker compose down
