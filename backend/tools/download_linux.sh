#!/bin/bash

set -e  # Exit on any error

# Ensure non-interactive mode for apt
export DEBIAN_FRONTEND=noninteractive

echo "Installing PostgreSQL and MySQL client tools for Linux (Debian/Ubuntu)..."
echo

# Check if running on supported system
if ! command -v apt-get &> /dev/null; then
    echo "Error: This script requires apt-get (Debian/Ubuntu-like system)"
    exit 1
fi

# Check if running as root or with sudo
if [[ $EUID -eq 0 ]]; then
    SUDO=""
else
    SUDO="sudo"
    echo "This script requires sudo privileges to install packages."
fi

# Create directories
mkdir -p postgresql
mkdir -p mysql

# Get absolute paths
POSTGRES_DIR="$(pwd)/postgresql"
MYSQL_DIR="$(pwd)/mysql"

echo "Installing PostgreSQL client tools to: $POSTGRES_DIR"
echo "Installing MySQL client tools to: $MYSQL_DIR"
echo

# ========== PostgreSQL Installation ==========
echo "========================================"
echo "Installing PostgreSQL client tools (versions 12-18)..."
echo "========================================"

# Add PostgreSQL official APT repository
echo "Adding PostgreSQL official APT repository..."
$SUDO apt-get update -qq -y
$SUDO apt-get install -y -qq wget ca-certificates gnupg lsb-release

# Add GPG key
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | $SUDO apt-key add - 2>/dev/null

# Add repository
echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" | $SUDO tee /etc/apt/sources.list.d/pgdg.list >/dev/null

# Update package list
echo "Updating package list..."
$SUDO apt-get update -qq -y

# Install PostgreSQL client tools for each version
pg_versions="12 13 14 15 16 17 18"

for version in $pg_versions; do
    echo "Installing PostgreSQL $version client tools..."
    
    # Install client tools only
    $SUDO apt-get install -y -qq postgresql-client-$version
    
    # Create version-specific directory and symlinks
    version_dir="$POSTGRES_DIR/postgresql-$version"
    mkdir -p "$version_dir/bin"
    
    # On Debian/Ubuntu, PostgreSQL binaries are located in /usr/lib/postgresql/{version}/bin/
    pg_bin_dir="/usr/lib/postgresql/$version/bin"
    
    if [ -d "$pg_bin_dir" ] && [ -f "$pg_bin_dir/pg_dump" ]; then
        # Create symlinks to the version-specific binaries
        ln -sf "$pg_bin_dir/pg_dump" "$version_dir/bin/pg_dump"
        ln -sf "$pg_bin_dir/pg_dumpall" "$version_dir/bin/pg_dumpall"
        ln -sf "$pg_bin_dir/psql" "$version_dir/bin/psql"
        ln -sf "$pg_bin_dir/pg_restore" "$version_dir/bin/pg_restore"
        ln -sf "$pg_bin_dir/createdb" "$version_dir/bin/createdb"
        ln -sf "$pg_bin_dir/dropdb" "$version_dir/bin/dropdb"
        
        echo "PostgreSQL $version client tools installed successfully"
    else
        echo "Error: PostgreSQL $version binaries not found in expected location: $pg_bin_dir"
        echo "Available PostgreSQL directories:"
        ls -la /usr/lib/postgresql/ 2>/dev/null || echo "No PostgreSQL directories found in /usr/lib/postgresql/"
        if [ -d "$pg_bin_dir" ]; then
            echo "Contents of $pg_bin_dir:"
            ls -la "$pg_bin_dir" 2>/dev/null || echo "Directory exists but cannot list contents"
        fi
        exit 1
    fi
    echo
done

# ========== MySQL Installation ==========
echo "========================================"
echo "Installing MySQL client tools (versions 5.7, 8.0, 8.4)..."
echo "========================================"

# Download and extract MySQL client tools
mysql_versions="5.7 8.0 8.4"

