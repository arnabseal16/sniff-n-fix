
# Image URL to use all building/pushing image targets
IMG ?= 996752749769.dkr.ecr.us-west-2.amazonaws.com/centurion-listener:latest
NAMESPACE= avondale

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: listener

# Run tests
test:

	go test ./... -coverprofile cover.out --json

# Build manager binary
listener:
	go build -o bin/listener main.go

# ci-build manager binary
ci-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o app/listener main.go

# Run against the configured Kubernetes cluster in current atmos environment
run:
	go run ./main.go --kubeconfig=${HOME}/.kube/config-${ATMOS_ENVIRONMENT_IDENTIFIER} --queuename=${ATMOS_ENVIRONMENT_IDENTIFIER}-datadog-events-queue