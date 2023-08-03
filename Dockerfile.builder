FROM golang:1.19-bullseye

# Install our required version of Kustomize to generate the examples
RUN wget https://github.com/kubernetes-sigs/kustomize/releases/download/v1.0.11/kustomize_1.0.11_linux_amd64 -q -O /usr/local/bin/kustomize && chmod 755 /usr/local/bin/kustomize

# Install our current version of Helm.  We can probably upgrade to a new version
# but this one has been tested and verified to work.
RUN wget https://get.helm.sh/helm-v2.16.10-linux-amd64.tar.gz -q -O - | tar zx -C /bin --strip-components=1 linux-amd64/helm

# Install our required version of Kubebuilder.  We cannot upgrade to a later
# version without significant effort.
RUN curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH); chmod +x kubebuilder && mv kubebuilder /usr/local/bin/

# Install the latest version of Docker although we should probably try and
# align the container version and the host version to ensure compatibility.
RUN apt-get update && \
apt-get -y --no-install-recommends install software-properties-common && \
curl -fsSL https://download.docker.com/linux/debian/gpg > /tmp/dkey; apt-key add /tmp/dkey && \
add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/debian \
   $(lsb_release -cs) \
   stable" && \
apt-get update && \
apt-get -y install docker-ce

ENV PATH="${PATH}:/usr/local/kubebuilder/bin:/bin"

# Set the workdir into which we will will be working within this container
WORKDIR /go/src/github.com/wind-river/cloud-platform-deployment-manager

# Initialize helm within the container otherwise no helm commands will work.
RUN helm init --stable-repo-url=https://charts.helm.sh/stable --client-only

# The entry command can be overwritten when launched but by default these are
# the build steps that we will be running.
CMD make && DEBUG=yes make docker-build
