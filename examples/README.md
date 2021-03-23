# Example deployment configurations

This directory contains some sample deployment configurations for the typical
system types (i.e., standard, AIO-SX, AIO-DX).  The intent is to provide some
examples for different configuration so that an end user has enough knowledge
to define new configurations for their own system installations.  These examples 
combined with the CRD schema definitions in ```config/crds``` should provide a
basis for learning how to define custom configurations.

The example directories listed here are structured as Kustomize hierarchies so
that a base configuration can be modified or refined for a different system
configuration type without needing to repeat the entire configuration multiple
times.  For example, the ```standard/default``` directory contains a deployment
configuration for a regular 2+2 standard system.  The configuration does not
include any HTTPS configuration nor any VxLAN data networks or Bond interface
configurations.  Separate Kustomize directories are provided to enable
modifications of the ```standard/default``` configuration for those special 
configurations.  That is ```standard/vxlan``` provides an overlay which sets up
VxLAN data network rather than the default VLAN data networks, and 
```standard/https``` modifies the standard system to include HTTPS certificates.

The Kustomize configurations can be rendered to their final form using the 
following commands:

```bash
export EXAMPLES=/tmp/wind-river-cloud-platform-deployment-manager
mkdir -p ${EXAMPLES}
kustomize build examples/standard/default  > ${EXAMPLES}/standard.yaml
kustomize build examples/standard/vxlan > ${EXAMPLES}/standard-vxlan.yaml
kustomize build examples/standard/https > ${EXAMPLES}/standard-https.yaml
kustomize build examples/standard/bond > ${EXAMPLES}/standard-bond.yaml
kustomize build examples/storage/default > ${EXAMPLES}/storage.yaml
kustomize build examples/aio-sx/default > ${EXAMPLES}/aio-sx.yaml
kustomize build examples/aio-sx/vxlan > ${EXAMPLES}/aio-sx-vxlan.yaml
kustomize build examples/aio-sx/https > ${EXAMPLES}/aio-sx-https.yaml
kustomize build examples/aio-sx/geo-location > ${EXAMPLES}/aio-sx-geo-location.yaml
kustomize build examples/aio-dx/default > ${EXAMPLES}/aio-dx.yaml
kustomize build examples/aio-dx/vxlan > ${EXAMPLES}/aio-dx-vxlan.yaml
kustomize build examples/aio-dx/https > ${EXAMPLES}/aio-dx-https.yaml
```

The rendered examples will not have valid values for any host MAC addresses
defined within the final deployment YAML.  The templates have the MAC address
values defined as placeholder values that can be replaced with standard
command line tools.  For example, given the following contents of a sed script
file the configuration can be rendered with valid MAC addresses using this 
approach.

```bash
cat << EOF > macs.sed
s/CONTROLLER0MAC/08:00:27:06:19:4f/
s/CONTROLLER1MAC/08:00:27:af:1f:fe/
EOF

kustomize build examples/aio-dx/default | sed -f macs.sed > ${EXAMPLES}/aio-dx.yaml
```

