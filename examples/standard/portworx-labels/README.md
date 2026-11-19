# Portworx Host Labels Example

This example demonstrates how to use Deployment Manager `HostProfile` labels
to declaratively assign Portworx placement labels on a Standard system with
mixed worker roles (storage, storageless, and pure compute).

## Prerequisites

- 2 controllers (HA)
- **3 storage workers minimum** (Portworx quorum requirement)
- 0+ storageless or pure-compute workers (optional)

> **Note:** Portworx requires a minimum of 3 storage nodes for quorum. Reducing
> storage workers below 3 will result in a non-functional PX cluster.

## Label Strategy

Two labels control Portworx node placement:

| Label | Purpose |
|---|---|
| `px-node: "true"` | Shared gate — KMM loads `px.ko` on any node with this label |
| `portworx.io/node-type: "storage"` or `"storageless"` | Selects whether the node contributes disks to the PX cluster |

## Profiles

| Profile | Labels | Role |
|---|---|---|
| `px-storage-worker-profile` | `px-node=true`, `node-type=storage` | Portworx storage node |
| `px-storageless-worker-profile` | `px-node=true`, `node-type=storageless` | Portworx storageless (compute-only) node |
| `worker-profile` | *(none)* | Pure compute — no Portworx |

The `px-storage-worker-profile` and `px-storageless-worker-profile` both use
`base: worker-profile` and only add their Portworx-specific labels.

## How It Works

1. Labels are declared in `HostProfile.spec.labels`
2. DM applies them as StarlingX sysinv host-labels during reconcile
3. StarlingX propagates host-labels to Kubernetes node labels
4. KMM `Module` selector matches `px-node: "true"` and loads `px.ko`
5. Portworx `StorageCluster` uses `node-type` to assign storage vs storageless role

No imperative `system host-label-assign` or `kubectl label node` commands needed.

## Standalone Example

This example is a standalone configuration (not a Kustomize overlay on
`standard/default`). It uses `spec.match.bootMAC` with dynamic provisioning
(DM discovers existing hosts by MAC), whereas `standard/default` uses
`spec.overrides.bootMAC` with static provisioning (DM creates hosts). These
are fundamentally different provisioning approaches, making overlay
composition impractical.

## Usage

1. Replace `CONTROLLER0MAC`, `WORKER0MAC`, etc. with actual boot MAC addresses
2. Replace `CHANGEME_BASE64` with your base64-encoded admin password
3. Replace `CHANGEME_REGION_UUID` with your region name
4. Adjust `bootDevice`/`rootDevice` paths and interface port names for your hardware
5. Apply:

```bash
kubectl apply -k .
```
