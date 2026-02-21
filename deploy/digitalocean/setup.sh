#!/usr/bin/env bash
# =============================================================================
# setup.sh — DigitalOcean Droplet initial setup for Sentinel
#
# Run once as root on a fresh Ubuntu 22.04 LTS Droplet:
#   curl -fsSL https://raw.githubusercontent.com/.../setup.sh | bash
# Or copy the script and run: bash setup.sh
#
# What this script does:
#   1. Installs Docker CE + Docker Compose plugin
#   2. Creates the application directory structure
#   3. Generates RSA key pair (4096 bits) for JWT RS256
#   4. Sets up a systemd service for auto-start
# =============================================================================
set -euo pipefail

APP_DIR="/opt/sentinel"
REPO_URL="${SENTINEL_REPO_URL:-}"  # Set before running: export SENTINEL_REPO_URL=...

echo "======================================================"
echo "  Sentinel — Droplet Setup"
echo "======================================================"

# ---- 1. System packages ----
echo "[1/5] Updating system and installing dependencies..."
apt-get update -qq
apt-get install -y -qq \
  curl \
  git \
  openssl \
  ca-certificates \
  gnupg \
  lsb-release \
  ufw

# ---- 2. Docker CE ----
echo "[2/5] Installing Docker CE..."
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

# ---- 3. Application directory ----
echo "[3/5] Setting up application directory at ${APP_DIR}..."
mkdir -p "${APP_DIR}/deploy/digitalocean/files/keys"

if [[ -n "${REPO_URL}" ]]; then
  if [[ -d "${APP_DIR}/.git" ]]; then
    echo "  Repository already cloned. Pulling latest..."
    git -C "${APP_DIR}" pull
  else
    echo "  Cloning repository..."
    git clone "${REPO_URL}" "${APP_DIR}"
  fi
else
  echo "  SENTINEL_REPO_URL not set — skipping git clone."
  echo "  Manually copy the repository to ${APP_DIR} before deploying."
fi

# ---- 4. RSA key pair ----
KEYS_DIR="${APP_DIR}/deploy/digitalocean/files/keys"
if [[ -f "${KEYS_DIR}/private.pem" ]]; then
  echo "[4/5] RSA keys already exist — skipping generation."
else
  echo "[4/5] Generating RSA-4096 key pair for JWT RS256..."
  openssl genrsa -out "${KEYS_DIR}/private.pem" 4096 2>/dev/null
  openssl rsa -in "${KEYS_DIR}/private.pem" \
              -pubout -out "${KEYS_DIR}/public.pem" 2>/dev/null
  chmod 600 "${KEYS_DIR}/private.pem"
  chmod 644 "${KEYS_DIR}/public.pem"
  echo "  Keys generated at: ${KEYS_DIR}/"
fi

# ---- 5. Firewall ----
echo "[5/5] Configuring UFW firewall..."
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
echo "Next steps:"
echo "  1. Copy .env.example to ${APP_DIR}/deploy/digitalocean/.env"
echo "     and fill in all required values:"
echo "     cp ${APP_DIR}/deploy/digitalocean/.env.example \\"
echo "        ${APP_DIR}/deploy/digitalocean/.env"
echo "     nano ${APP_DIR}/deploy/digitalocean/.env"
echo ""
echo "  2. Deploy the stack:"
echo "     cd ${APP_DIR}"
echo "     docker compose -f deploy/digitalocean/docker-compose.yml \\"
echo "       --env-file deploy/digitalocean/.env up -d --build"
echo ""
echo "  3. Wait ~30s for Let's Encrypt certificate, then verify:"
echo "     curl https://\${DOMAIN}/health"
echo ""
echo "  4. Get the bootstrap secret_key from logs:"
echo "     docker compose -f deploy/digitalocean/docker-compose.yml \\"
echo "       logs auth-service | grep secret_key"
echo ""
echo "  RSA keys location: ${KEYS_DIR}/"
echo "  Keep a secure backup of private.pem!"
