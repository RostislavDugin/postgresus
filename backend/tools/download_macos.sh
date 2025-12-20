#!/bin/bash

set -e  # Exit on any error

echo "Installing PostgreSQL and MySQL client tools for MacOS..."
echo

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo "Error: This script requires Homebrew to be installed."
    echo "Install Homebrew from: https://brew.sh/"
    exit 1
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

# Update Homebrew
echo "Updating Homebrew..."
brew update

# Install build dependencies
echo "Installing build dependencies..."
brew install wget openssl readline zlib cmake

# ========== PostgreSQL Installation ==========
echo "========================================"
echo "Building PostgreSQL client tools (versions 12-18)..."
echo "========================================"

# PostgreSQL source URLs
declare -A PG_URLS=(
    ["12"]="https://ftp.postgresql.org/pub/source/v12.20/postgresql-12.20.tar.gz"
    ["13"]="https://ftp.postgresql.org/pub/source/v13.16/postgresql-13.16.tar.gz"
    ["14"]="https://ftp.postgresql.org/pub/source/v14.13/postgresql-14.13.tar.gz"
    ["15"]="https://ftp.postgresql.org/pub/source/v15.8/postgresql-15.8.tar.gz"
    ["16"]="https://ftp.postgresql.org/pub/source/v16.4/postgresql-16.4.tar.gz"
    ["17"]="https://ftp.postgresql.org/pub/source/v17.0/postgresql-17.0.tar.gz"
    ["18"]="https://ftp.postgresql.org/pub/source/v18.0/postgresql-18.0.tar.gz"
)

# Create temporary build directory
BUILD_DIR="/tmp/db_tools_build_$$"
mkdir -p "$BUILD_DIR"

echo "Using temporary build directory: $BUILD_DIR"
echo

# Function to build PostgreSQL client tools
build_postgresql_client() {
    local version=$1
    local url=$2
    local version_dir="$POSTGRES_DIR/postgresql-$version"
    
    echo "Building PostgreSQL $version client tools..."
    
    # Skip if already exists
    if [ -f "$version_dir/bin/pg_dump" ]; then
        echo "PostgreSQL $version already installed, skipping..."
        return
    fi
    
    cd "$BUILD_DIR"
    
    # Download source
    echo "  Downloading PostgreSQL $version source..."
    wget -q "$url" -O "postgresql-$version.tar.gz"
    
    # Extract
    echo "  Extracting source..."
    tar -xzf "postgresql-$version.tar.gz"
    cd "postgresql-$version"*
    
    # Configure (client tools only)
    echo "  Configuring build..."
    ./configure \
        --prefix="$version_dir" \
        --with-openssl \
        --with-readline \
        --without-zlib \
        --disable-server \
        --disable-docs \
        --quiet
    
    # Build client tools only
    echo "  Building client tools (this may take a few minutes)..."
    make -s -C src/bin install
    make -s -C src/include install
    make -s -C src/interfaces install
    
    # Verify installation
    if [ -f "$version_dir/bin/pg_dump" ]; then
        echo "  PostgreSQL $version client tools installed successfully"
        
        # Test the installation
        local pg_version=$("$version_dir/bin/pg_dump" --version | cut -d' ' -f3)
        echo "  Verified version: $pg_version"
    else
        echo "  Warning: PostgreSQL $version may not have installed correctly"
    fi
    
    # Clean up source
    cd "$BUILD_DIR"
    rm -rf "postgresql-$version"*
    
    echo
}

# Build each PostgreSQL version
pg_versions="12 13 14 15 16 17 18"

for version in $pg_versions; do
    url=${PG_URLS[$version]}
    if [ -n "$url" ]; then
        build_postgresql_client "$version" "$url"
    else
        echo "Warning: No URL defined for PostgreSQL $version"
    fi
done

# ========== MySQL Installation ==========
echo "========================================"
echo "Installing MySQL client tools (versions 5.7, 8.0, 8.4)..."
echo "========================================"

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    MYSQL_ARCH="arm64"
else
    MYSQL_ARCH="x86_64"
fi

