SHORT_NAME := storage
PLATFORM ?= linux/amd64,linux/arm64

# container development environment variables
REPO_PATH := github.com/drycc/${SHORT_NAME}
DEV_ENV_IMAGE := ${DEV_REGISTRY}/drycc/go-dev
DEV_ENV_WORK_DIR := /opt/drycc/go/src/${REPO_PATH}
DEV_ENV_PREFIX := podman run --env CGO_ENABLED=0 --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR}
DEV_ENV_CMD := ${DEV_ENV_PREFIX} ${DEV_ENV_IMAGE}

LDFLAGS := "-s -X main.version=${VERSION}"
BINDIR := ./rootfs/bin
DRYCC_REGISTRY ?= ${DEV_REGISTRY}

IMAGE_PREFIX ?= drycc

include versioning.mk

all: build podman-build podman-push

bootstrap:
	${DEV_ENV_CMD} go mod vendor

build:
	mkdir -p ${BINDIR}
	${DEV_ENV_CMD} go build -ldflags '-s' -o $(BINDIR)/boot boot.go || exit 1

test: test-style
	${DEV_ENV_CMD} go test ./...

test-style:
	${DEV_ENV_CMD} lint

test-cover:
	${DEV_ENV_CMD} test-cover.sh

podman-build:
	# build the main image
	podman build --build-arg CODENAME=${CODENAME} -t ${IMAGE} .
	podman tag ${IMAGE} ${MUTABLE_IMAGE}

deploy: build podman-build podman-push

.PHONY: all bootstrap build test podman-build deploy
