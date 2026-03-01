#!/usr/bin/env bash
# 01_system.sh - System endpoint tests (T-001 to T-003)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "SYSTEM ENDPOINTS"

# T-001 | GET /health | Service healthy
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request GET "${BASE_URL}/health"
if assert_status 200 && assert_json_field '.status' 'healthy'; then
  print_result "T-001" "GET /health - Service healthy" "PASS"
else
  print_result "T-001" "GET /health - Service healthy" "FAIL" "Expected 200 + healthy"
fi

# T-002 | GET /.well-known/jwks.json | JWKS keys
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request GET "${BASE_URL}/.well-known/jwks.json"
if assert_status 200 && assert_json_is_array '.keys'; then
  local_kty=$(get_json_field '.keys[0].kty')
  local_alg=$(get_json_field '.keys[0].alg')
  if [ "$local_kty" = "RSA" ] && [ "$local_alg" = "RS256" ]; then
    print_result "T-002" "GET /.well-known/jwks.json - JWKS keys" "PASS"
  else
    print_result "T-002" "GET /.well-known/jwks.json - JWKS keys" "FAIL" "kty=$local_kty alg=$local_alg"
  fi
else
  print_result "T-002" "GET /.well-known/jwks.json - JWKS keys" "FAIL" "Expected 200 + keys array"
fi

# T-003 | GET /.well-known/jwks.json | Public (no X-App-Key)
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request GET "${BASE_URL}/.well-known/jwks.json"
if assert_status 200; then
  print_result "T-003" "GET /.well-known/jwks.json - No X-App-Key needed" "PASS"
else
  print_result "T-003" "GET /.well-known/jwks.json - No X-App-Key needed" "FAIL"
fi
