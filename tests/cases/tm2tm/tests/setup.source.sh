rm -rf $HOME/.yui-relayer
DIR=$(dirname $BASH_SOURCE)
CONFIG=$HOME/.yui-relayer/config/config.json

cd $DIR/..
./scripts/fixture
./scripts/init-rly

source $DIR/../../scripts/util

SCRIPT_DIR=$DIR/scripts
RLY_BINARY=${SCRIPT_DIR}/../../../../build/yrly
RLY="${RLY_BINARY} --debug"

CHAINID_ONE=ibc0
RLYKEY=testkey
CHAINID_TWO=ibc1
PATH_NAME=ibc01

$RLY tendermint keys show $CHAINID_ONE $RLYKEY
$RLY tendermint keys show $CHAINID_TWO $RLYKEY

# initialize the light client for {{chain_id}}
retry 5 $RLY tendermint light init $CHAINID_ONE -f
retry 5 $RLY tendermint light init $CHAINID_TWO -f

# you should see a balance for the rly key now
# $RLY q bal $CHAINID_ONE
# $RLY q bal $CHAINID_TWO

# add a path between chain0 and chain1
$RLY paths add $CHAINID_ONE $CHAINID_TWO $PATH_NAME --file=${SCRIPT_DIR}/../configs/path.json

