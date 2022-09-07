.PHONY: build
build:
	go build -o ./build/yrly .

.PHONY: test
test:
	go test -v ./...

.PHONY: proto-gen
proto-gen:
	@echo "Generating Protobuf files"
	docker run -v $(CURDIR):/workspace --workdir /workspace tendermintdev/sdk-proto-gen:v0.3 sh ./scripts/protocgen.sh