for version in $mysql_versions; do
    echo "Installing MySQL $version client tools..."
    
    version_dir="$MYSQL_DIR/mysql-$version"
    mkdir -p "$version_dir/bin"
    
    # Download MySQL client tools from official CDN
    # Note: 5.7 is in Downloads, 8.0 and 8.4 specific versions are in archives
    case $version in
        "5.7")
            MYSQL_URL="https://cdn.mysql.com/Downloads/MySQL-5.7/mysql-5.7.44-linux-glibc2.12-x86_64.tar.gz"
            ;;
        "8.0")
            MYSQL_URL="https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.40-linux-glibc2.17-x86_64-minimal.tar.xz"
            ;;
        "8.4")
            MYSQL_URL="https://cdn.mysql.com/archives/mysql-8.4/mysql-8.4.3-linux-glibc2.17-x86_64-minimal.tar.xz"
            ;;
    esac
    
    TEMP_DIR="/tmp/mysql_install_$version"
    mkdir -p "$TEMP_DIR"
    cd "$TEMP_DIR"
    
    echo "  Downloading MySQL $version..."
    wget -q "$MYSQL_URL" -O "mysql-$version.tar.gz" || wget -q "$MYSQL_URL" -O "mysql-$version.tar.xz"
    
    echo "  Extracting MySQL $version..."
    if [[ "$MYSQL_URL" == *.xz ]]; then
        tar -xJf "mysql-$version.tar.xz" 2>/dev/null || tar -xJf "mysql-$version.tar.gz" 2>/dev/null
    else
        tar -xzf "mysql-$version.tar.gz" 2>/dev/null || tar -xzf "mysql-$version.tar.xz" 2>/dev/null
    fi
    
    # Find extracted directory
    EXTRACTED_DIR=$(ls -d mysql-*/ 2>/dev/null | head -1)
    
    if [ -d "$EXTRACTED_DIR" ] && [ -f "$EXTRACTED_DIR/bin/mysqldump" ]; then
        # Copy client binaries
        cp "$EXTRACTED_DIR/bin/mysql" "$version_dir/bin/" 2>/dev/null || true
        cp "$EXTRACTED_DIR/bin/mysqldump" "$version_dir/bin/" 2>/dev/null || true
        chmod +x "$version_dir/bin/"*
        
        echo "  MySQL $version client tools installed successfully"
    else
        echo "  Warning: Could not extract MySQL $version binaries"
        echo "  You may need to install MySQL $version client tools manually"
    fi
    
    # Cleanup
    cd - >/dev/null
    rm -rf "$TEMP_DIR"
    echo
done

echo "========================================"
echo "Installation completed!"
echo "========================================"
echo
echo "PostgreSQL client tools are available in: $POSTGRES_DIR"
echo "MySQL client tools are available in: $MYSQL_DIR"
echo

# List installed PostgreSQL versions
echo "Installed PostgreSQL client versions:"
for version in $pg_versions; do
    version_dir="$POSTGRES_DIR/postgresql-$version"
    if [ -f "$version_dir/bin/pg_dump" ]; then
        echo "  postgresql-$version: $version_dir/bin/"
        version_output=$("$version_dir/bin/pg_dump" --version 2>/dev/null | grep -o "pg_dump (PostgreSQL) [0-9]\+\.[0-9]\+")
        echo "    Version check: $version_output"
    fi
done

echo
echo "Installed MySQL client versions:"
for version in $mysql_versions; do
    version_dir="$MYSQL_DIR/mysql-$version"
    if [ -f "$version_dir/bin/mysqldump" ]; then
        echo "  mysql-$version: $version_dir/bin/"
        version_output=$("$version_dir/bin/mysqldump" --version 2>/dev/null | head -1)
        echo "    Version check: $version_output"
    fi
done

echo
echo "Usage examples:"
echo "  $POSTGRES_DIR/postgresql-15/bin/pg_dump --version"
echo "  $MYSQL_DIR/mysql-8.0/bin/mysqldump --version" 