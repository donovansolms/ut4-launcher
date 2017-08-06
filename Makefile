#
# A simple Makefile to easily build, test and run the code
#

.PHONY: default build fmt lint run run_race test clean vet docker_build docker_run docker_clean

APP_NAME := ut4-launcher

default: build

build:
	go build -o ./bin/${APP_NAME} ./src/*.go

run: build
	./bin/${APP_NAME}

# http://golang.org/cmd/go/#hdr-Run_gofmt_on_package_sources
fmt:
	go fmt ./...

test:
	go test ./...

test_cover:
	go test ./ -v -cover -covermode=count -coverprofile=./coverage.out

clean:
	rm ./bin/*
