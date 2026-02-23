#!/usr/bin/env bash
# =============================================================================
# setup.sh — DigitalOcean Droplet initial setup for Sentinel
#
# Run once as root on a fresh Ubuntu 22.04 LTS Droplet:
#   export DO_REGISTRY_TOKEN=<your_do_personal_access_token>
#   bash setup.sh
#
# Required environment variables:
#   DO_REGISTRY_TOKEN  — DigitalOcean personal access token with registry
#                        read access (Settings → API → Generate New Token)
#
# What this script does:
#   1. Installs Docker CE + Docker Compose plugin
#   2. Logs in to DigitalOcean Container Registry (DOCR)
#   3. Creates the application directory and generates RSA key pair (4096 bits)
#   4. Configures UFW firewall
# =============================================================================
set -euo pipefail

APP_DIR="/opt/sentinel"
DO_REGISTRY_TOKEN="${DO_REGISTRY_TOKEN:-}"

echo "======================================================"
echo "  Sentinel — Droplet Setup"
echo "======================================================"

# ---- Validate required variables ----
if [[ -z "${DO_REGISTRY_TOKEN}" ]]; then
  echo "ERROR: DO_REGISTRY_TOKEN is not set."
  echo "  Set it before running: export DO_REGISTRY_TOKEN=<your_do_token>"
  exit 1
fi

# ---- 1. System packages ----
echo "[1/4] Updating system and installing dependencies..."
apt-get update -qq
apt-get install -y -qq \
  curl \
  openssl \
  ca-certificates \
  gnupg \
  lsb-release \
  ufw

# ---- 2. Docker CE + DOCR login ----
echo "[2/4] Installing Docker CE..."
if ! command -v docker &>/dev/null; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" \
    | tee /etc/apt/sources.list.d/docker.list > /dev/null

  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
  systemctl enable docker
  systemctl start docker
  echo "  Docker installed: $(docker --version)"
else
  echo "  Docker already installed: $(docker --version)"
fi

echo "  Logging in to registry.digitalocean.com..."
echo "${DO_REGISTRY_TOKEN}" | docker login registry.digitalocean.com \
  --username token \
  --password-stdin
echo "  DOCR login successful."

# ---- 3. Application directory and RSA key pair ----
echo "[3/4] Setting up application directory at ${APP_DIR}..."
mkdir -p "${APP_DIR}/files/keys"

KEYS_DIR="${APP_DIR}/files/keys"
if [[ -f "${KEYS_DIR}/private.pem" ]]; then
  echo "  RSA keys already exist — skipping generation."
else
  echo "  Generating RSA-4096 key pair for JWT RS256..."
  openssl genrsa -out "${KEYS_DIR}/private.pem" 4096 2>/dev/null
  openssl rsa -in "${KEYS_DIR}/private.pem" \
              -pubout -out "${KEYS_DIR}/public.pem" 2>/dev/null
  chmod 600 "${KEYS_DIR}/private.pem"
  chmod 644 "${KEYS_DIR}/public.pem"
  echo "  Keys generated at: ${KEYS_DIR}/"
fi

# ---- 4. Firewall ----
echo "[4/4] Configuring UFW firewall..."
ufw --force reset > /dev/null
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https
ufw --force enable
echo "  UFW enabled: SSH(22), HTTP(80), HTTPS(443) open."

# ---- Summary ----
echo ""
echo "======================================================"
echo "  Setup complete!"
echo "======================================================"
echo ""
echo "Next steps (from your local machine):"
echo ""
echo "  1. Build and push images to DOCR:"
echo "     export REGISTRY=registry.digitalocean.com/TU_ORG"
echo "     export IMAGE_TAG=v1.0"
echo ""
echo "     docker build -t \${REGISTRY}/sentinel-auth:\${IMAGE_TAG} ."
echo "     docker push \${REGISTRY}/sentinel-auth:\${IMAGE_TAG}"
echo ""
echo "     docker build -f deploy/digitalocean/Dockerfile.frontend \\"
echo "       --build-arg VITE_APP_KEY=OBTENER_DEL_PRIMER_DEPLOY \\"
echo "       -t \${REGISTRY}/sentinel-frontend:\${IMAGE_TAG} ."
echo "     docker push \${REGISTRY}/sentinel-frontend:\${IMAGE_TAG}"
echo ""
echo "  2. Copy configuration files to the Droplet:"
echo "     scp deploy/digitalocean/docker-compose.yml \\"
echo "         deploy/digitalocean/.env \\"
echo "         root@<DROPLET_IP>:${APP_DIR}/"
echo ""
echo "  3. Pull images and start the stack (on the Droplet):"
echo "     ssh root@<DROPLET_IP>"
echo "     cd ${APP_DIR} && docker compose pull && docker compose up -d"
echo ""
echo "  4. Get the bootstrap secret_key from logs:"
echo "     docker compose logs auth-service | grep secret_key"
echo ""
echo "  RSA keys location: ${KEYS_DIR}/"
echo "  Keep a secure backup of private.pem!"
