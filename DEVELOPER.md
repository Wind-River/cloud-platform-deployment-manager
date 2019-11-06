# Developer Info

This document contains information intended for developers and/or maintainers
of the Deployment Manager.  

## Environment Setup

Developing new features or maintaining existing features requires a workstation
configured for Go development as well as some specific tool/package versions.
The instructions that follow are intended to install the current minimum
package requirements to develop and maintain Deployment Manager content.  These
instructions were developed for installing the required packages onto a Ubuntu
16.04 workstation therefore some tweaks may be required on different Linux 
distributions. 

#### GoLang
These instructions assume that your Go directory is directly on your home
directory but it can be setup in any arbitrary directory.  Go packages must be
installed with a proper Go path.  Go tools will look for the "go/src" directory
structure in the parent directory tree therefore regardless of where you create
your Go directory it must be structured to have a "go/src" and "go/bin" 
directory within it.

```bash
cd ${HOME}
mkdir -p go/{src,bin}
export GOPATH=${HOME}/go
export PATH=${PATH}:${HOME}/go/bin
mkdir downloads
cd downloads
```

The Deployment Manager was developed during the period when Golang version
1.12.9 was prominent.  A newer version may work fine, but the minimum guaranteed
version that will work with the tools and Makefiles provided is v1.12.9.
 
```bash
wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
sudo tar -C /usr/local -zxf go1.12.9.linux-amd64.tar.gz
export PATH=${PATH}:/usr/local/go/bin
```

#### Helm

The recommended installation method of the Deployment Manager is to use a Helm
chart.  This ensures that the required CRD resources are installed before the
Deployment Manager pods are created.  It also ensures that recommended default
values for specific Kubernetes attributes are used.  The minimum required 
version of Helm is v2.14.3 and can be installed on your workstation using the 
following commands.
 
```bash
wget https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz
tar zxf helm-v2.14.3-linux-amd64.tar.gz
sudo cp linux-amd64/helm /usr/local/bin/
```

#### Kubebuilder

The basic structure of the Deployment Manager project is defined by the 
Kubebuilder project.  Kubebuilder is a code generator that implements the more
repetitive and template type code.  The StarlingX specific business logic is 
custom developed. At the time of initial development Kubebuilder was at version
v1.0.8 therefore this specific version of the tool must be installed.  

In the future the Deployment Manager should be upgraded to the latest 
controller-tools runtime but for now the current version meets the Deployment
Manager requirements.  Upgrading to the latest controller-tools runtime will 
require upgrading Kubebuilder to the latest version.  This will not be a trivial
task as it will involving changes to the underlying directory structure,
internal API changes, certificate handling changes, etc. 
  
```bash
wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.8/kubebuilder_1.0.8_linux_amd64.tar.gz
tar zxf kubebuilder_1.0.8_linux_amd64.tar.gz
sudo cp -r kubebuilder_1.0.8_linux_amd64 /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin
```

The Kubebuilder tool scaffolds a config file directory structure that is only 
compatible with Kustomize v1.0.11 therefore this specific package version must
be installed until the Deployment Manager is upgraded to using the latest
version of Kubebuilder.

```bash
wget https://github.com/kubernetes-sigs/kustomize/releases/download/v1.0.11/kustomize_1.0.11_linux_amd64
sudo cp kustomize_1.0.11_linux_amd64 /usr/local/bin/kustomize
sudo chmod 755 /usr/local/bin/kustomize
```


#### GoLangCI

The GoLangCI-lint tool is not a requirement of the Kubebuilder project 
scaffolding, but the Deployment Manager Makefile has been extended since its
initial creation.  Rather than relying on "go vet" for static analysis the
project now uses the GoLangCI-lint tool which is also in use by newer
Kubebuilder versions.

```bash
curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.17.1
```

#### Docker

The Docker version that is shipped with most distributions is out of date
therefore it must be removed and the latest stable version must be installed
manually.

```bash
sudo apt-get remove docker docker-engine docker.io containerd runc
sudo apt-get update
sudo apt-get install     apt-transport-https     ca-certificates     curl     gnupg-agent     software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository    "deb [arch=amd64] https://download.docker.com/linux/ubuntu   $(lsb_release -cs) stable"
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io
```

Finally, to avoid having to use "sudo" to run docker commands you should
consider adding your user id to the local docker group.  

