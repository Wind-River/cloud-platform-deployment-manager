# Developer Info

This document contains information intended for developers and/or maintainers
of the Deployment Manager.  

## Environment Setup

Developing new features or maintaining existing features requires a workstation
configured for Go development as well as some specific tool/package versions.
The instructions that follow are intended to install the current minimum
package requirements to develop and maintain Deployment Manager content.  These
instructions were developed for installing the required packages onto a Ubuntu
22.04 workstation therefore some tweaks may be required on different Linux 
distributions. The workstation should have minum 20G disk space, recommend to
have disk space above 50G.

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

The Deployment Manager was developed during the period when GoLang version
1.17.9 was prominent.  A newer version may work fine, but the minimum guaranteed
version that will work with the tools and Makefile provided is v1.17.9.
 
```bash
wget https://dl.google.com/go/go1.17.9.linux-amd64.tar.gz
sudo tar -C /usr/local -zxf go1.17.9.linux-amd64.tar.gz
export PATH=${PATH}:/usr/local/go/bin
```

#### Helm

The recommended installation method of the Deployment Manager is to use a Helm
chart.  This ensures that the required CRD resources are installed before the
Deployment Manager pods are created.  It also ensures that recommended default
values for specific Kubernetes attributes are used.  The minimum required 
version of Helm is v3.6.2 and can be installed on your workstation using the
following commands.
 
```bash
wget https://get.helm.sh/helm-v3.6.2-linux-amd64.tar.gz
tar zxf helm-v3.6.2-linux-amd64.tar.gz
sudo cp linux-amd64/helm /usr/local/bin/
```

#### Kubebuilder

The basic structure of the Deployment Manager project is defined by the 
Kubebuilder project.  Kubebuilder is a code generator that implements the more
repetitive and template type code.  The StarlingX specific business logic is 
custom developed. At the time of initial development Kubebuilder was at version
v1.0.8.

Since deployment manager v2.0.9, all codes were migrated to kubebuilder v3
base. There is no need to install kubebuilder to compile existing code of deployment
manager. But kubebuilder is needed to add new type or new webhook.
To install the latest kubebuilder,

```bash
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
sudo chmod +x kubebuilder && sudo mv kubebuilder /usr/local/bin/
export PATH=$PATH:/usr/local/bin
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
## Install golangci
curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.17.1

## Install other dependencies
sudo apt-get install make
sudo apt-get install gcc

## Environment Test/Verification

If you have setup your environment properly you should be able to clone the
Deployment Manager repo and build the Docker image.

```bash
cd ${HOME}/go/src
mkdir -p github.com/wind-river
cd github.com/wind-river
git clone https://github.com/Wind-River/cloud-platform-deployment-manager
cd cloud-platform-deployment-manager
make && DEBUG=yes make docker-build
```

With GoLang version 1.17.9, the source directory is not mandatory in ${HOME}/go/src.
You can create a directory anywhere except under GOPATH, and run the "git clone" step described above.

## Working with a private fork
With GoLang version 1.17.9, Go source path is not strictly under
GOPATH. You can create the working directory anywhere except under GOPATH,
and clone your private fork in it.

To use the GoLang structure under $GOPATH/src,
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

Alternatively, you can create the working directory anywhere except under GOPATH,
and clone your private fork, which will also work fine.

## Working with a private fork of a vendor package
In addition to the issues discussed in the preceding section, making changes to
vendored packages poses an additional problem.  That is, how to update the 
local vendored package with the latest commit in a local copy of the fork
instead of pulling the latest upstream commit.  For instance, if you need to
make a change to one of the vendor packages (e.g., gophercloud) then you will
fork that repo, clone it locally, edit your local clone, and run the unit 
tests provided by that repo.  But, before making a pull request to the upstream
repo (or pushing to the upstream repo if you are the owner) you should integrate 
your changes into a DM image and run proper integration tests.

Currently Go environment is module based. We need to modify go.mod to
direct your private fork. Also, we need to consider private fork directory
to create DM container (private fork directory needs to be visible during
container creation).

To include your private fork to compile and to create DM container;

```bash
cd [project working directory]/cloud-platform-deployment-manager
mkdir external
cd external
git clone [your private fork]
```
To compile DM including your private fork, you need to modify go.mod
to specify the directory to replace the upstream URL.
Here is an example for gophercloud;

```bash
replace github.com/gophercloud/gophercloud => ./external/gophercloud
// replace github.com/gophercloud/gophercloud => github.com/wind-river/gophercloud v0.0.0-20220714135506-aac06588b172
```

This change will refer to your private fork instead of upstream.

To generate DM container with your private fork, you need to copy your
private fork into builder container. To achieve this, you need to
add the copy in Dockerfile. This modification should have been written before
"RUN go mod download" because go mod will refer go.mod in your
private fork.

```bash
COPY external/ external/
```

***note:*** The Gophercloud repo is a special case because we actually pull from
a Wind River fork rather than the true upstream repo.

## Working with a private fork of a vendor command
DM use the command called "deepequal-gen" to generate deep equal code
from type code. "deepequal-gen" will download to bin directory automatically
during compilation. But if you'd like to modify "deepequal-gen", place
your private fork of "deepequal-gen", modify it and generate executable
by go build.

After you have created "deepequal-gen", please place it in bin directory.

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

## Complete example setup.

This section provides a complete step-by-step example of how to setup local
clones of the three Github repos that are directly related to this project.
These steps assume that the developer cloning the repos is an admin of these
repos and will be pulling and pushing directly from the repo rather than making
Pull Requests or working with private forks.  This section also assumes that
SSH keys are already setup in Github to enable Push without needing to input 
credentials on each Push operation.

```bash
mkdir -p ${GOPATH}/src/github.com/wind-river/
cd ${GOPATH}/src/github.com/wind-river/
git clone https://github.com/Wind-River/cloud-platform-deployment-manager
cd cloud-platform-deployment-manager
git checkout master
git remote set-url origin git@github.com:Wind-River/cloud-platform-deployment-manager

