.PHONY: images

DOCKER_REGISTRY?=$(shell whoami)
DOCKER_TAG?=latest

IMG ?= ${DOCKER_REGISTRY}/networking-agent:${DOCKER_TAG}

all:
	go build ./...
	go test ./...

gofmt:
	gofmt -w -s cmd/ pkg/

goimports:
	goimports -w cmd/ pkg/

.PHONY: docker-build
docker-build:
	docker buildx build -t ${IMG} --load .

.PHONY: docker-push
docker-push: docker-build
	docker push ${IMG}

bounce:
	kubectl delete pod -n kube-system -l name=kopeio-networking-agent
