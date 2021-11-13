#!/bin/bash

set -e

kaml concat \
  config/minimal/target-permissions.yaml \
  <(kustomize build config) | \
kaml replace-image operator=justinsb/kopeio-networking-operator:latest | \
kaml normalize-labels |
kaml prefix-name --kind ClusterRole --kind ClusterRoleBinding "kopeio-networking-system:"
