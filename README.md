# wind-river-cloud-platform-deployment-manager

The Wind River Cloud Platform Deployment Manager provides a data driven method
for configuring the platform components of a StarlingX® installation.  The
intent of this implementation is to ease adoption of StarlingX systems by
automated deployment systems (e.g., CI/CD pipelines). By providing a data
driven method of defining each system installation, the end user is no longer
required to manually interact with the system thru the system CLI, GUI, or
directly with the System API to perform the initial system installation.


## Scope

The current scope of the Deployment Manager supports both initial system
installation (i.e., so called Day-1 operations) and Day-2 operations
(configuration update, hardware replacement and cluster expansion).

On a Day-1 situation, the Deployment Manager consumes a ```YAML``` based
deployment configuration file that is provided by an end user, or automation
framework.  It attempts to reconcile the system state to match the desired
state that is defined in the supplied deployment configuration file.

On a Day-2 operation, the end user is able to update the YAML config file to
reflect the new desired state of the cluster and re-apply the updated
configuration.

The end goal is that once it has completed reconciling the system state,
each host has been transitioned to the ```unlocked/enabled``` state.  When each
host has reached the desired state, the system should be ready to deploy an
application workload.

## Prerequisites/Requirements


The Deployment Manager expects that the target system is in a specific system
state prior to beginning reconciling the system state to the desired state.
Failure to meet these requirements will cause the full system deployment to
fail or not complete.

### Software Requirements
The Deployment Manager depends on the System API to execute configuration
changes on the target system.  Following the initial software installation the
System API is not functional until the system has been bootstrapped.  The first
controller can be bootstrapped using Ansible®. This method is described in
"Installation guides" section at the following wiki.

https://docs.starlingx.io/

Following the bootstrapping of the system by the Ansible deployment method, the
System API is enabled and the Deployment Manager can continue the system
installation by configuring all system, networking and host level resources
according to the state specified in the deployment configuration model.

### Hardware Requirements

The Deployment Manager supports two different host provisioning modes.  The 
provisioning mode of a host is controlled by the ```provisioningMode``` host 
profile schema attribute which can be one of two values. 

+ Dynamic
+ Static

When a host is provisioned using the ***dynamic*** provisioning mode, the
deployment manager waits for the host to appear in system inventory before
applying any configuration changes to the system for that host.  It is the end
user's responsibility to power on the target host and to ensure that it network
boots from the active controller.  The netboot process will trigger a new host
to appear in the system inventory host list.  The Deployment Manager will detect
the new host and if it can correlate it to the ```match``` attributes specified
in the host resource ```spec``` section then it will proceed to configuring the
host to the desired state.  If a new host appears that cannot be correlated to
a host in the supplied deployment configuration then it is ignored by the
deployment manager until a configuration is supplied that matches that host.

When a host is provisioned using the ***static*** provisioning mode, the
Deployment Manager actively searches for an existing host record in the system
inventory database.  If one is not found it inserts a new record into the
system database.  If that host is configured with a Board Management Controller
(BMC) then the Deployment Manager will submit a "re-install" action via the
System API.  This action will cause the system to power-on the host via the BMC
and command it to netboot to force a re-installation.  If the host is not
configured with a BMC then it is the responsibility of the end user to power-on
the host and to force it to netboot.


## Schema Definitions

The end user must supply a deployment configuration model which conforms to the
supported system definition schema.  The schema is defined as a set of
Kubernetes® Custom Resource Definitions (CRD) instances and provides
documentation on each attribute along with validation rules that conform to the
OpenAPI v3 schema validation specification.  The full schema definition is
stored in this repo and can be found in the ```config/crds``` directory.  The
CRD instances are automatically generated based on annotations added directly
to the source code found under ```api/v1```.

A full system deployment configuration is composed of several Kubernetes Custom
Resource (CR) instances.  Each CR conforms to a CRD instance defined under the
```config/crds``` directory.  For example, a system deployment may be composed
of several instances of each of the following CRD types.

 + System
 + Platform Network
 + Data Network
 + Host Profile
 + Host
 + PTP Instances
 + PTP Interfaces

To streamline the process of defining many Host records it is possible to move
common host attributes into a HostProfile definition and to re-use that
definition from many Host resources.  Similarly, it is possible to define
multiple layers of HostProfile resources so that attributes common to multiple
HostProfile resources can be group together into a common HostProfile resource
and re-used by other HostProfile resources.  A Host resource can inherit from
a HostProfile but still provide overrides for individual attributes that may
be host specific.

