# App Config — ConfigMaps, Secrets, and Volume-Mounted Configuration

**Complexity:** Intermediate  
**Workload Types:** `stateless` (Deployment)

An application demonstrating externalized configuration using ConfigMaps for settings, Secrets for credentials, and volume-mounted config files.

## What This Example Demonstrates

### Core Concepts
- **`#ConfigMaps` resource** → ConfigMapTransformer → `v1 ConfigMap` (per entry)
- **`#Secrets` resource** → SecretTransformer → `v1 Secret` (per entry)
- **Volume-mounted ConfigMaps** — Config files from ConfigMap volumes
- **Environment variables from config** — Static env var wiring from values
- **Base64 encoding** — CUE `encoding/base64` for secret data

### OPM Patterns
- ConfigMap for non-sensitive settings
- Secret for credentials and API keys
- Volume mount for structured config files (YAML, JSON, etc.)
- Direct env var wiring (alternative to `envFrom` / `configMapKeyRef`)

## Architecture

```
┌─────────────────────────────────────────┐
│ ConfigMap: app-settings                 │
│   log_level: "info"                     │
│   max_connections: "100"                │
│   timeout: "30s"                        │
│   cache_enabled: "true"                 │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ ConfigMap: app-config-file              │
│   app.yaml: |                           │
│     server:                             │
│       port: 3000                        │
│     database:                           │
│       pool_size: 10                     │
└─────────────────────────────────────────┘
           │
           │ mounted as volume
           ▼
┌─────────────────────────────────────────┐
│ Deployment: app (2 replicas)            │
│                                          │
│  Container: app (node:20-alpine)        │
│    Port: 3000                           │
│    Env vars:                            │
│      LOG_LEVEL, MAX_CONNECTIONS, etc.   │
│      DB_HOST, DB_USERNAME, DB_PASSWORD  │
│      GITHUB_API_KEY, SLACK_WEBHOOK_URL  │
│    Volumes:                             │
│      /etc/app/app.yaml (from ConfigMap) │
└─────────────────────────────────────────┘
           ▲
           │ reads from
           │
┌─────────────────────────────────────────┐
│ Secret: db-credentials (base64)         │
│   host: cG9zdGdyZXMuZGF0YWJhc2Uuc3Zj │
│   port: NTQzMg==                        │
│   database: bXlhcHA=                    │
│   username: YXBwdXNlcg==                │
│   password: Y2hhbmdlLW1lLWluLXByb2R1... │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ Secret: api-keys (base64)               │
│   github: Z2hwX2V4YW1wbGVfdG9rZW4=     │
│   slack: aHR0cHM6Ly9ob29rcy5zbGFjay5... │
│   datadog: ZGRfYXBpX2tleV9yZXBsYWNl... │
└─────────────────────────────────────────┘
```

## Configuration Schema

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `app.image` | string | `"node:20-alpine"` | Container image |
| `app.port` | int | `3000` | Service port |
| `app.replicas` | int | `2` | Number of replicas |
| `app.settings.logLevel` | string | `"info"` | Log level (debug/info/warn/error) |
| `app.settings.maxConnections` | int | `100` | Max concurrent connections |
| `app.settings.timeout` | string | `"30s"` | Request timeout |
| `app.settings.cacheEnabled` | bool | `true` | Enable caching |
| `app.database.host` | string | `"postgres.database.svc.cluster.local"` | Database hostname |
| `app.database.port` | int | `5432` | Database port |
| `app.database.name` | string | `"myapp"` | Database name |
| `app.database.username` | string | `"appuser"` | Database username |
| `app.database.password` | string | `"change-me-in-production"` | Database password (SENSITIVE) |
| `app.apiKeys.github` | string | `"ghp_example_token_replace_in_prod"` | GitHub API token (SENSITIVE) |
| `app.apiKeys.slack` | string | `"https://hooks.slack.com/services/EXAMPLE"` | Slack webhook URL (SENSITIVE) |
| `app.apiKeys.datadog` | string | `"dd_api_key_replace_in_prod"` | Datadog API key (SENSITIVE) |
| `app.configFile.fileName` | string | `"app.yaml"` | Config file name |
| `app.configFile.content` | string | _(multiline YAML)_ | Config file content |

