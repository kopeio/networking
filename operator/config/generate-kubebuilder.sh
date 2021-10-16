#!/bin/bash

set -e

LATEST=1.0.20210815
kaml concat channels/packages/networking/${LATEST}/manifest.yaml | \
  rbac-gen --name operator --yaml - --crd networkings.addons.kope.io --limit-resource-names --supervisory --format kubebuilder
