#!/usr/bin/env bash
set -euo pipefail

go mod download
go mod tidy

gofmt -s -w .
fmt_out=$(gofmt -s -l . || true)
if [ -n "$fmt_out" ]; then
  echo "$fmt_out" && exit 1
fi

MODULE=$(go list -m)
goimports -local "$MODULE" -w .
imports_out=$(goimports -l -local "$MODULE" ./ || true)
if [ -n "$imports_out" ]; then
  echo "$imports_out" && exit 1
fi

go vet ./...

golangci-lint run --config tools/.golangci-lint.yml --timeout 3m --fix ./...
golangci-lint run --config tools/.golangci-lint.yml --timeout 3m ./...

go test ./... -covermode=atomic -coverprofile=var/coverage.out
go tool cover -func=var/coverage.out | tail -n 1

go test -race -count=0 ./...

# Проверка, что сборка приложения проходит успешно
go build -v ./cmd/server
