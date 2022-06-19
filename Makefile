# Placeholder environment variables. These should be present when building
# and pushing new docker images within the context of github actions.
# Target version
VERSION ?= 0.1.6
# Target docker image URL for building/pushing actions.
IMAGE ?= storageos/metrics-exporter:${VERSION}


# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GOBIN = $(shell pwd)/bin
CONTROLLER_GEN = $(GOBIN)/controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN):
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: lint
lint:
	golangci-lint run --config .github/linters-cfg/.golangci.yml

.PHONY: test
test:
	go test ./...

.PHONY: run
run:
	go run .

##@ Build

.PHONY: build
build:
	go build -o bin/metrics-exporter .

.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object paths=./api/config.storageos.com/...

.PHONY: bundle
bundle:
	kustomize build manifests > manifests/bundle.yaml

.PHONY: docker-build
docker-build:
	docker build -t ${IMAGE} --build-arg VERSION=$(VERSION) .

##@ Publish

.PHONY: docker-push
docker-push:
	docker push ${IMAGE}
