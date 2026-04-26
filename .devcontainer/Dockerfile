FROM mcr.microsoft.com/devcontainers/go:2-1.25-trixie

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    curl \
    git \
    gnupg \
    locales \
    lsb-release \
    make \
    openssh-client \
    postgresql-client \
    python3 \
    python3-pip \
    python3-venv \
    sudo \
    unzip \
    wget \
    && echo "en_US.UTF-8 UTF-8" >> /etc/locale.gen \
    && locale-gen \
    && rm -rf /var/lib/apt/lists/*

# Node 22 LTS + pnpm (via corepack)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && corepack enable pnpm

# Ansible in a venv
RUN python3 -m venv /opt/ansible \
    && /opt/ansible/bin/pip install --upgrade pip \
    && /opt/ansible/bin/pip install ansible
ENV PATH="/opt/ansible/bin:$PATH"

# golang-migrate
RUN ARCH=$(dpkg --print-architecture) \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-${ARCH}.tar.gz \
    | tar -xz -C /usr/local/bin \
    && chmod +x /usr/local/bin/migrate

# golangci-lint v2
RUN ARCH=$(dpkg --print-architecture) \
    && curl -sSfL \
    "https://github.com/golangci/golangci-lint/releases/download/v2.11.3/golangci-lint-2.11.3-linux-${ARCH}.tar.gz" \
    | tar -xz --strip-components=1 -C /usr/local/bin \
    golangci-lint-2.11.3-linux-${ARCH}/golangci-lint

# Go tools
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install golang.org/x/tools/cmd/goimports@latest
