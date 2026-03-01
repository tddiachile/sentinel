#!/usr/bin/env bash
# config.sh - Configuration for Sentinel API tests
# All sensitive values are read from environment variables with fallback defaults.

export BASE_URL="${SENTINEL_BASE_URL:-http://localhost:8080}"
export ADMIN_USER="${SENTINEL_ADMIN_USER:-admin}"
export ADMIN_PASS="${SENTINEL_ADMIN_PASS:-Admin@Local1!}"
export ADMIN_PASS_ALT="${SENTINEL_ADMIN_PASS_ALT:-Admin@Sentinel2!}"
export TEST_PASS="TestP@ssw0rd1!"
# Generate a unique NEW_PASS to avoid password history conflicts
_TIMESTAMP=$(date +%s | tail -c 5)
export NEW_PASS="NewP@ss${_TIMESTAMP}w0rd!"

# Shared state file for inter-script communication
export STATE_FILE="/tmp/sentinel_api_test_state.env"

# Initialize state file
init_state() {
  rm -f "$STATE_FILE"
  touch "$STATE_FILE"
}

# Save a variable to state
save_state() {
  local key="$1"
  local value="$2"
  # Remove existing key if present, then append
  if [ -f "$STATE_FILE" ]; then
    grep -v "^${key}=" "$STATE_FILE" > "${STATE_FILE}.tmp" 2>/dev/null || true
    mv "${STATE_FILE}.tmp" "$STATE_FILE"
  fi
  echo "${key}=${value}" >> "$STATE_FILE"
}

# Load state file
load_state() {
  if [ -f "$STATE_FILE" ]; then
    # shellcheck disable=SC1090
    source "$STATE_FILE"
  fi
}

# Truncate sensitive values for logging
truncate_token() {
  local token="$1"
  if [ ${#token} -gt 20 ]; then
    echo "${token:0:20}..."
  else
    echo "$token"
  fi
}
