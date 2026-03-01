#!/usr/bin/env bash
# common.sh - Shared test functions for Sentinel API tests

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# ---- HTTP helpers ----

# Make an HTTP request and capture status code + body
# Usage: http_request METHOD URL [BODY] [EXTRA_CURL_ARGS...]
# Sets global vars: HTTP_STATUS, HTTP_BODY
http_request() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  shift 2
  if [ $# -gt 0 ]; then shift; fi

  local curl_args=(
    -s
    -w "\n%{http_code}"
    -X "$method"
    "$url"
  )

  # Add headers from global arrays
  if [ -n "$HEADER_APP_KEY" ]; then
    curl_args+=(-H "X-App-Key: $HEADER_APP_KEY")
  fi
  if [ -n "$HEADER_AUTH" ]; then
    curl_args+=(-H "Authorization: Bearer $HEADER_AUTH")
  fi

  if [ -n "$body" ] && [ "$body" != "N/A" ] && [ "$body" != "" ]; then
    curl_args+=(-H "Content-Type: application/json" -d "$body")
  fi

  # Add any extra args
  for arg in "$@"; do
    curl_args+=("$arg")
  done

  local response
  response=$(curl "${curl_args[@]}" 2>/dev/null)

  HTTP_STATUS=$(echo "$response" | tail -1)
  HTTP_BODY=$(echo "$response" | sed '$d')
}

# ---- Assertion helpers ----

# Assert HTTP status code
# Usage: assert_status EXPECTED_STATUS [ALTERNATE_STATUS]
assert_status() {
  local expected="$1"
  local alternate="${2:-}"

  if [ "$HTTP_STATUS" = "$expected" ]; then
    return 0
  elif [ -n "$alternate" ] && [ "$HTTP_STATUS" = "$alternate" ]; then
    return 0
  else
    return 1
  fi
}

# Assert a JSON field has expected value using jq
# Usage: assert_json_field JQ_EXPRESSION EXPECTED_VALUE
assert_json_field() {
  local jq_expr="$1"
  local expected="$2"

  local actual
  actual=$(echo "$HTTP_BODY" | jq -r "$jq_expr" 2>/dev/null)

  if [ "$actual" = "$expected" ]; then
    return 0
  else
    return 1
  fi
}

# Assert a JSON field is not empty/null
# Usage: assert_json_not_empty JQ_EXPRESSION
assert_json_not_empty() {
  local jq_expr="$1"

  local actual
  actual=$(echo "$HTTP_BODY" | jq -r "$jq_expr" 2>/dev/null)

  if [ -n "$actual" ] && [ "$actual" != "null" ] && [ "$actual" != "" ]; then
    return 0
  else
    return 1
  fi
}

# Assert a JSON field is an array
# Usage: assert_json_is_array JQ_EXPRESSION
assert_json_is_array() {
  local jq_expr="$1"

  local type
  type=$(echo "$HTTP_BODY" | jq -r "$jq_expr | type" 2>/dev/null)

  if [ "$type" = "array" ]; then
    return 0
  else
    return 1
  fi
}

# Get a JSON field value
# Usage: get_json_field JQ_EXPRESSION
get_json_field() {
  local jq_expr="$1"
  echo "$HTTP_BODY" | jq -r "$jq_expr" 2>/dev/null
}

# ---- Test reporting ----

