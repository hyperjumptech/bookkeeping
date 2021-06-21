# awards makefile

CURRENT_PATH ?= $(shell pwd)
IMAGE_NAME ?= awards-go-img

.PHONY: all test clean build docker

build: build-static
	#go build -a -o $(IMAGE_NAME) cmd/Main.go
	GO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o $(IMAGE_NAME) cmd/Main.go

build-static:
# disable for go1.16.. using a new embed technicq
# go-resource -base "$(CURRENT_PATH)/api/swagger" -path "/docs" -filter "/**/*" -go "$(CURRENT_PATH)/api/StaticApi.go" -package api
	go fmt ./...

clean:
	go clean
	rm -f $(IMAGE_NAME)

lint: build
	golint -set_exit_status ./...

test-short: lint
	go test ./... -v -covermode=count -coverprofile=coverage.out -short

test: lint
	go test ./... -v -race -covermode=atomic -coverprofile=coverage.out

test-coverage: test
	go tool cover -html=coverage.out

docker:
	docker build -t $(IMAGE_NAME) -f .docker/Dockerfile .

docker-run:
	docker run -p 7000:7000 -d $(IMAGE_NAME)
