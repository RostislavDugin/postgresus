# Databasus Helm Chart

## Installation

Install directly from the OCI registry (no need to clone the repository):

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

The `-n databasus --create-namespace` flags control which namespace the chart is installed into. You can use any namespace name you prefer.

## Accessing Databasus

By default, the chart creates a ClusterIP service. Use port-forward to access:

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
```

Then open `http://localhost:4005` in your browser.

## Configuration

### Main Parameters

| Parameter          | Description        | Default Value               |
| ------------------ | ------------------ | --------------------------- |
| `image.repository` | Docker image       | `databasus/databasus` |
| `image.tag`        | Image tag          | `latest`                    |
| `image.pullPolicy` | Image pull policy  | `Always`                    |
| `imagePullSecrets` | Image pull secrets | `[]`                        |
| `replicaCount`     | Number of replicas | `1`                         |

### Custom Root CA

| Parameter      | Description                              | Default Value |
| -------------- | ---------------------------------------- | ------------- |
| `customRootCA` | Name of Secret containing CA certificate | `""`          |

To trust a custom CA certificate (e.g., for internal services with self-signed certificates):

1. Create a Secret with your CA certificate:

```bash
kubectl create secret generic my-root-ca \
  --from-file=ca.crt=./path/to/ca-certificate.crt
```

2. Reference it in values:

```yaml
customRootCA: my-root-ca
```

The certificate will be mounted to `/etc/ssl/certs/custom-root-ca.crt` and the `SSL_CERT_FILE` environment variable will be set automatically.

### Extra Environment Variables

| Parameter       | Description                                                                       | Default Value |
| --------------- | --------------------------------------------------------------------------------- | ------------- |
| `extraEnv`      | Extra env vars on the container; standard Kubernetes EnvVar shape.                | `[]`          |
| `extraEnvFrom`  | Extra envFrom sources; standard Kubernetes EnvFromSource (secretRef/configMapRef).| `[]`          |

Use `extraEnv` for arbitrary container env injection — for example, disabling anonymous telemetry in air-gapped or compliance-restricted clusters:

```yaml
extraEnv:
  - name: IS_DISABLE_ANONYMOUS_TELEMETRY
    value: "true"
```

Or pull a secret value into the container:

```yaml
extraEnv:
  - name: SOME_TOKEN
    valueFrom:
      secretKeyRef:
        name: databasus-extras
        key: token
```

Use `extraEnvFrom` to mount entire Secrets/ConfigMaps as env (handy with External Secrets Operator or the secrets-store-csi-driver):

```yaml
extraEnvFrom:
  - secretRef:
      name: databasus-extras
  - configMapRef:
      name: databasus-extras-cm
```

Notes:

- Env values must be strings. Use quoted `"true"` / `"false"` rather than bare booleans, and pass `--set-string` on the CLI (e.g. `--set-string 'extraEnv[0].value=true'`) to avoid Kubernetes rejecting non-string values.
- `extraEnv` is appended after `SSL_CERT_FILE` (when `customRootCA` is set), so an entry with the same name will override `SSL_CERT_FILE`. Avoid duplicate `name` entries unless that override is intentional.

### Service

| Parameter                  | Description             | Default Value |
| -------------------------- | ----------------------- | ------------- |
| `service.type`             | Service type            | `ClusterIP`   |
| `service.port`             | Service port            | `4005`        |
| `service.targetPort`       | Container port          | `4005`        |
| `service.headless.enabled` | Enable headless service | `true`        |

### Storage

| Parameter                      | Description               | Default Value          |
| ------------------------------ | ------------------------- | ---------------------- |
| `persistence.enabled`          | Enable persistent storage | `true`                 |
| `persistence.storageClassName` | Storage class             | `""` (cluster default) |
| `persistence.accessMode`       | Access mode               | `ReadWriteOnce`        |
| `persistence.size`             | Storage size              | `10Gi`                 |
| `persistence.mountPath`        | Mount path                | `/databasus-data`     |
| `persistence.annotations`      | Annotations for the PVC   | `{}`                   |

### Resources

| Parameter                   | Description    | Default Value |
| --------------------------- | -------------- | ------------- |
| `resources.requests.memory` | Memory request | `1Gi`         |
| `resources.requests.cpu`    | CPU request    | `500m`        |
| `resources.limits.memory`   | Memory limit   | `1Gi`         |
| `resources.limits.cpu`      | CPU limit      | `500m`        |

## External Access Options

### Option 1: Port Forward (Default)

Best for development or quick access:

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
```

Access at `http://localhost:4005`

### Option 2: NodePort

For direct access via node IP:

```yaml
# nodeport-values.yaml
service:
  type: NodePort
  port: 4005
  targetPort: 4005
  nodePort: 30080
```

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  -f nodeport-values.yaml
```

Access at `http://<NODE-IP>:30080`

### Option 3: LoadBalancer

