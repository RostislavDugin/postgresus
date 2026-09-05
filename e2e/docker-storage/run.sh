#!/usr/bin/env bash
set -Eeuo pipefail

readonly harness_label_key="com.databasus.storage-test"
run_id="${RANDOM}-$$-$(date +%s)"
readonly run_id
readonly harness_label="${harness_label_key}=${run_id}"
readonly selected_case="${CASE:-all}"
readonly databasus_image="${DATABASUS_IMAGE:?DATABASUS_IMAGE must be set}"
readonly samba_fixture_image="dockurr/samba@sha256:dbd8ebd6392cc1e8e98312d96d163f0039b644b84aee741d89efdd0abd3d32bf"
readonly nfs_fixture_image="itsthenetwork/nfs-server-alpine@sha256:7fa99ae65c23c5af87dd4300e543a86b119ed15ba61422444207efc7abd0ba20"

case_root=""

cleanup() {
    local container_id
    local container_name
    local volume_name
    local volume_type
    local -a client_container_ids=()
    local -a server_container_ids=()
    local -a volume_names=()
    local -a network_ids=()

    while read -r container_id container_name; do
        [[ -n "${container_id}" ]] || continue
        if [[ "${container_name}" == *-server-* ]]; then
            server_container_ids+=("${container_id}")
        else
            client_container_ids+=("${container_id}")
        fi
    done < <(docker ps -a --filter "label=${harness_label}" --format '{{.ID}} {{.Names}}' 2>/dev/null)

    if (("${#client_container_ids[@]}")); then
        docker rm -f "${client_container_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t volume_names < <(docker volume ls -q --filter "label=${harness_label}" 2>/dev/null)
    for volume_name in "${volume_names[@]}"; do
        volume_type="$(docker volume inspect --format '{{index .Options "type"}}' "${volume_name}" 2>/dev/null || true)"
        if [[ "${volume_type}" == "cifs" || "${volume_type}" == "nfs" ]]; then
            docker volume rm -f "${volume_name}" >/dev/null 2>&1 || true
        fi
    done

    if (("${#server_container_ids[@]}")); then
        docker rm -f "${server_container_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t network_ids < <(docker network ls -q --filter "label=${harness_label}" 2>/dev/null)
    if (("${#network_ids[@]}")); then
        docker network rm "${network_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t volume_names < <(docker volume ls -q --filter "label=${harness_label}" 2>/dev/null)
    if (("${#volume_names[@]}")); then
        docker volume rm -f "${volume_names[@]}" >/dev/null 2>&1 || true
    fi

    if [[ "${case_root}" == /tmp/databasus-storage-test.* ]]; then
        docker run --rm --label "${harness_label}" --entrypoint /bin/chmod \
            --volume "${case_root}:/case-root" "${databasus_image}" \
            -R 0777 /case-root >/dev/null 2>&1 || true
        rm -rf -- "${case_root}" || true
    fi
}

trap cleanup EXIT
case_root="$(mktemp -d /tmp/databasus-storage-test.XXXXXXXX)"

wait_for_health() {
    local container_name="$1"
    local health_status

    for ((readiness_attempt = 1; readiness_attempt <= 90; readiness_attempt++)); do
        health_status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${container_name}")"
        if [[ "${health_status}" == "healthy" ]]; then
            return
        fi
        if [[ "$(docker inspect --format '{{.State.Running}}' "${container_name}")" != "true" ]]; then
            docker logs "${container_name}" >&2
            return 1
        fi
        sleep 1
    done

    docker logs "${container_name}" >&2
    echo "container did not become healthy" >&2
    return 1
}

wait_for_log() {
    local container_name="$1"
    local required_log_line="$2"

    for ((log_attempt = 1; log_attempt <= 60; log_attempt++)); do
        if docker logs "${container_name}" 2>&1 | grep -Fq "${required_log_line}"; then
            return
        fi
        sleep 1
    done

    docker logs "${container_name}" >&2
    echo "container did not report readiness: ${required_log_line}" >&2
    return 1
}

wait_for_exit() {
    local container_name="$1"

    for ((exit_attempt = 1; exit_attempt <= 100; exit_attempt++)); do
        if [[ "$(docker inspect --format '{{.State.Running}}' "${container_name}")" != "true" ]]; then
            [[ "$(docker inspect --format '{{.State.ExitCode}}' "${container_name}")" != "0" ]]
            return
        fi
        sleep 0.1
    done

    echo "container did not stop" >&2
    return 1
}

