## Why

Docker storage permission changes can work on one mount type and fail on another. The project needs a real-container test suite for bind mounts, Docker volumes, CIFS, NFS, split mounts, and numeric ownership so these failures are caught before an image is published.

## What Changes

- Add one Bash harness behind `make test-filesystems`. It builds or accepts a local candidate image and runs the same cases locally and in GitHub Actions.
- Test file creation, writing, reading, and removal directly inside the running container under the Databasus and PostgreSQL accounts.
- Cover default and custom account IDs, bind mounts, named volumes, separate mounts, CIFS, NFS root squash, and rejected permission layouts.
- Declare `DATABASUS_PUID=65532`, `DATABASUS_PGID=65532`, `POSTGRES_PUID=999`, and `POSTGRES_PGID=999` in the Docker image. Operators can override them when mounted storage requires other numeric owners.
- Keep identity settings out of ordinary configuration. Document them only on the six Advanced Configuration pages.
- Change production code only when a container case reproduces a Docker entrypoint or image defect. Keep the inline entrypoint and backend local-storage model simple.

## Capabilities

### New Capabilities

- `docker-storage-compatibility`: Defines the real-container filesystem matrix, Docker account variables, local execution, CI execution, and cleanup.

## Impact

The change affects the Docker image, one filesystem test harness, the CI workflow, and Advanced Configuration documentation. Backend local-storage behavior, backup file publication, and connection-test behavior remain unchanged.

The suite tests the current candidate image only. It has no version-to-version upgrade fixtures and does not require a maintained ZFS, Proxmox, or user-namespace runner.
