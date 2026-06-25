# syntax=docker/dockerfile:1

# Databasus verification image: the official Postgres image plus a curated set of
# common extensions, so a dump's CREATE EXTENSION statements succeed when the
# agent restores a backup for verification (issue #619 PostGIS, plus auxiliary
# extensions like pg_stat_kcache / set_user). This mirrors how the agent already
# uses timescale/timescaledb for TimescaleDB. Built per PG major and published as
# databasus/verification-postgres:<major>.
#
# BASE_IMAGE is digest-pinned by CI (postgres:<major>@sha256:...). The tag-only
# default exists solely for local `docker build --build-arg PG_MAJOR=16` runs.
ARG PG_MAJOR=16
ARG BASE_IMAGE=postgres:${PG_MAJOR}

FROM ${BASE_IMAGE}

# ARGs declared before FROM are not visible in the build stage — redeclare it.
ARG PG_MAJOR

# Extensions whose CREATE EXTENSION works once the files are present, plus the
# Tier-2 monitoring extensions that additionally need shared_preload_libraries
# (set below). The official image already configures the apt.postgresql.org repo
# these packages come from, so every entry must be installable from PGDG apt for
# all majors 12-18 (verified). A missing package fails the build loudly rather
# than silently shipping a half-populated image — important because the agent
# tolerates missing extensions at restore time, so a thinned image would mark
# such backups "verified" without actually exercising the extension.
#
# Intentionally NOT bundled: topn (Citus), pg_hashids, anonymizer
# (postgresql_anonymizer) and pg_stat_monitor (Percona). PGDG apt does not
# package these — they need source builds or Percona's own repo. Add them via
# that route (not this list) if verification coverage for them is ever required.
ARG EXTENSION_PACKAGES="\
postgresql-${PG_MAJOR}-postgis-3 \
postgresql-${PG_MAJOR}-pgrouting \
postgresql-${PG_MAJOR}-h3 \
postgresql-${PG_MAJOR}-pgvector \
postgresql-${PG_MAJOR}-rum \
postgresql-${PG_MAJOR}-partman \
postgresql-${PG_MAJOR}-repack \
postgresql-${PG_MAJOR}-hypopg \
postgresql-${PG_MAJOR}-hll \
postgresql-${PG_MAJOR}-set-user \
postgresql-${PG_MAJOR}-pgaudit \
postgresql-${PG_MAJOR}-ip4r \
postgresql-${PG_MAJOR}-semver \
postgresql-${PG_MAJOR}-http \
postgresql-${PG_MAJOR}-mysql-fdw \
postgresql-${PG_MAJOR}-tds-fdw \
postgresql-${PG_MAJOR}-pg-stat-kcache \
postgresql-${PG_MAJOR}-pg-qualstats \
postgresql-${PG_MAJOR}-pg-wait-sampling \
postgresql-plpython3-${PG_MAJOR} \
postgresql-plperl-${PG_MAJOR} \
postgresql-pltcl-${PG_MAJOR}"

# Tier-2 extensions refuse to CREATE unless their library is preloaded. This list
# must match the packages installed above: a name whose library is absent makes
# Postgres refuse to start. Appending to the initdb sample means every fresh
# PGDATA (the verification container is ephemeral) inherits it; the agent does not
# pass -c shared_preload_libraries, so this value stands at runtime.
ARG PRELOAD_LIBRARIES="pg_stat_statements,pg_stat_kcache,pg_qualstats,pg_wait_sampling,pgaudit,set_user"

# Root is the base image's user; the official entrypoint still gosu-drops to the
# postgres user before exec, so the agent's hardened runtime is unchanged.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ${EXTENSION_PACKAGES}; \
    rm -rf /var/lib/apt/lists/*; \
    printf "shared_preload_libraries = '%s'\n" "${PRELOAD_LIBRARIES}" \
        >> "/usr/share/postgresql/${PG_MAJOR}/postgresql.conf.sample"
