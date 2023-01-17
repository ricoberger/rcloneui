REPO        ?= github.com/ricoberger/rcloneui
PWD         ?= $(shell pwd)
VERSION     ?= $(shell git describe --tags)
REVISION    ?= $(shell git rev-parse HEAD)
BRANCH      ?= $(shell git rev-parse --abbrev-ref HEAD)
BUILDUSER   ?= $(shell id -un)
BUILDTIME   ?= $(shell date '+%Y%m%d-%H:%M:%S')

.PHONY: build
build:
	go build -ldflags "-X ${REPO}/pkg/version.Version=${VERSION} \
		-X ${REPO}/pkg/version.Revision=${REVISION} \
		-X ${REPO}/pkg/version.Branch=${BRANCH} \
		-X ${REPO}/pkg/version.BuildUser=${BUILDUSER} \
		-X ${REPO}/pkg/version.BuildDate=${BUILDTIME}" \
		-o ./bin/rcloneui ./cmd/rcloneui; \
