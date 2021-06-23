#!/usr/bin/env bash

set -eu

# https://docs.docker.com/storage/volumes/#backup-restore-or-migrate-data-volumes
# https://stackoverflow.com/questions/46349071/commit-content-of-mounted-volumes-as-well

docker run \
--rm \
--volumes-from fabric-orderer.example.com \
-v $(pwd)/backup/orderer/ledger:/backup/orderer/ledger \
ubuntu tar cvf /backup/orderer/ledger/backup.tar /var/hyperledger

docker run \
--rm \
--volumes-from fabric-peer0.org1.example.com \
-v $(pwd)/backup/peer0-org1/ledger:/backup/peer0-org1/ledger \
ubuntu tar cvf /backup/peer0-org1/ledger/backup.tar /var/hyperledger

docker run \
--rm \
--volumes-from fabric-peer0.org2.example.com \
-v $(pwd)/backup/peer0-org2/ledger:/backup/peer0-org2/ledger \
ubuntu tar cvf /backup/peer0-org2/ledger/backup.tar /var/hyperledger

