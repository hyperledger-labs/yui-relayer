#!/bin/bash
source ./setup.source.sh

$RLY tx clients --src-height 2 ibc01
r=$?
if [ $r -ne 0 ]; then
  echo "fail to create client"
  exit 1
fi

$RLY tx connection ibc01
r=$?

if [ $r -eq 0 ]; then
    echo "success"
    exit 0
else
    echo "fail"
    exit 1
fi

cat $CONFIG \
    | jq '.paths.ibc01.src["chan-id"] |= "ibc999"' \
    | jq '.paths.ibc01.dst["chan-id"] |= "ibc999"' \
    > $CONFIG.tmp
mv $CONFIG.tmp $CONFIG

expect <<EOF
set timeout 5
set expect_result 0
spawn $RLY tx channel ibc01
expect {
  "src channel id is given but that channel is not exists" {
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

