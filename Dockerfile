# Build the manager binary
FROM golang:1.17 as dlvbuilder

# Build delve debugger
RUN apt-get update && apt-get install -y git
RUN go get github.com/go-delve/delve/cmd/dlv

# Build the manager binary
FROM dlvbuilder as builder
ARG GOBUILD_GCFLAGS=""

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY common/ common/
COPY platform/ platform/
COPY controllers/ controllers/
COPY scripts/ scripts/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags "${GOBUILD_GCFLAGS}" -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM scratch as production
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]

# Copy the delve debugger into a debug image
FROM ubuntu:latest as debug
WORKDIR /
RUN apt-get update && apt-get install -y tcpdump net-tools iputils-ping iproute2
COPY --from=dlvbuilder /go/bin/dlv /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/scripts/dlv-wrapper.sh /

CMD ["/dlv-wrapper.sh", "/manager"]
