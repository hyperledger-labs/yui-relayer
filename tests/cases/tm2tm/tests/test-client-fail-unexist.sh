#!/bin/bash
source ./setup.source.sh

cat $CONFIG \
    | jq '.paths.ibc01.src["client-id"] |= "07-tendermint-999"' \
    | jq '.paths.ibc01.dst["client-id"] |= "07-tendermint-999"' \
    > $CONFIG.tmp
mv $CONFIG.tmp $CONFIG

expect <<EOF
set timeout 5
set expect_result 0
spawn $RLY tx clients --src-height 2 ibc01
expect {
  "07-tendermint-999: light client not found" {
     set expect_result 1
     exp_continue
  }
  eof
}
catch wait spawn_result
set ryly_result [lindex \$spawn_result 3]
set r [expr \$ryly_result * 100 + \$expect_result ]
exit \$r
EOF

r=$?

if [ $r -eq 101 ]; then
    echo "success"
    exit 0
else
    echo "fail: $r"
    exit 1
fi

