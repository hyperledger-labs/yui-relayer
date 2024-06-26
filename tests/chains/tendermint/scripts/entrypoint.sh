#!/bin/sh

export IBC_AUTHORITY=`jq -r .address /root/${CHAINDIR}/${CHAINID}/key_seed.json`

simd --home /root/${CHAINDIR}/${CHAINID} start --pruning=nothing --grpc.address="0.0.0.0:${GRPCPORT}"
