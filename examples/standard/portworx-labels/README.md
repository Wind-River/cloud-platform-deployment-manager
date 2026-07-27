# Portworx Host Labels Example

This example demonstrates how to use Deployment Manager `HostProfile` labels
to declaratively assign Portworx placement labels on a Standard system with
mixed worker roles (storage, storageless, and pure compute).

## Label Strategy

Two labels control Portworx node placement:

| Label | Purpose |
|---|---|
| `portworx.io/px-node: "true"` | Shared gate — KMM loads `px.ko` on any node with this label |
| `portworx.io/node-type: "storage"` or `"storageless"` | Selects whether the node contributes disks to the PX cluster |

## Profiles

| Profile | Labels | Role |
|---|---|---|
| `px-storage-worker-profile` | `px-node=true`, `node-type=storage` | Portworx storage node |
| `px-storageless-worker-profile` | `px-node=true`, `node-type=storageless` | Portworx storageless (compute-only) node |
| `worker-profile` | *(none)* | Pure compute — no Portworx |

## How It Works

1. Labels are declared in `HostProfile.spec.labels`
2. DM applies them as StarlingX sysinv host-labels during reconcile
3. StarlingX propagates host-labels to Kubernetes node labels
4. KMM `Module` selector matches `portworx.io/px-node: "true"` and loads `px.ko`
5. Portworx `StorageCluster` uses `node-type` to assign storage vs storageless role

No imperative `system host-label-assign` or `kubectl label node` commands needed.

## Usage

1. Replace `CONTROLLER0MAC`, `WORKER0MAC`, etc. with actual boot MAC addresses
2. Replace `CHANGEME_BASE64_ENCODED` with your base64-encoded admin password
3. Adjust `bootDevice`/`rootDevice` paths and interface port names for your hardware
4. Apply:

```bash
kubectl apply -k .
```

## Notes

- Author all labels in the HostProfile **before** the host first reconciles.
  DM applies the full label set on initial reconcile.
- If a label is removed out-of-band, DM detects `INSYNC=false` and re-applies
  the declared state automatically.
- KMM's `Module.spec.selector` uses a plain label map (AND logic). A single
  shared `px-node=true` gate covers both storage and storageless nodes; the
  `node-type` label is consumed separately by the Portworx StorageCluster spec.
