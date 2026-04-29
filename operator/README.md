# databasus-operator

A Kubernetes operator that manages databasus configuration declaratively via Custom Resource Definitions (CRDs). Instead of configuring databases, backups, storages, and notifiers through the web UI, define them as Kubernetes resources and let the operator reconcile them against the databasus API.

## How it works

The operator watches three CRDs and syncs their state to a running databasus instance via its REST API:

- **`Storage`** -- storage backends where backups are saved (S3, SFTP, Azure Blob, etc.)
- **`Notifier`** -- notification channels for alerts (Discord, Slack, Telegram, etc.)
- **`DatabaseBackup`** -- database connection + backup schedule + healthcheck config, referencing Storage and Notifier resources by name

On create/update, the operator calls the databasus API to upsert resources. On delete, a finalizer ensures cleanup via the API before the Kubernetes resource is removed.

## Prerequisites

- A running databasus instance accessible from the cluster
- `kubectl` configured for your cluster
- `make`, `go 1.25+`, `docker` (for building)

## Setup

### 1. Create the credentials Secret

The operator authenticates to databasus using an email/password stored in a Kubernetes Secret. It signs in on startup to get a JWT token and resolves the workspace automatically.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: databasus-operator-credentials
  namespace: databasus
type: Opaque
stringData:
  email: admin@example.com
  password: your-password
  # Optional: target a specific workspace by name (defaults to first available)
  # workspaceName: "My Workspace"
  # Optional: or by UUID (takes precedence over workspaceName)
  # workspaceId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

```bash
kubectl apply -f credentials-secret.yaml
```

### 2. Create Secrets for your resources

Each CRD references sensitive values via `secretKeyRef` fields pointing to Kubernetes Secrets. Create these before applying the CRDs.

```bash
# Storage credentials (e.g., S3)
kubectl create secret generic rustfs-credentials \
  --from-literal=access-key=YOUR_ACCESS_KEY \
  --from-literal=secret-key=YOUR_SECRET_KEY \
  -n databasus-operator-system

# Notifier credentials (e.g., Discord webhook)
kubectl create secret generic discord-webhook \
  --from-literal=url=https://discord.com/api/webhooks/... \
  -n databasus-operator-system

# Database password
kubectl create secret generic gitea-db-credentials \
  --from-literal=password=YOUR_DB_PASSWORD \
  -n databasus-operator-system
```

### 3. Build and deploy the operator

```bash
# Build the image
make docker-build IMG=databasus-operator:latest

# Import into k3s (if using local images)
docker save databasus-operator:latest | sudo k3s ctr images import -

# Install CRDs and deploy the operator
make install
make deploy IMG=databasus-operator:latest
```

### 4. Apply your resources

```bash
kubectl apply -f config/samples/storage_s3.yaml
kubectl apply -f config/samples/notifier_discord.yaml
kubectl apply -f config/samples/databasebackup_gitea.yaml
```

### 5. Verify

```bash
kubectl get storages,notifiers,databasebackups -n databasus-operator-system
```

```
NAME                                    TYPE   READY   AGE
storage.databasus.databasus.io/rustfs   S3     True    5m

NAME                                                TYPE      READY   AGE
notifier.databasus.databasus.io/operator-discord    DISCORD   True    5m

NAME                                              DB TYPE    HEALTH      READY   LAST BACKUP   AGE
databasebackup.databasus.databasus.io/gitea       POSTGRES   AVAILABLE   True    <timestamp>   5m
```

## CRD Reference

### Storage

Defines a backup storage backend. Supported types: `S3`, `SFTP`, `AZURE_BLOB`, `LOCAL`, `FTP`, `RCLONE`, `NAS`, `GOOGLE_DRIVE`.

```yaml
apiVersion: databasus.databasus.io/v1alpha1
kind: Storage
metadata:
  name: my-s3-storage
  namespace: databasus
spec:
  name: my-s3-storage
  type: S3
  s3:
    bucket: my-backup-bucket
    region: us-east-1
    endpoint: http://minio.minio.svc.cluster.local:9000  # optional for AWS
    prefix: backups/  # optional
    accessKeySecretRef:
      name: s3-credentials
      key: access-key
    secretKeySecretRef:
      name: s3-credentials
      key: secret-key
    isSkipTLSVerify: false  # optional
    storageClass: STANDARD  # optional
```

### Notifier

Defines a notification channel. Supported types: `DISCORD`, `SLACK`, `TELEGRAM`, `EMAIL`, `WEBHOOK`, `TEAMS`.

```yaml
apiVersion: databasus.databasus.io/v1alpha1
kind: Notifier
metadata:
  name: my-discord
  namespace: databasus
spec:
  name: My Discord Channel
  type: DISCORD
  discord:
    webhookURLSecretRef:
      name: discord-webhook
      key: url
```

### DatabaseBackup

The main resource combining database connection, backup configuration, and health checks. References Storage and Notifier CRDs by their metadata name.

```yaml
apiVersion: databasus.databasus.io/v1alpha1
kind: DatabaseBackup
metadata:
  name: my-database
  namespace: databasus
spec:
  database:
    name: my-database
    type: POSTGRES  # POSTGRES, MYSQL, MARIADB, MONGODB
    notifierRefs:
      - my-discord  # metadata.name of a Notifier CRD
    postgresql:
      version: "17"
      host: postgres.default.svc.cluster.local
      port: 5432
      username: myuser
      passwordSecretRef:
        name: db-credentials
        key: password
      database: mydb
      backupType: PG_DUMP  # PG_DUMP or WAL_V1

  backup:
    isEnabled: true
    storageRef: my-s3-storage  # metadata.name of a Storage CRD
    encryption: ENCRYPTED      # NONE or ENCRYPTED
    interval:
      type: DAILY              # HOURLY, DAILY, WEEKLY, MONTHLY, CRON
      timeOfDay: "09:00"
    retentionPolicy:
      type: TIME_PERIOD        # TIME_PERIOD, COUNT, GFS
      timePeriod: "90d"
    isRetryIfFailed: true
    maxFailedTriesCount: 3
    sendNotificationsOn:
      - BACKUP_FAILED          # BACKUP_FAILED, BACKUP_SUCCESS

  healthcheck:
    isEnabled: true
    isSentNotificationWhenUnavailable: true
    intervalMinutes: 1
    attemptsBeforeConsideredAsDown: 3
    storeAttemptsDays: 7
```

## Configuration

The operator reads its configuration from environment variables set in the deployment:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASUS_API_URL` | `http://databasus-service.databasus.svc.cluster.local:4005` | databasus API endpoint |
| `DATABASUS_CREDENTIALS_SECRET` | `databasus-operator-credentials` | Name of the credentials Secret |
| `DATABASUS_CREDENTIALS_NAMESPACE` | `databasus` | Namespace of the credentials Secret |

## Uninstall

```bash
# Remove CRs (triggers finalizer cleanup via databasus API)
kubectl delete databasebackups,notifiers,storages --all -n databasus-operator-system

# Remove operator and CRDs
make undeploy
make uninstall
```
