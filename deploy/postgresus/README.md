# Postgresus Helm Chart

## Installation

```bash
helm install postgresus ./deploy/postgresus -n postgresus --create-namespace
```

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

| Parameter                  | Description             | Default Value |
| -------------------------- | ----------------------- | ------------- |
| `service.type`             | Service type            | `ClusterIP`   |
| `service.port`             | Service port            | `4005`        |
| `service.targetPort`       | Target port             | `4005`        |
| `service.headless.enabled` | Enable headless service | `true`        |

### Ingress

| Parameter               | Description       | Default Value            |
| ----------------------- | ----------------- | ------------------------ |
| `ingress.enabled`       | Enable Ingress    | `true`                   |
| `ingress.className`     | Ingress class     | `nginx`                  |
| `ingress.hosts[0].host` | Hostname          | `postgresus.example.com` |
| `ingress.tls`           | TLS configuration | See values.yaml          |

### Health Checks

| Parameter                | Description            | Default Value |
| ------------------------ | ---------------------- | ------------- |
| `livenessProbe.enabled`  | Enable liveness probe  | `true`        |
| `readinessProbe.enabled` | Enable readiness probe | `true`        |

## Custom Ingress Example

```yaml
# custom-values.yaml
ingress:
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
helm install postgresus ./deploy/postgresus -n postgresus --create-namespace -f custom-values.yaml
```
