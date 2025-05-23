.DEFAULT_GOAL := build

# globals
BINARY_NAME?=tracelens-cli
BUILD_DIR?="./build"
CGO_ENABLED?=0
COMMIT?=$(shell git rev-parse --short HEAD)
DATE?=$(shell date -u '+%Y-%m-%dT%H:%M:%S %Z')
REPO=github.com/optiflow-os/tracelens-cli
VERSION?=<local-build>

# ld flags for go build
LD_FLAGS=-s -w -X '${REPO}/pkg/version.BuildDate=${DATE}' -X ${REPO}/pkg/version.Commit=${COMMIT} -X ${REPO}/pkg/version.Version=${VERSION}

# basic Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# linting
define get_latest_lint_release
	curl -s "https://api.github.com/repos/golangci/golangci-lint/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
endef
LATEST_LINT_VERSION=$(shell $(call get_latest_lint_release))
INSTALLED_LINT_VERSION=$(shell golangci-lint --version 2>/dev/null | awk '{print "v"$$4}')

# get GOPATH, GOOS and GOARCH according to OS
ifeq ($(OS),Windows_NT) # is Windows_NT on XP, 2000, 7, Vista, 10...
    GOPATH=$(go env GOPATH)
	GOOS=$(shell cmd /c go env GOOS)
	GOARCH=$(shell cmd /c go env GOARCH)
else
    GOPATH=$(shell go env GOPATH)
	GOOS=$(shell go env GOOS)
	GOARCH=$(shell go env GOARCH)
endif

# targets
build-all: build-all-android build-darwin build-freebsd build-linux build-netbsd build-openbsd build-windows

build-all-android: build-android-arm build-android-arm64

# to build for android arm, you need to have the android ndk installed, enable CGO and
# set CC to the path of the android ndk toolchain
# example: CC=/path/to/Android/sdk/ndk/26.0.10792818/toolchains/llvm/prebuilt/darwin-x86_64/bin/armv7a-linux-androideabi34-clang
build-android-arm:
	GOOS=android GOARCH=arm CGO_ENABLED=1 $(MAKE) build

build-android-arm64:
	GOOS=android GOARCH=arm64 $(MAKE) build

build-all-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) build

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) build

build-all-freebsd: build-freebsd-386 build-freebsd-amd64 build-freebsd-arm

build-freebsd-386:
	GOOS=freebsd GOARCH=386 $(MAKE) build

build-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 $(MAKE) build

build-freebsd-arm:
	GOOS=freebsd GOARCH=arm $(MAKE) build

build-all-linux: build-linux-386 build-linux-amd64 build-linux-arm build-linux-arm64 build-linux-riscv64

build-linux-386:
	GOOS=linux GOARCH=386 $(MAKE) build

build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) build

build-linux-arm:
	GOOS=linux GOARCH=arm $(MAKE) build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) build

build-linux-riscv64:
	GOOS=linux GOARCH=riscv64 $(MAKE) build

build-all-netbsd: build-netbsd-386 build-netbsd-amd64 build-netbsd-arm

build-netbsd-386:
	GOOS=netbsd GOARCH=386 $(MAKE) build

build-netbsd-amd64:
	GOOS=netbsd GOARCH=amd64 $(MAKE) build

build-netbsd-arm:
	GOOS=netbsd GOARCH=arm $(MAKE) build

build-all-openbsd: build-openbsd-386 build-openbsd-amd64 build-openbsd-arm build-openbsd-arm64

build-openbsd-386:
	GOOS=openbsd GOARCH=386 $(MAKE) build

build-openbsd-amd64:
	GOOS=openbsd GOARCH=amd64 $(MAKE) build

build-openbsd-arm:
	GOOS=openbsd GOARCH=arm $(MAKE) build

build-openbsd-arm64:
	GOOS=openbsd GOARCH=arm64 $(MAKE) build

build-all-windows: build-windows-386 build-windows-amd64 build-windows-arm64

build-windows-386:
	GOOS=windows GOARCH=386 $(MAKE) build-windows

build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(MAKE) build-windows

build-windows-arm64:
	GOOS=windows GOARCH=arm64 $(MAKE) build-windows

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -v \
		-ldflags "${LD_FLAGS} -X ${REPO}/pkg/version.OS=$(GOOS) -X ${REPO}/pkg/version.Arch=$(GOARCH)" \
		-o ${BUILD_DIR}/$(BINARY_NAME)-$(GOOS)-$(GOARCH)

.PHONY: build-windows
build-windows:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -v \
		-ldflags "${LD_FLAGS} -X ${REPO}/pkg/version.OS=$(GOOS) -X ${REPO}/pkg/version.Arch=$(GOARCH)" \
		-o ${BUILD_DIR}/$(BINARY_NAME)-$(GOOS)-$(GOARCH).exe

install: install-go-modules install-linter

.PHONY: install-linter
install-linter:
ifneq "$(INSTALLED_LINT_VERSION)" "$(LATEST_LINT_VERSION)"
	@echo "new golangci-lint version found:" $(LATEST_LINT_VERSION)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin latest
endif

.PHONY: install-go-modules
install-go-modules:
	go mod vendor

# run static analysis tools, configuration in ./.golangci.yml file
.PHONY: lint
lint: install-linter
	golangci-lint run ./...

.PHONY: vulncheck
vulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	./bin/govulncheck-with-excludes.sh ./...

.PHONY: test
test:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: test-integration
test-integration:
	go test -race -tags=integration ./main_test.go

.PHONY: test-ip
test-ip:
	go test -race -tags=ip ./ip_test.go

.PHONY: test-shell-script
test-shell-script:
	bats ./bin/tests

test-all: lint test test-integration test-shell-script test-ip
