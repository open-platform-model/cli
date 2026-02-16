# Minecraft Server Module

A production-ready Minecraft Java Edition server with automated backup support using OPM (Open Platform Model).

## Features

- **Multiple Server Types**: Vanilla, Paper, Forge, Fabric, Spigot, Bukkit, Purpur, Magma
- **Automated Backups**: RCON-coordinated backups with zero data loss
- **Flexible Storage**: PVC (cloud), hostPath (bare-metal), or emptyDir (testing)
- **Multiple Backup Methods**: tar, rsync, restic (cloud), rclone (remote)
- **Production Ready**: Health checks, resource limits, persistent storage
- **Sidecar Pattern**: Backup runs alongside server without interruption

## Quick Start

### 1. Basic Deployment (Paper Server with Daily Backups)

```bash
# Build the module with defaults
opm mod build --module-dir ./examples/minecraft --output-dir ./dist

# Apply to Kubernetes cluster
kubectl apply -f ./dist
```

This deploys:

- **Paper 1.20+ server** (latest version)
- **20 max players**, normal difficulty, survival mode
- **10Gi PVC** for game data (worlds, configs, plugins)
- **20Gi PVC** for backups (1 week retention)
- **Daily backups** at 24h intervals with 7-day pruning
- **LoadBalancer service** on port 25565

### 2. Check Status

```bash
# Watch pod startup
kubectl get pods -n minecraft -w

# View server logs
kubectl logs -n minecraft -l app=minecraft -c minecraft -f

# View backup logs
kubectl logs -n minecraft -l app=minecraft -c backup -f
```

### 3. Connect to Server

Once the LoadBalancer has an external IP:

```bash
kubectl get svc -n minecraft

# Copy EXTERNAL-IP and connect from Minecraft client
# Example: mc-server.example.com:25565
```

## Configuration Examples

### Example 1: Modded Forge Server with Local Storage

For bare-metal deployments with existing data directories:

```cue
// values_forge.cue
package main

values: {
 server: {
  type:    "FORGE"
  version: "1.20.1"
  image:   "itzg/minecraft-server:latest"
  eula:    true
  motd:    "Modded Forge Server"
  
  // More players for modded servers
  maxPlayers: 50
  
  rcon: {
   password: "CHANGE-THIS-PASSWORD"
   port:     25575
  }
 }

 storage: {
  data: {
   type:         "hostPath"
   path:         "/mnt/minecraft/modded-server"
   hostPathType: "DirectoryOrCreate"
  }
  backups: {
   type:         "hostPath"
   path:         "/mnt/minecraft/backups"
   hostPathType: "DirectoryOrCreate"
  }
 }

 resources: {
  limits: {
   memory: "12Gi" // Modded servers need more RAM
  }
 }

 serviceType: "NodePort" // For bare-metal deployments
}
```

**Deploy with:**

```bash
opm mod build --module-dir ./examples/minecraft --values-file values_forge.cue --output-dir ./dist
kubectl apply -f ./dist
```

### Example 2: Production Server with Restic Cloud Backups

For cloud environments with automated off-site backups to S3:

```cue
// values_production.cue
package main

values: {
 server: {
  type:       "PAPER"
  version:    "1.20.4"
  eula:       true
  motd:       "Production SMP Server"
  maxPlayers: 100

  // Server operators (admins)
  ops: ["player1", "player2"]

  // Whitelist for private servers
  whitelist: ["player1", "player2", "player3"]

  rcon: {
   password: "USE-A-SECURE-PASSWORD-HERE"
   port:     25575
  }
 }

 storage: {
  data: {
   type:         "pvc"
   size:         "50Gi"
   storageClass: "fast-ssd" // High-performance storage
  }
  backups: {
   type: "pvc"
   size: "10Gi" // Smaller - restic deduplicates
  }
 }

 backup: {
  enabled: true
  method:  "restic"

  // Backup every 6 hours
  interval:     "6h"
  initialDelay: "10m"

  restic: {
   repository: "s3:s3.amazonaws.com/my-minecraft-backups"
   password:   "restic-repo-password"
   hostname:   "production-smp"
   
   // Keep: 7 daily, 4 weekly, 6 monthly backups
   retention: "--keep-daily 7 --keep-weekly 4 --keep-monthly 6"
  }
 }

 resources: {
  requests: {
   cpu:    "2000m"
   memory: "4Gi"
  }
  limits: {
   cpu:    "8000m"
   memory: "16Gi"
  }
 }

 serviceType: "LoadBalancer"
}
```

**Setup AWS S3 for Restic:**

```bash
# Create S3 bucket
aws s3 mb s3://my-minecraft-backups

# Initialize restic repository (run once)
kubectl exec -it <pod-name> -c backup -- restic init
```