mkdir -p ${GOPATH}/src/github.com/gophercloud/
cd ${GOPATH}/src/github.com/gophercloud/
git clone https://github.com/Wind-River/gophercloud
cd gophercloud
git checkout starlingx
git remote set-url origin git@github.com:Wind-River/gophercloud

cd ${GOPATH}/src/github.com/wind-river/
git clone https://github.com/Wind-River/deepequal-gen
cd deepequal-gen
git checkout master 
git remote set-url origin git@github.com:Wind-River/deepequal-gen
```

The following two commands create rules to rewrite URL values for the purpose
of forcing pull requests done by "dep ensure -update ..." so that it is
possible to test local changes to vendor packages prior to pushing those changes
to their respective Github repos.  Refer to the [Working with a private fork of
 a vendor package](#working-with-a-private-fork-of-a-vendor-package) section for further clarification.

```bash
git config --global url."ssh://${USER}@localhost${GOPATH}/src/github.com/gophercloud/gophercloud".insteadOf "https://github.com/wind-river/gophercloud"
git config --global url."ssh://${USER}@localhost${GOPATH}/src/github.com/wind-river/deepequal-gen".insteadOf "https://github.com/wind-river/deepequal-gen"
```


## Development workflow

This section provides a complete step-by-step example of what a normal developer
workflow looks like when making a change.  This workflow assumes that the
developer making the change is an admin of this repo and will be pulling and
pushing directly to the repo rather than making a Pull Request or working with a
private fork.  It also assumes that the developer will be publishing the final
Docker images directly rather through, a more likely, automated CI/CD pipeline 
production process (i.e., Jenkins).

Prior to executing of the following steps the [Complete example setup](#complete-example-setup) must
have been completed successfully.

The first step is to checkout a feature branch in the main repo and to make
whatever changes are necessary for the feature or bug being addressed.

```bash
cd ${GOPATH}/src/github.com/wind-river/cloud-platform-deployment-manager
git checkout master
git pull --rebase
git checkout -b feature1
# ... make changes + add/update unit tests (if applicable)
```

This workflow also assumes that the change being implemented will require a
change to the Gophercloud vendor package to enable a add a new or modifying an
existing system API schema. Refer to previous sections on how to clone and setup
the Gophercloud repo clone properly.

```bash
pushd ${GOPATH}/src/github.com/gophercloud/gophercloud
git checkout starlingx
git pull --rebase
# ... make changes + add/update unit tests (if applicable)
go fmt ./starlingx/...
go vet ./starlingx/...
go test ./starlingx/...
```

Once you are satisfied that the change to the Gophercloud repo looks good enough
to begin integration testing you must create a commit so that it can be used
to update the main DM repo.  ***Note:*** Do not push this commit to github until
it has been integrated tested with an updated DM Docker image that contains this
change.

```bash
git add -A
git commit -s
# ... update commit message
```

To pick up the Gophercloud change within the main DM repo you must return to the
main DM repo directory.

When you are confident that your changes to your Gophercloud clone have been
properly included in your DM repo clone then you can re-build both the 
production and debug Docker images.  This will run formatting, static analysis, 
and unit tests before building the Docker images and Helm charts (if necessary).

```bash
make && DEBUG=yes make docker-build
```

If the above commands are successful then the updated Docker images and Helm
charts (if necessary) need to be tested against an actual StarlingX installation
either on a real hardware system, or in some type of virtualized test
environment.   Before you can test against the newly built images you will
need to publish them to a private Docker registry from where it can be pulled
from the StarlingX installation when the DM Helm chart is installed. 

The following commands will publish your newly built production and debug images
to a Docker registry of your choice.

```bash
export MY_REGISTRY=some.registry.com
docker tag wind-river/cloud-platform-deployment-manager:latest ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:latest
docker tag wind-river/cloud-platform-deployment-manager:debug ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:debug
docker push ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:latest
docker push ${MY_REGISTRY}/${USER}/wind-river-cloud-platform-deployment-manager:debug
```

Once the images are pushed to your Docker registry you can used them to test
on your hardware or virtualized test environment.  All affected functionality
needs to be retested before moving ahead with the steps required to publish the
changes to Github.  If any tests fail then you must return to the beginning of
this section and make the necessary changes and continue with each of the
intermediate steps before retesting a new image.

If all tests have succeeded then you can proceed with creating a commit and
pushing it to the public Github repo.  The following commands will create a new
commit and push it to the master branch of the Github repo.

```bash
git add -A
git commit -s
# ... edit commit message
git push origin HEAD:master
```

Again, these instructions assume that a dependent change was required in the
Gophercloud vendor package.  Therefore, you must also return to your local
Gophercloud clone and push any outstanding commit in that repo.

```bash
pushd ${GOPATH}/src/github.com/gophercloud/gophercloud
git push origin HEAD:starlingx
popd
```

The final step is to publish your production and debug Docker images to your
production Docker registry.  This step may not be necessary if your environment
includes an automated CI/CD pipeline that will automatically pickup your change
to the Github repo and automatically re-build and re-publish both images.  For
the sake of this example, we assume that as a developer you are responsible for
this step and can complete this task with the following steps.

```bash
export MY_REGISTRY=some.registry.com
docker tag wind-river/cloud-platform-deployment-manager:latest ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
docker tag wind-river/cloud-platform-deployment-manager:debug ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:debug
docker push ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
docker push ${MY_REGISTRY}/wind-river/cloud-platform-deployment-manager:debug
```

## Troubleshooting dep
If you encounter issues with dep (eg, dep command hangs), check the following:

#### ~/.cache directory

Your home directory is a shared file system. If disk quotas are enabled due to
limited space, the .cache directory could end up filling up your quota. Use a
symbolic link instead.

```bash
mv ${HOME}/.cache /localdisk/loadbuild/${USER}/.cache and
ln -s /localdisk/loadbuild/${USER}/.cache ${HOME}/.cache
```

#### ssh keys

Ensure your ssh keys are set up properly. Check that the commands work without
prompting for your password:
```bash
> ssh -T git@github.com
Warning: Permanently added the RSA host key for IP address '140.82.113.4' to the list of known hosts.
Hi <your-git-username>! You've successfully authenticated, but GitHub does not provide shell access.

> ssh -T localhost ls
Pictures
public
```

If not, check that you added your public key to github. If you ran ssh-keygen
again after ~/.ssh/authorized_keys was already created/added, copy your latest
public key to ~/.ssh/authorized_keys.

#### Versions

Try using Go 1.17 if newer versions aren't working. Check with:
```bash
> go version
go version go1.17.9 linux/amd64
```
