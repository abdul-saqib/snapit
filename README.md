## SnapIt

**SnapIt** is a lightweight Kubernetes controller for automating **PersistentVolumeClaim (PVC) snapshots**. It allows users to create snapshots **on-demand via custom resource policies** or **periodically** according to a schedule. SnapIt works with any CSI driver that supports the Kubernetes VolumeSnapshot API.

---

### Features

- **CRD-Based Snapshot Policies**  
  Define snapshot policies using a custom resource. SnapIt creates snapshots immediately when a policy is applied.

- **Periodic Snapshotting**  
  Schedule recurring snapshots to automatically protect your PVC-backed workloads.

- **CSI VolumeSnapshot Support**  
  Works with any CSI-compliant storage provider.

- **Lightweight and Simple**  
  Minimal complexity, easy to deploy and manage.

---

### Use Cases

- On-demand snapshots for backup or debugging  
- Scheduled snapshots for stateful applications  
- Versioning of PVC-backed environments  

---

### How It Works

1. User applies a `SnapshotPolicy` custom resource defining the target PVC(s) and snapshot configuration.  
2. SnapIt watches these resources and triggers snapshots:
   - Immediately if a one-time snapshot is requested.  
   - Periodically if a schedule is defined.  
3. SnapIt updates the status of the resource so users can track snapshot progress and completion.  

---

### Build and Deploy

Make sure your cluster has a CSI driver installed that supports `VolumeSnapshots`.

Clone the repository:

~~~
git clone https://github.com/<your-org>/snapit.git
cd snapit
~~~

Build, load into Kind cluster, and deploy:

```
make all
```