### Example 3: Testing/Development Server (Ephemeral)

For quick testing without persistent storage:

```cue
// values_testing.cue
package main

values: {
 server: {
  type:       "PAPER"
  version:    "LATEST"
  eula:       true
  motd:       "Test Server - Data Not Saved"
  maxPlayers: 5
  mode:       "creative"

  rcon: {
   password: "test"
   port:     25575
  }
 }

 storage: {
  data: {
   type: "emptyDir" // Data deleted when pod restarts
  }
  backups: {
   type: "emptyDir"
  }
 }

 backup: {
  enabled: false // No backups for testing
 }

 resources: {
  requests: {
   cpu:    "500m"
   memory: "1Gi"
  }
  limits: {
   cpu:    "2000m"
   memory: "4Gi"
  }
 }

 serviceType: "ClusterIP" // Internal only
}
```

### Example 4: High-Performance PvP Server

Optimized for competitive gameplay:

```cue
// values-pvp.cue
package main

values: {
 server: {
  type:       "PAPER"
  version:    "LATEST"
  eula:       true
  motd:       "PvP Arena - May the best player win!"
  maxPlayers: 50
  difficulty: "hard"
  mode:       "survival"
  pvp:        true

  // Optimize for PvP
  viewDistance: 8 // Lower view distance = better performance

  rcon: {
   password: "pvp-server-password"
   port:     25575
  }
 }

 storage: {
  data: {
   type:         "pvc"
   size:         "20Gi"
   storageClass: "nvme-ssd" // Fastest storage class
  }
  backups: {
   type: "pvc"
   size: "30Gi"
  }
 }

 backup: {
  enabled:          true
  method:           "tar"
  interval:         "12h" // Twice daily backups
  pruneBackupsDays: 3     // Keep only 3 days

  tar: {
   compressMethod: "zstd" // Fast compression
   linkLatest:     true
  }
 }

 resources: {
  requests: {
   cpu:    "4000m"
   memory: "8Gi"
  }
  limits: {
   cpu:    "8000m"
   memory: "16Gi"
  }
 }

 serviceType: "LoadBalancer"
}
```

## Storage Strategy Guide

### PersistentVolumeClaim (PVC) - Recommended for Cloud

**Best for**: Cloud environments (AWS, GCP, Azure, DigitalOcean)

**Pros**:

- ✅ Cloud-native and portable across clusters
- ✅ Automated provisioning with StorageClasses
- ✅ Snapshots and backup support from cloud provider
- ✅ Can survive node failures

**Cons**:

- ❌ Costs money (charged per GB-month)
- ❌ Performance depends on storage class

**Example**:

```cue
storage: {
 data: {
  type:         "pvc"
  size:         "50Gi"
  storageClass: "gp3" // AWS EBS gp3 - good balance
 }
}
```

### hostPath - For Bare-Metal and Home Servers

**Best for**: Self-hosted servers, existing data migration, NAS integration

**Pros**:

- ✅ No additional cost
- ✅ Direct access to host filesystem
- ✅ Can use existing data directories
- ✅ Works with NFS/CIFS network mounts

**Cons**:

