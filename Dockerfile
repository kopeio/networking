FROM golang:1.16 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache and prebuild some dependencies for faster builds.
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v k8s.io/client-go/kubernetes

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY version.go version.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o kopeio-networking-agent ./cmd/networking-agent/

# Use distroless as base image https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/kopeio-networking-agent .
USER 65532:65532

ENTRYPOINT ["/kopeio-networking-agent"]
