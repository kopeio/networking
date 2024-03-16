.PHONY: images

IMAGE_REPO?=$(shell whoami)
IMAGE_TAG?=latest

all:
	go build ./...
	go test ./...

gofmt:
	gofmt -w -s cmd/ pkg/

goimports:
	goimports -w cmd/ pkg/

push:
	KO_DOCKER_REPO=${IMAGE_REPO} go run github.com/google/ko@v0.15.2 build -B --tags=${IMAGE_TAG} ./cmd/networking-agent

bounce:
	kubectl delete pod -n kube-system -l name=kopeio-networking-agent
