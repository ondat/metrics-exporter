# Build the manager binary
FROM golang:1.17.3 as builder

WORKDIR /workspace
COPY go.mod go.sum ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o metrics-exporter .

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /
ARG VERSION=v0.0.1
LABEL name="Ondat Metrics Exporter" \
    vendor="Ondat" \
    url="https://docs.ondat.com" \
    maintainer="support@ondat.com" \
    version="${VERSION}" \
    distribution-scope="public" \
    architecture="x86_64" \
    summary="The Ondat Metrics Exporter exports metrics of all Ondat volumes on the node" \
    io.k8s.display-name="Ondat Metrics Exporter" \
    io.k8s.description="The Ondat Metrics Exporter exports metrics of all Ondat volumes on the node" \
    io.openshift.tags=""
COPY --from=builder /workspace/metrics-exporter .
COPY --from=builder /workspace/index.html .

ENTRYPOINT ["/metrics-exporter"]