```bash
sudo usermod -a -G docker ${USER}
newgrp docker
```

## Environment Test/Verification

If you have setup your environment properly you should be able to clone the
Deployment Manager repo and build the Docker image.

```bash
cd ${HOME}/go/src
mkdir -p github.com/wind-river
cd github.com/wind-river
git clone https://github.com/Wind-River/cloud-platform-deployment-manager
cd cloud-platform-deployment-manager
make docker-build && DEBUG=yes make docker-build
```

## Working with a private fork
GoLang projects use fully qualified paths for imports and the GoLang tools
expect to find modules in a matching directory structure within your Go path.
Working with a direct clone of a Go project is straightforward as you can
simply clone the repo in a path that exactly matches the repo's github.com
path.  You cannot use the same method if working with a private fork of an
existing repo because import paths to the project's own module will not be
resolvable.  For example, if you clone your fork of this repo to the following
directory then any imports looking for "github.com/wind-river/cloud-platform-deployment-manager/*"
will fail.

    ${HOME}/go/src/github.com/${USER}/cloud-platform-deployment-manager

To work around this limitation you need clone your fork to the path used by the
main repo.

    ${HOME}/go/src/github.com/wind-river/cloud-platform-deployment-manager
    
From this directory you can maintain remotes for both the main repo and your
fork if this is needed by your workflow.  You can setup a remote to your own
fork if you have already cloned the main repo using the following commands.

```bash
git remote rename origin upstream
git remote add origin git@github.com:<my_username>/cloud-platform-deployment-manager.git
```

## Working with a private fork of a vendor package
In addition to the issues discussed in the preceding section, making changes to
vendored packages poses an additional problem.  That is, how to update the 
local vendored package with the latest commit in a local copy of the fork
instead of pulling the latest upstream commit.  For instance, if you need to
make a change to one of the vendor packages (e.g., gophercloud) then you will
fork that repo, clone it locally, edit your local clone, and run the unit 
tests provided by that repo.  But, before making a pull request to the upstream
repo (or pushing to the upstream repo if you are the owner) you should integrate 
your changes into a DM image and run proper integration tests.  The normal 
method to pull in the latest vendor package is to run "dep ensure" to pull in 
the latest package version.  For example, the following command will update only
a single vendored package:

    dep ensure -update github.com/gophercloud/gophercloud
    
By default, this command will go to the actual github.com URL provided and pull
down the latest commit.  Since you have local changes to your local clone that
behaviour is undesirable.  Instead, you want your "dep ensure" command to pull
from the local clone of your fork.  You can redirect the requests automatically
by adding a few lines to your ${HOME}/.gitconfig file.  

```
[url "ssh://USER@localhost/home/USER/go/src/github.com/gophercloud/gophercloud"]
    insteadOf = https://github.com/wind-river/gophercloud
```

***note:*** The gophercloud repo is a special case because we actually pull from
a Wind River fork rather than the true upstream repo; therefore, there is an
extra layer of redirection found in the top-level Gopkg.toml file.

The above two lines added to the ${HOME}/.gitconfig will intercept any pull
operations directed to https://github.com/wind-river/gophercloud and replace
them with the local path provided in the "url" element.  In the example above
"USER" is meant to be replaced by your user id.


## Publishing
Building the Deployment Manager image using the "make docker-build" command
builds the Docker image using the default image name embedded within the
Makefile.  Unless you are using the image directly on the machine where it was
built you will likely need to publish this image to a public or private Docker
registry so that it can be pulled from whatever Kubernetes platform is being
used as your test platform.   

### Tag/Push for private developer builds
The following commands tag and push the image as a privately named image using
your user id the top level image path.

```bash
export MY_REGISTRY=some.registry.com
docker tag wind-river/cloud-platform-deployment-manager:latest ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:latest
docker tag wind-river/cloud-platform-deployment-manager:debug ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:debug
docker push ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:latest
docker push ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:debug
```

### Tag/Push for production builds
If your private image has been tested and is ready to publish for consumption by
a wider audience then it can tagged and pushed using the official image name
rather than your user id based private image name.

```bash
export MY_REGISTRY=some.registry.com
docker tag wind-river/cloud-platform-deployment-manager:latest ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
docker tag wind-river/cloud-platform-deployment-manager:debug ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:debug
docker push ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
docker push ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:debug
```
