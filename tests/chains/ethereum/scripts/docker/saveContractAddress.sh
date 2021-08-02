#!/usr/bin/env bash

set -eu

NETWORK_ID=$1

SAVE_DIR=./contract/build/addresses/${NETWORK_ID}
mkdir -p ${SAVE_DIR}
jq -r ".networks | .[\"${NETWORK_ID}\"].address" < ./contract/build/contracts/IBCHost.json > ${SAVE_DIR}/IBCHost
jq -r ".networks | .[\"${NETWORK_ID}\"].address" < ./contract/build/contracts/IBCHandler.json > ${SAVE_DIR}/IBCHandler
