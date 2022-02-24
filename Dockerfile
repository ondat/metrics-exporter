# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /workspace

# Copy the source code
COPY . .

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o metrics-exporter .

FROM debian:latest
WORKDIR /
COPY --from=builder /workspace/metrics-exporter .

ENTRYPOINT ["/metrics-exporter"]