start_databasus() {
    local container_name="$1"
    shift

    docker run --detach \
        --name "${container_name}" \
        --label "${harness_label}" \
        "$@" \
        "${databasus_image}" >/dev/null
    wait_for_health "${container_name}"
}

probe_path() {
    local container_name="$1"
    local account_name="$2"
    local mounted_path="$3"

    docker exec --user "${account_name}" "${container_name}" /bin/sh -eu -c '
        probe_path="$1/.databasus-filesystem-probe-$$"
        printf filesystem-probe > "$probe_path"
        [ "$(cat "$probe_path")" = filesystem-probe ]
        rm "$probe_path"
        [ ! -e "$probe_path" ]
    ' sh "${mounted_path}"
}

assert_service_boundaries() {
    local container_name="$1"

    docker exec --user databasus "${container_name}" test ! -r /databasus-data/pgdata/PG_VERSION
    docker exec --user databasus "${container_name}" test ! -x /databasus-data/pgsocket
}

probe_service_paths() {
    local container_name="$1"

    probe_path "${container_name}" databasus /databasus-data/temp
    probe_path "${container_name}" databasus /databasus-data/backups
    probe_path "${container_name}" postgres /databasus-data/pgdata
    assert_service_boundaries "${container_name}"
}

run_default_ids() {
    local container_name="databasus-storage-default-ids-${run_id}"
    local volume_name="databasus-storage-default-ids-${run_id}"
    local rejected_remap_command="${case_root}/reject-identity-remap"
    local image_environment
    local container_environment
    local identity_setting

    image_environment="$(docker image inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${databasus_image}")"
    for identity_setting in DATABASUS_PUID=65532 DATABASUS_PGID=65532 POSTGRES_PUID=999 POSTGRES_PGID=999; do
        grep -Fxq "${identity_setting}" <<<"${image_environment}"
    done

    printf '#!/bin/sh\nexit 1\n' > "${rejected_remap_command}"
    chmod 0755 "${rejected_remap_command}"
    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --mount "type=bind,source=${rejected_remap_command},target=/usr/sbin/groupmod,readonly" \
        --mount "type=bind,source=${rejected_remap_command},target=/usr/sbin/usermod,readonly" \
        --volume "${volume_name}:/databasus-data"

    container_environment="$(docker exec "${container_name}" env)"
    for identity_setting in DATABASUS_PUID=65532 DATABASUS_PGID=65532 POSTGRES_PUID=999 POSTGRES_PGID=999; do
        grep -Fxq "${identity_setting}" <<<"${container_environment}"
    done

    [[ "$(docker exec "${container_name}" id -u databasus)" == "65532" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "65532" ]]
    [[ "$(docker exec "${container_name}" id -u postgres)" == "999" ]]
    [[ "$(docker exec "${container_name}" id -g postgres)" == "999" ]]
    probe_service_paths "${container_name}"
}

run_custom_ids() {
    local container_name="databasus-storage-custom-ids-${run_id}"
    local volume_name="databasus-storage-custom-ids-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --env DATABASUS_PUID=25001 \
        --env DATABASUS_PGID=25002 \
        --env POSTGRES_PUID=26001 \
        --env POSTGRES_PGID=26002 \
        --volume "${volume_name}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "25001" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "25002" ]]
    [[ "$(docker exec "${container_name}" id -u postgres)" == "26001" ]]
    [[ "$(docker exec "${container_name}" id -g postgres)" == "26002" ]]
    probe_service_paths "${container_name}"
}

run_bind_root() {
    local container_name="databasus-storage-bind-${run_id}"
    local data_root="${case_root}/bind-root"

    mkdir -p "${data_root}"
    chmod 0770 "${data_root}"
    start_databasus "${container_name}" --volume "${data_root}:/databasus-data"
    probe_service_paths "${container_name}"
}

run_named_volume() {
    local container_name="databasus-storage-volume-${run_id}"
    local volume_name="databasus-storage-volume-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" --volume "${volume_name}:/databasus-data"
    probe_service_paths "${container_name}"
}

