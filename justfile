BIN := justfile_directory() / '.bin'

[private]
default:
    just --list

# ---- dependencies

GOLANGCI_LINT_VERSION := 'v2.11.4'
GOLANGCI_LINT_PATH := BIN / 'golangci-lint'
GOLANGCI_LINT := GOLANGCI_LINT_PATH + '@' + GOLANGCI_LINT_VERSION

[doc('https://github.com/golangci/golangci-lint')]
[private]
install-golangci-lint:
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b {{ BIN }} {{ GOLANGCI_LINT_VERSION }}
    mv {{ GOLANGCI_LINT_PATH }} {{ GOLANGCI_LINT }}

BETTERALIGN_VERSION := 'v0.9.1'
BETTERALIGN_PATH := BIN / 'betteralign'
BETTERALIGN := BETTERALIGN_PATH + '@' + BETTERALIGN_VERSION

[doc('https://github.com/dkorunic/betteralign')]
[private]
install-betteralign:
    GOBIN={{ BIN }} go install github.com/dkorunic/betteralign/cmd/betteralign@{{ BETTERALIGN_VERSION }}
    mv {{ BETTERALIGN_PATH }} {{ BETTERALIGN }}

# ---- end dependencies
# ---- testing

[private]
smoke:
    go test ./...

[private]
cover:
    go test -count 1 -parallel 8 -race -coverprofile=coverage.out ./...
    go tool cover -func coverage.out

# ---- end testing

[doc('Install needed tools and dependencies')]
install: install-golangci-lint install-betteralign

[doc('Run static checkers and formatters')]
code:
    go fix ./...
    {{ BETTERALIGN }} -apply ./...
    {{ GOLANGCI_LINT }} run --path-prefix $(pwd) ./...

[doc('Run tests by type: `smoke` for quick checks, `cover` for detailed analysis')]
test type='smoke':
    just {{ type }}
