#!/bin/bash

set -e  # Exit on any error

# Ensure non-interactive mode for apt
export DEBIAN_FRONTEND=noninteractive

echo "Installing PostgreSQL client tools versions 12-18 for Linux (Debian/Ubuntu)..."
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

# Create postgresql directory
mkdir -p postgresql

# Get absolute path
POSTGRES_DIR="$(pwd)/postgresql"

echo "Installing PostgreSQL client tools to: $POSTGRES_DIR"
echo

# Add PostgreSQL official APT repository
echo "Adding PostgreSQL official APT repository..."
$SUDO apt-get update -qq -y
$SUDO apt-get install -y -qq wget ca-certificates

# Add GPG key
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | $SUDO apt-key add - 2>/dev/null

# Add repository
echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" | $SUDO tee /etc/apt/sources.list.d/pgdg.list >/dev/null

# Update package list
echo "Updating package list..."
$SUDO apt-get update -qq -y

# Install client tools for each version
versions="12 13 14 15 16 17 18"

for version in $versions; do
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

echo "Installation completed!"
echo "PostgreSQL client tools are available in: $POSTGRES_DIR"
echo

# List installed versions
echo "Installed PostgreSQL client versions:"
for version in $versions; do
    version_dir="$POSTGRES_DIR/postgresql-$version"
    if [ -f "$version_dir/bin/pg_dump" ]; then
        echo "  postgresql-$version: $version_dir/bin/"
        # Verify the correct version
        version_output=$("$version_dir/bin/pg_dump" --version 2>/dev/null | grep -o "pg_dump (PostgreSQL) [0-9]\+\.[0-9]\+")
        echo "    Version check: $version_output"
    fi
done

echo
echo "Usage example:"
echo "  $POSTGRES_DIR/postgresql-15/bin/pg_dump --version" 