#!/usr/bin/env bash
# 02_auth.sh - Authentication endpoint tests (T-004 to T-027)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "AUTHENTICATION - LOGIN"

# T-004 | POST /auth/login | Login success (web)
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"web\"}"
if assert_status 200 && assert_json_not_empty '.access_token' && assert_json_not_empty '.refresh_token' && assert_json_field '.token_type' 'Bearer'; then
  print_result "T-004" "POST /auth/login - Login success (web)" "PASS"
  # Save tokens for later tests
  LOGIN_ACCESS=$(get_json_field '.access_token')
  LOGIN_REFRESH=$(get_json_field '.refresh_token')
  save_state "LOGIN_ACCESS" "$LOGIN_ACCESS"
  save_state "LOGIN_REFRESH" "$LOGIN_REFRESH"
else
  print_result "T-004" "POST /auth/login - Login success (web)" "FAIL"
fi

# T-005 | POST /auth/login | Login success (mobile)
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"mobile\"}"
if assert_status 200 && assert_json_not_empty '.access_token'; then
  print_result "T-005" "POST /auth/login - Login success (mobile)" "PASS"
  # Save this refresh for rotation test later
  MOBILE_REFRESH=$(get_json_field '.refresh_token')
  save_state "MOBILE_REFRESH" "$MOBILE_REFRESH"
else
  print_result "T-005" "POST /auth/login - Login success (mobile)" "FAIL"
fi

# T-006 | POST /auth/login | Login success (desktop)
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"desktop\"}"
if assert_status 200 && assert_json_not_empty '.access_token'; then
  print_result "T-006" "POST /auth/login - Login success (desktop)" "PASS"
else
  print_result "T-006" "POST /auth/login - Login success (desktop)" "FAIL"
fi

# T-007 | POST /auth/login | No X-App-Key
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"web\"}"
if assert_status 401; then
  print_result "T-007" "POST /auth/login - No X-App-Key" "PASS"
else
  print_result "T-007" "POST /auth/login - No X-App-Key" "FAIL" "Expected 401"
fi

# T-008 | POST /auth/login | Invalid X-App-Key
HEADER_APP_KEY="invalid-key-12345"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"web\"}"
if assert_status 401; then
  print_result "T-008" "POST /auth/login - Invalid X-App-Key" "PASS"
else
  print_result "T-008" "POST /auth/login - Invalid X-App-Key" "FAIL" "Expected 401"
fi

# T-009 | POST /auth/login | Wrong username
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  '{"username":"nonexistent_user_xyz","password":"AnyP@ss123!","client_type":"web"}'
if assert_status 401; then
  print_result "T-009" "POST /auth/login - Wrong username" "PASS"
else
  print_result "T-009" "POST /auth/login - Wrong username" "FAIL" "Expected 401"
fi

# T-010 | POST /auth/login | Wrong password
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"WrongP@ssw0rd!\",\"client_type\":\"web\"}"
if assert_status 401; then
  print_result "T-010" "POST /auth/login - Wrong password" "PASS"
else
  print_result "T-010" "POST /auth/login - Wrong password" "FAIL" "Expected 401"
fi

# T-011 | POST /auth/login | Empty body
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" '{}'
if assert_status 400; then
  print_result "T-011" "POST /auth/login - Empty body" "PASS"
else
  print_result "T-011" "POST /auth/login - Empty body" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

# T-012 | POST /auth/login | Invalid client_type
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"tablet\"}"
if assert_status 400; then
  print_result "T-012" "POST /auth/login - Invalid client_type" "PASS"
else
  print_result "T-012" "POST /auth/login - Invalid client_type" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

# T-013 | POST /auth/login | Missing client_type
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\"}"
if assert_status 400; then
  print_result "T-013" "POST /auth/login - Missing client_type" "PASS"
else
  print_result "T-013" "POST /auth/login - Missing client_type" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

print_section "AUTHENTICATION - REFRESH"

# T-014 | POST /auth/refresh | Success
# First get a fresh login to have a valid refresh token
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"web\"}"
REFRESH_FOR_TEST=$(get_json_field '.refresh_token')
OLD_REFRESH_FOR_ROTATION="$REFRESH_FOR_TEST"
save_state "OLD_REFRESH_FOR_ROTATION" "$OLD_REFRESH_FOR_ROTATION"

HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" \
  "{\"refresh_token\":\"${REFRESH_FOR_TEST}\"}"
if assert_status 200 && assert_json_not_empty '.access_token' && assert_json_not_empty '.refresh_token'; then
  NEW_REFRESH=$(get_json_field '.refresh_token')
  if [ "$NEW_REFRESH" != "$REFRESH_FOR_TEST" ]; then
    print_result "T-014" "POST /auth/refresh - Success (rotation)" "PASS"
  else
    print_result "T-014" "POST /auth/refresh - Success (rotation)" "FAIL" "New token same as old"
  fi
else
  print_result "T-014" "POST /auth/refresh - Success (rotation)" "FAIL"
fi

# T-015 | POST /auth/refresh | Invalid token
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" \
  '{"refresh_token":"totally-invalid-refresh-token-12345"}'
if assert_status 401; then
  print_result "T-015" "POST /auth/refresh - Invalid token" "PASS"
else
  print_result "T-015" "POST /auth/refresh - Invalid token" "FAIL" "Expected 401"
fi

# T-016 | POST /auth/refresh | Empty body
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" '{}'
if assert_status 400 401; then
  print_result "T-016" "POST /auth/refresh - Empty body" "PASS"
else
  print_result "T-016" "POST /auth/refresh - Empty body" "FAIL" "Expected 400 or 401"
fi

# T-017 | POST /auth/refresh | No X-App-Key
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" \
  '{"refresh_token":"any-token"}'
if assert_status 401; then
  print_result "T-017" "POST /auth/refresh - No X-App-Key" "PASS"
else
  print_result "T-017" "POST /auth/refresh - No X-App-Key" "FAIL" "Expected 401"
fi

print_section "AUTHENTICATION - LOGOUT"

# Get a fresh session for logout tests
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/login" \
  "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_WORKING_PASS}\",\"client_type\":\"web\"}"
LOGOUT_TOKEN=$(get_json_field '.access_token')
LOGOUT_REFRESH=$(get_json_field '.refresh_token')
save_state "LOGOUT_REFRESH" "$LOGOUT_REFRESH"

# T-018 | POST /auth/logout | Success
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$LOGOUT_TOKEN"
http_request POST "${BASE_URL}/auth/logout"
if assert_status 204; then
  print_result "T-018" "POST /auth/logout - Success" "PASS"
else
  print_result "T-018" "POST /auth/logout - Success" "FAIL" "Expected 204"
fi

# T-019 | POST /auth/logout | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/logout"
if assert_status 401; then
  print_result "T-019" "POST /auth/logout - No Authorization" "PASS"
else
  print_result "T-019" "POST /auth/logout - No Authorization" "FAIL" "Expected 401"
fi

# T-020 | POST /auth/logout | Invalid token
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="invalid.jwt.token"
http_request POST "${BASE_URL}/auth/logout"
if assert_status 401; then
  print_result "T-020" "POST /auth/logout - Invalid token" "PASS"
else
  print_result "T-020" "POST /auth/logout - Invalid token" "FAIL" "Expected 401"
fi

print_section "AUTHENTICATION - TOKEN ROTATION & POST-LOGOUT"

# T-136 | POST /auth/refresh | Already used token (rotation)
load_state
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" \
  "{\"refresh_token\":\"${OLD_REFRESH_FOR_ROTATION}\"}"
if assert_status 401; then
  print_result "T-136" "POST /auth/refresh - Already rotated token" "PASS"
else
  print_result "T-136" "POST /auth/refresh - Already rotated token" "FAIL" "Expected 401"
fi

# T-137 | POST /auth/refresh | After logout
load_state
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/auth/refresh" \
  "{\"refresh_token\":\"${LOGOUT_REFRESH}\"}"
if assert_status 401; then
  print_result "T-137" "POST /auth/refresh - After logout (revoked)" "PASS"
else
  print_result "T-137" "POST /auth/refresh - After logout (revoked)" "FAIL" "Expected 401"
fi

# Re-login admin for subsequent tests
do_admin_login
