SHORT_NAME := minio
PLATFORM ?= linux/amd64,linux/arm64

# dockerized development environment variables
REPO_PATH := github.com/drycc/${SHORT_NAME}
DEV_ENV_IMAGE := drycc/go-dev
DEV_ENV_WORK_DIR := /go/src/${REPO_PATH}
DEV_ENV_PREFIX := docker run --env CGO_ENABLED=0 --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR}
DEV_ENV_CMD := ${DEV_ENV_PREFIX} ${DEV_ENV_IMAGE}

LDFLAGS := "-s -X main.version=${VERSION}"
BINDIR := ./rootfs/bin
DEV_REGISTRY ?= 
DRYCC_REGISTRY ?= ${DEV_REGISTRY}

IMAGE_PREFIX ?= drycc

include versioning.mk

all: build docker-build docker-push

bootstrap:
	${DEV_ENV_CMD} go mod vendor

build:
	mkdir -p ${BINDIR}
	${DEV_ENV_CMD} go build -ldflags '-s' -o $(BINDIR)/boot boot.go || exit 1

test: test-style
	${DEV_ENV_CMD} go test ./...

test-style:
	${DEV_ENV_CMD} lint --deadline

test-cover:
	${DEV_ENV_CMD} test-cover.sh

docker-build:
	# build the main image
	docker build ${DOCKER_BUILD_FLAGS} -t ${IMAGE} .
	docker tag ${IMAGE} ${MUTABLE_IMAGE}

docker-buildx:
	docker buildx build --platform ${PLATFORM} ${DOCKER_BUILD_FLAGS} -t ${IMAGE} . --push

deploy: build docker-build docker-push

.PHONY: all bootstrap build test docker-build deploy
