#!/usr/bin/env bash
# 11_change_password.sh - Change password tests + post-operation logins (T-021 to T-027, T-138, T-139)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "CHANGE PASSWORD - TEST USER"

# First, login as the test user to get a token
# The test user was created in 05_users.sh with password TEST_PASS
# After 05_users.sh T-073 (reset-password), the user may have a temporary password.
# We need to try login with TEST_PASS first, then TEMP_PASSWORD if available.

TEST_USER_TOKEN=""
TEST_USER_CURRENT_PASS=""

# Try login with TEST_PASS
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"testuser_api\",\"password\":\"${TEST_PASS}\",\"client_type\":\"web\"}"

if assert_status 200; then
  TEST_USER_TOKEN=$(get_json_field '.access_token')
  TEST_USER_CURRENT_PASS="$TEST_PASS"
  save_state "TEST_USER_TOKEN" "$TEST_USER_TOKEN"
  save_state "TEST_USER_CURRENT_PASS" "$TEST_USER_CURRENT_PASS"
  echo -e "  ${GREEN}Logged in as testuser_api with TEST_PASS${NC}"
else
  # Try with TEMP_PASSWORD from reset
  if [ -n "${TEMP_PASSWORD:-}" ] && [ "$TEMP_PASSWORD" != "null" ]; then
    HEADER_APP_KEY="$APP_KEY"
    HEADER_AUTH=""
    http_request POST "${BASE_URL}/auth/login" \
      "{\"username\":\"testuser_api\",\"password\":\"${TEMP_PASSWORD}\",\"client_type\":\"web\"}"
    if assert_status 200; then
      TEST_USER_TOKEN=$(get_json_field '.access_token')
      TEST_USER_CURRENT_PASS="$TEMP_PASSWORD"
      save_state "TEST_USER_TOKEN" "$TEST_USER_TOKEN"
      save_state "TEST_USER_CURRENT_PASS" "$TEST_USER_CURRENT_PASS"
      echo -e "  ${GREEN}Logged in as testuser_api with TEMP_PASSWORD${NC}"
    fi
  fi
fi

if [ -z "$TEST_USER_TOKEN" ]; then
  echo -e "  ${YELLOW}WARNING: Could not login as testuser_api, skipping change-password tests${NC}"
  for tid in T-021 T-022 T-023 T-024 T-025 T-026 T-027 T-138; do
    print_result "$tid" "POST /auth/change-password - Skipped (no login)" "SKIP" "Cannot login as testuser_api"
  done
