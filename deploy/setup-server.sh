#!/usr/bin/env bash
# Setup Hetzner CX43 (or any Ubuntu 22.04/24.04) for B2B stack: Docker, Docker Compose, UFW.
# Run with: sudo bash deploy/setup-server.sh

set -e

echo "=== Updating system ==="
apt-get update -qq && apt-get upgrade -y -qq

echo "=== Installing Docker ==="
if ! command -v docker &>/dev/null; then
  curl -fsSL https://get.docker.com | sh
  usermod -aG docker "${SUDO_USER:-$USER}" 2>/dev/null || true
else
  echo "Docker already installed."
fi

echo "=== Installing Docker Compose plugin ==="
apt-get install -y -qq docker-compose-plugin 2>/dev/null || {
  echo "If docker-compose-plugin is not available, install standalone: apt-get install -y docker-compose"
}

echo "=== Configuring firewall (UFW) ==="
ufw allow 22/tcp   # SSH
ufw allow 80/tcp   # HTTP (redirect to HTTPS or serve)
ufw allow 443/tcp  # HTTPS
# Uncomment to expose API directly without reverse proxy:
# ufw allow 8081/tcp
# ufw allow 3000/tcp
# ufw allow 5000/tcp
ufw --force enable || true
ufw status

echo "=== Done. Log out and back in (or run 'newgrp docker') so docker runs without sudo. ==="
