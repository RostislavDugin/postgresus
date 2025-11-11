This directory is needed only for development and CI\CD.

We have to download and install all the PostgreSQL versions from 12 to 18 locally.
This is needed so we can call pg_dump, pg_dumpall, etc. on each version of the PostgreSQL database.

You do not need to install PostgreSQL fully with all the components.
We only need the client tools (pg_dump, pg_dumpall, psql, etc.) for each version.

We have to install the following:

- PostgreSQL 12
- PostgreSQL 13
- PostgreSQL 14
- PostgreSQL 15
- PostgreSQL 16
- PostgreSQL 17
- PostgreSQL 18

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
- Installs client tools only (no server components)
- May require administrator privileges during installation

### Linux (Debian/Ubuntu)

- Uses the official PostgreSQL APT repository
- Requires sudo privileges to install packages
- Creates symlinks in version-specific directories for consistency

### MacOS

- Requires Homebrew to be installed
- Compiles PostgreSQL from source (client tools only)
- Takes longer than other platforms due to compilation

## Manual Installation

If something goes wrong with the automated scripts, install manually.
The final directory structure should match:

```
./tools/postgresql/postgresql-{version}/bin/pg_dump
./tools/postgresql/postgresql-{version}/bin/pg_dumpall
./tools/postgresql/postgresql-{version}/bin/psql
```

For example:

- `./tools/postgresql/postgresql-12/bin/pg_dump`
- `./tools/postgresql/postgresql-13/bin/pg_dump`
- `./tools/postgresql/postgresql-14/bin/pg_dump`
- `./tools/postgresql/postgresql-15/bin/pg_dump`
- `./tools/postgresql/postgresql-16/bin/pg_dump`
- `./tools/postgresql/postgresql-17/bin/pg_dump`
- `./tools/postgresql/postgresql-18/bin/pg_dump`

## Usage

After installation, you can use version-specific tools:

```bash
# Windows
./postgresql/postgresql-15/bin/pg_dump.exe --version

# Linux/MacOS
./postgresql/postgresql-15/bin/pg_dump --version
```
