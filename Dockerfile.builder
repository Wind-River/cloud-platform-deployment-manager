FROM golang:1.12.9

# Install our current version of Helm.  We can probably upgrade to a new version
# but this one has been tested and verified to work.
RUN wget https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz -q -O - | tar zx -C /bin --strip-components=1 linux-amd64/helm

# Install our required version of Kubebuilder.  We cannot upgrade to a later
# version without significant effort.
RUN wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.8/kubebuilder_1.0.8_linux_amd64.tar.gz -q -O - | tar zx -C /usr/local/ --transform 's/kubebuilder_1.0.8_linux_amd64/kubebuilder/'

# Install our current version of golangci-lint.  We can probably upgrade to a
# new version but this one has been tested and verified to work.
RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.17.1

# Install the latest version of Docker although we should probably try and
# align the container version and the host version to ensure compatibility.
RUN apt-get update && \
apt-get -y install apt-transport-https \
     ca-certificates \
     curl \
     gnupg2 \
     software-properties-common && \
curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg > /tmp/dkey; apt-key add /tmp/dkey && \
add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
   $(lsb_release -cs) \
   stable" && \
apt-get update && \
apt-get -y install docker-ce

ENV PATH="${PATH}:/usr/local/kubebuilder/bin:/bin"

# Set the workdir into which we will will be working within this container
WORKDIR /go/src/github.com/wind-river/cloud-platform-deployment-manager

# Initialize helm within the container otherwise no helm commands will work.
RUN helm init --client-only

# The entry command can be overwritten when launched but by default these are
# the build steps that we will be running.
CMD make && DEBUG=yes make docker-build
