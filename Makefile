.PHONY: build run-worker

build:
	go build -o .bin/job-board .

run-worker:
	./.bin/job-board worker
