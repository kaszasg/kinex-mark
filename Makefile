APP         := kinex-mark
CMD         := ./cmd/kinex-mark
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.version=$(VERSION)
GOBUILD     := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"
TARGETS     := linux/amd64 linux/arm64 windows/amd64 darwin/arm64
RACE        ?= -race
COVER_TOTAL ?= 85
COVER_PKG   ?= 80
NODE_DEPS   := web/node_modules/.package-lock.json
WEB_INDEX   := web/build/index.html
GOFILES     := cmd internal tools $(wildcard web/*.go)

.PHONY: all tidy web build run cross check test test-go test-web test-e2e e2e-install verify graphics image binary-image clean

all: build

tidy:
	go mod tidy

web:
	cd web && npm ci && npm run build

# Installs the web dependencies only when the lockfile changed.
$(NODE_DEPS): web/package-lock.json
	cd web && npm ci

# go:embed needs web/build; build the UI once if it is missing.
$(WEB_INDEX):
	$(MAKE) web

build: web
	$(GOBUILD) -o dist/$(APP) $(CMD)

run: build
	./dist/$(APP)

cross: web
	@for target in $(TARGETS); do \
		os=$${target%/*}; arch=$${target#*/}; ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		echo "$$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o dist/$(APP)-$$os-$$arch$$ext $(CMD) || exit 1; \
	done

check: $(NODE_DEPS) $(WEB_INDEX)
	cd web && npm run check && npm run lint
	@unformatted="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go run ./tools/langcheck

test: test-go test-web

test-go: $(WEB_INDEX)
	go test $(RACE) -covermode=atomic -coverprofile=coverage.out ./...
	go run ./tools/covercheck -profile coverage.out -total $(COVER_TOTAL) -package $(COVER_PKG)

test-web: $(NODE_DEPS)
	cd web && npm run test:coverage

e2e-install: $(NODE_DEPS)
	cd web && npx playwright install chromium

test-e2e: build
	cd web && KINEX_MARK_BIN="$(CURDIR)/dist/$(APP)" npm run test:e2e

verify: check test test-e2e

# Generates only the images that are still missing (a regeneration is a paid call: use --force by hand).
graphics:
	python3 graphics/generate_graphics.py

image:
	docker build --target runtime --build-arg VERSION=$(VERSION) -t $(APP):$(VERSION) .

# Builds the Linux binary with Docker only, without a local Go or Node.
binary-image:
	docker build --target binary --build-arg VERSION=$(VERSION) --output type=local,dest=dist .

clean:
	rm -rf dist web/build web/.svelte-kit web/coverage web/test-results web/playwright-report web/blob-report coverage.out coverage.html
