# TODO: Move entirely to bazel?
.PHONY: images

all: images

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

push: images
	docker push kopeio/krouton-network-agent:latest

images:
	bazel run //images:krouton-network-agent kopeio/krouton-network-agent:latest
