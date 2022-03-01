# Build the operator binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the Operator go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Copy the Listener go source
COPY centurion-listener/go.mod centurion-listener/go.mod
COPY centurion-listener/go.sum centurion-listener/go.sum
COPY centurion-listener/main.go centurion-listener/main.go
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o operator main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o listener centurion-listener/main.go

# Use distroless as minimal base image to package the operator binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/operator .
COPY --from=builder /workspace/listener .
USER 65532:65532

LABEL org.opencontainers.image.title="centurion-operator"
LABEL org.opencontainers.image.authors="pod-avondale"
LABEL org.opencontainers.image.url="https://stash.sniff-n-fix.com/projects/CCS/repos/centurion-operator/browse"
LABEL org.opencontainers.image.description="Operator for taking action on Datadog events"
LABEL org.opencontainers.image.ref.name="centurion"
LABEL org.opencontainers.image.vendor="sniff-n-fix"
ENTRYPOINT ["/operator"]
