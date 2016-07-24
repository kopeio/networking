all: image

route-controller:
	go install github.com/kopeio/route-controller/cmd/route-controller

test:
	go test -v github.com/kopeio/route-controller/pkg/...

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

builder-image:
	docker build -f images/builder/Dockerfile -t builder .

build-in-docker: builder-image
	docker run -it -v `pwd`:/src builder /onbuild.sh

image: build-in-docker
	docker build -t kope/route-controller  -f images/route-controller/Dockerfile .

push: image
	docker push kope/route-controller:latest

copydeps:
	rsync -avz _vendor/ vendor/ --exclude vendor/
