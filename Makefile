GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
BINARY_NAME=lrucache

all: test build

build:
	$(GOCMD) build -o ./bin/$(BINARY_NAME) .

test:
	$(GOTEST) -v ./...

vet:
	$(GOVET) ./...

clean:
	rm -f $(BINARY_NAME)

run:
	$(GOCMD) run main.go

.PHONY: all build test vet clean run