## Rendered Kubernetes Resources

| Resource | Name | Type | Notes |
|----------|------|------|-------|
| Deployment | `app` | `apps/v1` | 2 replicas |
| Service | `app` | `v1` | ClusterIP (port 3000) |
| ConfigMap | `app-settings` | `v1` | Application settings (4 keys) |
| ConfigMap | `app-config-file` | `v1` | YAML config file (1 key) |
| Secret | `db-credentials` | `v1` | Database credentials (5 keys, base64) |
| Secret | `api-keys` | `v1` | API keys (3 keys, base64) |

**Total:** 6 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/app-config

# Render to split files
opm mod build --split ./examples/app-config
```

**Important:** The default `values.cue` contains placeholder credentials. **DO NOT** deploy to production without overriding sensitive values.

### Apply to Kubernetes

```bash
# Apply with defaults (DEV ONLY)
opm mod apply ./examples/app-config

# Apply to production namespace with custom values
opm mod apply --namespace production -f values_prod.cue ./examples/app-config
```

### Override Sensitive Values

Create a production values file that overrides sensitive fields:

```cue
// values_prod.cue
package main

values: {
    app: {
        database: {
            host:     "prod-postgres.us-east-1.rds.amazonaws.com"
            password: "REAL_PRODUCTION_PASSWORD"  // Read from vault/env
        }

        apiKeys: {
            github:  "ghp_REAL_GITHUB_TOKEN"
            slack:   "https://hooks.slack.com/services/REAL/WEBHOOK/URL"
            datadog: "dd_REAL_API_KEY"
        }
    }
}
```

Apply with overrides:

```bash
opm mod apply -f values_prod.cue ./examples/app-config
```

### Verify ConfigMaps and Secrets

```bash
# List ConfigMaps
kubectl get configmaps

# View ConfigMap contents
kubectl describe configmap app-settings
kubectl get configmap app-config-file -o yaml

# List Secrets
kubectl get secrets

# View Secret (base64 decoded)
kubectl get secret db-credentials -o jsonpath='{.data.password}' | base64 -d
```

## Files

```
app-config/
├── cue.mod/module.cue    # CUE dependencies
├── module.cue            # Module metadata and config schema
├── components.cue        # App component with ConfigMaps/Secrets
└── values.cue            # Default configuration values
```

## Key Code Snippets

### ConfigMap Definition

```cue
spec: {
    configMaps: {
        "app-settings": {
            data: {
                "log_level":        #config.app.settings.logLevel
                "max_connections":  "\(#config.app.settings.maxConnections)"
                "timeout":          #config.app.settings.timeout
                "cache_enabled":    "\(#config.app.settings.cacheEnabled)"
            }
        }
    }
}
```

Produces:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-settings
data:
  log_level: "info"
  max_connections: "100"
  timeout: "30s"
  cache_enabled: "true"
```

### Secret Definition (Base64 Encoded)

```cue
import "encoding/base64"

spec: {
    secrets: {
        "db-credentials": {
            type: "Opaque"
            data: {
                "host":     base64.Encode(null, #config.app.database.host)
                "port":     base64.Encode(null, "\(#config.app.database.port)")
                "database": base64.Encode(null, #config.app.database.name)
                "username": base64.Encode(null, #config.app.database.username)
                "password": base64.Encode(null, #config.app.database.password)
            }
        }
    }
}
```

Produces:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
type: Opaque
data:
  host: cG9zdGdyZXMuZGF0YWJhc2Uuc3ZjLmNsdXN0ZXIubG9jYWw=  # base64("postgres.database.svc.cluster.local")
  port: NTQzMg==  # base64("5432")
  database: bXlhcHA=  # base64("myapp")
  username: YXBwdXNlcg==  # base64("appuser")
  password: Y2hhbmdlLW1lLWluLXByb2R1Y3Rpb24=  # base64("change-me-in-production")
