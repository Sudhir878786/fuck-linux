BINARY     = fuck-linux
PACKAGE    = fucklinux
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PREFIX    ?= /usr/local
BINDIR     = $(PREFIX)/bin
SOUNDDIR   = /usr/share/$(PACKAGE)/sounds
DOCDIR     = /usr/share/doc/$(PACKAGE)

.PHONY: all build install uninstall clean release

all: build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -dm755 $(DESTDIR)$(SOUNDDIR)
	cp -r sounds/* $(DESTDIR)$(SOUNDDIR)/
	install -Dm644 README.md $(DESTDIR)$(DOCDIR)/README.md
	@echo ""
	@echo "✅ fucklinux installed!"
	@echo "   Run: $(BINARY) --source mic --sound-dir $(SOUNDDIR) --threshold 0.4"
	@echo ""

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -rf $(DESTDIR)$(SOUNDDIR)
	rm -rf $(DESTDIR)$(DOCDIR)

clean:
	rm -f $(BINARY)
	rm -rf dist/

release:
	@which goreleaser > /dev/null || (echo "Install goreleaser: https://goreleaser.com/install/" && exit 1)
	goreleaser release --clean

snapshot:
	@which goreleaser > /dev/null || (echo "Install goreleaser: https://goreleaser.com/install/" && exit 1)
	goreleaser release --snapshot --clean

test:
	go test ./...