run_split_mounts() {
    local container_name="databasus-storage-split-${run_id}"
    local data_volume_name="databasus-storage-split-data-${run_id}"
    local backup_volume_name="databasus-storage-split-backup-${run_id}"
    local pgdata_volume_name="databasus-storage-split-pgdata-${run_id}"
    local temporary_device_id
    local backup_device_id

    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${backup_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${pgdata_volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --tmpfs /databasus-data/temp:rw,nosuid,size=64m,mode=0700,uid=65532,gid=65532 \
        --volume "${backup_volume_name}:/databasus-data/backups" \
        --volume "${pgdata_volume_name}:/databasus-data/pgdata"

    temporary_device_id="$(docker exec "${container_name}" stat -c %d /databasus-data/temp)"
    backup_device_id="$(docker exec "${container_name}" stat -c %d /databasus-data/backups)"
    [[ "${temporary_device_id}" != "${backup_device_id}" ]]
    probe_service_paths "${container_name}"
}

run_read_only() {
    local container_name="databasus-storage-read-only-${run_id}"
    local data_root="${case_root}/read-only"

    mkdir -p "${data_root}/pgdata" "${data_root}/pgsocket" "${data_root}/temp" "${data_root}/backups"
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${data_root}:/data" "${databasus_image}" -c \
        'chown 65532:65532 /data /data/temp /data/backups && chown 999:999 /data/pgdata /data/pgsocket && chmod 0770 /data /data/backups && chmod 0700 /data/pgdata /data/pgsocket /data/temp'
    docker run --detach \
        --name "${container_name}" \
        --label "${harness_label}" \
        --mount "type=bind,source=${data_root},target=/databasus-data,readonly" \
        "${databasus_image}" >/dev/null
    wait_for_exit "${container_name}"
    docker logs "${container_name}" 2>&1 | grep -Fqi "Read-only file system"
}

run_inaccessible_path() {
    local container_name="databasus-storage-inaccessible-${run_id}"
    local volume_name="databasus-storage-inaccessible-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" --volume "${volume_name}:/databasus-data"
    docker exec "${container_name}" chmod 000 /databasus-data/backups
    if probe_path "${container_name}" databasus /databasus-data/backups >/dev/null 2>&1; then
        echo "Databasus wrote to an inaccessible backup path" >&2
        return 1
    fi
}

run_network_prerequisites() {
    [[ "$(uname -s)" == "Linux" ]]
    command -v mount.cifs >/dev/null || {
        echo "mount.cifs is required; install cifs-utils on the Docker host" >&2
        return 1
    }
    command -v mount.nfs >/dev/null || {
        echo "mount.nfs is required; install nfs-common on the Docker host" >&2
        return 1
    }

    docker image inspect "${samba_fixture_image}" >/dev/null 2>&1 || docker pull "${samba_fixture_image}" >/dev/null
    docker image inspect "${nfs_fixture_image}" >/dev/null 2>&1 || docker pull "${nfs_fixture_image}" >/dev/null
}

run_cifs() {
    local network_name="databasus-storage-cifs-${run_id}"
    local server_container_name="databasus-storage-cifs-server-${run_id}"
    local client_container_name="databasus-storage-cifs-client-${run_id}"
    local server_volume_name="databasus-storage-cifs-server-${run_id}"
    local data_volume_name="databasus-storage-cifs-data-${run_id}"
    local backup_volume_name="databasus-storage-cifs-backup-${run_id}"
    local temporary_volume_name="databasus-storage-cifs-temp-${run_id}"
    local server_address

    run_network_prerequisites
    docker network create --label "${harness_label}" "${network_name}" >/dev/null
    docker volume create --label "${harness_label}" "${server_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${server_volume_name}:/shared" "${databasus_image}" -c \
        'chown 25001:25002 /shared && chmod 0770 /shared'
    docker run --detach \
        --name "${server_container_name}" \
        --label "${harness_label}" \
        --network "${network_name}" \
        --health-interval 1s \
        --env NAME=Storage \
        --env USER=storage-test \
        --env PASS=storage-test-password \
        --env UID=25001 \
        --env GID=25002 \
        --volume "${server_volume_name}:/shared" \
        "${samba_fixture_image}" >/dev/null
    wait_for_health "${server_container_name}"

    server_address="$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${server_container_name}")"
    for volume_name in "${backup_volume_name}" "${temporary_volume_name}"; do
        docker volume create \
            --label "${harness_label}" \
            --driver local \
            --opt type=cifs \
            --opt "device=//${server_address}/Storage" \
            --opt "o=username=storage-test,password=storage-test-password,vers=3.0,uid=25001,gid=25002,file_mode=0660,dir_mode=0770" \
            "${volume_name}" >/dev/null
    done

    start_databasus "${client_container_name}" \
        --env DATABASUS_PUID=25001 \
        --env DATABASUS_PGID=25002 \
        --env TMPDIR=/databasus-data/temp \
        --volume "${data_volume_name}:/databasus-data" \
        --volume "${temporary_volume_name}:/databasus-data/temp" \
        --volume "${backup_volume_name}:/databasus-data/backups"

    probe_path "${client_container_name}" databasus /databasus-data/temp
    probe_path "${client_container_name}" databasus /databasus-data/backups
    docker exec --user databasus "${client_container_name}" /bin/sh -c \
        "tr '\\0' '\\n' < /proc/1/environ | grep -Fxq TMPDIR=/tmp"
    assert_service_boundaries "${client_container_name}"
}

run_nfs_root_squash() {
    local network_name="databasus-storage-nfs-${run_id}"
    local server_container_name="databasus-storage-nfs-server-${run_id}"
    local client_container_name="databasus-storage-nfs-client-${run_id}"
    local server_volume_name="databasus-storage-nfs-server-${run_id}"
    local data_volume_name="databasus-storage-nfs-data-${run_id}"
    local client_volume_name="databasus-storage-nfs-client-${run_id}"
    local server_address

    run_network_prerequisites
    docker network create --label "${harness_label}" "${network_name}" >/dev/null
    docker volume create --label "${harness_label}" "${server_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${server_volume_name}:/nfsshare" "${databasus_image}" -c \
        'chown 65532:65532 /nfsshare && chmod 0770 /nfsshare'
    docker run --detach \
        --name "${server_container_name}" \
        --label "${harness_label}" \
        --network "${network_name}" \
        --privileged \
        --env SHARED_DIRECTORY=/nfsshare \
        --volume "${server_volume_name}:/nfsshare" \
        --entrypoint /bin/bash \
        "${nfs_fixture_image}" -c \
        "mount -t nfsd nfsd /proc/fs/nfsd && printf 10 > /proc/fs/nfsd/nfsv4gracetime && sed -i 's/no_root_squash/root_squash/' /etc/exports && exec /usr/bin/nfsd.sh" >/dev/null
    wait_for_log "${server_container_name}" "Startup successful."

    server_address="$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${server_container_name}")"
    docker volume create \
        --label "${harness_label}" \
        --driver local \
        --opt type=nfs \
        --opt "o=addr=${server_address},rw,nfsvers=4" \
        --opt device=:/ \
        "${client_volume_name}" >/dev/null

    if docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${client_volume_name}:/export" "${databasus_image}" -c \
        'chown 1:1 /export' >/dev/null 2>&1; then
        echo "root-squashed NFS accepted a client-root ownership change" >&2
        return 1
    fi
    start_databasus "${client_container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --volume "${client_volume_name}:/databasus-data/backups"
    probe_service_paths "${client_container_name}"
    docker rm -f "${client_container_name}" >/dev/null
    start_databasus "${client_container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --volume "${client_volume_name}:/databasus-data/backups"
    probe_service_paths "${client_container_name}"
}

