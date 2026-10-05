.PHONY: build test lint vet race clean

BINARY := portwatch
PKG    := ./...
build:
			go build -o $(BINARY) ./cmd/portwatch

test:
			go test $(PKG)

race:
			go test -race $(PKG)

vet:
			go vet $(PKG)

lint:
			golangci-lint run

clean:
			rm -f $(BINARY)