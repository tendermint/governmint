.PHONY: all test get_deps

all: test install

install: get_deps
	go install github.com/tendermint/governmint/cmd/...

test:
	go test github.com/tendermint/governmint/...

get_deps:
	go get -d github.com/tendermint/governmint/...