When the Deployment Manager prepares to configure a Host resource it first
resolves the final Host attributes by merging the hierarchy of HostProfile
resources related to that particular Host and applies any Host specific
overrides.  The final Host attributes are validated and processed. Refer to the
HostProfile schema documentation for more information regarding individual
attributes and how they are handled during the resolution of the HostProfile
hierarchy.

***Warning***: The Schema definition is currently at a Beta release status.
Non-backward compatible changes may be required prior to the first official GA
release.


### Example Deployment Configurations

Example configurations are included here to help end users understand the
scope and context of the Deployment Manager functionality.  To generate the
deployment configuration YAML for each of the supplied examples run the
following command from a cloned copy of this repo.

 ```bash
 $ make examples
 ```

The above command will produce this output and the generated examples can be
found within the listed files.

 ```bash
 mkdir -p /tmp/wind-river-cloud-platform-deployment-manager/system
 kustomize build examples/standard/default  > /tmp/wind-river-cloud-platform-deployment-manager/system/standard.yaml
 kustomize build examples/standard/vxlan > /tmp/wind-river-cloud-platform-deployment-manager/system/standard-vxlan.yaml
 kustomize build examples/standard/bond > /tmp/wind-river-cloud-platform-deployment-manager/system/standard-bond.yaml
 kustomize build examples/storage/default  > /tmp/wind-river-cloud-platform-deployment-manager/system/storage.yaml
 kustomize build examples/aio-sx/default > /tmp/wind-river-cloud-platform-deployment-manager/system/aio-sx.yaml
 kustomize build examples/aio-sx/vxlan > /tmp/wind-river-cloud-platform-deployment-manager/system/aio-sx-vxlan.yaml
 kustomize build examples/aio-dx/default > /tmp/wind-river-cloud-platform-deployment-manager/system/aio-dx.yaml
 kustomize build examples/aio-dx/vxlan > /tmp/wind-river-cloud-platform-deployment-manager/system/aio-dx-vxlan.yaml
 ```

***Note***: The examples provided assume a certain hardware configuration and
may need to be modified to work in your environment.  For instance, it is
assumed that each server is equipped with at least two NICs that can be used
for either OAM or Management networks and are named "enp0s3", and "enp0s8"
respectively. You may need to adjust these values to align with the specific
configuration of your system. Similarly, the MAC addresses specified for each
host have been set to placeholder values that the end user must update before
using.  For example, for the controller-0 host the MAC address is set to
```CONTROLLER0MAC```.  This value is invalid as an actual configuration value
and needs to be set to a real MAC address before using.  The same is true for
the MAC address for each host defined in the example deployment configuration
files.


### Generating Deployment Configurations From Existing Installations

End users can migrate existing installations to use the Deployment Manager by
using the ```deployctl``` tool to generate a deployment configuration from a
running system.  The tool is designed to access the System API to extract
resources from the running system and format them as a deployment configuration
file that conforms to the Kubernetes CRD instances defined by the Deployment
Manager and can be consumed by the Deployment Manager to configure the same
system.


### Building The ```deployctl``` Tool

The deployctl tool can be built using the following command.  These instructions
assume that you have cloned this repo and that your local environment is
suitable for building Go packages.  The resulting binary program will be written
to the ```bin``` directory of this repo.

```bash
$ make tools
```

### Using The ```deployctl``` Tool

To access the System API the system endpoint credentials must be sourced
into the shell environment variables.  The following command example assumes that
the ```deployctl``` tool has already been copied up to the target system.
Alternatively, the tool can be run remotely, but the "openrc" credentials file
must be downloaded to the local system beforehand and the endpoint URL contained
within it must be updated to point to the system's public Keystone endpoint.

```bash
$ source /etc/platform/openrc
$ ./deployctl build -n deployment -s vbox --minimal-config
```

The above command will produce output similar to the following example.  By
default, the generated deployment configuration file is stored in ```deployment-config.yaml```
unless the output path is customized with the ```-o``` option.

