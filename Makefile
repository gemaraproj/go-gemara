# Makefile for go-gemara
# Purpose: provide convenient local targets for building, testing and basic CI parity.
# NOTE: Some tasks (CUE generation, repo-level tidy) are delegated to the repo root. See TODOs.

REPO_ROOT := $(shell git rev-parse --show-toplevel)
PKGS := ./...
BINDIR := ./bin
# Binaries to build (paths to cmd packages)
BINS := ./cmd/oscal_export ./cmd/types_tagger
GOFLAGS :=
COVERFILE := coverage.out
TESTCOVERAGE_THRESHOLD := 71
GOLANGCI_LINT := golangci-lint

.PHONY: all tidy fmtcheck fmt vet lint test testcov race coverage-check build install generate ci-local clean help

# Default target
all: tidy fmtcheck vet lint testcov build
	@echo "All done."

# Run `go mod tidy` at repo root (required because module files live at repo root)
tidy:
	@echo " > Tidying module at repo root ($(REPO_ROOT))"
	@cd $(REPO_ROOT) && go mod tidy

# Formatting check (fail if any files are unformatted)
fmtcheck:
	@echo " > Checking gofmt"
	@sh -c "test -z \"$$(gofmt -l .)\" || (echo 'gofmt -l found non-formatted files:' && gofmt -l . && exit 1)"

# Apply gofmt (destructive)
fmt:
	@echo " > Formatting files with gofmt"
	@gofmt -w .

vet:
	@echo " > Running go vet"
	@go vet $(PKGS)

lint:
	@echo " > Running golangci-lint (requires $(GOLANGCI_LINT) to be installed)"
	@$(GOLANGCI_LINT) run ./...

# Run unit tests
test:
	@echo " > Running go test"
	@go test $(GOFLAGS) $(PKGS)

# Run tests and write coverage
testcov:
	@echo " > Running tests with coverage"
	@go test $(GOFLAGS) $(PKGS) -coverprofile=$(COVERFILE) -covermode=count
	@echo " > Coverage summary:"
	@go tool cover -func=$(COVERFILE) | grep total || true

race:
	@echo " > Running tests with race detector"
	@go test -race $(PKGS)

# Check coverage threshold (requires testcov)
coverage-check:
	@echo " > Checking coverage threshold ($(TESTCOVERAGE_THRESHOLD)%)"
	@sh -c '\
	if [ ! -f "$(COVERFILE)" ]; then \
	  echo "$(COVERFILE) not found; run make testcov first"; exit 1; \
	fi; \
	cov=$$(go tool cover -func=$(COVERFILE) | awk '/total/ {gsub("%","",$$3); print $$3}'); \
	awk -v cov=$$cov -v th=$(TESTCOVERAGE_THRESHOLD) 'BEGIN { if ((cov+0) < (th+0)) { print "Coverage "cov"% is below threshold "th"%"; exit 1 } else { print "Coverage "cov"% meets threshold "th"%" } }' ; \
'

# Build CLI binaries listed in BINS
build:
	@echo " > Building binaries to $(BINDIR)"
	@mkdir -p $(BINDIR)
	@for b in $(BINS); do \
	  bn=$$(basename $$b); \
	  echo "  - building $$b -> $(BINDIR)/$$bn"; \
	  go build -o $(BINDIR)/$$bn $$b || exit 1; \
	done

# Install package/binaries (simple wrapper)
install:
	@echo " > Installing module/binaries"
	@go install ./...

# Generate files (delegates to repo root cuegen). TODO: consider replicating or making a local generator
generate:
	@echo " > Delegating generation to repo root (make cuegen). Ensure generated artifacts are committed."
	@cd $(REPO_ROOT) && make cuegen || (echo "Warning: repo-level cuegen failed or is not present; update generate target" && exit 1)

# Runs the small subset used by CI for a quick local check
ci-local: fmtcheck vet lint generate testcov coverage-check
	@echo "CI-local checks complete"

clean:
	@echo " > Cleaning build artifacts"
	@rm -rf $(BINDIR) $(COVERFILE) *.coverprofile

help:
	@echo "make targets:"
	@echo "  all            - tidy -> fmtcheck -> vet -> lint -> testcov -> build"
	@echo "  tidy           - run 'go mod tidy' at repo root"
	@echo "  fmtcheck       - fail if formatting issues found"
	@echo "  fmt            - run gofmt -w"
	@echo "  vet            - go vet"
	@echo "  lint           - golangci-lint run ./... (needs tool installed)"
	@echo "  test           - go test"
	@echo "  testcov        - go test -coverprofile=$(COVERFILE)"
	@echo "  coverage-check - ensure coverage >= $(TESTCOVERAGE_THRESHOLD)%"
	@echo "  build          - build binaries listed in BINS -> $(BINDIR)"
	@echo "  generate       - runs 'make cuegen' at repo root (TODO: consider implementing locally)"
	@echo "  ci-local       - run quick CI-like checks (fmtcheck vet lint generate testcov coverage-check)"
	@echo "  clean          - remove build artifacts"

# TODOs / notes:
# - Consider adding staticcheck or a separate 'staticcheck' target if desired.
# - The 'generate' target delegates to repo-level 'make cuegen'. If you prefer local generation without touching the repo root, implement the steps here and ensure they produce the same outputs.
# - golangci-lint must be installed locally for 'lint' to succeed; CI uses the golangci-lint GitHub Action.
# - Coverage parsing uses 'awk' and should work on macOS; if you have a different shell environment, adjust accordingly.
# - If you want to add a 'vet' or 'lint' subset, add variables to configure packages.
