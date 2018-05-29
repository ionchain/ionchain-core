.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/ionchain/ionchain-core/cmd/swarm
	go build -o bin/geth     github.com/ionchain/ionchain-core/cmd/geth
	go build -o bin/bootnode github.com/ionchain/ionchain-core/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/ionchain/ionchain-core/swarm/...
