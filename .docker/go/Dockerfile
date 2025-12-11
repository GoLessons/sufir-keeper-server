FROM golangci/golangci-lint:v2.7.2-alpine AS lint-base
FROM golang:1.23-bookworm

ENV CGO_ENABLED=1
ENV PATH="/usr/local/go/bin:/go/bin:${PATH}"

COPY --from=lint-base /usr/bin/golangci-lint /usr/local/bin/golangci-lint

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

RUN go install golang.org/x/tools/cmd/goimports@v0.28.0 && \
    ln -sf /go/bin/goimports /usr/local/bin/goimports && \
    ln -sf /usr/local/go/bin/go /usr/local/bin/go && \
    ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt && \
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 && \
    ln -sf /go/bin/oapi-codegen /usr/local/bin/oapi-codegen

WORKDIR /workspace
COPY go.mod go.sum /workspace/
RUN cd /workspace && go mod download