# MySQL download URLs for macOS (using CDN)
# Note: 5.7 is in Downloads, 8.0 and 8.4 specific versions are in archives
declare -A MYSQL_URLS=(
    ["5.7"]="https://cdn.mysql.com/Downloads/MySQL-5.7/mysql-5.7.44-macos10.14-x86_64.tar.gz"
    ["8.0"]="https://cdn.mysql.com/archives/mysql-8.0/mysql-8.0.40-macos14-${MYSQL_ARCH}.tar.gz"
    ["8.4"]="https://cdn.mysql.com/archives/mysql-8.4/mysql-8.4.3-macos14-${MYSQL_ARCH}.tar.gz"
)

# Function to install MySQL client tools
install_mysql_client() {
    local version=$1
    local url=$2
    local version_dir="$MYSQL_DIR/mysql-$version"
    
    echo "Installing MySQL $version client tools..."
    
    # Skip if already exists
    if [ -f "$version_dir/bin/mysqldump" ]; then
        echo "MySQL $version already installed, skipping..."
        return
    fi
    
    mkdir -p "$version_dir/bin"
    cd "$BUILD_DIR"
    
    # Download
    echo "  Downloading MySQL $version..."
    wget -q "$url" -O "mysql-$version.tar.gz" || {
        echo "  Warning: Could not download MySQL $version for $MYSQL_ARCH"
        echo "  You may need to install MySQL $version client tools manually"
        return
    }
    
    # Extract
    echo "  Extracting MySQL $version..."
    tar -xzf "mysql-$version.tar.gz"
    
    # Find extracted directory
    EXTRACTED_DIR=$(ls -d mysql-*/ 2>/dev/null | head -1)
    
    if [ -d "$EXTRACTED_DIR" ] && [ -f "$EXTRACTED_DIR/bin/mysqldump" ]; then
        # Copy client binaries
        cp "$EXTRACTED_DIR/bin/mysql" "$version_dir/bin/" 2>/dev/null || true
        cp "$EXTRACTED_DIR/bin/mysqldump" "$version_dir/bin/" 2>/dev/null || true
        chmod +x "$version_dir/bin/"*
        
        echo "  MySQL $version client tools installed successfully"
        
        # Test the installation
        local mysql_version=$("$version_dir/bin/mysqldump" --version 2>/dev/null | head -1)
        echo "  Verified: $mysql_version"
    else
        echo "  Warning: Could not extract MySQL $version binaries"
        echo "  You may need to install MySQL $version client tools manually"
    fi
    
    # Clean up
    rm -rf "mysql-$version.tar.gz" mysql-*/
    
    echo
}

# Install each MySQL version
mysql_versions="5.7 8.0 8.4"

for version in $mysql_versions; do
    url=${MYSQL_URLS[$version]}
    if [ -n "$url" ]; then
        install_mysql_client "$version" "$url"
    else
        echo "Warning: No URL defined for MySQL $version"
    fi
done

# Clean up build directory
echo "Cleaning up build directory..."
rm -rf "$BUILD_DIR"

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
        pg_version=$("$version_dir/bin/pg_dump" --version | cut -d' ' -f3)
        echo "  postgresql-$version ($pg_version): $version_dir/bin/"
    fi
done

echo
echo "Installed MySQL client versions:"
for version in $mysql_versions; do
    version_dir="$MYSQL_DIR/mysql-$version"
    if [ -f "$version_dir/bin/mysqldump" ]; then
        mysql_version=$("$version_dir/bin/mysqldump" --version 2>/dev/null | head -1)
        echo "  mysql-$version: $version_dir/bin/"
        echo "    $mysql_version"
    fi
done

echo
echo "Usage examples:"
echo "  $POSTGRES_DIR/postgresql-15/bin/pg_dump --version"
echo "  $MYSQL_DIR/mysql-8.0/bin/mysqldump --version"
echo
echo "To add specific versions to your PATH temporarily:"
echo "  export PATH=\"$POSTGRES_DIR/postgresql-15/bin:\$PATH\""
echo "  export PATH=\"$MYSQL_DIR/mysql-8.0/bin:\$PATH\"" 