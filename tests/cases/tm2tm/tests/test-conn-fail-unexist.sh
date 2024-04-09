#!/bin/bash
source ./setup.source.sh

$RLY tx clients --src-height 2 ibc01
r=$?
if [ $r -ne 0 ]; then
  echo "fail to create client"
  exit 1
fi

cat $CONFIG \
    | jq '.paths.ibc01.src["connection-id"] |= "connection-999"' \
    | jq '.paths.ibc01.dst["connection-id"] |= "connection-999"' \
    > $CONFIG.tmp
mv $CONFIG.tmp $CONFIG

expect <<EOF
set timeout 5
set expect_result 0
spawn $RLY tx connection ibc01
expect {
  "src connection id is given but that connection is not exists" {
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

