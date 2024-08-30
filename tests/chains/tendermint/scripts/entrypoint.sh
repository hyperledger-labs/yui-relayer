#!/bin/sh

export IBC_AUTHORITY=`jq -r .address /root/${CHAINDIR}/${CHAINID}/key_seed.json`
export IBC_CHANNEL_UPGRADE_TIMEOUT=20000000000 # 20sec = 20_000_000_000nsec

simd --home /root/${CHAINDIR}/${CHAINID} start --pruning=nothing --grpc.address="0.0.0.0:${GRPCPORT}"