For cloud environments with load balancer support:

```yaml
# loadbalancer-values.yaml
service:
  type: LoadBalancer
  port: 80
  targetPort: 4005
```

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  -f loadbalancer-values.yaml
```

Get the external IP:

```bash
kubectl get svc -n databasus
```

Access at `http://<EXTERNAL-IP>`

### Option 4: Ingress

For domain-based access with TLS:

```yaml
# ingress-values.yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: backup.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: backup-example-com-tls
      hosts:
        - backup.example.com
```

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  -f ingress-values.yaml
```

### Option 5: HTTPRoute (Gateway API)

For clusters using Istio, Envoy Gateway, Cilium, or other Gateway API implementations:

```yaml
# httproute-values.yaml
route:
  enabled: true
  hostnames:
    - backup.example.com
  parentRefs:
    - name: my-gateway
      namespace: istio-system
```

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  -f httproute-values.yaml
```

## Ingress Configuration

| Parameter               | Description       | Default Value            |
| ----------------------- | ----------------- | ------------------------ |
| `ingress.enabled`       | Enable Ingress    | `false`                  |
| `ingress.className`     | Ingress class     | `nginx`                  |
| `ingress.hosts[0].host` | Hostname          | `databasus.example.com` |
| `ingress.tls`           | TLS configuration | `[]`                     |

## HTTPRoute Configuration

| Parameter          | Description             | Default Value                  |
| ------------------ | ----------------------- | ------------------------------ |
| `route.enabled`    | Enable HTTPRoute        | `false`                        |
| `route.apiVersion` | Gateway API version     | `gateway.networking.k8s.io/v1` |
| `route.hostnames`  | Hostnames for the route | `["databasus.example.com"]`   |
| `route.parentRefs` | Gateway references      | `[]`                           |

## Health Checks

| Parameter                | Description            | Default Value |
| ------------------------ | ---------------------- | ------------- |
| `livenessProbe.enabled`  | Enable liveness probe  | `true`        |
| `readinessProbe.enabled` | Enable readiness probe | `true`        |

## Custom Storage Size

```yaml
# storage-values.yaml
persistence:
  size: 50Gi
  storageClassName: "fast-ssd"
  annotations:
    k8up.io/backup: "false"
```

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  -f storage-values.yaml
```

## Pod Security / Hardening

On a cluster that enforces the restricted Pod Security Standard (or runs scanners
like Kyverno, Kubescape or Polaris), tune the following values.

| Parameter                                    | Description                                              | Default Value |
| -------------------------------------------- | -------------------------------------------------------- | ------------- |
| `serviceAccount.create`                      | Provision a dedicated ServiceAccount                     | `false`       |
| `serviceAccount.name`                        | Override the ServiceAccount name (else default/fullname) | `""`          |
| `serviceAccount.annotations`                 | Annotations for the created ServiceAccount               | `{}`          |
| `serviceAccount.automountServiceAccountToken`| Mount the SA token into the pod                          | `false`       |
| `podSecurityContext`                         | Pod-level security context                               | `{}`          |
| `securityContext`                            | Container-level security context                         | `{}`          |
| `podAnnotations`                             | Pod annotations (e.g. AppArmor profile)                  | `{}`          |
| `extraVolumes` / `extraVolumeMounts`         | Extra volumes/mounts (e.g. emptyDir for read-only rootfs)| `[]`          |

The app does **not** automount the ServiceAccount token by default
(`automountServiceAccountToken: false`), which clears the common
`AutomountServiceAccountTokenTrueAndDefaultSA` finding even on the namespace
`default` ServiceAccount.

A restricted-PSS values example wiring seccomp, no privilege escalation, AppArmor,
a dedicated ServiceAccount and a read-only root filesystem:

```yaml
# hardened-values.yaml
serviceAccount:
  create: true

podAnnotations:
  container.apparmor.security.beta.kubernetes.io/databasus: runtime/default

podSecurityContext:
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop:
      - ALL
    # gosu drops privileges from the root entrypoint; these are required.
    add:
      - SETUID
      - SETGID

# Writable paths required while the root filesystem is read-only.
extraVolumes:
  - name: tmp
    emptyDir: {}
  - name: var-run
    emptyDir: {}
extraVolumeMounts:
  - name: tmp
    mountPath: /tmp
  - name: var-run
    mountPath: /var/run
```

> **Not supported by design:** `runAsNonRoot: true` and dropping **all**
> capabilities. The entrypoint must start as root to handle PUID/PGID remapping,
> volume `chown` and PostgreSQL `initdb`, then drops to a non-root user with
> `gosu` (which requires `SETUID`/`SETGID`). See the note in `values.yaml`
> next to `podSecurityContext`.

## Upgrade

```bash
helm upgrade databasus oci://ghcr.io/databasus/charts/databasus -n databasus
```

## Uninstall

```bash
helm uninstall databasus -n databasus
```
