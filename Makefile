# TODO: Move entirely to bazel?
.PHONY: images

DOCKER_REGISTRY?=kopeio
DOCKER_TAG=1.0.20170406

all: images

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

push: images
	docker push ${DOCKER_REGISTRY}/networking-agent:${DOCKER_TAG}

images:
	bazel run //images:networking-agent ${DOCKER_REGISTRY}/networking-agent:${DOCKER_TAG}

# Targets for building inside docker

buildindocker: bazelbuild
	mkdir -p build/
	docker run -v `pwd`:/src -v ~/.cache/bazeldocker/:/root/.cache -v `pwd`/build:/build bazelbuild make indockertarget

bazelbuild:
	cd images/bazelbuild; docker build -t bazelbuild .

indockertarget:
	bazel build --spawn_strategy=standalone --genrule_strategy=standalone  //images:networking-agent.tar
	cp /src/bazel-bin/images/networking-agent.tar /build/

