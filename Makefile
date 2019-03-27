# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ionc android ios ionc-cross swarm evm all test clean
.PHONY: ionc-linux ionc-linux-386 ionc-linux-amd64 ionc-linux-mips64 ionc-linux-mips64le
.PHONY: ionc-linux-arm ionc-linux-arm-5 ionc-linux-arm-6 ionc-linux-arm-7 ionc-linux-arm64
.PHONY: ionc-darwin ionc-darwin-386 ionc-darwin-amd64
.PHONY: ionc-windows ionc-windows-386 ionc-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

ionc:
	build/env.sh go run build/ci.go install ./cmd/ionc
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ionc\" to launch ionc."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/ionc.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

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
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ionc
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep amd64

ionc-linux-arm: ionc-linux-arm-5 ionc-linux-arm-6 ionc-linux-arm-7 ionc-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm

ionc-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ionc
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-5

ionc-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ionc
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-6

ionc-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ionc
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm-7

ionc-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ionc
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep arm64

ionc-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips

ionc-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mipsle

ionc-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips64

ionc-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ionc
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ionc-linux-* | grep mips64le

ionc-darwin: ionc-darwin-386 ionc-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-*

ionc-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ionc
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-* | grep 386

ionc-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ionc
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-darwin-* | grep amd64

ionc-windows: ionc-windows-386 ionc-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-*

ionc-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ionc
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-* | grep 386

ionc-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ionc
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ionc-windows-* | grep amd64
