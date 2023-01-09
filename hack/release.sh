#!/bin/bash

TODAY=`date +%Y%m%d`
echo "# Run this command to do a release"
echo "IMAGE_REGISTRY=kopeio IMAGE_TAG=1.0.${TODAY} make push"
echo "DOCKER_REGISTRY=kopeio DOCKER_TAG=1.0.${TODAY} make -C operator docker-build docker-push"
