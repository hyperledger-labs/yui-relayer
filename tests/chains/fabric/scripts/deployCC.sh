#!/usr/bin/env bash
set -eux pipefail

export FABRIC_CFG_PATH=${PWD}/configtx

VERSION=${VERSION:="1"}

# import utils
. ./scripts/setEnv.sh

OREDERER_ENDPOINT=localhost:7050
CC_NAME=${CC_NAME:="fabibc"}

packageChaincode() {
    ORG_NAME=$1
    CC_NAME=$2

    echo "### Package Chaincode ${CC_NAME}"
    pushd ./external-builders/config/${ORG_NAME}/${CC_NAME}
    tar cfz code.tar.gz connection.json
    tar cfz ${ORG_NAME}-${CC_NAME}.tar.gz code.tar.gz metadata.json
    mv ${ORG_NAME}-${CC_NAME}.tar.gz ../../../../build
    popd
}

installChaincode() {
    ORG_NAME=$1
    CC_NAME=$2
    CHANNEL_NAME=$3

    setGlobals ${ORG_NAME}
    set -x
    peer lifecycle chaincode install ./build/${ORG_NAME}-${CC_NAME}.tar.gz
    set +x
}

queryInstalled() {
    ORG_NAME=$1
    CC_NAME=$2

    setGlobals ${ORG_NAME}
    set -x
    peer lifecycle chaincode queryinstalled >&log.txt
    set +x
    cat log.txt
    PACKAGE_ID=$(sed -n "/${CC_NAME}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)
    echo ${PACKAGE_ID} &> ./build/${ORG_NAME}-${CC_NAME}-ccid.txt
}

approveForMyOrg() {
    ORG_NAME=$1
    CC_NAME=$2
    CHANNEL_NAME=$3
    SIGNATURE_POLICY=$4
    COLLECTION_CFG_JSON=$5

    setGlobals ${ORG_NAME}
    PACKAGE_ID=$(cat ./build/${ORG_NAME}-${CC_NAME}-ccid.txt)
    set -x
    peer lifecycle chaincode approveformyorg \
    -o ${OREDERER_ENDPOINT} \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${VERSION} \
    --sequence ${VERSION} \
    --package-id ${PACKAGE_ID} \
    --signature-policy ${SIGNATURE_POLICY} \
    --collections-config ${COLLECTION_CFG_JSON}

    peer lifecycle chaincode checkcommitreadiness \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${VERSION} \
    --sequence ${VERSION} \
    --signature-policy ${SIGNATURE_POLICY} \
    --collections-config ${COLLECTION_CFG_JSON} \
    --output json
    set +x
}

commitChaincode() {
    ORG_NAME=$1
    CC_NAME=$2
    CHANNEL_NAME=$3
    SIGNATURE_POLICY=$4
    COLLECTION_CFG_JSON=$5

    PEER_CONN_PARAMS=""
    for org in ${@:6}; do
      setGlobals ${org}
      PEER_CONN_PARAMS="$PEER_CONN_PARAMS --peerAddresses $CORE_PEER_ADDRESS"
    done

    setGlobals ${ORG_NAME}
    set -x
    peer lifecycle chaincode commit \
    -o ${OREDERER_ENDPOINT} \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${VERSION} \
    --sequence ${VERSION} ${PEER_CONN_PARAMS} \
    --signature-policy ${SIGNATURE_POLICY} \
    --collections-config ${COLLECTION_CFG_JSON}

    peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CC_NAME}
    set +x
}

mkdir -p ./build

ORG_NAME="Org1"
MSP_ID="Org1MSP"
CHANNEL_NAME=channel1
packageChaincode ${ORG_NAME} ${CC_NAME}
installChaincode ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME}
SIGNATURE_POLICY="OR('${MSP_ID}.peer')"
COLLECTION_CFG_JSON=./collection-policy/${CHANNEL_NAME}.json
queryInstalled ${ORG_NAME} ${CC_NAME}
approveForMyOrg ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME} ${SIGNATURE_POLICY} ${COLLECTION_CFG_JSON}
commitChaincode ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME} ${SIGNATURE_POLICY} ${COLLECTION_CFG_JSON}

ORG_NAME="Org2"
MSP_ID="Org2MSP"
CHANNEL_NAME=channel2
packageChaincode ${ORG_NAME} ${CC_NAME}
installChaincode ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME}
SIGNATURE_POLICY="OR('${MSP_ID}.peer')"
COLLECTION_CFG_JSON=./collection-policy/${CHANNEL_NAME}.json
queryInstalled ${ORG_NAME} ${CC_NAME}
approveForMyOrg ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME} ${SIGNATURE_POLICY} ${COLLECTION_CFG_JSON}
commitChaincode ${ORG_NAME} ${CC_NAME} ${CHANNEL_NAME} ${SIGNATURE_POLICY} ${COLLECTION_CFG_JSON}
