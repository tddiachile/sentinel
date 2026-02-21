#!/bin/sh
# =============================================================================
# entrypoint-appplatform.sh
#
# Wrapper entrypoint for App Platform deployments.
# Decodes base64 RSA keys from environment variables and writes them to disk,
# then starts the auth-service.
#
# Required environment variables:
#   RSA_PRIVATE_KEY_B64  — base64-encoded private.pem
#   RSA_PUBLIC_KEY_B64   — base64-encoded public.pem
#   JWT_PRIVATE_KEY_PATH — path where to write private key (e.g. /tmp/keys/private.pem)
#   JWT_PUBLIC_KEY_PATH  — path where to write public key (e.g. /tmp/keys/public.pem)
#
# To encode your PEM files:
#   base64 -w 0 keys/private.pem   # Linux
#   base64 -i keys/private.pem     # macOS
# =============================================================================
set -e

echo "[entrypoint] Decoding RSA keys from environment variables..."

PRIVATE_KEY_DIR=$(dirname "${JWT_PRIVATE_KEY_PATH:-/tmp/keys/private.pem}")
mkdir -p "${PRIVATE_KEY_DIR}"

if [ -z "${RSA_PRIVATE_KEY_B64:-}" ]; then
  echo "[entrypoint] ERROR: RSA_PRIVATE_KEY_B64 is not set" >&2
  exit 1
fi

if [ -z "${RSA_PUBLIC_KEY_B64:-}" ]; then
  echo "[entrypoint] ERROR: RSA_PUBLIC_KEY_B64 is not set" >&2
  exit 1
fi

printf '%s' "${RSA_PRIVATE_KEY_B64}" | base64 -d > "${JWT_PRIVATE_KEY_PATH:-/tmp/keys/private.pem}"
printf '%s' "${RSA_PUBLIC_KEY_B64}"  | base64 -d > "${JWT_PUBLIC_KEY_PATH:-/tmp/keys/public.pem}"

chmod 600 "${JWT_PRIVATE_KEY_PATH:-/tmp/keys/private.pem}"

echo "[entrypoint] RSA keys written. Starting auth-service..."
exec ./auth-service