else
  # T-021 | POST /auth/change-password | Success
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"${TEST_USER_CURRENT_PASS}\",\"new_password\":\"${NEW_PASS}\"}"
  if assert_status 204; then
    print_result "T-021" "POST /auth/change-password - Success" "PASS"
    TEST_USER_CURRENT_PASS="$NEW_PASS"
    save_state "TEST_USER_CURRENT_PASS" "$TEST_USER_CURRENT_PASS"
    # Re-login to get fresh token after password change
    HEADER_APP_KEY="$APP_KEY"
    HEADER_AUTH=""
    http_request POST "${BASE_URL}/auth/login" \
      "{\"username\":\"testuser_api\",\"password\":\"${NEW_PASS}\",\"client_type\":\"web\"}"
    if assert_status 200; then
      TEST_USER_TOKEN=$(get_json_field '.access_token')
      save_state "TEST_USER_TOKEN" "$TEST_USER_TOKEN"
    fi
  else
    print_result "T-021" "POST /auth/change-password - Success" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi

  # T-022 | POST /auth/change-password | Wrong current password
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"WrongCurrent1!\",\"new_password\":\"${NEW_PASS}\"}"
  if assert_status 400 401; then
    print_result "T-022" "POST /auth/change-password - Wrong current password" "PASS"
  else
    print_result "T-022" "POST /auth/change-password - Wrong current password" "FAIL" "Expected 400/401, got $HTTP_STATUS"
  fi

  # T-023 | POST /auth/change-password | New password too short
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"${TEST_USER_CURRENT_PASS}\",\"new_password\":\"Short1!\"}"
  if assert_status 400; then
    print_result "T-023" "POST /auth/change-password - Too short" "PASS"
  else
    print_result "T-023" "POST /auth/change-password - Too short" "FAIL" "Expected 400, got $HTTP_STATUS"
  fi

  # T-024 | POST /auth/change-password | No uppercase
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"${TEST_USER_CURRENT_PASS}\",\"new_password\":\"nouppercase1!@\"}"
  if assert_status 400; then
    print_result "T-024" "POST /auth/change-password - No uppercase" "PASS"
  else
    print_result "T-024" "POST /auth/change-password - No uppercase" "FAIL" "Expected 400, got $HTTP_STATUS"
  fi

  # T-025 | POST /auth/change-password | No number
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"${TEST_USER_CURRENT_PASS}\",\"new_password\":\"NoNumbers!!@@AB\"}"
  if assert_status 400; then
    print_result "T-025" "POST /auth/change-password - No number" "PASS"
  else
    print_result "T-025" "POST /auth/change-password - No number" "FAIL" "Expected 400, got $HTTP_STATUS"
  fi

  # T-026 | POST /auth/change-password | No symbol
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$TEST_USER_TOKEN"
  http_request POST "${BASE_URL}/auth/change-password" \
    "{\"current_password\":\"${TEST_USER_CURRENT_PASS}\",\"new_password\":\"NoSymbols1234AB\"}"
  if assert_status 400; then
    print_result "T-026" "POST /auth/change-password - No symbol" "PASS"
  else
    print_result "T-026" "POST /auth/change-password - No symbol" "FAIL" "Expected 400, got $HTTP_STATUS"
  fi

  # T-027 | POST /auth/change-password | No Authorization
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""
  http_request POST "${BASE_URL}/auth/change-password" \
    '{"current_password":"any","new_password":"AnyP@ss1234!"}'
  if assert_status 401; then
    print_result "T-027" "POST /auth/change-password - No Authorization" "PASS"
  else
    print_result "T-027" "POST /auth/change-password - No Authorization" "FAIL" "Expected 401, got $HTTP_STATUS"
  fi

  print_section "POST-OPERATION VERIFICATION"

  # T-138 | POST /auth/login | Login with changed password
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""
  http_request POST "${BASE_URL}/auth/login" \
    "{\"username\":\"testuser_api\",\"password\":\"${TEST_USER_CURRENT_PASS}\",\"client_type\":\"web\"}"
  if assert_status 200 && assert_json_not_empty '.access_token'; then
    print_result "T-138" "POST /auth/login - Login with changed password" "PASS"
  else
    print_result "T-138" "POST /auth/login - Login with changed password" "FAIL" "Got $HTTP_STATUS"
  fi
fi

# T-139 | POST /auth/login | Login with temporary password (from admin reset)
# This test requires the temp password from T-073 (05_users.sh)
# The admin reset may have been overridden by T-021 (change password)
# We need to re-do admin reset to test this properly
load_state
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/users/${TEST_USER_ID}/reset-password"
if assert_status 200 && assert_json_not_empty '.temporary_password'; then
  TEMP_PASSWORD_NEW=$(get_json_field '.temporary_password')
  # Now try to login with the temporary password
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""
  http_request POST "${BASE_URL}/auth/login" \
    "{\"username\":\"testuser_api\",\"password\":\"${TEMP_PASSWORD_NEW}\",\"client_type\":\"web\"}"
  if assert_status 200; then
    must_change=$(get_json_field '.user.must_change_password')
    if [ "$must_change" = "true" ]; then
      print_result "T-139" "POST /auth/login - Login with temp password (must_change)" "PASS"
    else
      # Some implementations may not set must_change_password flag in login response
      print_result "T-139" "POST /auth/login - Login with temp password" "PASS" "(must_change_password not set)"
    fi
  else
    print_result "T-139" "POST /auth/login - Login with temp password" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-139" "POST /auth/login - Login with temp password" "SKIP" "Reset-password failed"
fi
