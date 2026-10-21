set shell := ["bash", "-euo", "pipefail", "-c"]

run *processes:
    scripts/dev.sh {{ processes }}

test:
    scripts/test.sh

pull-testcontainers:
    just --justfile backend/Justfile pull-testcontainers

lint:
    just --justfile backend/Justfile lint
    just --justfile agent/verification/Justfile lint
    pnpm --dir frontend format:check
    pnpm --dir frontend lint
    pnpm --dir frontend typecheck

doctor:
    .agent-hooks/run.sh doctor

format:
    just --justfile backend/Justfile format
    just --justfile agent/verification/Justfile format
    pnpm --dir frontend format

test-filesystem:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ -z "${DATABASUS_IMAGE:-}" ]]; then
        export DATABASUS_IMAGE=databasus-storage-test:local
        docker build --tag "$DATABASUS_IMAGE" .
    fi
    exec bash e2e/docker-storage/run.sh
