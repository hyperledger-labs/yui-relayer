#!/bin/bash
source ./setup.source.sh

$RLY tx clients --src-height 2 ibc01

r=$?

if [ $r -eq 0 ]; then
    echo "success"
    exit 0
else
    echo "fail"
    exit 1
fi