- ❌ Tied to specific node (pod can't move)
- ❌ Data lost if node dies (unless using network storage)
- ❌ Manual backup responsibility

**Example**:

```cue
storage: {
 data: {
  type:         "hostPath"
  path:         "/mnt/ssd/minecraft/world"
  hostPathType: "DirectoryOrCreate"
 }
 backups: {
  type:         "hostPath"
  path:         "/mnt/backup-drive/minecraft"
  hostPathType: "DirectoryOrCreate"
 }
}
```

**Important**: Use `nodeSelector` or `nodeAffinity` to ensure pod always schedules on the correct node!

### emptyDir - Testing Only

**Best for**: Development, CI/CD testing, throw-away servers

**Pros**:

- ✅ No setup required
- ✅ Fast (uses node's disk or RAM)
- ✅ No cost

**Cons**:

- ❌ **DATA IS DELETED WHEN POD RESTARTS**
- ❌ Not suitable for any real gameplay

**Example**:

```cue
storage: {
 data: {
  type: "emptyDir"
 }
}
```

## Backup Methods

### Tar (Default) - Simple and Reliable

**Best for**: Local backups, easy restore, getting started

- Creates compressed `.tgz` files in `/backups`
- Easy to inspect and extract manually
- Supports symbolic link to latest backup
- Prune old backups with `pruneBackupsDays`

**Restore Process**:

```bash
# Copy backup file from PVC to local machine
kubectl cp minecraft/<pod-name>:/backups/world-backup-2024-01-15.tgz ./backup.tgz -c backup

# Extract on server
tar -xzf backup.tgz -C /path/to/minecraft/data
```

### Restic - Cloud Backups with Deduplication

**Best for**: Production servers, off-site backups, disaster recovery

- Encrypted, deduplicated, incremental backups
- Supports S3, B2, Azure, GCS, SFTP, local
- Space-efficient (only stores changed data)
- Built-in retention policies

**Supported Backends**:

- **AWS S3**: `s3:s3.amazonaws.com/bucket-name`
- **Backblaze B2**: `b2:bucket-name`
- **Google Cloud Storage**: `gs:bucket-name:`
- **Azure Blob**: `azure:container-name:`
- **Local**: `rclone:remote:path` (via rclone backend)

**Example Configuration**:

```cue
backup: {
 method: "restic"
 restic: {
  repository: "s3:s3.us-west-2.amazonaws.com/my-backups"
  password:   "secure-restic-password"
  retention:  "--keep-daily 7 --keep-weekly 4 --keep-monthly 12"
  hostname:   "minecraft-prod" // Identifies this server
 }
}
```

**Restore Process**:

```bash
# List snapshots
kubectl exec -it <pod> -c backup -- restic snapshots

# Restore specific snapshot
kubectl exec -it <pod> -c backup -- restic restore <snapshot-id> --target /data
```

### Rsync - Incremental Backups

**Best for**: Network storage, minimal bandwidth usage

- Only copies changed files
- Fast incremental backups
- Creates directory per backup timestamp
- Works well with NFS/CIFS

### Rclone - Remote Cloud Storage

**Best for**: Advanced users, custom cloud providers

- Supports 40+ cloud storage providers
- Flexible configuration via `rclone.conf`
- Combines tar compression with remote upload

## Manual Restore Procedure

Since this module does **not** include automatic restore on startup, follow these steps for disaster recovery:

### Restore from Tar Backups

```bash
# 1. Scale down the Minecraft server
kubectl scale statefulset minecraft -n minecraft --replicas=0

# 2. Find the latest backup
kubectl exec -it minecraft-0 -n minecraft -c backup -- ls -lh /backups

# 3. Start a temporary pod with the PVC attached
kubectl run restore-helper --image=busybox --restart=Never -n minecraft \
  --overrides='{"spec":{"volumes":[{"name":"data","persistentVolumeClaim":{"claimName":"minecraft-data-minecraft-0"}},{"name":"backups","persistentVolumeClaim":{"claimName":"minecraft-backups-minecraft-0"}}],"containers":[{"name":"restore","image":"busybox","command":["sleep","3600"],"volumeMounts":[{"name":"data","mountPath":"/data"},{"name":"backups","mountPath":"/backups"}]}]}}'

# 4. Extract the backup
kubectl exec -it restore-helper -n minecraft -- sh
cd /data
rm -rf *  # CAUTION: This deletes current data!
tar -xzf /backups/world-backup-YYYY-MM-DD-HHMM.tgz
exit

# 5. Clean up and restart
kubectl delete pod restore-helper -n minecraft
kubectl scale statefulset minecraft -n minecraft --replicas=1
```

### Restore from Restic Backups

```bash
# 1. Scale down server
kubectl scale statefulset minecraft -n minecraft --replicas=0

# 2. List available snapshots
kubectl exec -it minecraft-0 -n minecraft -c backup -- restic snapshots

# 3. Restore specific snapshot
kubectl exec -it minecraft-0 -n minecraft -c backup -- restic restore <snapshot-id> --target /data

# 4. Restart server
kubectl scale statefulset minecraft -n minecraft --replicas=1
```

## On-Demand Backups

Trigger a backup immediately without waiting for the interval:

```bash
# Execute backup command in the sidecar container
kubectl exec -it <pod-name> -n minecraft -c backup -- backup now
```

This is useful before:

- Installing new plugins/mods
- Major world edits
- Minecraft version upgrades
- Planned maintenance

## Security Best Practices

### 1. Change Default RCON Password

**IMPORTANT**: The default RCON password (`minecraft`) is insecure!

```cue
values: {
 server: {
  rcon: {
   password: "USE-A-STRONG-RANDOM-PASSWORD"
  }
 }
}
```

Generate a secure password:

```bash
openssl rand -base64 32
```

### 2. Enable Whitelist for Private Servers

```cue
values: {
 server: {
  whitelist: ["friend1", "friend2", "family_member"]
 }
}
```

### 3. Restrict Network Access

For internal/testing servers, use `ClusterIP` instead of `LoadBalancer`:

```cue
values: {
 serviceType: "ClusterIP"
}
```

Then access via port-forward:

```bash
kubectl port-forward svc/minecraft -n minecraft 25565:25565
```

### 4. Use Secrets for Sensitive Data

Instead of hardcoding passwords in values.cue, use Kubernetes Secrets:

```bash
# Create secret with RCON password
kubectl create secret generic minecraft-secrets \
  --from-literal=rcon-password="$(openssl rand -base64 32)" \
  -n minecraft
```

Then reference in the module (requires custom configuration - not shown in basic example).

## Performance Tuning

### Memory Allocation

Minecraft servers are Java-based and memory-hungry:

| Players | Recommended Memory | CPU Cores |
|---------|-------------------|-----------|
| 1-10    | 2-4 GB           | 1-2       |
| 10-20   | 4-6 GB           | 2-3       |
| 20-50   | 6-10 GB          | 3-4       |
| 50-100  | 10-16 GB         | 4-8       |
| 100+    | 16-32 GB         | 8+        |

### Storage Performance

- **SSD strongly recommended** for `/data` volume
- NVMe for best performance (reduces lag during world saves)
- Use high IOPS storage classes in cloud environments

### Network Latency

- Deploy server close to players (choose appropriate region)
- Use LoadBalancer with static IP for consistent connection
- Consider DDoS protection for public servers

## Troubleshooting

### Server Won't Start

**Check EULA acceptance**:

```bash
kubectl logs -n minecraft <pod-name> -c minecraft
# Look for: "You need to agree to the EULA"
```

**Solution**: Ensure `server.eula: true` in values.

**Check resource limits**:

```bash
kubectl describe pod <pod-name> -n minecraft
# Look for: "OOMKilled" or "Insufficient memory"
```

**Solution**: Increase `resources.limits.memory`.

### Backup Not Working

**Check RCON connectivity**:

```bash
kubectl logs -n minecraft <pod-name> -c backup
# Look for: "RCON authentication failed" or "connection refused"
```

**Solution**: Verify `server.rcon.password` matches in both containers.

**Check disk space**:

```bash
kubectl exec -it <pod-name> -n minecraft -c backup -- df -h /backups
```

**Solution**: Increase `storage.backups.size` or reduce `pruneBackupsDays`.

### Players Can't Connect

**Check service external IP**:

```bash
kubectl get svc -n minecraft
# LoadBalancer should show EXTERNAL-IP (not <pending>)
```

**Check firewall rules**: Ensure port 25565 is open in cloud provider firewall.

**Check server logs**:

```bash
kubectl logs -n minecraft <pod-name> -c minecraft | grep -i "error\|exception"
```

### High Memory Usage

**Enable JVM garbage collection tuning** (requires custom JVM flags - advanced).

**Reduce view distance**:

```cue
values: {
 server: {
  viewDistance: 8 // Lower = less memory, smaller visible area
 }
}
```

## Resource Outputs

When you build this module, OPM generates:

- **StatefulSet** (`minecraft`): 1 replica with server + backup sidecar
- **Service** (`minecraft`): Exposes port 25565 (LoadBalancer/NodePort/ClusterIP)
- **PersistentVolumeClaim** (`minecraft-data-minecraft-0`): Game data volume
- **PersistentVolumeClaim** (`minecraft-backups-minecraft-0`): Backup storage (if backup enabled)

## Advanced Customization

### Add Custom Java Flags

Modify server image environment variables (requires extending the module):

```cue
container: {
 env: {
  JVM_OPTS: {
   name:  "JVM_OPTS"
   value: "-Xms4G -Xmx8G -XX:+UseG1GC -XX:+ParallelRefProcEnabled"
  }
 }
}
```

### Install Plugins/Mods

For plugins (Paper/Spigot) or mods (Forge/Fabric):

1. **Option A**: Use `MODPACK_URL` environment variable (image supports auto-download)
2. **Option B**: Add a volume mount to a ConfigMap/PVC with plugins:

```cue
volumes: {
 plugins: {
  name: "plugins"
  persistentClaim: {
   size: "1Gi"
  }
 }
}
container: {
 volumeMounts: {
  plugins: {
   name:      "plugins"
   mountPath: "/data/plugins"
  }
 }
}
```

Then manually copy plugins into the PVC.

## Related Resources

- **itzg/minecraft-server**: <https://github.com/itzg/docker-minecraft-server>
- **itzg/mc-backup**: <https://github.com/itzg/docker-mc-backup>
- **Minecraft Server Docs**: <https://docker-minecraft-server.readthedocs.io/>
- **Restic Documentation**: <https://restic.readthedocs.io/>

## License

This example module is provided as-is for educational purposes under the OPM project license.
