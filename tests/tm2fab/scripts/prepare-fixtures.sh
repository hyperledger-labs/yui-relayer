#!/usr/bin/env bash
set -eux

FIXTURES_DIR=./fixtures

## Setup test fixtures

rm -rf ${FIXTURES_DIR}/wallet && mkdir -p ${FIXTURES_DIR}/wallet
rm -rf ${FIXTURES_DIR}/output && mkdir -p ${FIXTURES_DIR}/output
rm -rf ${FIXTURES_DIR}/certs && mkdir -p ${FIXTURES_DIR}/certs
rm -rf ${FIXTURES_DIR}/keys && mkdir -p ${FIXTURES_DIR}/keys/clients ${FIXTURES_DIR}/keys/signers
rm -rf ${FIXTURES_DIR}/msps && mkdir -p ${FIXTURES_DIR}/msps

ln -s ${MSPS_DIR}/Org1MSP/users/User1@org1.dev.com/msp/signcerts/cert.pem ${FIXTURES_DIR}/certs/org1-user-signcert.pem
ln -s ${MSPS_DIR}/Org2MSP/users/User1@org2.dev.com/msp/signcerts/cert.pem ${FIXTURES_DIR}/certs/org2-user-signcert.pem
ln -s ${MSPS_DIR}/Org3MSP/users/User1@org3.dev.com/msp/signcerts/cert.pem ${FIXTURES_DIR}/certs/org3-user-signcert.pem
ln -s ${MSPS_DIR}/Org4MSP/users/User1@org4.dev.com/msp/signcerts/cert.pem ${FIXTURES_DIR}/certs/org4-user-signcert.pem

ln -s ${MSPS_DIR}/Org1MSP/users/User1@org1.dev.com/msp/keystore/$(basename ${MSPS_DIR}/Org1MSP/users/User1@org1.dev.com/msp/keystore/*_sk) ${FIXTURES_DIR}/keys/clients/org1-user-priv_sk
ln -s ${MSPS_DIR}/Org2MSP/users/User1@org2.dev.com/msp/keystore/$(basename ${MSPS_DIR}/Org2MSP/users/User1@org2.dev.com/msp/keystore/*_sk) ${FIXTURES_DIR}/keys/clients/org2-user-priv_sk
ln -s ${MSPS_DIR}/Org3MSP/users/User1@org3.dev.com/msp/keystore/$(basename ${MSPS_DIR}/Org3MSP/users/User1@org3.dev.com/msp/keystore/*_sk) ${FIXTURES_DIR}/keys/clients/org3-user-priv_sk
ln -s ${MSPS_DIR}/Org4MSP/users/User1@org4.dev.com/msp/keystore/$(basename ${MSPS_DIR}/Org4MSP/users/User1@org4.dev.com/msp/keystore/*_sk) ${FIXTURES_DIR}/keys/clients/org4-user-priv_sk

ln -s ${MSPS_DIR}/Org1MSP/users/IbcSigner@org1.dev.com/msp ${FIXTURES_DIR}/msps/Org1MSP
ln -s ${MSPS_DIR}/Org2MSP/users/IbcSigner@org2.dev.com/msp ${FIXTURES_DIR}/msps/Org2MSP
ln -s ${MSPS_DIR}/Org3MSP/users/IbcSigner@org3.dev.com/msp ${FIXTURES_DIR}/msps/Org3MSP
ln -s ${MSPS_DIR}/Org4MSP/users/IbcSigner@org4.dev.com/msp ${FIXTURES_DIR}/msps/Org4MSP
