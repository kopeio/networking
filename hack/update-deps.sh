#!/bin/bash

set -ex

GO111MODULE=on go mod vendor
find vendor -name "BUILD" -delete
find vendor -name "BUILD.bazel" -delete
bazel run //:gazelle -- -proto=disable

