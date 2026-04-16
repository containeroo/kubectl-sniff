# Makefile

GO ?= go
LOCALBIN ?= $(CURDIR)/bin
BINARY_NAME ?= kubectl-sniff
BINARY_PATH ?= $(LOCALBIN)/$(BINARY_NAME)
INSTALL_DIR ?= $(or $(shell $(GO) env GOBIN),$(shell $(GO) env GOPATH)/bin)
RUN_ARGS ?= --help

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.11.4

# Tagging
VERSION_PREFIX ?= v
LATEST_TAG := $(shell git tag --list '$(VERSION_PREFIX)*' --sort=-v:refname | head -n 1)
CURRENT_VERSION := $(if $(LATEST_TAG),$(patsubst $(VERSION_PREFIX)%,%,$(LATEST_TAG)),0.0.0)
NEXT_PATCH := $(shell echo "$(CURRENT_VERSION)" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3 + 1}')
NEXT_MINOR := $(shell echo "$(CURRENT_VERSION)" | awk -F. '{printf "%d.%d.0", $$1, $$2 + 1}')
NEXT_MAJOR := $(shell echo "$(CURRENT_VERSION)" | awk -F. '{printf "%d.0.0", $$1 + 1}')


$(LOCALBIN):
	mkdir -p $(LOCALBIN)

##@ Development

.PHONY: run
run: ## Run the CLI locally. Override with RUN_ARGS='...'.
	$(GO) run . $(RUN_ARGS)

.PHONY: build
build: $(LOCALBIN) ## Build the kubectl plugin binary into ./bin.
	$(GO) build -o $(BINARY_PATH) .

.PHONY: install
install: ## Install the kubectl plugin binary into Go's bin directory.
	mkdir -p $(INSTALL_DIR)
	$(GO) build -o $(INSTALL_DIR)/$(BINARY_NAME) .

.PHONY: fmt
fmt: ## Format Go code.
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: fmt vet ## Run unit tests.
	$(GO) test -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...

.PHONY: cover
cover: ## Write coverage artifacts to coverage.out and coverage.html.
	$(GO) test -coverprofile=coverage.out -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: clean
clean: ## Remove build and coverage artifacts.
	rm -f coverage.out coverage.html $(BINARY_PATH)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint with autofixes.
	$(GOLANGCI_LINT) run --fix

##@ Tagging

.PHONY: tag tag-patch tag-minor tag-major push-tags
tag-patch: ## Create the next patch tag locally.
	@git tag -a "$(VERSION_PREFIX)$(NEXT_PATCH)" -m "Release $(VERSION_PREFIX)$(NEXT_PATCH)"
	@echo "Created tag $(VERSION_PREFIX)$(NEXT_PATCH)"

tag-minor: ## Create the next minor tag locally.
	@git tag -a "$(VERSION_PREFIX)$(NEXT_MINOR)" -m "Release $(VERSION_PREFIX)$(NEXT_MINOR)"
	@echo "Created tag $(VERSION_PREFIX)$(NEXT_MINOR)"

tag-major: ## Create the next major tag locally.
	@git tag -a "$(VERSION_PREFIX)$(NEXT_MAJOR)" -m "Release $(VERSION_PREFIX)$(NEXT_MAJOR)"
	@echo "Created tag $(VERSION_PREFIX)$(NEXT_MAJOR)"

push-tags: ## Push commits and tags to origin.
	@git push --follow-tags

tag: ## Show latest tag.
	@echo "Latest version: $(if $(LATEST_TAG),$(LATEST_TAG),none (next: $(VERSION_PREFIX)$(NEXT_PATCH)))"


##@ Dependencies

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.

$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' a tool to a specific binary path if it is missing.
# $1 - target path with binary name
# $2 - package path
# $3 - specific version
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) $(GO) install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
