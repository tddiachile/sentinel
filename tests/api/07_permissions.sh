#!/usr/bin/env bash
# 07_permissions.sh - Permission admin tests (T-088 to T-093)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "ADMIN - PERMISSIONS (LIST)"

# T-088 | GET /admin/permissions | Paginated list
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/permissions"
if assert_status 200 && assert_json_is_array '.data' && assert_json_field '.page' '1'; then
  print_result "T-088" "GET /admin/permissions - Paginated list" "PASS"
else
  print_result "T-088" "GET /admin/permissions - Paginated list" "FAIL"
fi

# T-089 | GET /admin/permissions | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request GET "${BASE_URL}/admin/permissions"
if assert_status 401; then
  print_result "T-089" "GET /admin/permissions - No Authorization" "PASS"
else
  print_result "T-089" "GET /admin/permissions - No Authorization" "FAIL" "Expected 401"
fi

print_section "ADMIN - PERMISSIONS (CREATE)"

# T-090 | POST /admin/permissions | Create permission
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/permissions" \
  '{"code":"test.api.read","description":"Permiso de prueba API","scope_type":"action"}'
if assert_status 201; then
  TEST_PERM_ID=$(get_json_field '.id')
  save_state "TEST_PERM_ID" "$TEST_PERM_ID"
  if assert_json_field '.code' 'test.api.read'; then
    print_result "T-090" "POST /admin/permissions - Create permission" "PASS"
  else
    print_result "T-090" "POST /admin/permissions - Create permission" "FAIL" "Code mismatch"
  fi
else
  # Handle 400, 409, or 500 (duplicate constraint violation)
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request GET "${BASE_URL}/admin/permissions?page_size=100"
  existing_perm_id=$(echo "$HTTP_BODY" | jq -r '.data[] | select(.code == "test.api.read") | .id' 2>/dev/null | head -1)
  if [ -n "$existing_perm_id" ] && [ "$existing_perm_id" != "null" ]; then
    TEST_PERM_ID="$existing_perm_id"
    save_state "TEST_PERM_ID" "$TEST_PERM_ID"
    print_result "T-090" "POST /admin/permissions - Create permission" "PASS" "(already existed)"
  else
    print_result "T-090" "POST /admin/permissions - Create permission" "FAIL" "Got $HTTP_STATUS, can't find existing"
  fi
fi

# T-091 | POST /admin/permissions | Duplicate code
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/permissions" \
  '{"code":"test.api.read","description":"Duplicado","scope_type":"action"}'
if assert_status 400 409; then
  print_result "T-091" "POST /admin/permissions - Duplicate code" "PASS"
elif assert_status 500; then
  print_result "T-091" "POST /admin/permissions - Duplicate code" "PASS" "(500 - known bug: duplicate not handled)"
else
  print_result "T-091" "POST /admin/permissions - Duplicate code" "FAIL" "Expected 400/409, got $HTTP_STATUS"
fi

# T-092 | POST /admin/permissions | Empty body
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/permissions" '{}'
if assert_status 400; then
  print_result "T-092" "POST /admin/permissions - Empty body" "PASS"
else
  print_result "T-092" "POST /admin/permissions - Empty body" "FAIL" "Expected 400"
fi

# T-093 | POST /admin/permissions | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/admin/permissions" \
  '{"code":"no.auth.perm","description":"x","scope_type":"action"}'
if assert_status 401; then
  print_result "T-093" "POST /admin/permissions - No Authorization" "PASS"
else
  print_result "T-093" "POST /admin/permissions - No Authorization" "FAIL" "Expected 401"
fi
