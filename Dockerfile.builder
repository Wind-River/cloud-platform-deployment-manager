FROM golang:1.25.7-trixie

# Install our required version of Kustomize to generate the examples
RUN wget https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v5.4.3/kustomize_v5.4.3_linux_amd64.tar.gz -q -O /tmp/kustomize.tar.gz && \
    tar -zxf /tmp/kustomize.tar.gz -C /tmp && \
    mv /tmp/kustomize /usr/local/bin/kustomize && \
    chmod 755 /usr/local/bin/kustomize && \
    rm -rf /tmp/kustomize.tar.gz

# Install Helm v3.17.1
RUN wget https://get.helm.sh/helm-v3.17.1-linux-amd64.tar.gz -O /tmp/helm.tar.gz && \
    tar -zxf /tmp/helm.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    rm -rf /tmp/helm.tar.gz /tmp/linux-amd64

# Install our required version of Kubebuilder.  We cannot upgrade to a later
# version without significant effort.
RUN curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH); chmod +x kubebuilder && mv kubebuilder /usr/local/bin/

# Install the latest version of Docker although we should probably try and
# align the container version and the host version to ensure compatibility.
# https://docs.docker.com/engine/install/debian/
RUN <<EOF
apt update
apt install ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

cat > /etc/apt/sources.list.d/docker.sources <<-SOURCES
Types: deb
URIs: https://download.docker.com/linux/debian
Suites: $(. /etc/os-release && echo "$VERSION_CODENAME")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
SOURCES

apt update
apt -y install docker-ce
EOF


ENV PATH="${PATH}:/usr/local/kubebuilder/bin:/bin"

# Set the workdir into which we will will be working within this container
WORKDIR /go/src/github.com/wind-river/cloud-platform-deployment-manager
RUN git config --global --add safe.directory /go/src/github.com/wind-river/cloud-platform-deployment-manager

ARG BUILD_TEST=1

# Helm v3 is ready to use for packaging and linting
# The entry command can be overwritten when launched but by default these are
# the build steps that we will be running.
CMD ["sh", "-c", "DEBUG=yes BUILD_TEST=${BUILD_TEST} make"]
