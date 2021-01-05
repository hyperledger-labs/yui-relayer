.PHONY: build
build:
	go build -o ./build/uly .

.PHONY: test
test:
	go test -v ./...
