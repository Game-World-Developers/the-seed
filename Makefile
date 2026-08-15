GO             ?= go
BINARY         ?= seed
CMD_DIR        ?= ./cmd/seed
BUILD_DIR      ?= build
COVERAGE_OUT   ?= coverage.out
COVERAGE_MIN   ?= 30

.PHONY: all build fmt fmtcheck vet lint test coverage coverage-check tidy clean run install uninstall help ci

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

# -coverpkg=./... is not optional here: most of this repo's coverage
# comes from tests/*.go (black-box tests calling into internal/generators,
# internal/commands, internal/ir, etc.), not from per-package _test.go
# files. Plain `go test ./... -coverprofile=...` only attributes coverage
# to the package under test itself, so every package exercised solely
# through tests/ reports a false 0.0% — this previously made the total
# read ~2.7% instead of the real ~35%. -coverpkg=./... attributes
# coverage to whichever package the executed code actually lives in,
# regardless of which package's test invoked it.
coverage:
	$(GO) test ./... -coverpkg=./... -coverprofile=$(COVERAGE_OUT)
	$(GO) tool cover -func=$(COVERAGE_OUT) | tail -1

# Fails the build if total coverage drops below COVERAGE_MIN (percent).
# Wired into `make ci` so a coverage regression is caught the same way a
# test failure or a vet error already is, not left to be noticed later.
coverage-check: coverage
	@pct=$$($(GO) tool cover -func=$(COVERAGE_OUT) | tail -1 | grep -oE '[0-9]+\.[0-9]+' | head -1); \
	echo "Total coverage: $$pct% (minimum: $(COVERAGE_MIN)%)"; \
	awk -v pct="$$pct" -v min="$(COVERAGE_MIN)" 'BEGIN { exit !(pct >= min) }' || \
		(echo "Coverage $$pct% is below the required $(COVERAGE_MIN)%" && exit 1)

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

ci: tidy fmtcheck vet test coverage-check build

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
	@echo "  coverage    Run tests with a real cross-package coverage report"
	@echo "  coverage-check  coverage, failing the build below COVERAGE_MIN%"
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
