VETARGS?=-all

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

BUILD_DIR ?= $(abspath build)

$(BUILD_DIR):
	mkdir -p $@

GOBUILD ?= $(shell go env GOOS)-$(shell go env GOARCH)

GET_GOOS   = $(word 1,$(subst -, ,$1))
GET_GOARCH = $(word 2,$(subst -, ,$1))

BINARIES = $(addprefix $(BUILD_DIR)/datadog-service-helper-, $(GOBUILD))
$(BINARIES): $(BUILD_DIR)/datadog-service-helper-%: $(BUILD_DIR)
	@echo "=> building $@ ..."
	GOOS=$(call GET_GOOS,$*) GOARCH=$(call GET_GOARCH,$*) CGO_ENABLED=0 govendor build -o $@

.PHONY: install
install:
	go get -u github.com/kardianos/govendor

	@echo "=> govendor sync..."
	govendor sync

.PHONY: fmt
fmt:
	@echo "=> Running go fmt" ;
	@if [ -n "`go fmt ${GOFILES_NOVENDOR}`" ]; then \
		echo "[ERR] go fmt updated formatting. Please commit formatted code first."; \
		exit 1; \
	fi

.PHONY: vet
vet: fmt
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi

	@echo "=> Running go tool vet $(VETARGS) ${GOFILES_NOVENDOR}"
	@go tool vet $(VETARGS) ${GOFILES_NOVENDOR} ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "[LINT] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
	fi

.PHONY: build
build: install fmt vet
	@echo "=> building backend ..."
	$(MAKE) -j $(BINARIES)

.PHONY: rebuild
rebuild: clean
	@echo "=> rebuilding backend ..."
	$(MAKE) -j build

.PHONY: clean
clean:
	@echo "=> cleaning backend ..."
	rm -rf $(BUILD_DIR)

.PHONY: dist-clean
dist-clean: clean
	@echo "=> dist-cleaning backend ..."
	rm -rf vendor/{github.com,golang.org,gopkg.in}

test:
	@echo "==> Running $@..."
	@go test -v -tags $(shell go list ./... | grep -v vendor)