```bash
$ ../deployctl build -n deployment -s vbox --minimal-config
building deployment for system "vbox" in namespace "deployment"
building namespace configuration
building system configuration
...filtering system attributes
building system endpoint secret configuration
building system license secret configuration
building certificate secret configurations
building data network configurations
building platform network configurations
building PTP instance configurations
building PTP interface configurations
building host and profile configurations
...Building host configuration for "controller-0"
...Building host profile configuration for "controller-0"
...Running profile filters for "controller-0-profile"
...Building host configuration for "controller-1"
...Building host profile configuration for "controller-1"
...Running profile filters for "controller-1-profile"
...Building host configuration for "compute-0"
...Building host profile configuration for "compute-0"
...Running profile filters for "compute-0-profile"
...Building host configuration for "compute-1"
...Building host profile configuration for "compute-1"
...Running profile filters for "compute-1-profile"
re-running profile filters for second pass
...Running profile filters for "controller-0-profile"
...Running profile filters for "controller-1-profile"
...Running profile filters for "compute-0-profile"
...Running profile filters for "compute-1-profile"
simplifying profile configurations
...Profile "controller-1-profile" not unique using "controller-0-profile" instead
...Profile "compute-1-profile" not unique using "compute-0-profile" instead
done.
```

## Post Installation Updates - Day-2 Operations

The Deployment Manager in Wind River Cloud Platform has expanded its scope
beyond Day-1 operations to include Day-2 operations. While initially focused on
system installation, the Deployment Manager now supports ongoing configuration
changes, cluster expansion, and hardware replacements.

To begin, the Deployment Manager still relies on a YAML-based deployment
configuration file provided by the user or an automation framework. This
configuration file defines the desired state of the system. The Deployment
Manager's objective remains reconciling the system state with the specified
configuration, but now it extends beyond the initial installation.

Once the Deployment Manager completes the reconciliation process, it ensures
that each host is transitioned to the "unlocked/enabled" state. This guarantees
that the system is ready to deploy application workloads, as in the previous
version. However, the new functionality allows for more granular control over
the system, enabling users to make configuration changes, expand the cluster by
adding new hosts, and perform hardware replacements.

With these Day-2 operations, users can update various configuration parameters,
such as system entity configuration, host entity configuration, and more.
Additionally, the Deployment Manager provides enhanced reconciliation and sync
status reporting, giving users visibility into the delta between the target and
actual configuration.

Cluster expansion is now supported, enabling users to add new worker or storage
hosts to increase the system's capacity. Furthermore, hardware replacements,
both for nodes and devices, can be seamlessly handled through the Deployment
Manager.

### Scope Parameter

To distinguish between the initial deployment and a Day-2 update, a new status
field has been added to each resource in the Deployment Manager. This addition
aims to provide clearer visibility and differentiate between the two phases of
the system lifecycle.

To specify the status field in the YAML file, the following format must be used:

```yaml
status:
  deploymentScope: "principal"
```

The `deploymentScope` field indicates the scope of the deployment for the
resource. It can take on two possible values:

- `"bootstrap"` (default value): This indicates the initial deployment scope.
It is utilized during the system's initial installation or Day-1 operations.

- `"principal"`: This value is employed when applying a configuration for Day-2
operations. It signifies that the configuration update is intended for ongoing
management and changes to the system.

By setting the `deploymentScope` field to `"bootstrap"`, the Deployment Manager
acknowledges that the configuration is meant for the initial deployment.
Conversely, when set to `"principal"`, it indicates that the configuration
should be applied as part of Day-2 operations.

This differentiation in the `deploymentScope` value allows for better control
and provides the possibility to apply specific configurations based on the
desired deployment phase.

It is possible to check the deploymentScope value for each resource by running:

```bash
~(keystone_admin)]$ kubectl -n deployment get system
NAME     MODE     TYPE       VERSION   INSYNC   SCOPE         RECONCILED
system   duplex   standard   22.12     false    principal     true
```

### Applying Day-2 Configuration Updates

To update a value in the configuration file and apply it as part of Day-2
operations, these steps must be followed:

1. Open the YAML deployment configuration file and ensure that the
`deploymentScope` field is set to `"principal"`. This indicates that the
changes or modifications in the file are intended for Day-2 operations.

2. Make the changes or modifications for the desired resources and save the
file.

