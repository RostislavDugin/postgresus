# Postgresus Helm Chart

## Installation

Install directly from the OCI registry (no need to clone the repository):

```bash
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace
```

## Accessing Postgresus

By default, the chart creates a ClusterIP service. Use port-forward to access:

```bash
kubectl port-forward svc/postgresus-service 4005:4005 -n postgresus
```

Then open `http://localhost:4005` in your browser.

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
| `persistence.mountPath`        | Mount path                | `/postgresus-data`     |

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
kubectl port-forward svc/postgresus-service 4005:4005 -n postgresus
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
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace -f nodeport-values.yaml
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
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace -f loadbalancer-values.yaml
```

Get the external IP:

```bash
kubectl get svc -n postgresus
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
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace -f ingress-values.yaml
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
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace -f httproute-values.yaml
```

## Ingress Configuration

| Parameter               | Description       | Default Value            |
| ----------------------- | ----------------- | ------------------------ |
| `ingress.enabled`       | Enable Ingress    | `false`                  |
| `ingress.className`     | Ingress class     | `nginx`                  |
| `ingress.hosts[0].host` | Hostname          | `postgresus.example.com` |
| `ingress.tls`           | TLS configuration | `[]`                     |

## HTTPRoute Configuration

| Parameter          | Description             | Default Value                  |
| ------------------ | ----------------------- | ------------------------------ |
| `route.enabled`    | Enable HTTPRoute        | `false`                        |
| `route.apiVersion` | Gateway API version     | `gateway.networking.k8s.io/v1` |
| `route.hostnames`  | Hostnames for the route | `["postgresus.example.com"]`   |
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
```

```bash
helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus --create-namespace -f storage-values.yaml
```

## Upgrade

```bash
helm upgrade postgresus oci://ghcr.io/rostislavdugin/charts/postgresus -n postgresus
```

## Uninstall

```bash
helm uninstall postgresus -n postgresus
```
