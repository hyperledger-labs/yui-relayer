#!/usr/bin/env bash

set -eu

setGlobals() {
  USING_ORG=$1
  echo "Using organization ${USING_ORG}"
  if [[ ${USING_ORG} = "Org1" ]]; then
    export CORE_PEER_ID=peer0.org1.example.com
    export CORE_PEER_LOCALMSPID=Org1MSP
    export CORE_PEER_ADDRESS=localhost:7051
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
  elif [[ ${USING_ORG} = "Org2" ]]; then
    export CORE_PEER_ID=peer0.org2.example.com
    export CORE_PEER_LOCALMSPID=Org2MSP
    export CORE_PEER_ADDRESS=localhost:8051
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
  else
    echo "================== ERROR !!! ORG Unknown =================="
    exit 1
  fi
}

