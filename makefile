# ---- Variables ----
APP_NAME := koshime
VERSION  := $(shell git describe --tags --exact-match 2>/dev/null)
COMMIT   := $(shell git rev-parse --short HEAD)
DISPLAY_VERSION := $(if $(VERSION),$(VERSION),No Tag)
LDFLAGS  := -ldflags="-X 'github.com/Jaeiya/koshime/lib.Version=$(VERSION)' -X 'github.com/Jaeiya/koshime/lib.CommitHash=$(COMMIT)'"

.PHONY: all build install clean

all: build

build:
	@echo "🔨 Building $(APP_NAME)"
	@echo "   Version: $(DISPLAY_VERSION)"
	@echo "   Commit:  $(COMMIT)"
	@go build $(LDFLAGS) -o $(APP_NAME) ./...

install: tidy
	@echo "📦 Installing $(APP_NAME)"
	@echo "   Version: $(DISPLAY_VERSION)"
	@echo "   Commit:  $(COMMIT)"
	@go install $(LDFLAGS) ./...

tidy:
	@echo "🧩 Running go mod tidy"
	@go mod tidy

clean:
	@echo "🧹 Cleaning"
	@go clean
	@rm -f $(APP_NAME)
