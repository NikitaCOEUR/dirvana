FROM debian:bookworm-slim

# =====================
# Build variables
# =====================
ARG GO_VERSION=1.25.1
ARG KUBECTL_VERSION=stable
ARG TERRAFORM_VERSION=1.9.8
ARG AQUA_VERSION=2.55.0

# =====================
# Dépendances système
# =====================
RUN apt-get update && apt-get install -y \
    bash \
    bash-completion \
    zsh \
    zsh-common \
    expect \
    curl \
    wget \
    unzip \
    gnupg \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# =====================
# Go
# =====================
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"

# =====================
# kubectl
# =====================
RUN if [ "$KUBECTL_VERSION" = "stable" ]; then \
      KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt); \
    fi \
    && curl -fsSLo /usr/local/bin/kubectl \
      https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
    && chmod +x /usr/local/bin/kubectl

# =====================
# terraform
# =====================
RUN curl -fsSLo terraform.zip \
      https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip \
    && unzip terraform.zip \
    && mv terraform /usr/local/bin/ \
    && rm terraform.zip

# =====================
# aqua
# =====================
RUN curl -fsSL \
      https://github.com/aquaproj/aqua/releases/download/v${AQUA_VERSION}/aqua_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin \
    && chmod +x /usr/local/bin/aqua

# =====================
# Installation of native completions
# =====================
# kubectl completion pour bash et zsh
RUN kubectl completion bash > /etc/bash_completion.d/kubectl && \
    mkdir -p /usr/share/zsh/site-functions && \
    kubectl completion zsh > /usr/share/zsh/site-functions/_kubectl

# terraform completion
RUN terraform -install-autocomplete || true

# aqua completion pour bash et zsh
RUN aqua completion bash > /etc/bash_completion.d/aqua && \
    aqua completion zsh > /usr/share/zsh/site-functions/_aqua

# Configuration des shells pour charger les completions
RUN echo 'source /etc/bash_completion' >> /root/.bashrc && \
    echo 'fpath=(/usr/share/zsh/site-functions $fpath)' >> /root/.zshrc && \
    echo 'autoload -Uz compinit' >> /root/.zshrc && \
    echo 'compinit -i' >> /root/.zshrc

# =====================
# Configuration des tests
# =====================
ENV TEST_SHELL=bash
ENV ZDOTDIR=/root
WORKDIR /app

# Le code et le binaire seront montés via volume
# On crée un lien symbolique vers le binaire compilé
RUN ln -s /app/bin/dirvana /usr/local/bin/dirvana

CMD ["sh", "-c", "cd tests/completion && go test -v ./..."]
