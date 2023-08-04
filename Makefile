GO ?= latest
GOPATH := $(or $(GOPATH), $(shell go env GOPATH))
GORUN = env GOPATH=$(GOPATH) GO111MODULE=on go run

BIN = $(shell pwd)/build/bin
CONF = $(shell pwd)/build/conf/guardian.yaml

gdn:
	go build -o $(BIN)/guardian guardian/main.go

debug:
	go build -o $(BIN)/guardian -gcflags="-N -l" guardian/main.go

run:
	$(BIN)/guardian --conf $(CONF)