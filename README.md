<div align="center">
  <img src="assets/logo.svg" alt="Databasus Logo" width="250"/>

  <h3>Backup tool for PostgreSQL, MySQL and MongoDB</h3>
  <p>Databasus is a free, open source and self-hosted tool to backup databases. Make backups with different storages (S3, Google Drive, FTP, etc.) and notifications about progress (Slack, Discord, Telegram, etc.). Previously known as Postgresus (see migration guide).</p>
  
  <!-- Badges -->
   [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
  [![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
  [![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/rostislavdugin/postgresus?color=brightgreen)](https://hub.docker.com/r/rostislavdugin/postgresus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <a href="#-features">Features</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-usage">Usage</a> •
    <a href="#-license">License</a> •
    <a href="#-contributing">Contributing</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com" target="_blank"><strong>🌐 Databasus website</strong></a>
  </p>
  
  <img src="assets/dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="assets/dashboard.svg" alt="Databasus Dashboard" width="800"/>
  
 
</div>

---

## ✨ Features

### 💾 **Supported databases**

- **PostgreSQL**: 12, 13, 14, 15, 16, 17 and 18
- **MySQL**: 5.7, 8 and 9
- **MariaDB**: 10 and 11
- **MongoDB**: 4, 5, 6, 7 and 8

### 🔄 **Scheduled backups**

- **Flexible scheduling**: hourly, daily, weekly, monthly or cron
- **Precise timing**: run backups at specific times (e.g., 4 AM during low traffic)
- **Smart compression**: 4-8x space savings with balanced compression (~20% overhead)

### 🗄️ **Multiple storage destinations** <a href="https://databasus.com/storages">(view supported)</a>

- **Local storage**: Keep backups on your VPS/server
- **Cloud storage**: S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone and more
- **Secure**: All data stays under your control

### 📱 **Smart notifications** <a href="https://databasus.com/notifiers">(view supported)</a>

- **Multiple channels**: Email, Telegram, Slack, Discord, webhooks
- **Real-time updates**: Success and failure notifications
- **Team integration**: Perfect for DevOps workflows

### 🔒 **Enterprise-grade security** <a href="https://databasus.com/security">(docs)</a>

- **AES-256-GCM encryption**: Enterprise-grade protection for backup files
- **Zero-trust storage**: Backups are encrypted and remain useless to attackers, so you can safely store them in shared storage like S3, Azure Blob Storage, etc.
- **Encryption for secrets**: Any sensitive data is encrypted and never exposed, even in logs or error messages
- **Read-only user**: Databasus uses a read-only user by default for backups and never stores anything that can modify your data

### 👥 **Suitable for teams** <a href="https://databasus.com/access-management">(docs)</a>

- **Workspaces**: Group databases, notifiers and storages for different projects or teams
- **Access management**: Control who can view or manage specific databases with role-based permissions
- **Audit logs**: Track all system activities and changes made by users
- **User roles**: Assign viewer, member, admin or owner roles within workspaces

### 🎨 **UX-Friendly**

- **Designer-polished UI**: Clean, intuitive interface crafted with attention to detail
- **Dark & light themes**: Choose the look that suits your workflow
- **Mobile adaptive**: Check your backups from anywhere on any device

### ☁️ **Works with self-hosted & cloud databases**

Databasus works seamlessly with both self-hosted PostgreSQL and cloud-managed databases:

- **Cloud support**: AWS RDS, Google Cloud SQL, Azure Database for PostgreSQL
- **Self-hosted**: Any PostgreSQL instance you manage yourself
- **Why no PITR support?**: Cloud providers already offer native PITR, and external PITR backups cannot be restored to managed cloud databases — making them impractical for cloud-hosted PostgreSQL
- **Practical granularity**: Hourly and daily backups are sufficient for 99% of projects without the operational complexity of WAL archiving

### 🐳 **Self-hosted & secure**

- **Docker-based**: Easy deployment and management
- **Privacy-first**: All your data stays on your infrastructure
- **Open source**: Apache 2.0 licensed, inspect every line of code

### 📦 Installation <a href="https://databasus.com/installation">(docs)</a>

You have four ways to install Databasus:

- Automated script (recommended)
- Simple Docker run
- Docker Compose setup
- Kubernetes with Helm

<img src="assets/healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Installation

You have four ways to install Databasus: automated script (recommended), simple Docker run, or Docker Compose setup.

### Option 1: Automated installation script (recommended, Linux only)

The installation script will:

- ✅ Install Docker with Docker Compose (if not already installed)
- ✅ Set up Databasus
- ✅ Configure automatic startup on system reboot

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Option 2: Simple Docker run

The easiest way to run Databasus:

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

This single command will:

- ✅ Start Databasus
- ✅ Store all data in `./databasus-data` directory
- ✅ Automatically restart on system reboot

### Option 3: Docker Compose setup

Create a `docker-compose.yml` file with the following configuration:

```yaml
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
```

Then run:

```bash
docker compose up -d
```

### Option 4: Kubernetes with Helm

For Kubernetes deployments, install directly from the OCI registry.

**With ClusterIP + port-forward (development/testing):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**With LoadBalancer (cloud environments):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**With Ingress (domain-based access):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

For more options (NodePort, TLS, HTTPRoute for Gateway API), see the [Helm chart README](deploy/helm/README.md).

---

## 🚀 Usage

1. **Access the dashboard**: Navigate to `http://localhost:4005`
2. **Add your first database for backup**: Click "New Database" and follow the setup wizard
3. **Configure schedule**: Choose from hourly, daily, weekly, monthly or cron intervals
4. **Set database connection**: Enter your database credentials and connection details
5. **Choose storage**: Select where to store your backups (local, S3, Google Drive, etc.)
6. **Add notifications** (optional): Configure email, Telegram, Slack, or webhook notifications
7. **Save and start**: Databasus will validate settings and begin the backup schedule

### 🔑 Resetting password <a href="https://databasus.com/password">(docs)</a>

If you need to reset the password, you can use the built-in password reset command:

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="admin"
```

Replace `admin` with the actual email address of the user whose password you want to reset.

---

## 📝 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details

---

## 🤝 Contributing

Contributions are welcome! Read the <a href="https://databasus.com/contribute">contributing guide</a> for more details, priorities and rules. If you want to contribute but don't know where to start, message me on Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

--

## 📖 Migration guide

Databasus is the new name for Postgresus. You can stay with latest version of Postgresus if you wish. If you want to migrate - follow installation steps for Databasus itself.

Just renaming an image is not enough as Postgresus and Databasus use different data folders and internal database naming.

You can put a new Databasus image with updated volume near the old Postgresus and run it (stop Postgresus before):

```
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
```

Then manually move databases from Postgresus to Databasus.

### Why was Postgresus renamed to Databasus?

It was an important step for the project to grow. Actually, there are a couple of reasons:

1. Postgresus is no longer a little tool that just adds UI for pg_dump for little projects. It became a tool both for individual users, DevOps, DBAs, teams, companies and even large enterprises. Tens of thousands of users use Postgresus every day. Postgresus grew into a reliable backup management tool. Initial positioning is no longer suitable: the project is not just a UI wrapper, it's a solid backup management system now (despite it's still easy to use).

2. New databases are supported: although the primary focus is PostgreSQL (with 100% support in the most efficient way) and always will be, Databasus added support for MySQL, MariaDB and MongoDB. Later more databases will be supported.

3. Trademark issue: "postgres" is a trademark of PostgreSQL Inc. and cannot be used in the project name. So for safety and legal reasons, we had to rename the project.

## AI disclaimer

There have been questions about AI usage in project development in issues and discussions. As the project focuses on security, reliability and production usage, it's important to explain how AI is used in the development process.

AI is used as a helper for:

- verification of code quality and searching for vulnerabilities
- cleaning up and improving documentation, comments and code
- assistance during development
- double-checking PRs and commits after human review

AI is not used for:

- writing entire code
- "vibe code" approach
- code without line-by-line verification by a human
- code without tests

The project has:

- solid test coverage (both unit and integration tests)
- CI/CD pipeline automation with tests and linting to ensure code quality
- verification by experienced developers with experience in large and secure projects

So AI is just an assistant and a tool for developers to increase productivity and ensure code quality. The work is done by developers.

Moreover, it's important to note that we do not differentiate between bad human code and AI vibe code. There are strict requirements for any code to be merged to keep the codebase maintainable.

Even if code is written manually by a human, it's not guaranteed to be merged. Vibe code is not allowed at all and all such PRs are rejected by default (see [contributing guide](https://databasus.com/contribute)).

We also draw attention to fast issue resolution and security [vulnerability reporting](https://github.com/databasus/databasus?tab=security-ov-file#readme).
