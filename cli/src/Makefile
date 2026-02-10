BINARY := nebula
BUILD_DIR := ./build

.PHONY: build run test clean install fmt lint

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/nebula/...

run:
	go run ./cmd/nebula/...

test:
	go test ./... -v

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './build/*')

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "installed $(BINARY) to /usr/local/bin/$(BINARY)"
