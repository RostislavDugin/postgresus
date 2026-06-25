FROM golang:1.26-bookworm

ARG NODE_MAJOR=24
ARG PNPM_VERSION=9.15.9
ARG GOOSE_VERSION=v3.27.1
ARG SWAG_VERSION=v1.16.4
ARG GOLANGCI_LINT_VERSION=v2.11.3
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=1000

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl gnupg sudo git \
      postgresql-client \
      zstd \
      python3 python3-pip pre-commit \
      libncurses5 libncurses6 libtinfo5 libtinfo6 \
      libmariadb3 libgnutls30 \
 && mkdir -p /etc/apt/keyrings \
 && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key \
      | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main" \
      > /etc/apt/sources.list.d/nodesource.list \
 && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      | gpg --dearmor -o /etc/apt/keyrings/githubcli-archive-keyring.gpg \
 && chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
      > /etc/apt/sources.list.d/github-cli.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends nodejs gh \
 && rm -rf /var/lib/apt/lists/*

RUN corepack enable

RUN groupadd --gid ${USER_GID} ${USERNAME} \
 && useradd --uid ${USER_UID} --gid ${USER_GID} --create-home --shell /bin/bash ${USERNAME} \
 && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} \
 && chmod 0440 /etc/sudoers.d/${USERNAME} \
 && mkdir -p /home/${USERNAME}/go /home/${USERNAME}/.cache /home/${USERNAME}/.local/share/pnpm \
 && chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}

ENV GOPATH=/home/${USERNAME}/go
ENV PATH=${GOPATH}/bin:${PATH}

USER ${USERNAME}

RUN corepack prepare pnpm@${PNPM_VERSION} --activate

RUN go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION} \
 && go install github.com/swaggo/swag/cmd/swag@${SWAG_VERSION}

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}

USER root
