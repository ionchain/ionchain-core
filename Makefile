# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ionc android ios ionc-cross swarm evm all test clean
.PHONY: ionc-linux ionc-linux-386 ionc-linux-amd64 ionc-linux-mips64 ionc-linux-mips64le
.PHONY: ionc-linux-arm ionc-linux-arm-5 ionc-linux-arm-6 ionc-linux-arm-7 ionc-linux-arm64
.PHONY: ionc-darwin ionc-darwin-386 ionc-darwin-amd64
.PHONY: ionc-windows ionc-windows-386 ionc-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

ionc:
	$(GORUN) build/ci.go install ./cmd/ionc
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ionc\" to launch ionc."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."
	@echo "Import \"$(GOBIN)/geth-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"
	
ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

ionc-cross: ionc-linux ionc-darwin ionc-windows ionc-android ionc-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/ionc-*

ionc-linux: ionc-linux-386 ionc-linux-amd64 ionc-linux-arm ionc-linux-mips64 ionc-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-*

ionc-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/ionc
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep 386

ionc-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ionc
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep amd64

ionc-linux-arm: ionc-linux-arm-5 ionc-linux-arm-6 ionc-linux-arm-7 ionc-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm

ionc-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ionc
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-5

ionc-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ionc
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-6

ionc-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ionc
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-7

ionc-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ionc
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-linux-* | grep arm64

ionc-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips

ionc-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/geth
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mipsle

ionc-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips64

ionc-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips64le

ionc-darwin: ionc-darwin-386 ionc-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-*

ionc-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ionc
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-* | grep 386

ionc-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ionc
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-* | grep amd64

ionc-windows: ionc-windows-386 ionc-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-*

ionc-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ionc
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-* | grep 386

ionc-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ionc
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-* | grep amd64
