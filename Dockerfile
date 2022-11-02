# Build the manager binary
FROM golang:1.12.9 as dlvbuilder

# Build delve debugger
RUN apt-get update && apt-get install -y git ca-certificates libgnutls30
RUN GO111MODULE=on go get github.com/go-delve/delve/cmd/dlv@v1.2.0

FROM dlvbuilder as builder
ARG GOBUILD_GCFLAGS=""

# Copy in the go src
WORKDIR /go/src/github.com/wind-river/cloud-platform-deployment-manager
COPY vendor/  vendor/
COPY scripts/ scripts/
COPY cmd/     cmd/
COPY pkg/     pkg/

# Build manager
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags "${GOBUILD_GCFLAGS}" -a -o manager github.com/wind-river/cloud-platform-deployment-manager/cmd/manager

# Copy the controller-manager into a thin image
FROM scratch as production
WORKDIR /
COPY --from=builder /go/src/github.com/wind-river/cloud-platform-deployment-manager/manager .
CMD "/manager"

# Copy the delve debugger into a debug image
FROM ubuntu:latest as debug
WORKDIR /
RUN apt-get update && apt-get install -y tcpdump net-tools iputils-ping iproute2
COPY --from=dlvbuilder /go/bin/dlv /
COPY --from=builder /go/src/github.com/wind-river/cloud-platform-deployment-manager/manager .
COPY --from=builder /go/src/github.com/wind-river/cloud-platform-deployment-manager/scripts/dlv-wrapper.sh /

CMD ["/dlv-wrapper.sh", "/manager"]
