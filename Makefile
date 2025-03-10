DOCKER := $(shell which docker)

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --user 0 --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

.PHONY: build
build:
	go build -o ./build/yrly .

TESTMOCKS = core/strategies_testmock.go core/provers_testmock.go core/chain_testmock.go core/headers_testmock.go
.PHONY: test
test: $(TESTMOCKS)
	go test -v ./...

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-update-deps:
	@echo "Updating Protobuf dependencies"

$(TESTMOCKS):
	for f in $@; do mockgen -source `echo $$f | sed -e s/_testmock//g` -destination $$f -package core; done

.PHONY: proto-gen proto-update-deps
