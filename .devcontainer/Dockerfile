FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Install Rclone and system dependencies for testing & headless browser journeys
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        rclone \
        sqlite3 \
        curl \
        chromium \
        ca-certificates \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*
