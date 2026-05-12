default: build

LINTER_EXE := golangci-lint
LINTER := $(GOPATH)/bin/$(LINTER_EXE)

GOVULNCHECK_EXE := govulncheck
GOVULNCHECK := $(GOPATH)/bin/$(GOVULNCHECK_EXE)

$(GOVULNCHECK):
	go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: vulncheck
vulncheck: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

$(LINTER):
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin

.PHONY: lint
lint: $(LINTER)
	$(LINTER) run --fix

.PHONY: clean
clean:
	rm -f tflint-ruleset-uw-kafka-config

# builds our binary
.PHONY: build
build: clean
	CGO_ENABLED=0 go build -o tflint-ruleset-uw-kafka-config

.PHONY: install
install: build
	mkdir -p ~/.tflint.d/plugins
	mv ./tflint-ruleset-uw-kafka-config ~/.tflint.d/plugins

.PHONY: mod
mod: clean
	go mod tidy

.PHONY: test
test:
	go test -v --race -cover ./...

.PHONY: all
all: clean $(LINTER) lint test vulncheck build
