APP     := schmournal
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
OUTDIR  := dist

.PHONY: all clean test \
        build-mac-arm build-mac-intel build-mac \
        build-linux-amd64 build-linux-arm64 build-linux \
        build-windows-amd64 build-windows-arm64 build-windows \
        build \
        package-deb-amd64 package-deb-arm64 package-deb

all: build

# ── Tests ──────────────────────────────────────────────────────────────────────
test:
	go test ./...

# ── macOS ─────────────────────────────────────────────────────────────────────
build-mac-arm:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-darwin-arm64 .

build-mac-intel:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-darwin-amd64 .

build-mac: build-mac-arm build-mac-intel

# ── Linux ─────────────────────────────────────────────────────────────────────
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-linux-arm64 .

build-linux: build-linux-amd64 build-linux-arm64

# ── Windows ───────────────────────────────────────────────────────────────────
build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-windows-amd64.exe .

build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(OUTDIR)/$(APP)-windows-arm64.exe .

build-windows: build-windows-amd64 build-windows-arm64

# ── All platforms ─────────────────────────────────────────────────────────────
build: build-mac build-linux build-windows

# ── Package (Linux .deb) ──────────────────────────────────────────────────────
package-deb-amd64: build-linux-amd64
	ARCH=amd64 VERSION=$(VERSION) nfpm package --packager deb --target $(OUTDIR)/

package-deb-arm64: build-linux-arm64
	ARCH=arm64 VERSION=$(VERSION) nfpm package --packager deb --target $(OUTDIR)/

package-deb: package-deb-amd64 package-deb-arm64

# ── Cleanup ───────────────────────────────────────────────────────────────────
clean:
	rm -rf $(OUTDIR)
