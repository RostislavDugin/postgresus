#!/bin/bash

set -e  # Exit on any error

# Check if script is run as root
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root (sudo ./install-databasus.sh)" >&2
    exit 1
fi

# Set up logging
LOG_FILE="/var/log/databasus-install.log"
INSTALL_DIR="/opt/databasus"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Create log file if doesn't exist
touch "$LOG_FILE"
log "Starting Databasus installation..."

# Create installation directory
log "Creating installation directory..."
if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    log "Created directory: $INSTALL_DIR"
else
    log "Directory already exists: $INSTALL_DIR"
fi

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION_CODENAME=${VERSION_CODENAME:-}
    else
        log "ERROR: Cannot detect OS. /etc/os-release not found."
        exit 1
    fi
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log "Docker not found. Installing Docker..."
    
    detect_os
    log "Detected OS: $OS, Codename: $VERSION_CODENAME"
    
    # Install prerequisites
    apt-get update
    apt-get install -y ca-certificates curl gnupg

    # Set up Docker repository
    install -m 0755 -d /etc/apt/keyrings
    
    # Determine Docker repo URL based on OS
    case "$OS" in
        ubuntu)
            DOCKER_URL="https://download.docker.com/linux/ubuntu"
            # Fallback for unsupported versions
            case "$VERSION_CODENAME" in
                plucky|oracular) VERSION_CODENAME="noble" ;;  # Ubuntu 25.x -> 24.04
            esac
            ;;
        debian)
            DOCKER_URL="https://download.docker.com/linux/debian"
            # Fallback for unsupported versions
            case "$VERSION_CODENAME" in
                trixie|forky) VERSION_CODENAME="bookworm" ;;  # Debian 13/14 -> 12
            esac
            ;;
        *)
            log "ERROR: Unsupported OS: $OS. Please install Docker manually."
            exit 1
            ;;
    esac
    
    log "Using Docker repository: $DOCKER_URL with codename: $VERSION_CODENAME"
    
    # Download and add Docker GPG key (no sudo needed - already root)
    curl -fsSL "$DOCKER_URL/gpg" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    
    # Add Docker repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] $DOCKER_URL $VERSION_CODENAME stable" > /etc/apt/sources.list.d/docker.list
    
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # Verify Docker installation
    if ! command -v docker &> /dev/null; then
        log "ERROR: Docker installation failed!"
        exit 1
    fi
    
    log "Docker installed successfully"
else
    log "Docker already installed"
fi

# Check if docker compose is available
if ! docker compose version &> /dev/null; then
    log "ERROR: Docker Compose plugin not available!"
    exit 1
else
    log "Docker Compose available"
fi

# Write docker-compose.yml
log "Writing docker-compose.yml to $INSTALL_DIR"
cat > "$INSTALL_DIR/docker-compose.yml" << 'EOF'
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
EOF
log "docker-compose.yml created successfully"

# Start Databasus
log "Starting Databasus..."
cd "$INSTALL_DIR"
if docker compose up -d; then
    log "Databasus started successfully"
else
    log "ERROR: Failed to start Databasus!"
    exit 1
fi

log "Databasus installation completed successfully!"
log "-------------------------------------------"
log "To launch:"
log "> cd $INSTALL_DIR && docker compose up -d"
log "Access Databasus at: http://localhost:4005"