3. It is recommended to apply the new configuration using the ansible-playbook
command, which is also used during the Deployment Manager installation process.
Please refer to
[Using Ansible To Install The Deployment Manager](#using-ansible-to-install-the-deployment-manager)
Arternativaly, the new configuration can be applied running the kubectl command:
```bash
$ kubectl apply -f my-deployment.yaml
```

After applying the new configuration, the update process can require a reboot
of the hosts. In case it is needed, the reboot will be executed automatically
by DM using the VIM (Virtual Infrastructure Manager) Client.

4. In order to complete the Day-2 operation, when the configuration is
updated and the resources are reconciled and in sync again, it is
necessary to edit the YAML deployment configuration file and set the
`deploymentScope` field to `"bootstrap"`. Apply it using:
```bash
$ kubectl apply -f my-deployment.yaml
```
By doing this, DM will only update the resources Scope status to 'bootstrap'
again.

### Delta status

When a new configuration is applied, DM will detect the differences between the
current configuration and the new expected one. This information is stored on a
Status field called `Delta` for each resource.

In order to get the `Delta` values for the resource, it is possible to run the
following kubectl command:

```bash
kubectl get <resource> -n deployment --output=custom-columns=DELTA:.status.delta
```

Example:
```bash
sysadmin@controller-0:~/config/dm-config$ kubectl get host -n deployment --output=custom-columns=DELTA:.status.delta
DELTA
interfaces:
      "ethernet":
-           "class":                    "none",
+           "class":                    "platform",
-           "name":                     "enp0s3",
+           "name":                     "oam0",
-           "platformNetworks":         {},
+           "platformNetworks":         {"oam"},
-           "class":                    "none",
+           "class":                    "data",
-           "dataNetworks":             {},
+           "dataNetworks":             {"group0-data0"},
-           "name":                     "enp0s9",
+           "name":                     "data0",
-           "class":                    "none",
+           "class":                    "data",
-           "dataNetworks":             {},
+           "dataNetworks":             {"group0-data1"},
-           "name":                     "enp0s10",
+           "name":                     "data1",

labels:
+           "disable-nohz-full":          "enabled",
+           "elastic-client":             "enabled",


storage:
      "filesystems":
+           {"name":                      "instances",         "size":         11},

storage:
      "osds":
+           {
+           "cluster":                    "ceph_cluster",
+           "function":                   "osd",
+           "path":                       "/dev/disk/by-path/pci-0000:00:0d.0-ata-2.0",
+           },
+           },

```

If there is more than one instance for the resource, the command should be:

```bash
[sysadmin@controller-0 ~(keystone_admin)]$ kubectl get <RESOURCE> -n deployment -o go-template='{{range .items}}{{printf "NAME: %s\nDELTA: %s\n" .metadata.name .status.delta}}{{end}}'
```

Example:

```bash
[sysadmin@controller-0 ~(keystone_admin)]$ kubectl get datanetworks -n deployment -o go-template='{{range .items}}{{printf "NAME: %s\nDELTA: %s\n" .metadata.name .status.delta}}{{end}}'

NAME: group0-data0
DELTA:

NAME: group0-data1
DELTA:

NAME: group0-data2
DELTA:
MTU: 1300
Description: description2

NAME: group0-data3
DELTA:
MTU: 1400
Description: description-test
```

### Adjusting Generated Configuration Models With Private Information

On systems configured with HTTPS and/or BMC information, the generated
deployment configuration will be incomplete.  Private information such as HTTPS
private key information and BMC password information is not exposed by the
System API and therefore cannot be extracted from the system and inserted
automatically into the deployment configuration.  The end user is responsible
for editing the generated deployment configuration to populate the missing
information.  If private information is missing during the generation of the
configuration the following warning will be output.

```text
Warning: The generated deployment configuration contains kubernetes
  Secrets that must be manually edited to add information that is not
  retrievable from the system.  For example, any BMC Secrets must be
  edited to add the password and any SSL Secrets must be edited to add
  the certificate and key information.  Any such information must be
  added in base64 encoded format.
```

The missing private information must be Base64 encoded prior to inserting it
into the deployment configuration.  For example, given a BMC password of
"mysecret" the following command can be used to convert the cleartext version of
the password to a Base64 encoded version of the password.

```bash
$ VALUE=$(echo  "mysecret" | base64 -w0)
$ echo ${VALUE}
bXlzZWNyZXQKa
$
```

In the above command example the string "bXlzZWNyZXQKa" is the Base64 encoded
version of the "mysecret".  The generated deployment configuration must be
manually edited to insert this value into the BMC secret password attribute.
Look for the following section within the file and replace the empty password
string with the value from the above commands.

```yaml
apiVersion: v1
data:
  password: ""
  username: cm9vdA==
kind: Secret
metadata:
  name: bmc-secret
  namespace: deployment
type: kubernetes.io/basic-auth
```

***Note:*** Base64 is not an encryption method therefore any private information
encoded must still be handled with security in mind and must be kept safe from
unwanted disclosure.

Similarly, encoding certificate and private key information can be accomplished
using the following syntax.

```bash
$ VALUE=$(cat private-key.pem | base64 -w0)
$ echo ${VALUE}
... encoded value will be here ...
```

Look for the following section in the generated deployment configuration and
replace the "ca.crt", "tls.crt", and "tls.key" attributes with the encoded
information corresponding to the CA certificate, Public Certificate, and Private
Key respectively.  Refer to the schema documentation for more information
regarding which certificate types require which fields to be populated.

```yaml
apiVersion: v1
data:
  ca.crt: ""
  tls.crt: ""
  tls.key: ""
kind: Secret
metadata:
  name: openstack-cert-secret-1
  namespace: deployment
type: kubernetes.io/tls
```

## Post Factory Installation Updates

In cases the starlingx system was already deployed once by the Deployment
Manager with generic configurations, additional customized configurations are
expected to be applied once again to finalize the configuration of the system.
The Deployment Manager can read/update a configmap and orchestrate the
configuration process accordingly.

Note: only one host is expected to deployed before updating again, and the host
is expected to be locked. Otherwise, the configMap will be set as finalized
to block the further operation.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: factory-install
  namespace: deployment
data:
  factory-installed: "true"  # Initial state (change to "true" if factory-installed)
  factory-config-finalized: "false" # Optional (change to "true" if configuration is complete)
  <system-name>-default-updated: "false" # Optional (change to "true" if resource default updated)
  <controller-0-name>-default-updated: "false" # Optional (change to "true" if resource default updated)
  <controller-1-name>-default-updated: "false" # Optional (change to "true" if resource default updated)
```

Once the factory-installed data is set to true, the Deployment Manager will
recollect the default resource configuration(which is expected to be set up
before the Deployment Manager and should not be updated by the Deployment
Manager), and force update the reconciled status as false until finalized.

The factory-config-finalized value is expected to be set as true once a host
is unlocked.

Once the factory-config-finalized is set to true, or the configmap is deleted
from the namespace, this operation will not be triggered.

The "-default-updated" data are expected to be updated by the deployment manager,
they are expected to be set to true if the default configurations are updated once
based on the configurations before the deployment manager.


## Operational Models

The Deployment Manager is implemented independent of the existing StarlingX
system architecture.   It is intended to be deployed and run as a container
within a Kubernetes cluster.  That cluster could be the Kubernetes cluster
hosted by the StarlingX system being configured (i.e., the Deployment Manager
can be run on the system it is configuring), or it can be a standalone
Kubernetes cluster provided by the end user (i.e., the Deployment Manager can
configure StarlingX systems remotely).

***Warning***:  Remote deployment is currently not supported due to technical
issues with how the OpenStack endpoint list is configured on the StarlingX
system.  Currently, this functionality only works if the IP address of the
first controller is configured (e.g., DHCP) to be the eventual OAM floating
IP address.  For all other address configurations a implementation change is
required to the StarlingX software.

Depending on which operational model is chosen, the system URL
and endpoint type (e.g., public, internal) must specify the correct access
method to reach the target system.  For example, the ```system-endpoint```
Secret defined within the system's deployment configuration file will contain a
URL similar to this example:

```yaml
apiVersion: v1
data:
  OS_PASSWORD: U3Q4cmxpbmdYKg==
  OS_USERNAME: YWRtaW4=
kind: Secret
metadata:
  name: system-endpoint
  namespace: deployment
stringData:
  OS_AUTH_TYPE: password
  OS_AUTH_URL: http://192.168.204.1:5000/v3
  OS_ENDPOINT_TYPE: internalURL
  OS_IDENTITY_API_VERSION: "3"
  OS_INTERFACE: internal
  OS_KEYSTONE_REGION_NAME: RegionOne
  OS_PROJECT_DOMAIN_NAME: Default
  OS_PROJECT_NAME: admin
  OS_REGION_NAME: RegionOne
type: Opaque
```

### Local Deployment

Running the Deployment Manager locally, on the target system, requires that the
system URL specify the local management floating IP address and the endpoint
type be set to ```internal```.

***Note:*** If the target system is to be configured with HTTPS or BMC is to be
enabled on hosts then the URL provided must point to the OAM floating IP
address. This is to ensure that BMC credentials and HTTPS private key
information is always transmitted over an encrypted HTTPS session.

### Remote Deployment

Running the Deployment Manager remotely, on a system other than the target
system, requires that the system URL specify the OAM floating IP address and
the endpoint type be set to ```public```.  For example, these attribute values
would need to be modified to configure a remote system.

```yaml
  OS_AUTH_URL: http://10.10.10.3:5000/v3
  OS_ENDPOINT_TYPE: publicURL
  OS_INTERFACE: public
```

If the system being configured remotely was installed using a temporary IP
address (i.e., an address different than the one to be configured as the OAM
floating IP address during bootstrapping) then the Deployment Manager needs to
be configured with both the temporary IP address and the eventual OAM Floating
IP address.  In this case, the OS_AUTH_URL attribute can be specified as a comma
separated list of URL values and may look similar to the following example where
10.10.10.3 is the OAM floating IP address, and 172.16.3.17 is the temporary IP
address used to install the node.

```yaml
  OS_AUTH_URL: http://10.10.10.3:5000/v3,http://172.16.3.17:5000/v3
```

When specifying two different URL values as a comma separated list it is
important to list the OAM floating IP address value first, and the temporary
installation IP address second.  This ensures that, subsequent to the initial
configuration, the Deployment Manager will succeed immediately when connecting
to the first IP and will not be subject to a connection timeout delay incurred
when accessing the temporary installation IP address when it is no longer valid.

### Debugging API requests

For debug purposes, it is possible to log all API requests between the
Deployment Manager and the StarlinX system API.  To enable this functionality
the OS_DEBUG attribute must be a string representation of a boolean value.  At
the time of writing this document those values are "1", "t", "T", "true", 
"TRUE", and "True".

```yaml
  OS_DEBUG: true
```

## Building The Deployment Manager Image

The Deployment Manager Docker Image is not currently posted on any public Docker
Registry.  To install and use the Deployment Manager a image must be built.  To
build an image the following command can be used.

```bash
make docker-build
```

This will build the image using the default name "wind-river/cloud-platform-deployment-manager".
To change the default image name the DEFAULT_IMG variable can be overridden
using the following command syntax.

```bash
DEFAULT_IMG=${USER}/wind-river-cloud-platform-deployment-manager  make docker-build
```

To use the built image to install the Deployment Manager onto a StarlingX system
it must either be tagged and pushed to a private Docker Registry (i.e., using
docker tag + docker push) or exported to an archive that can be used to load the
image directly into the StarlingX local private Docker Registry (i.e., using
docker save + docker load).  For more information on how to manipulate Docker 
images please refer to Docker documentation.  The following subsections provide
example commands to publish and use a custom Deployment Manager image.  Actual
commands may vary based on your local environment.

### Pushing The Image To A Private Registry

The following commands tag a custom image and pushes it to a private Docker 
registry.  This private image can later be accessed from the StarlingX system
if the Deployment Manager is configured to pull its image from this private
Docker registry rather than the default local Docker registry.

```bash
export PRIVATE_REGISTRY="your.registry.com"
docker tag wind-river/cloud-platform-deployment-manager:latest ${PRIVATE_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
docker push ${PRIVATE_REGISTRY}/wind-river/cloud-platform-deployment-manager:latest
```

### Exporting The Image For Offline Use

The following commands tag a custom image and exports it to an compressed
archive.  This archive can later be uploaded to a StarlingX system and imported
directly into the StarlingX local Docker registry.

```bash
export OFFLINE_IMAGE_PATH="/some/path/to/images"
docker tag wind-river/cloud-platform-deployment-manager:latest wind-river/cloud-platform-deployment-manager:stable
docker save wind-river/cloud-platform-deployment-manager | gzip >  ${OFFLINE_IMAGE_PATH}/wind-river-cloud-platform-deployment-manager-images.tgz
```

### Importing The Image From Offline Storage

The following command imports a custom image into the StarlingX local Docker
registry.  This command assumes that the compressed image archive has already
been copied onto the local file system.

```bash
sudo docker load -i /home/wrsroot/wind-river-cloud-platform-deployment-manager-images.tgz
```

## Installing The Deployment Manager

The Deployment Manager is intended to be deployed in two ways 

# Deployment Manager as Helm™ chart.

The chart source definition can be found under the ```helm/wind-river-cloud-platform-deployment-manager```
directory and all overridable configuration values are defined in the
```helm/wind-river-cloud-platform-deployment-manager/values.yaml``` file.

A pre-built copy of the Deployment Manager Helm chart can be downloaded from
this repo at the following location.  Alternatively, it can be accessed
directly from the cloned repo in the ```docs/charts``` directory.

https://github.com/Wind-River/cloud-platform-deployment-manager/raw/master/docs/charts/wind-river-cloud-platform-deployment-manager-24.09.300.tgz

It can be deployed using the following command.

```bash
helm upgrade --install deployment-manager wind-river-cloud-platform-deployment-manager-24.09.300.tgz
```

If any configuration values need to be overridden at installation time then a
file containing those overrides can be supplied using the following syntax.  For
further details on managing and deploying Helm charts please refer to Helm
documentation for more information.

```bash
helm upgrade --install deployment-manager --values overrides.yaml wind-river-cloud-platform-deployment-manager-24.09.300.tgz
```

The default Helm chart assumes that the Deployment Manager image is present in
the StarlingX local Docker Registry.  That is, its image name is not prefixed
with the path to a public Docker Registry.  The image must be loaded into the
StarlingX local Docker Registry prior to installing the Helm chart; otherwise
the system will fail to instantiate the Deployment Manager due to the
unavailability of the image.

If the image has already been uploaded to a private Docker Registry then the
Helm chart must be overridden to supply the path to the image.  For example, the
image location can be overridden to specify a private URL using the following
syntax assuming that the private registry is hosted at "your.registry.org".

```bash
helm upgrade --install deployment-manager --set "manager.image.repository=your.registry.com/wind-river/cloud-platform-deployment-manager" wind-river-cloud-platform-deployment-manager-24.09.300.tgz
```

# Deployment Manager as System application

The procedure for building the deployment-manager system application helm chart is not designed for open source. 
The open-source version does not include the necessary code to generate the system application helm chart.

The application chart source definition is of the form ```deployment-manager-*.tgz```

This chart is built with helm-overrides.

1. Upload the system application

```bash
system application-upload deployment-manager-*.tgz
```

2. Apply the system application

```bash
system application-apply deployment-manager
```

3. Check system application is successfully applied

```bash
[sysadmin@controller-0 ~(keystone_admin)]$ system application-list | grep deployment-manager
| deployment-manager       | <app_version>  | deployment-manager-fluxcd-manifests       | fluxcd-manifests | applied | completed |
```

## Loading A Deployment Configuration Model

Since deployment configurations are defined using standard Kubernetes CRD
instances they can be applied using standard Kubernetes tools and APIs.  Once
the Deployment Manager has been installed and is running, a deployment
configuration can be loaded using the ```kubectl``` command with the following
syntax.

```bash
$ kubectl apply -f my-deployment.yaml
```

***Note:*** The Deployment Manager must be operational before applying a
deployment configuration; otherwise the apply will timeout while waiting
for a response to validation callbacks which are serviced by the Deployment
Manager.

### Working With Multiple Configurations.

The Deployment Manager is capable of installing multiple systems, but it expects
that resources for a single system are all contained within a single Kubernetes
Namespace.  This can be most useful when the Deployment Manager is run remotely
and each system deployment configuration points to a different public endpoint
URL with a unique set of authentication credentials.

***Note:*** There is currently no support for sharing resources across multiple
namespaces.


### Deleting A Deployment Configuration

A deployment configuration can be deleted using the ```kubectl``` command or
other Kubernetes APIs.  The Deployment Manager will make a best effort attempt
at deleting resources that were added by the deployment configuration.  It may
not be possible to delete certain resources (e.g., the last active controller,
the last remaining storage node, etc) and therefore it may not be possible to
delete some dependent resources (i.e., networks still in use by resources that
could not be deleted).

Since the intended purpose of the Deployment Manager is to install a system
rather than to manage a system thru its full life cycle the delete functionality
is currently viewed as a best effort functionality.


## Using Ansible To Install The Deployment Manager

To provide better integration into existing CI/CD pipelines an Ansible playbook
has been provided.  The playbook is stored in this repo as the ```docs/playbooks/wind-river-cloud-platform-deployment-manager-playbook.yaml```
file.

The playbook will download the latest Helm chart to the local Ansible host,
upload a copy to the target system, install the chart on the target system,
upload a copy of the desired deployment configuration, and apply the desired
deployment configuration.  Once the playbook has completed the deployment
manager will be running, and will begin consuming the desired deployment
configuration file.  From this point on the remaining system installation is
automated and once the Deployment Manager has completed consuming the deployment
configuration file the system state will match the desired state.  Normally,
this means that all hosts will be ```unlocked/enabled``` and the system will be
ready to take on additional workloads.

The playbook is designed to download a copy of the latest Deployment Manager
Helm chart from this repo.  If a specific version needs to be used or if the
Helm chart has simply already been stored locally, then a playbook variable must
be overridden to provide an alternative source for the chart.

The playbook consists of tasks that require privilege escalation. For this reason
it is important to provide the ansible become password through an ansible overrides
file. This password is supplied to Ansible through command line using the ```-e```
option. For reference, a sample ansible overrides file is given below.

```bash
$ cat ansible-overrides.yaml

ansible_become_pass: TestSysadminPassword
```

To override the location of the chart to a local file simply set the ```deployment_manager_chart```
variable to the absolute or relative path to the chart.  This can be accomplished
at the command line using a ```-e``` override using the following syntax. For
more detailed information on how to set playbook variables and how to run
playbooks please refer to the Ansible documentation.

```bash
$ ansible-playbook docs/playbooks/wind-river-cloud-platform-deployment-manager-playbook.yaml -e "deployment_manager_chart==/some/other/path/wind-river-cloud-platform-deployment-manager-24.09.300.tgz" -e @ansible-overrides.yaml
```

The system deployment configuration file must be specified using the
```deployment_config``` variable.  If this variable is not set the playbook will
simply install the Deployment Manager and skip the deployment configuration
step.  If this is the case then, at a later time, the end user can manually
apply a deployment configuration file.


### Integrating The playbook Into A Master playbook.

Since StarlingX supports bootstrapping the system using an Ansible playbook it
is possible to combine both the bootstrap playbook and the Deployment Manager
playbook into a single master playbook.  This simplifies the number of steps required
to install the system and further helps enable CI/CD pipeline integration.  For
example, a sample master playbook could resemble the following.  This assumes
that the StarlingX bootstrap playbook directory has already been downloaded and
stored on a filesystem accessible by the Ansible controller host.

```yaml
- import_playbook: /some/path/to/playbookconfig/playbooks/playbooks/bootstrap/bootstrap.yml
- import_playbook: /some/other/path/to/docs/charts/wind-river-cloud-platform-deployment-manager-playbook.yaml
```

An end user can further expand this master playbook by adding additional
playbooks to deploy their custom applications, but subsequent steps must be
written to wait for the Deployment Manager to complete the
system installation before continuing.  This is required because the
Deployment Manager playbook will complete as soon as the deployment configuration
file has been applied and does not wait for that deployment configuration to be
consumed.


## Project License

The license for this project is the Apache 2.0 license. Text of the Apache 2.0
license and other applicable license notices can be found in the LICENSE file
in the top level directory. Each source file should include a license notice
that designates the licensing terms for the respective file.


## Legal Notices

All product names, logos, and brands are property of their respective owners.
All company, product and service names used in this software are for
identification purposes only. Wind River is a registered trademark of Wind River
Systems, Inc.  StarlingX is a registered trademark of the OpenStack.
Kubernetes is a registered trademark of Google Inc.  Helm is a trademark of The
Linux Foundation®. Ansible  is a registered trademark of Red Hat, Inc. in the
United States and other countries.

Disclaimer of Warranty / No Support: Wind River does not provide support and
maintenance services for this software, under Wind River’s standard Software
Support and Maintenance Agreement or otherwise. Unless required by applicable
law, Wind River provides the software (and each contributor provides its
contribution) on an “AS IS” BASIS, WITHOUT WARRANTIES OF ANY KIND, either
express or implied, including, without limitation, any warranties of TITLE,
NONINFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A PARTICULAR PURPOSE. You are
solely responsible for determining the appropriateness of using or
redistributing the software and assume any risks associated with your exercise
of permissions under the license.
