GO           ?= go
BINARY       ?= seed
CMD_DIR      ?= ./cmd/seed
BUILD_DIR    ?= build
COVERAGE_OUT ?= coverage.out

.PHONY: all build fmt fmtcheck vet lint test coverage tidy clean run install uninstall help ci

all: lint test build

build:
	$(GO) build -o $(BINARY) $(CMD_DIR)

fmt:
	$(GO) fmt ./...

fmtcheck:
	@! $(GO) fmt ./... 2>&1 | grep -q '^'
	@echo "All files are properly formatted."

vet:
	$(GO) vet ./...

lint: fmtcheck vet

test:
	$(GO) test ./tests/... -v

coverage: test
	$(GO) test ./... -coverprofile=$(COVERAGE_OUT)
	$(GO) tool cover -func=$(COVERAGE_OUT)

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BINARY) $(BUILD_DIR) $(COVERAGE_OUT)

run: build
	./$(BINARY)

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

ci: tidy fmtcheck vet test build

help:
	@echo "seed Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all         Run lint, test, then build (default)"
	@echo "  build       Build the seed binary"
	@echo "  fmt         Format all Go source files"
	@echo "  fmtcheck    Check formatting (fails if not formatted)"
	@echo "  vet         Run go vet"
	@echo "  lint        Run fmtcheck + vet"
	@echo "  test        Run all tests (verbose)"
	@echo "  coverage    Run tests with coverage report"
	@echo "  tidy        Run go mod tidy"
	@echo "  clean       Remove build artifacts"
	@echo "  run         Build and run seed"
	@echo "  install     Install seed binary"
	@echo "  uninstall   Remove installed binary"
	@echo "  ci          Full CI pipeline (tidy + lint + test + build)"
	@echo ""
	@echo "Variables:"
	@echo "  GO        Go compiler (default: go)"
	@echo "  PREFIX    Install prefix (default: /usr/local)"
	@echo "  DESTDIR   Staging root for packaging"
