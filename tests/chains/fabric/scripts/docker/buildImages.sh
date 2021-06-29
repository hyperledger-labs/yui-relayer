#!/usr/bin/env bash

set -eux

DOCKER_BUILD="docker build --rm --no-cache --pull"

DOCKER_REPO=$1
DOCKER_TAG=$2

## orderer
${DOCKER_BUILD} -f ./Dockerfiles/orderer/Dockerfile \
		--build-arg ORDERER_BLOCK_FILE_PATH=artifacts/orderer.block \
		--build-arg LEDGER_BACKUP_PATH=backup/orderer/ledger \
		--build-arg ORDERER_CONFIG_FILE_PATH=configtx/orderer.yaml \
		--build-arg MSP_CONFIG_FILE_PATH=organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp \
		--tag ${DOCKER_REPO}fabric-orderer:${DOCKER_TAG} .

## org1 peer0
${DOCKER_BUILD} -f ./Dockerfiles/peer/Dockerfile \
    --build-arg PEER_CONFIG_FILE_PATH=configtx/core.yaml \
    --build-arg EXTERNAL_BUILDER_BIN_PATH=external-builders/bin \
    --build-arg LEDGER_BACKUP_PATH=backup/peer0-org1/ledger \
    --build-arg MSP_CONFIG_FILE_PATH=organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/msp \
    --tag ${DOCKER_REPO}fabric-peer0-org1:${DOCKER_TAG} .

## org2 peer0
${DOCKER_BUILD} -f ./Dockerfiles/peer/Dockerfile \
  --build-arg PEER_CONFIG_FILE_PATH=configtx/core.yaml \
  --build-arg EXTERNAL_BUILDER_BIN_PATH=external-builders/bin \
  --build-arg LEDGER_BACKUP_PATH=backup/peer0-org2/ledger \
  --build-arg MSP_CONFIG_FILE_PATH=organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/msp \
  --tag ${DOCKER_REPO}fabric-peer0-org2:${DOCKER_TAG} .

## org1 chaincode
${DOCKER_BUILD} -f ./Dockerfiles/chaincode/Dockerfile \
  --build-arg CHAINCODE_CCID="$(cat ./build/Org1-fabibc-ccid.txt)" \
  --build-arg CHAINCODE_ADDRESS="$(jq -r .address < external-builders/config/Org1/fabibc/connection.json)" \
  --tag ${DOCKER_REPO}fabric-chaincode-org1:${DOCKER_TAG} .

## org2 chaincode
${DOCKER_BUILD} -f ./Dockerfiles/chaincode/Dockerfile \
  --build-arg CHAINCODE_CCID="$(cat ./build/Org2-fabibc-ccid.txt)" \
  --build-arg CHAINCODE_ADDRESS="$(jq -r .address < external-builders/config/Org2/fabibc/connection.json)" \
  --tag ${DOCKER_REPO}fabric-chaincode-org2:${DOCKER_TAG} .

# msp conf data container
${DOCKER_BUILD} -f ./Dockerfiles/data/Dockerfile \
  --build-arg MSP_CONF_DIR_PATH=organizations \
  --build-arg CONNECTION_PROFILE_DIR_PATH=connection-profile \
  --tag ${DOCKER_REPO}fabric-data:${DOCKER_TAG} .
