Pre-built DB client binaries committed to the repo so that local dev,
CI, and the Docker image all read from one place. The Go backend
resolves them at runtime via `runtime.GOOS`+`runtime.GOARCH` →
`assets/tools/<arch-key>/<db>/<db>-<v>/bin/<command>[.exe]`.

Layout (one subtree per arch, identical shape):

```
assets/tools/<arch>/
  postgresql/postgresql-{12,13,14,15,16,17,18}/bin/
    pg_dump, pg_restore, psql   (+ minimal *.dll set on Windows)
  mysql/mysql-{5.7,8.0,8.4,9}/bin/
    mysql, mysqldump
  mariadb/mariadb-{10.6,12.1}/bin/
    mariadb, mariadb-dump
  mongodb/bin/
    mongodump, mongorestore
```

`<arch>` keys (mapping in `backend/internal/util/tools/paths.go`):

| GOOS / GOARCH       | key       | size  |
|---------------------|-----------|-------|
| `linux` / `amd64`   | `x64`     | ~160 MB |
| `linux` / `arm64`   | `arm`     | ~125 MB |
| `windows` / `amd64` | `win-x64` | ~120 MB |

Notes:
- MySQL `5.7` is amd64-only — `arm/mysql/mysql-5.7/` is intentionally absent.
- MariaDB ships two client versions: legacy `10.6` (for MariaDB servers 5.5 / 10.1) and modern `12.1` (for 10.2+). The mapping lives in `tools/mariadb.go`.
- MongoDB Database Tools are version 100.16.1 across all arches and are backward-compatible with all supported server versions (4.2 – 8.2). MongoDB 4.0 is not supported (wire version 7, requires older mongodump).

To refresh a tool set, drop the corresponding `bin/` contents in place
and commit. There are no install scripts in this directory; binaries
are sourced from the upstream vendor downloads.
