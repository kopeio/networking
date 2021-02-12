# TODO: Move entirely to bazel?
.PHONY: images

DOCKER_REGISTRY?=$(shell whoami)
DOCKER_TAG?=latest

all:
	bazel build //...
	bazel test //...

gofmt:
	gofmt -w -s cmd/ pkg/

goimports:
	goimports -w cmd/ pkg/

push:
	bazel run //images:push-networking-agent

deps:
	go mod vendor
	find vendor/ -name "BUILD" -delete
	find vendor/ -name "BUILD.bazel" -delete
	bazel run //:gazelle -- --proto=disable

gazelle:
	bazel run //:gazelle -- fix --proto=disable

bounce:
	kubectl delete pod -n kube-system -l name=kopeio-networking-agent
