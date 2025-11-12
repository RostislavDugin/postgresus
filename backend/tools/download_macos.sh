#!/bin/bash

set -e  # Exit on any error

echo "Installing PostgreSQL client tools versions 12-18 for MacOS..."
echo

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo "Error: This script requires Homebrew to be installed."
    echo "Install Homebrew from: https://brew.sh/"
    exit 1
fi

# Create postgresql directory
mkdir -p postgresql

# Get absolute path
POSTGRES_DIR="$(pwd)/postgresql"

echo "Installing PostgreSQL client tools to: $POSTGRES_DIR"
echo

# Update Homebrew
echo "Updating Homebrew..."
brew update

# Install build dependencies
echo "Installing build dependencies..."
brew install wget openssl readline zlib

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
BUILD_DIR="/tmp/postgresql_build_$$"
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

# Build each version
versions="12 13 14 15 16 17 18"

for version in $versions; do
    url=${PG_URLS[$version]}
    if [ -n "$url" ]; then
        build_postgresql_client "$version" "$url"
    else
        echo "Warning: No URL defined for PostgreSQL $version"
    fi
done

# Clean up build directory
echo "Cleaning up build directory..."
rm -rf "$BUILD_DIR"

echo "Installation completed!"
echo "PostgreSQL client tools are available in: $POSTGRES_DIR"
echo

# List installed versions
echo "Installed PostgreSQL client versions:"
for version in $versions; do
    version_dir="$POSTGRES_DIR/postgresql-$version"
    if [ -f "$version_dir/bin/pg_dump" ]; then
        pg_version=$("$version_dir/bin/pg_dump" --version | cut -d' ' -f3)
        echo "  postgresql-$version ($pg_version): $version_dir/bin/"
    fi
done

echo
echo "Usage example:"
echo "  $POSTGRES_DIR/postgresql-15/bin/pg_dump --version"
echo
echo "To add a specific version to your PATH temporarily:"
echo "  export PATH=\"$POSTGRES_DIR/postgresql-15/bin:\$PATH\"" 