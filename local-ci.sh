#!/bin/bash

GO_VERSION="1.22.5"
PROJECT_DIR="server"

confirm_installation() {
  read -p "This script will install the following packages: goimports, golangci-lint, gosec, govulncheck. Do you want to continue? (y/n): " -n 1 -r
  echo  # move to a new line
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Installation aborted."
    exit 1
  fi
}

check_go_version() {
  INSTALLED_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  REQUIRED_GO_VERSION="${GO_VERSION}"

  if [[ "$(printf '%s\n' "$INSTALLED_GO_VERSION" "$REQUIRED_GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_GO_VERSION" ]]; then
    echo "Error: Go version '$REQUIRED_GO_VERSION' or higher is required, but '$INSTALLED_GO_VERSION' is installed."
    exit 1
  fi
}

setup() {
  printf "\n=== Setup ===\n"
  cd $PROJECT_DIR || exit 1
  go mod tidy
  go mod download
  cd - || exit 1
}

lint() {
  printf "\n=== Linting ===\n"
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  cd $PROJECT_DIR || exit 1
  golangci-lint run --timeout=5m
  cd - || exit 1
}

format() {
  printf "\n=== Format Stage ===\n"
  cd $PROJECT_DIR || exit 1
  go install golang.org/x/tools/cmd/goimports@latest
  gofmt -l -w .
  goimports -l -w .
  cd - || exit 1
}

#test() {
#  printf "\n=== Test Stage ===\n"
#  cd $PROJECT_DIR || exit 1
#  check_pkg_config
#  go test -v -coverprofile=coverage.out ./...
#  go install github.com/t-yuki/gocover-cobertura@latest
#  gocover-cobertura < coverage.out > coverage.xml
#  cd - || exit 1
#}

security() {
  printf "\n=== Security Stage ===\n"
  go install github.com/securego/gosec/v2/cmd/gosec@latest
  go install golang.org/x/vuln/cmd/govulncheck@latest
  cd $PROJECT_DIR || exit 1
  gosec ./...
  govulncheck ./...
  cd - || exit 1
}

confirm_installation
check_go_version
setup
lint
format
# test
security

echo "CI Pipeline completed successfully!"