# Print test result
# Usage: print_result TEST_ID DESCRIPTION PASS_OR_FAIL [DETAILS]
print_result() {
  local test_id="$1"
  local description="$2"
  local result="$3"
  local details="${4:-}"

  TOTAL_TESTS=$((TOTAL_TESTS + 1))

  if [ "$result" = "PASS" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
    echo -e "  ${GREEN}PASS${NC} | ${test_id} | ${description} | HTTP ${HTTP_STATUS}"
  elif [ "$result" = "SKIP" ]; then
    SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
    echo -e "  ${YELLOW}SKIP${NC} | ${test_id} | ${description} | ${details}"
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "  ${RED}FAIL${NC} | ${test_id} | ${description} | HTTP ${HTTP_STATUS} ${details}"
    if [ -n "$HTTP_BODY" ]; then
      echo -e "       ${RED}Body: $(echo "$HTTP_BODY" | head -c 200)${NC}"
    fi
  fi
}

# Print section header
print_section() {
  echo ""
  echo -e "${CYAN}=== $1 ===${NC}"
}

# Print summary
print_summary() {
  echo ""
  echo -e "${CYAN}========================================${NC}"
  echo -e "${CYAN}  TEST SUMMARY${NC}"
  echo -e "${CYAN}========================================${NC}"
  echo -e "  Total:   ${TOTAL_TESTS}"
  echo -e "  ${GREEN}Passed:  ${PASSED_TESTS}${NC}"
  echo -e "  ${RED}Failed:  ${FAILED_TESTS}${NC}"
  echo -e "  ${YELLOW}Skipped: ${SKIPPED_TESTS}${NC}"
  echo -e "${CYAN}========================================${NC}"
}

# ---- Setup helpers ----

# Login as admin and save tokens to state
do_admin_login() {
  load_state

  # Try primary password first, then alternate
  local pass="$ADMIN_PASS"
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""

  http_request POST "${BASE_URL}/auth/login" \
    "{\"username\":\"${ADMIN_USER}\",\"password\":\"${pass}\",\"client_type\":\"web\"}"

  if [ "$HTTP_STATUS" != "200" ]; then
    pass="$ADMIN_PASS_ALT"
    http_request POST "${BASE_URL}/auth/login" \
      "{\"username\":\"${ADMIN_USER}\",\"password\":\"${pass}\",\"client_type\":\"web\"}"
  fi

  if [ "$HTTP_STATUS" = "200" ]; then
    ADMIN_TOKEN=$(get_json_field '.access_token')
    ADMIN_REFRESH=$(get_json_field '.refresh_token')
    ADMIN_USER_ID=$(get_json_field '.user.id')
    ADMIN_WORKING_PASS="$pass"
    save_state "ADMIN_TOKEN" "$ADMIN_TOKEN"
    save_state "ADMIN_REFRESH" "$ADMIN_REFRESH"
    save_state "ADMIN_USER_ID" "$ADMIN_USER_ID"
    save_state "ADMIN_WORKING_PASS" "$ADMIN_WORKING_PASS"
    return 0
  else
    echo -e "${RED}ERROR: Could not login as admin${NC}"
    return 1
  fi
}

# Obtain APP_KEY by listing applications (system app key)
obtain_app_key() {
  # First try to login without app key to find it
  # The APP_KEY might already be set via environment
  if [ -n "${SENTINEL_APP_KEY:-}" ]; then
    APP_KEY="$SENTINEL_APP_KEY"
    save_state "APP_KEY" "$APP_KEY"
    return 0
  fi

  # Try to read from the deploy/local/.env file
  local env_file="/home/enunez/proyectos/github.com/enunezf/sentinel/deploy/local/.env"
  if [ -f "$env_file" ]; then
    local key
    key=$(grep -E "^(BOOTSTRAP_APP_KEY|APP_KEY)" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2- | tr -d '"' | tr -d "'")
    if [ -n "$key" ]; then
      APP_KEY="$key"
      save_state "APP_KEY" "$APP_KEY"
      return 0
    fi
  fi

  # Try to query the database via Docker container
  if command -v docker &> /dev/null; then
    local key
    key=$(docker exec local-postgres-1 psql -U sentinel -d sentinel_local -t -c "SELECT secret_key FROM applications WHERE slug='system' LIMIT 1;" 2>/dev/null | tr -d ' \n')
    if [ -n "$key" ] && [ "$key" != "" ]; then
      APP_KEY="$key"
      save_state "APP_KEY" "$APP_KEY"
      return 0
    fi
  fi

  # As a last resort, try to get it from config.yaml
  local config_file="/home/enunez/proyectos/github.com/enunezf/sentinel/config.yaml"
  if [ -f "$config_file" ]; then
    local key
    key=$(grep -E "app_key|secret_key|bootstrap.*key" "$config_file" 2>/dev/null | head -1 | awk '{print $NF}' | tr -d '"' | tr -d "'")
    if [ -n "$key" ]; then
      APP_KEY="$key"
      save_state "APP_KEY" "$APP_KEY"
      return 0
    fi
  fi

  echo -e "${RED}ERROR: Could not determine APP_KEY. Set SENTINEL_APP_KEY env var.${NC}"
  return 1
}

# Cleanup test data created during the run
cleanup_test_data() {
  load_state
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"

  echo -e "${YELLOW}Cleaning up test data...${NC}"

  # Deactivate test user
  if [ -n "${TEST_USER_ID:-}" ]; then
    http_request PUT "${BASE_URL}/admin/users/${TEST_USER_ID}" '{"is_active":false}'
  fi

  # Note: We don't hard-delete resources since most are soft-delete.
  # The test user is deactivated, other resources remain but are identifiable by prefix.
}