run_case() {
    local case_name="$1"

    echo "Running Docker filesystem case: ${case_name}"
    case "${case_name}" in
        build-only) docker image inspect "${databasus_image}" >/dev/null ;;
        default-ids) run_default_ids ;;
        custom-ids) run_custom_ids ;;
        bind-root) run_bind_root ;;
        named-volume) run_named_volume ;;
        split-mounts) run_split_mounts ;;
        read-only) run_read_only ;;
        inaccessible-path) run_inaccessible_path ;;
        network-prerequisites) run_network_prerequisites ;;
        cifs) run_cifs ;;
        nfs-root-squash) run_nfs_root_squash ;;
        *)
            echo "unknown Docker filesystem case: ${case_name}" >&2
            return 2
            ;;
    esac
}

# Issue coverage:
# #45, #64 and #224: bind and split mounts.
# #141, #478 and #738: default and custom service IDs without default remapping.
# #274: CIFS forced modes plus a private process TMPDIR.
# #751: PostgreSQL traversal through the bind-mounted data root.
# #763: CIFS numeric ownership.
# #96 and #199 require an arbitrary container user, which cannot start both services.
# #401 concerns application-level transfer policy, not container permissions.
# #431 exercises the existing cross-device move fallback, not container permissions.

if [[ "${selected_case}" == all ]]; then
    for case_name in \
        build-only default-ids custom-ids bind-root named-volume split-mounts \
        read-only inaccessible-path cifs nfs-root-squash; do
        run_case "${case_name}"
        cleanup
        case_root="$(mktemp -d /tmp/databasus-storage-test.XXXXXXXX)"
    done
else
    run_case "${selected_case}"
fi
