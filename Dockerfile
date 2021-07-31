FROM golang:1.16 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache and prebuild some dependencies for faster builds.
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v k8s.io/client-go/kubernetes github.com/vishvananda/netlink

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY version.go version.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o kopeio-networking-agent ./cmd/networking-agent/

# Use distroless as base image https://github.com/GoogleContainerTools/distroless

# Note: we can't use non-root; we need to do privileged network operations
FROM gcr.io/distroless/static:latest
#FROM gcr.io/distroless/static:nonroot
#USER 65532:65532

WORKDIR /
COPY --from=builder /workspace/kopeio-networking-agent .

ENTRYPOINT ["/kopeio-networking-agent"]
