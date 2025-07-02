# Makefile for building Koshime with metadata

MODULE=github.com/Jaeiya/koshime/lib

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell powershell -NoProfile -Command "Get-Date -Format o")

ifeq ($(VERSION),)
	BUILD_TAG := $(COMMIT)
	VERSION_STR :=
else
	BUILD_TAG := $(VERSION)
	VERSION_STR := $(VERSION)
endif

OUTPUT=dist/koshime
ARCHIVE_OUTPUT=dist/koshime-$(BUILD_TAG)

.PHONY: all build archive

all: build archive
	@echo
	@echo "Version: $(VERSION)"
	@echo " Commit: $(COMMIT)"
	@echo "   Date: $(DATE)"
	@echo
	@echo "✅ Built binary: $(OUTPUT).exe"

build:
	@echo "🔨 Building binary..."
	@go build -ldflags="-s -w -X $(MODULE).Version=$(VERSION_STR) -X $(MODULE).CommitHash=$(COMMIT) -X $(MODULE).BuildDate=$(DATE)" -trimpath -o $(OUTPUT).exe
	@if [ $$? -ne 0 ]; then echo "❌ Build failed."; exit 1; fi

archive:
	@command -v 7z >/dev/null 2>&1 || { echo "⚠️ 7z not found in PATH. Skipping packaging."; exit 0; }
	@if [ -f "$(ARCHIVE_OUTPUT).7z" ]; then rm -f "$(ARCHIVE_OUTPUT).7z"; fi
	@echo "📦 Creating archive $(ARCHIVE_OUTPUT).7z ..."
	@7z a -t7z -mx=7 "$(ARCHIVE_OUTPUT).7z" "$(OUTPUT).exe" >/dev/null
	@if [ $$? -ne 0 ]; then echo "❌ 7z packaging failed."; exit 1; fi
	@echo "✅ Built Archive: $(ARCHIVE_OUTPUT).7z"
