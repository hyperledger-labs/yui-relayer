#!/usr/bin/env bash

set -eux

RLY_BINARY=../build/uly
RLY="${RLY_BINARY} --debug"
FIXTURES_DIR=./fixtures

$RLY fabric wallet populate ibc1 --cert ${FIXTURES_DIR}/certs/org1-user-signcert.pem --key ${FIXTURES_DIR}/keys/clients/org1-user-priv_sk
# $RLY fabric wallet populate channel2 --cert ${FIXTURES_DIR}/certs/org3-user-signcert.pem --key ${FIXTURES_DIR}/keys/clients/org3-user-priv_sk

# ${CLI_COMMAND} tx init channel1
# ${CLI_COMMAND} tx init channel2

# ${CLI_COMMAND} query sequence channel1
# ${CLI_COMMAND} query sequence channel2

# ${CLI_COMMAND} tx raw client channel1 channel2 channel1client
# ${CLI_COMMAND} tx raw client channel2 channel1 channel2client

# sleep 5

# ${CLI_COMMAND} tx raw update-sequence channel2
# ${CLI_COMMAND} tx create cc-info channel1 channel2 -o ./tests/output/noSigCCInfoChannel2.json
# ${CLI_COMMAND} tx sign cc-info \
# --msp-id Org3MSP \
# --msp-conf-dir ${FIXTURES_DIR}/msps/Org3MSP \
# -i ./tests/output/noSigCCInfoChannel2.json \
# -o ./tests/output/org3SignedCCInfoChannel2.json

# ${CLI_COMMAND} tx sign cc-info \
# --msp-id Org4MSP \
# --msp-conf-dir ${FIXTURES_DIR}/msps/Org4MSP \
# -i ./tests/output/org3SignedCCInfoChannel2.json \
# -o ./tests/output/bothSignedCCInfoChannel2.json

# ${CLI_COMMAND} tx raw update-client channel1 channel2 channel1client -i ./tests/output/bothSignedCCInfoChannel2.json

# ${CLI_COMMAND} tx raw update-sequence channel1
# ${CLI_COMMAND} tx create cc-info channel2 channel1 -o ./tests/output/noSigCCInfoChannel1.json
# ${CLI_COMMAND} tx sign cc-info \
# --msp-id Org1MSP \
# --msp-conf-dir ${FIXTURES_DIR}/msps/Org1MSP \
# -i ./tests/output/noSigCCInfoChannel1.json \
# -o ./tests/output/org1SignedCCInfoChannel1.json

# ${CLI_COMMAND} tx sign cc-info \
# --msp-id Org2MSP \
# --msp-conf-dir ${FIXTURES_DIR}/msps/Org2MSP \
# -i ./tests/output/org1SignedCCInfoChannel1.json \
# -o ./tests/output/bothSignedCCInfoChannel1.json

# ${CLI_COMMAND} tx raw update-client channel2 channel1 channel2client  -i ./tests/output/bothSignedCCInfoChannel1.json

# ${CLI_COMMAND} tx connection channel1_channel2
# ${CLI_COMMAND} tx channel channel1_channel2

# ${CLI_COMMAND} tx transfer channel1_channel2 10ftk true
# # ensure that increments sequence correctly
# ${CLI_COMMAND} tx transfer channel1_channel2 10ftk true