```

### Volume-Mounted ConfigMap

```cue
spec: {
    volumes: {
        "config-file": {
            name: "config-file"
            configMap: {
                name: "app-config-file"
            }
        }
    }

    container: {
        volumeMounts: {
            "config-file": {
                name:      "config-file"
                mountPath: "/etc/app"
                readOnly:  true
            }
        }
    }
}
```

This mounts the ConfigMap `app-config-file` at `/etc/app/`, making the file available at `/etc/app/app.yaml` inside the container.

Pod spec includes:

```yaml
spec:
  volumes:
    - name: config-file
      configMap:
        name: app-config-file
  containers:
    - name: app
      volumeMounts:
        - name: config-file
          mountPath: /etc/app
          readOnly: true
```

### Environment Variables from Config

```cue
spec: {
    container: {
        env: {
            // From ConfigMap values
            LOG_LEVEL: {
                name:  "LOG_LEVEL"
                value: #config.app.settings.logLevel
            }

            // From Secret values
            DB_PASSWORD: {
                name:  "DB_PASSWORD"
                value: #config.app.database.password
            }
        }
    }
}
```

**Note:** This wires values directly as static env vars. The values are resolved at build time from `#config`, which gets populated from `values.cue`.

In a real production setup, you'd use Kubernetes' `envFrom` with `configMapKeyRef` and `secretKeyRef` to reference the ConfigMap/Secret resources dynamically (see RFC-0005 for OPM's planned `#EnvVarSchema` support).

## ConfigMap vs. Secret

| Use ConfigMap for: | Use Secret for: |
|-------------------|----------------|
| Application settings | Passwords |
| Feature flags | API keys |
| Non-sensitive URLs | Tokens |
| Timeouts, limits | TLS certificates |
| Log levels | SSH keys |

**Why?** Secrets:
- Are base64-encoded (obfuscation, not encryption)
- Can be encrypted at rest (if cluster configured)
- Have stricter RBAC controls
- Are not logged by default
- Can be mounted as tmpfs (memory-only, not disk)

## Configuration Patterns

### Pattern 1: ConfigMap + Volume Mount (Structured Files)

**Use case:** YAML/JSON/TOML config files

```cue
configMaps: {
    "app-config": {
        data: {
            "app.yaml": """
                server:
                  port: 3000
                """
        }
    }
}
volumes: {
    "config": {
        name: "config"
        configMap: { name: "app-config" }
    }
}
```

**Pros:** Clean separation, supports complex config structures  
**Cons:** Requires app to read file at startup

### Pattern 2: ConfigMap + Env Vars (Simple Settings)

**Use case:** Simple key-value settings

```cue
configMaps: {
    "app-settings": {
        data: {
            "log_level": "info"
            "timeout":   "30s"
        }
    }
}
env: {
    LOG_LEVEL: { name: "LOG_LEVEL", value: #config.logLevel }
}
```

**Pros:** No file reading required, 12-factor friendly  
**Cons:** String-only values, less structure

### Pattern 3: Secret + Volume Mount (Certificates, Keys)

**Use case:** TLS certs, SSH keys, large secrets

```cue
secrets: {
    "tls-cert": {
        type: "kubernetes.io/tls"
        data: {
            "tls.crt": base64.Encode(null, #config.tlsCert)
            "tls.key": base64.Encode(null, #config.tlsKey)
        }
    }
}
volumes: {
    "tls": {
        name: "tls"
        secret: { name: "tls-cert" }
    }
}
```

**Pros:** Standard K8s secret type, works with cert-manager  
**Cons:** Requires file reading

## Security Best Practices

1. **Never commit secrets to git** — Use `.gitignore` for `values_prod.cue`
2. **Use external secret managers** — Integrate with Vault, AWS Secrets Manager, etc.
3. **Rotate credentials regularly** — Update secrets periodically
4. **Limit RBAC permissions** — Only grant secret access to necessary service accounts
5. **Enable encryption at rest** — Configure cluster-level secret encryption
6. **Mount secrets as read-only** — Use `readOnly: true` for volume mounts

## Next Steps

- **Environment-specific config:** See upcoming `values-layering/` example for dev/staging/prod overrides
- **Simplify with Blueprints:** See upcoming `blueprint-module/` example
- **Add external secret integration:** See OPM RFC-0002 for `#Secret` type (planned)

## Related Examples

- [blog/](../blog/) — Simple multi-component stateless app
- [webapp-ingress/](../webapp-ingress/) — Production web app with Ingress and HPA
- [jellyfin/](../jellyfin/) — Stateful workload with persistent storage
