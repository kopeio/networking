# TODO: Move entirely to bazel?
.PHONY: images

DOCKER_REGISTRY?=kopeio
DOCKER_TAG=20161116

all: images

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

push: images
	docker push ${DOCKER_REGISTRY}/networking-agent:${DOCKER_TAG}

images:
	bazel run //images:networking-agent ${DOCKER_REGISTRY}/networking-agent:${DOCKER_TAG}
