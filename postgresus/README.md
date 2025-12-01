# Postgresus Helm Chart

Helm chart for deploying Postgresus - a PostgreSQL backup and management system.

## Description

This Helm chart deploys Postgresus in a Kubernetes cluster using StatefulSet to ensure persistent data storage.

## Installation

### Install with Default Values

```bash
helm install postgresus ./helm-chart/postgresus -n postgresus --create-namespace
```

### Install with Custom Values

```bash
helm install postgresus ./helm-chart/postgresus -n postgresus --create-namespace -f custom-values.yaml
```

### Upgrade Release

```bash
helm upgrade postgresus ./helm-chart/postgresus -n postgresus
```

### Uninstall Release

```bash
helm uninstall postgresus -n postgresus
```

## Configuration

### Main Parameters

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `namespace.create` | Create namespace | `true` |
| `namespace.name` | Namespace name | `postgresus` |
| `image.repository` | Docker image | `rostislavdugin/postgresus` |
| `image.tag` | Image tag | `v1.34.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `replicaCount` | Number of replicas | `1` |

### Resources

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `resources.requests.memory` | Memory request | `512Mi` |
| `resources.requests.cpu` | CPU request | `250m` |
| `resources.limits.memory` | Memory limit | `1Gi` |
| `resources.limits.cpu` | CPU limit | `500m` |

### Storage

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.storageClassName` | Storage class | `longhorn` |
| `persistence.accessMode` | Access mode | `ReadWriteOnce` |
| `persistence.size` | Storage size | `10Gi` |
| `persistence.mountPath` | Mount path | `/postgresus-data` |

### Service

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `4005` |
| `service.targetPort` | Target port | `4005` |
| `service.headless.enabled` | Enable headless service | `true` |

### Ingress

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `ingress.enabled` | Enable Ingress | `true` |
| `ingress.className` | Ingress class | `nginx` |
| `ingress.hosts[0].host` | Hostname | `postgresus.home.fwz.ru` |
| `ingress.tls` | TLS configuration | See values.yaml |

### Health Checks

| Parameter | Description | Default Value |
|----------|----------|----------------------|
| `livenessProbe.enabled` | Enable liveness probe | `false` |
| `readinessProbe.enabled` | Enable readiness probe | `false` |

## Usage Examples

### Change Image Version

```yaml
# custom-values.yaml
image:
  tag: v1.20.0
```

```bash
helm upgrade postgresus ./helm-chart/postgresus -n postgresus -f custom-values.yaml
```

### Change Storage Size

```yaml
# custom-values.yaml
persistence:
  size: 50Gi
```

### Configure Ingress for Different Domain

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

### Enable Health Checks

```yaml
# custom-values.yaml
livenessProbe:
  enabled: true

readinessProbe:
  enabled: true
```

## Verify Deployment

```bash
# Check release status
helm status postgresus -n postgresus

# View pods
kubectl get pods -n postgresus

# View services
kubectl get svc -n postgresus

# View ingress
kubectl get ingress -n postgresus

# View PVC
kubectl get pvc -n postgresus
```

## Debugging

```bash
# View logs
kubectl logs -n postgresus -l app=postgresus

# View events
kubectl get events -n postgresus

# Describe StatefulSet
kubectl describe statefulset postgresus -n postgresus

# Render templates without installation
helm template postgresus ./helm-chart/postgresus -n postgresus
```

## Requirements

- Kubernetes 1.19+
- Helm 3.0+
- PV provisioner (e.g., Longhorn)
- Nginx Ingress Controller (optional)
- Cert-Manager (optional, for automatic certificates)

## Notes

- The chart uses StatefulSet to ensure stable data storage
- Headless service is created automatically for StatefulSet
- By default, one application instance is created (replica=1)
- It's recommended to configure a backup policy for PVC
