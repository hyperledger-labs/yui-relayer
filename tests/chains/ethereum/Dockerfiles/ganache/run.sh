#!/bin/sh

node "/app/ganache-core.docker.cli.js" \
  --chainId $CHAINID \
  --networkId $NETWORKID \
  --db /root/.ethereum \
  --defaultBalanceEther 10000 \
  --mnemonic "math razor capable expose worth grape metal sunset metal sudden usage scheme" \
  $@
