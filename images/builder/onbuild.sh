#!/bin/bash

mkdir -p /go
export GOPATH=/go

mkdir -p /go/src/github.com/kopeio/
ln -s /src /go/src/github.com/kopeio/route-controller

cd /go/src/github.com/kopeio/route-controller
/usr/bin/glide install --strip-vendor --strip-vcs

go install github.com/kopeio/route-controller/cmd/route-controller

mkdir -p /src/.build/artifacts/
cp /go/bin/route-controller /src/.build/artifacts/
