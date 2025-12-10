FROM golang:1.23.0-bullseye

# Install our required version of Kustomize to generate the examples
RUN wget https://github.com/kubernetes-sigs/kustomize/releases/download/v1.0.11/kustomize_1.0.11_linux_amd64 -q -O /usr/local/bin/kustomize && chmod 755 /usr/local/bin/kustomize

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
RUN apt-get update && \
apt-get -y --no-install-recommends install software-properties-common ca-certificates gnupg && \
install -m 0755 -d /etc/apt/keyrings && \
curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
chmod a+r /etc/apt/keyrings/docker.gpg && \
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null && \
apt-get update && \
apt-get -y install docker-ce

ENV PATH="${PATH}:/usr/local/kubebuilder/bin:/bin"

# Set the workdir into which we will will be working within this container
WORKDIR /go/src/github.com/wind-river/cloud-platform-deployment-manager
RUN git config --global --add safe.directory /go/src/github.com/wind-river/cloud-platform-deployment-manager

ARG BUILD_TEST=1

# Helm v3 is ready to use for packaging and linting
# The entry command can be overwritten when launched but by default these are
# the build steps that we will be running.
CMD ["sh", "-c", "DEBUG=yes BUILD_TEST=${BUILD_TEST} make"]
