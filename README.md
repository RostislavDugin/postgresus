<div align="center">
  <img src="assets/logo.svg" style="margin-bottom: 20px;" alt="Postgresus Logo" width="250"/>

  <h3>PostgreSQL backup</h3>
  <p>Free, open source and self-hosted solution for automated PostgreSQL backups. With multiple storage options and notifications</p>
  
  <!-- Badges -->
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/rostislavdugin/postgresus?color=brightgreen)](https://hub.docker.com/r/rostislavdugin/postgresus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/RostislavDugin/postgresus)
  
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12%20%7C%2013%20%7C%2014%20%7C%2015%20%7C%2016%20%7C%2017%20%7C%2018-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/RostislavDugin/postgresus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/RostislavDugin/postgresus)
  
  <p>
    <a href="#-features">Features</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-usage">Usage</a> •
    <a href="#-license">License</a> •
    <a href="#-contributing">Contributing</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://postgresus.com" target="_blank"><strong>🌐 Postgresus website</strong></a>
  </p>
  
  <img src="assets/dashboard-dark.svg" alt="Postgresus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="assets/dashboard.svg" alt="Postgresus Dashboard" width="800"/>
  
 
</div>

---

## ✨ Features

### 🔄 **Scheduled Backups**

- **Flexible scheduling**: hourly, daily, weekly, monthly
- **Precise timing**: run backups at specific times (e.g., 4 AM during low traffic)
- **Smart compression**: 4-8x space savings with balanced compression (~20% overhead)

### 🗄️ **Multiple Storage Destinations** <a href="https://postgresus.com/storages">(view supported)</a>

- **Local storage**: Keep backups on your VPS/server
- **Cloud storage**: S3, Cloudflare R2, Google Drive, NAS, Dropbox and more
- **Secure**: All data stays under your control

### 📱 **Smart Notifications** <a href="https://postgresus.com/notifiers">(view supported)</a>

- **Multiple channels**: Email, Telegram, Slack, Discord, webhooks
- **Real-time updates**: Success and failure notifications
- **Team integration**: Perfect for DevOps workflows

### 🐘 **PostgreSQL Support**

- **Multiple versions**: PostgreSQL 12, 13, 14, 15, 16, 17 and 18
- **SSL support**: Secure connections available
- **Easy restoration**: One-click restore from any backup

### 🔒 **Enterprise-grade security** <a href="https://postgresus.com/security">(docs)</a>

- **AES-256-GCM encryption**: Enterprise-grade protection for backup files
- **Zero-trust storage**: Backups are encrypted and they are useless to attackers, so you can keep them in shared storages like S3, Azure Blob Storage, etc.
- **Encryption for secrets**: Any sensitive data is encrypted and never exposed, even in logs or error messages
- **Read-only user**: Postgresus uses by default a read-only user for backups and never stores anything that can change your data

### 👥 **Suitable for Teams** <a href="https://postgresus.com/access-management">(docs)</a>

- **Workspaces**: Group databases, notifiers and storages for different projects or teams
- **Access management**: Control who can view or manage specific databases with role-based permissions
- **Audit logs**: Track all system activities and changes made by users
- **User roles**: Assign viewer, member, admin or owner roles within workspaces

### 🎨 **UX-Friendly**

- **Designer-polished UI**: Clean, intuitive interface crafted with attention to detail
- **Dark & light themes**: Choose the look that suits your workflow
- **Mobile adaptive**: Check your backups from anywhere on any device

### 🐳 **Self-Hosted & Secure**

- **Docker-based**: Easy deployment and management
- **Privacy-first**: All your data stays on your infrastructure
- **Open source**: Apache 2.0 licensed, inspect every line of code

### 📦 Installation <a href="https://postgresus.com/installation">(docs)</a>

You have three ways to install Postgresus:

- Script (recommended)
- Simple Docker run
- Docker Compose setup

<img src="assets/healthchecks.svg" alt="Postgresus Dashboard" width="800"/>

---

## 📦 Installation

You have three ways to install Postgresus: automated script (recommended), simple Docker run, or Docker Compose setup.

### Option 1: Automated Installation Script (Recommended, Linux only)

The installation script will:

- ✅ Install Docker with Docker Compose(if not already installed)
- ✅ Set up Postgresus
- ✅ Configure automatic startup on system reboot

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh \
| sudo bash
```

### Option 2: Simple Docker Run

The easiest way to run Postgresus with embedded PostgreSQL:

```bash
docker run -d \
  --name postgresus \
  -p 4005:4005 \
  -v ./postgresus-data:/postgresus-data \
  --restart unless-stopped \
  rostislavdugin/postgresus:latest
```

This single command will:

- ✅ Start Postgresus
- ✅ Store all data in `./postgresus-data` directory
- ✅ Automatically restart on system reboot

### Option 3: Docker Compose Setup

Create a `docker-compose.yml` file with the following configuration:

```yaml
services:
  postgresus:
    container_name: postgresus
    image: rostislavdugin/postgresus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./postgresus-data:/postgresus-data
    restart: unless-stopped
```

Then run:

```bash
docker compose up -d
```

### Option 4: Kubernetes with Helm

For Kubernetes deployments, use the official Helm chart.

**Step 1:** Clone the repository:

```bash
git clone https://github.com/RostislavDugin/postgresus.git
cd postgresus
```

**Step 2:** Install with Helm:

```bash
helm install postgresus ./deploy/helm -n postgresus --create-namespace
```

**Step 3:** Get the external IP:

```bash
kubectl get svc -n postgresus
```

Access Postgresus at `http://<EXTERNAL-IP>` (port 80).

To customize the installation (e.g., storage size, NodePort instead of LoadBalancer), see the [Helm chart README](deploy/helm/README.md) for all configuration options.

---

## 🚀 Usage

1. **Access the dashboard**: Navigate to `http://localhost:4005`
2. **Add first DB for backup**: Click "New Database" and follow the setup wizard
3. **Configure schedule**: Choose from hourly, daily, weekly or monthly intervals
4. **Set database connection**: Enter your PostgreSQL credentials and connection details
5. **Choose storage**: Select where to store your backups (local, S3, Google Drive, etc.)
6. **Add notifications** (optional): Configure email, Telegram, Slack, or webhook notifications
7. **Save and start**: Postgresus will validate settings and begin the backup schedule

### 🔑 Resetting Password <a href="https://postgresus.com/password">(docs)</a>

If you need to reset the password, you can use the built-in password reset command:

```bash
docker exec -it postgresus ./main --new-password="YourNewSecurePassword123" --email="admin"
```

Replace `admin` with the actual email address of the user whose password you want to reset.

---

## 📝 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

---

## 🤝 Contributing

Contributions are welcome! Read <a href="https://postgresus.com/contributing">contributing guide</a> for more details, prioerities and rules are specified there. If you want to contribute, but don't know what and how - message me on Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)
