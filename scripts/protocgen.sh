#!/usr/bin/env bash

set -eo pipefail

protoc_gen_gocosmos() {
  if ! grep "github.com/gogo/protobuf => github.com/regen-network/protobuf" go.mod &>/dev/null ; then
    echo -e "\tPlease run this command from somewhere inside the cosmos-sdk folder."
    return 1
  fi

  go get github.com/regen-network/cosmos-proto/protoc-gen-gocosmos 2>/dev/null
}

protoc_gen_gocosmos

proto_dirs=$(find ./proto -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  buf protoc \
  -I "proto" \
  -I "third_party/proto" \
  --gocosmos_out=plugins=interfacetype+grpc,\
Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:. \
  $(find "${dir}" -maxdepth 1 -name '*.proto')

  # command to generate gRPC gateway (*.pb.gw.go in respective modules) files
  buf protoc \
  -I "proto" \
  -I "third_party/proto" \
  --grpc-gateway_out=logtostderr=true:. \
  $(find "${dir}" -maxdepth 1 -name '*.proto')

done

buf protoc \
	-I third_party/proto \
	--gocosmos_out=plugins=interfacetype+grpc,Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:. \
	$(find ./third_party/proto/jp -type f -name '*.proto')
rm -rf ./jp_datachain_corda_ibc_grpc
mkdir ./jp_datachain_corda_ibc_grpc
mv jp/datachain/corda/ibc/grpc/*.go jp_datachain_corda_ibc_grpc
rmdir -p jp/datachain/corda/ibc/grpc

cp -r github.com/datachainlab/relayer/* ./
rm -rf github.com
