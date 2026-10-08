.PHONY: test vet build run docker

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o bin/heimda11 ./cmd/heimda11

run:
	go run ./cmd/heimda11

docker:
	docker compose up --build
