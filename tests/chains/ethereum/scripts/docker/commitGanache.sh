#!/usr/bin/env bash

set -eu

DOCKER_BUILD="docker build --rm --no-cache --pull"

DOCKER_REPO=$1
DOCKER_TAG=$2
DOCKER_IMAGE=$3
SCAFFOLD_IMAGE=$4
NETWORK_ID=$5

docker cp ./contract/build/addresses/${NETWORK_ID} ethereum-ganache0-scaffold:/root/contracts
docker commit --pause=true ${SCAFFOLD_IMAGE} ${DOCKER_REPO}${DOCKER_IMAGE}:${DOCKER_TAG}
