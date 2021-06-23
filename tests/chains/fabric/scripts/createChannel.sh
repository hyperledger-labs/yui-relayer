#!/bin/bash
set -eu pipefail

export FABRIC_CFG_PATH=${PWD}/configtx

# import utils
. ./scripts/setEnv.sh

OREDERER_ENDPOINT=localhost:7050

createChannel() {
    CHANNEL_NAME=$1
    ORG_NAME=$2

    setGlobals "${ORG_NAME}"
    echo "### Creating channel ${CHANNEL_NAME}"
    peer channel create -c ${CHANNEL_NAME} -f ./artifacts/${CHANNEL_NAME}.tx -o ${OREDERER_ENDPOINT}
    mv ./${CHANNEL_NAME}.block ./artifacts/${CHANNEL_NAME}.block
    echo "### Creating channel ${CHANNEL_NAME} Done"
}

joinChannel() {
    CHANNEL_NAME=$1
    ORG_NAME=$2

    setGlobals "${ORG_NAME}"
    echo "### Join channel ${CHANNEL_NAME} Org=${ORG_NAME}"
    peer channel join -b ./artifacts/${CHANNEL_NAME}.block -o ${OREDERER_ENDPOINT}
    echo "### Join channel ${CHANNEL_NAME} Org=${ORG_NAME} Done"
}

updateAnchorPeers() {
    CHANNEL_NAME=$1
    ORG_NAME=$2

    setGlobals "${ORG_NAME}"
    peer channel update -o ${OREDERER_ENDPOINT} -c ${CHANNEL_NAME} -f ./artifacts/"${CHANNEL_NAME}"-"${ORG_NAME}"Anchors.tx
}

createChannel channel1 "Org1"
joinChannel channel1 "Org1"
updateAnchorPeers channel1 "Org1"

createChannel channel2 "Org2"
joinChannel channel2 "Org2"
updateAnchorPeers channel2 "Org2"

exit 0
