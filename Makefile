.PHONY: test build run docker-build

test:
	go test ./...

build:
	go build -o bin/eis-customer ./cmd/eis-customer

run:
	go run ./cmd/eis-customer

docker-build:
	docker build -t eis-customer:local .

