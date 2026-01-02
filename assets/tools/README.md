We keep binaries here to speed up CI \ CD tasks and building.

Docker image needs:
- PostgreSQL client tools (versions 12-18)
- MySQL client tools (versions 5.7, 8.0, 8.4, 9)
- MariaDB client tools (versions 10.6, 12.1)
- MongoDB Database Tools (latest)

For the most of tools, we need a couple of binaries for each version. However, if we download them on each run, it will download a couple of GBs each time.

So, for speed up we keep only required executables (like pg_dump, mysqldump, mariadb-dump, mongodump, etc.).

It takes:
- ~ 100MB for ARM
- ~ 100MB for x64

Instead of GBs. See Dockefile for usage details.