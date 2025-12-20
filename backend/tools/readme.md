This directory is needed only for development and CI\CD.

We have to download and install all the PostgreSQL versions from 12 to 18 and MySQL versions 5.7, 8.0, 8.4 locally.
This is needed so we can call pg_dump, pg_restore, mysqldump, mysql, etc. on each version of the database.

You do not need to install the databases fully with all the components.
We only need the client tools for each version.

## Required Versions

### PostgreSQL

- PostgreSQL 12
- PostgreSQL 13
- PostgreSQL 14
- PostgreSQL 15
- PostgreSQL 16
- PostgreSQL 17
- PostgreSQL 18

### MySQL

- MySQL 5.7
- MySQL 8.0
- MySQL 8.4

## Installation

Run the appropriate download script for your platform:

### Windows

```cmd
download_windows.bat
```

### Linux (Debian/Ubuntu)

```bash
chmod +x download_linux.sh
./download_linux.sh
```

### MacOS

```bash
chmod +x download_macos.sh
./download_macos.sh
```

## Platform-Specific Notes

### Windows

- Downloads official PostgreSQL installers from EnterpriseDB
- Downloads official MySQL ZIP archives from dev.mysql.com
- Installs client tools only (no server components)
- May require administrator privileges during PostgreSQL installation

### Linux (Debian/Ubuntu)

- Uses the official PostgreSQL APT repository
- Downloads MySQL client tools from official archives
- Requires sudo privileges to install packages
- Creates symlinks in version-specific directories for consistency

### MacOS

- Requires Homebrew to be installed
- Compiles PostgreSQL from source (client tools only)
- Downloads pre-built MySQL binaries from dev.mysql.com
- Takes longer than other platforms due to PostgreSQL compilation
- Supports both Intel (x86_64) and Apple Silicon (arm64)

## Manual Installation

If something goes wrong with the automated scripts, install manually.
The final directory structure should match:

### PostgreSQL

```
./tools/postgresql/postgresql-{version}/bin/pg_dump
./tools/postgresql/postgresql-{version}/bin/pg_dumpall
./tools/postgresql/postgresql-{version}/bin/psql
./tools/postgresql/postgresql-{version}/bin/pg_restore
```

For example:

- `./tools/postgresql/postgresql-12/bin/pg_dump`
- `./tools/postgresql/postgresql-13/bin/pg_dump`
- `./tools/postgresql/postgresql-14/bin/pg_dump`
- `./tools/postgresql/postgresql-15/bin/pg_dump`
- `./tools/postgresql/postgresql-16/bin/pg_dump`
- `./tools/postgresql/postgresql-17/bin/pg_dump`
- `./tools/postgresql/postgresql-18/bin/pg_dump`

### MySQL

```
./tools/mysql/mysql-{version}/bin/mysqldump
./tools/mysql/mysql-{version}/bin/mysql
```

For example:

- `./tools/mysql/mysql-5.7/bin/mysqldump`
- `./tools/mysql/mysql-8.0/bin/mysqldump`
- `./tools/mysql/mysql-8.4/bin/mysqldump`

## Usage

After installation, you can use version-specific tools:

```bash
# Windows - PostgreSQL
./postgresql/postgresql-15/bin/pg_dump.exe --version

# Windows - MySQL
./mysql/mysql-8.0/bin/mysqldump.exe --version

# Linux/MacOS - PostgreSQL
./postgresql/postgresql-15/bin/pg_dump --version

# Linux/MacOS - MySQL
./mysql/mysql-8.0/bin/mysqldump --version
```

## Environment Variables

The application expects these environment variables to be set (or uses defaults):

```env
# PostgreSQL tools directory (default: ./tools/postgresql)
POSTGRES_INSTALL_DIR=C:\path\to\tools\postgresql

# MySQL tools directory (default: ./tools/mysql)
MYSQL_INSTALL_DIR=C:\path\to\tools\mysql
```

## Troubleshooting

### MySQL 5.7 on Apple Silicon (M1/M2/M3)

MySQL 5.7 does not have native ARM64 binaries for macOS. The script will attempt to download the x86_64 version, which may work under Rosetta 2. If you encounter issues:

1. Ensure Rosetta 2 is installed: `softwareupdate --install-rosetta`
2. Or skip MySQL 5.7 if you don't need to support that version

### Permission Errors on Linux

If you encounter permission errors, ensure you have sudo privileges:

```bash
sudo ./download_linux.sh
```

### Download Failures

If downloads fail, you can manually download the files:

- PostgreSQL: https://www.postgresql.org/ftp/source/
- MySQL: https://dev.mysql.com/downloads/mysql/
