# awards makefile

CURRENT_PATH ?= $(shell pwd)
IMAGE_NAME ?= awards-go-img

.PHONY: all test clean build docker

build:
	#go build -a -o $(IMAGE_NAME) cmd/Main.go
	GO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o $(IMAGE_NAME) cmd/Main.go

clean:
	go clean
	rm -f $(IMAGE_NAME)

lint: build
	# golint -set_exit_status ./...

test-short: lint
	go test ./... -v -covermode=count -coverprofile=coverage.out -short

test: lint
	go test ./... -v -race -covermode=atomic -coverprofile=coverage.out

run: build
	go run cmd/Main.go

test-coverage: test
	go tool cover -html=coverage.out

docker:
	docker build -t $(IMAGE_NAME) -f ./.docker/Dockerfile .

docker-run:
	docker run -p 7000:7000 -d $(IMAGE_NAME)
