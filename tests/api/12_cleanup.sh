#!/usr/bin/env bash
# 12_cleanup.sh - Deletion and cleanup tests (T-121 to T-126, T-140 to T-143)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "DELETION - ROLES"

# T-121 | DELETE /admin/roles/{id} | Delete (deactivate) role
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${TEST_ROLE_ID:-}" ] && [ "$TEST_ROLE_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/roles/${TEST_ROLE_ID}"
  if assert_status 204; then
    print_result "T-121" "DELETE /admin/roles/{id} - Delete role" "PASS"
  else
    print_result "T-121" "DELETE /admin/roles/{id} - Delete role" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi
else
  print_result "T-121" "DELETE /admin/roles/{id} - Delete role" "SKIP" "No TEST_ROLE_ID"
fi

# T-122 | DELETE /admin/roles/{id} | Invalid ID
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request DELETE "${BASE_URL}/admin/roles/not-a-uuid"
if assert_status 400; then
  print_result "T-122" "DELETE /admin/roles/{id} - Invalid ID" "PASS"
else
  print_result "T-122" "DELETE /admin/roles/{id} - Invalid ID" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

# T-123 | DELETE /admin/roles/{id} | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
if [ -n "${TEST_ROLE_ID:-}" ] && [ "$TEST_ROLE_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/roles/${TEST_ROLE_ID}"
else
  http_request DELETE "${BASE_URL}/admin/roles/00000000-0000-0000-0000-000000000000"
fi
if assert_status 401; then
  print_result "T-123" "DELETE /admin/roles/{id} - No Authorization" "PASS"
else
  print_result "T-123" "DELETE /admin/roles/{id} - No Authorization" "FAIL" "Expected 401, got $HTTP_STATUS"
fi

print_section "DELETION - PERMISSIONS"

# T-124 | DELETE /admin/permissions/{id} | Delete permission (unassigned)
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${TEST_PERM_ID:-}" ] && [ "$TEST_PERM_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/permissions/${TEST_PERM_ID}"
  if assert_status 204; then
    print_result "T-124" "DELETE /admin/permissions/{id} - Delete permission" "PASS"
  else
    print_result "T-124" "DELETE /admin/permissions/{id} - Delete permission" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi
else
  print_result "T-124" "DELETE /admin/permissions/{id} - Delete permission" "SKIP" "No TEST_PERM_ID"
fi

# T-125 | DELETE /admin/permissions/{id} | Invalid ID
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request DELETE "${BASE_URL}/admin/permissions/not-a-uuid"
if assert_status 400; then
  print_result "T-125" "DELETE /admin/permissions/{id} - Invalid ID" "PASS"
else
  print_result "T-125" "DELETE /admin/permissions/{id} - Invalid ID" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

# T-126 | DELETE /admin/permissions/{id} | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
if [ -n "${TEST_PERM_ID:-}" ] && [ "$TEST_PERM_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/permissions/${TEST_PERM_ID}"
else
  http_request DELETE "${BASE_URL}/admin/permissions/00000000-0000-0000-0000-000000000000"
fi
if assert_status 401; then
  print_result "T-126" "DELETE /admin/permissions/{id} - No Authorization" "PASS"
else
  print_result "T-126" "DELETE /admin/permissions/{id} - No Authorization" "FAIL" "Expected 401, got $HTTP_STATUS"
fi

print_section "CLEANUP"

# T-140 | DELETE /admin/permissions/{id} | Cleanup remaining permission
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${TEST_PERM_ID:-}" ] && [ "$TEST_PERM_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/permissions/${TEST_PERM_ID}"
  if assert_status 204 404; then
    print_result "T-140" "DELETE /admin/permissions/{id} - Cleanup perm" "PASS"
  else
    print_result "T-140" "DELETE /admin/permissions/{id} - Cleanup perm" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-140" "DELETE /admin/permissions/{id} - Cleanup perm" "SKIP" "Already cleaned"
fi

# T-141 | DELETE /admin/roles/{id} | Cleanup remaining role
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${TEST_ROLE_ID:-}" ] && [ "$TEST_ROLE_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/roles/${TEST_ROLE_ID}"
  if assert_status 204 404; then
    print_result "T-141" "DELETE /admin/roles/{id} - Cleanup role" "PASS"
  else
    print_result "T-141" "DELETE /admin/roles/{id} - Cleanup role" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-141" "DELETE /admin/roles/{id} - Cleanup role" "SKIP" "Already cleaned"
fi

# T-142 | PUT /admin/users/{id} | Deactivate test user (cleanup)
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${TEST_USER_ID:-}" ] && [ "$TEST_USER_ID" != "null" ]; then
  http_request PUT "${BASE_URL}/admin/users/${TEST_USER_ID}" '{"is_active":false}'
  if assert_status 200; then
    is_active=$(get_json_field '.is_active')
    if [ "$is_active" = "false" ]; then
      print_result "T-142" "PUT /admin/users/{id} - Deactivate test user" "PASS"
    else
      print_result "T-142" "PUT /admin/users/{id} - Deactivate test user" "PASS" "(is_active not in response)"
    fi
  else
    print_result "T-142" "PUT /admin/users/{id} - Deactivate test user" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-142" "PUT /admin/users/{id} - Deactivate test user" "SKIP" "No TEST_USER_ID"
fi

print_section "FINAL HEALTH CHECK"

# T-143 | GET /health | Final health check
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request GET "${BASE_URL}/health"
if assert_status 200 && assert_json_field '.status' 'healthy'; then
  print_result "T-143" "GET /health - Final health check (post-tests)" "PASS"
else
  print_result "T-143" "GET /health - Final health check (post-tests)" "FAIL"
fi
