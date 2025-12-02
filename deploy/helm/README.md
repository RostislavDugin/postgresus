# Postgresus Helm Chart

## Installation

```bash
helm install postgresus ./deploy/helm -n postgresus --create-namespace
```

After installation, get the external IP:

```bash
kubectl get svc -n postgresus
```

Access Postgresus at `http://<EXTERNAL-IP>` (port 80).

## Configuration

### Main Parameters

| Parameter          | Description        | Default Value               |
| ------------------ | ------------------ | --------------------------- |
| `namespace.create` | Create namespace   | `true`                      |
| `namespace.name`   | Namespace name     | `postgresus`                |
| `image.repository` | Docker image       | `rostislavdugin/postgresus` |
| `image.tag`        | Image tag          | `latest`                    |
| `image.pullPolicy` | Image pull policy  | `Always`                    |
| `replicaCount`     | Number of replicas | `1`                         |

### Resources

| Parameter                   | Description    | Default Value |
| --------------------------- | -------------- | ------------- |
| `resources.requests.memory` | Memory request | `1Gi`         |
| `resources.requests.cpu`    | CPU request    | `500m`        |
| `resources.limits.memory`   | Memory limit   | `1Gi`         |
| `resources.limits.cpu`      | CPU limit      | `500m`        |

### Storage

| Parameter                      | Description               | Default Value          |
| ------------------------------ | ------------------------- | ---------------------- |
| `persistence.enabled`          | Enable persistent storage | `true`                 |
| `persistence.storageClassName` | Storage class             | `""` (cluster default) |
| `persistence.accessMode`       | Access mode               | `ReadWriteOnce`        |
| `persistence.size`             | Storage size              | `10Gi`                 |
| `persistence.mountPath`        | Mount path                | `/postgresus-data`     |

### Service

| Parameter                  | Description             | Default Value  |
| -------------------------- | ----------------------- | -------------- |
| `service.type`             | Service type            | `LoadBalancer` |
| `service.port`             | External port           | `80`           |
| `service.targetPort`       | Container port          | `4005`         |
| `service.headless.enabled` | Enable headless service | `true`         |

### Ingress (Optional)

Ingress is disabled by default. The chart uses LoadBalancer service for direct IP access.

| Parameter               | Description       | Default Value            |
| ----------------------- | ----------------- | ------------------------ |
| `ingress.enabled`       | Enable Ingress    | `false`                  |
| `ingress.className`     | Ingress class     | `nginx`                  |
| `ingress.hosts[0].host` | Hostname          | `postgresus.example.com` |
| `ingress.tls`           | TLS configuration | `[]`                     |

### Health Checks

| Parameter                | Description            | Default Value |
| ------------------------ | ---------------------- | ------------- |
| `livenessProbe.enabled`  | Enable liveness probe  | `true`        |
| `readinessProbe.enabled` | Enable readiness probe | `true`        |

## Examples

### Basic Installation (LoadBalancer on port 80)

Default installation exposes Postgresus via LoadBalancer on port 80:

```bash
helm install postgresus ./deploy/helm -n postgresus --create-namespace
```

Access via `http://<EXTERNAL-IP>`

### Using NodePort

If your cluster doesn't support LoadBalancer:

```yaml
# nodeport-values.yaml
service:
  type: NodePort
  port: 80
  targetPort: 4005
  nodePort: 30080
```

```bash
helm install postgresus ./deploy/helm -n postgresus --create-namespace -f nodeport-values.yaml
```

Access via `http://<NODE-IP>:30080`

### Enable Ingress with HTTPS

For domain-based access with TLS:

```yaml
# ingress-values.yaml
service:
  type: ClusterIP
  port: 4005
  targetPort: 4005

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
helm install postgresus ./deploy/helm -n postgresus --create-namespace -f ingress-values.yaml
```

### Custom Storage Size

```yaml
# storage-values.yaml
persistence:
  size: 50Gi
  storageClassName: "fast-ssd"
```

```bash
helm install postgresus ./deploy/helm -n postgresus --create-namespace -f storage-values.yaml
```
