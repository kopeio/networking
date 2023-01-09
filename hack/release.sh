#!/bin/bash

TODAY=`date +%Y%m%d`
echo "# Run this command to do a release"
<<<<<<< HEAD
echo "DOCKER_REGISTRY=kopeio DOCKER_TAG=1.0.${TODAY} make docker-push"
echo "DOCKER_REGISTRY=kopeio DOCKER_TAG=1.0.${TODAY} make -C operator docker-build docker-push"
=======
echo "IMAGE_REGISTRY=kopeio IMAGE_TAG=1.0.${TODAY} make push"
>>>>>>> 57a23e0 (Replace bazel with ko)
