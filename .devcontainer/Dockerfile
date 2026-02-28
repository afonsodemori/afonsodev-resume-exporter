FROM mcr.microsoft.com/devcontainers/base:trixie

RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  ca-certificates gnupg lsb-release \
  vim tig curl \
  && rm -rf /var/lib/apt/lists/*
