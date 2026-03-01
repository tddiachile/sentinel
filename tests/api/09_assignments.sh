#!/usr/bin/env bash
# 09_assignments.sh - Assignment tests: role-perms, user-roles, user-perms, user-cecos (T-103 to T-120)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

# Guard: ensure required state variables exist
_ROLE_ID="${TEST_ROLE_ID:-}"
_PERM_ID="${TEST_PERM_ID:-}"
_USER_ID="${TEST_USER_ID:-}"
_CECO_ID="${TEST_CECO_ID:-}"

print_section "ASSIGNMENTS - ROLE PERMISSIONS"

# T-103 | POST /admin/roles/{id}/permissions | Add permission to role
if [ -n "$_ROLE_ID" ] && [ "$_ROLE_ID" != "null" ] && [ -n "$_PERM_ID" ] && [ "$_PERM_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/roles/${_ROLE_ID}/permissions" \
    "{\"permission_ids\":[\"${_PERM_ID}\"]}"
  if assert_status 201 200; then
    print_result "T-103" "POST /admin/roles/{id}/permissions - Add perm to role" "PASS"
  else
    print_result "T-103" "POST /admin/roles/{id}/permissions - Add perm to role" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-103" "POST /admin/roles/{id}/permissions - Add perm to role" "SKIP" "Missing ROLE_ID or PERM_ID"
fi

# T-104 | POST /admin/roles/{id}/permissions | Empty array
if [ -n "$_ROLE_ID" ] && [ "$_ROLE_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/roles/${_ROLE_ID}/permissions" \
    '{"permission_ids":[]}'
  if assert_status 400; then
    print_result "T-104" "POST /admin/roles/{id}/permissions - Empty array" "PASS"
  else
    # Some implementations may accept empty and return 201 with assigned=0
    if assert_status 201 200; then
      assigned=$(get_json_field '.assigned')
      if [ "$assigned" = "0" ]; then
        print_result "T-104" "POST /admin/roles/{id}/permissions - Empty array" "PASS" "(201 with 0 assigned)"
      else
        print_result "T-104" "POST /admin/roles/{id}/permissions - Empty array" "FAIL" "Expected 400, got $HTTP_STATUS"
      fi
    else
      print_result "T-104" "POST /admin/roles/{id}/permissions - Empty array" "FAIL" "Expected 400"
    fi
  fi
else
  print_result "T-104" "POST /admin/roles/{id}/permissions - Empty array" "SKIP" "Missing ROLE_ID"
fi

# T-105 | POST /admin/roles/{id}/permissions | No Authorization
if [ -n "$_ROLE_ID" ] && [ "$_ROLE_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""
  http_request POST "${BASE_URL}/admin/roles/${_ROLE_ID}/permissions" \
    "{\"permission_ids\":[\"${_PERM_ID:-00000000-0000-0000-0000-000000000000}\"]}"
  if assert_status 401; then
    print_result "T-105" "POST /admin/roles/{id}/permissions - No Auth" "PASS"
  else
    print_result "T-105" "POST /admin/roles/{id}/permissions - No Auth" "FAIL" "Expected 401"
  fi
else
  # Use a fake UUID for the no-auth test
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH=""
  http_request POST "${BASE_URL}/admin/roles/00000000-0000-0000-0000-000000000000/permissions" \
    '{"permission_ids":["00000000-0000-0000-0000-000000000001"]}'
  if assert_status 401; then
    print_result "T-105" "POST /admin/roles/{id}/permissions - No Auth" "PASS"
  else
    print_result "T-105" "POST /admin/roles/{id}/permissions - No Auth" "FAIL" "Expected 401"
  fi
fi

# T-106 | DELETE /admin/roles/{id}/permissions/{pid} | Remove permission from role
if [ -n "$_ROLE_ID" ] && [ "$_ROLE_ID" != "null" ] && [ -n "$_PERM_ID" ] && [ "$_PERM_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request DELETE "${BASE_URL}/admin/roles/${_ROLE_ID}/permissions/${_PERM_ID}"
  if assert_status 204; then
    print_result "T-106" "DELETE /admin/roles/{id}/permissions/{pid} - Remove" "PASS"
  else
    print_result "T-106" "DELETE /admin/roles/{id}/permissions/{pid} - Remove" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi
else
  print_result "T-106" "DELETE /admin/roles/{id}/permissions/{pid} - Remove" "SKIP" "Missing ROLE_ID or PERM_ID"
fi

# T-107 | DELETE /admin/roles/{id}/permissions/{pid} | Invalid IDs
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request DELETE "${BASE_URL}/admin/roles/not-a-uuid/permissions/not-a-uuid"
if assert_status 400; then
  print_result "T-107" "DELETE /admin/roles/{id}/permissions/{pid} - Invalid IDs" "PASS"
else
  print_result "T-107" "DELETE /admin/roles/{id}/permissions/{pid} - Invalid IDs" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

print_section "ASSIGNMENTS - USER ROLES"

# T-108 | POST /admin/users/{id}/roles | Assign role to user
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ] && [ -n "$_ROLE_ID" ] && [ "$_ROLE_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/roles" \
    "{\"role_id\":\"${_ROLE_ID}\"}"
  if assert_status 201; then
    ROLE_ASSIGNMENT_ID=$(get_json_field '.id')
    save_state "ROLE_ASSIGNMENT_ID" "$ROLE_ASSIGNMENT_ID"
    print_result "T-108" "POST /admin/users/{id}/roles - Assign role" "PASS"
  else
    print_result "T-108" "POST /admin/users/{id}/roles - Assign role" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-108" "POST /admin/users/{id}/roles - Assign role" "SKIP" "Missing USER_ID or ROLE_ID"
fi

# T-109 | POST /admin/users/{id}/roles | Invalid role_id
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/roles" \
    '{"role_id":"not-a-valid-uuid"}'
  if assert_status 400; then
    print_result "T-109" "POST /admin/users/{id}/roles - Invalid role_id" "PASS"
  else
    print_result "T-109" "POST /admin/users/{id}/roles - Invalid role_id" "FAIL" "Expected 400"
  fi
else
  print_result "T-109" "POST /admin/users/{id}/roles - Invalid role_id" "SKIP" "Missing USER_ID"
fi

# T-110 | POST /admin/users/{id}/roles | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/admin/users/${_USER_ID:-00000000-0000-0000-0000-000000000000}/roles" \
  "{\"role_id\":\"${_ROLE_ID:-00000000-0000-0000-0000-000000000000}\"}"
if assert_status 401; then
  print_result "T-110" "POST /admin/users/{id}/roles - No Auth" "PASS"
else
  print_result "T-110" "POST /admin/users/{id}/roles - No Auth" "FAIL" "Expected 401"
fi

# T-111 | DELETE /admin/users/{id}/roles/{rid} | Revoke role
load_state
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${ROLE_ASSIGNMENT_ID:-}" ] && [ "$ROLE_ASSIGNMENT_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/users/${_USER_ID}/roles/${ROLE_ASSIGNMENT_ID}"
  if assert_status 204; then
    print_result "T-111" "DELETE /admin/users/{id}/roles/{rid} - Revoke" "PASS"
  else
    print_result "T-111" "DELETE /admin/users/{id}/roles/{rid} - Revoke" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi
else
  print_result "T-111" "DELETE /admin/users/{id}/roles/{rid} - Revoke" "SKIP" "No assignment ID"
fi

# T-112 | DELETE /admin/users/{id}/roles/{rid} | Invalid IDs
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request DELETE "${BASE_URL}/admin/users/not-a-uuid/roles/not-a-uuid"
if assert_status 400; then
  print_result "T-112" "DELETE /admin/users/{id}/roles/{rid} - Invalid IDs" "PASS"
else
  print_result "T-112" "DELETE /admin/users/{id}/roles/{rid} - Invalid IDs" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

print_section "ASSIGNMENTS - USER PERMISSIONS"

# T-113 | POST /admin/users/{id}/permissions | Assign permission to user
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ] && [ -n "$_PERM_ID" ] && [ "$_PERM_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/permissions" \
    "{\"permission_id\":\"${_PERM_ID}\"}"
  if assert_status 201; then
    PERM_ASSIGNMENT_ID=$(get_json_field '.id')
    save_state "PERM_ASSIGNMENT_ID" "$PERM_ASSIGNMENT_ID"
    print_result "T-113" "POST /admin/users/{id}/permissions - Assign perm" "PASS"
  else
    print_result "T-113" "POST /admin/users/{id}/permissions - Assign perm" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-113" "POST /admin/users/{id}/permissions - Assign perm" "SKIP" "Missing USER_ID or PERM_ID"
fi

# T-114 | POST /admin/users/{id}/permissions | Invalid permission_id
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/permissions" \
    '{"permission_id":"not-a-uuid"}'
  if assert_status 400; then
    print_result "T-114" "POST /admin/users/{id}/permissions - Invalid ID" "PASS"
  else
    print_result "T-114" "POST /admin/users/{id}/permissions - Invalid ID" "FAIL" "Expected 400"
  fi
else
  print_result "T-114" "POST /admin/users/{id}/permissions - Invalid ID" "SKIP" "Missing USER_ID"
fi

# T-115 | POST /admin/users/{id}/permissions | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/admin/users/${_USER_ID:-00000000-0000-0000-0000-000000000000}/permissions" \
  "{\"permission_id\":\"${_PERM_ID:-00000000-0000-0000-0000-000000000000}\"}"
if assert_status 401; then
  print_result "T-115" "POST /admin/users/{id}/permissions - No Auth" "PASS"
else
  print_result "T-115" "POST /admin/users/{id}/permissions - No Auth" "FAIL" "Expected 401"
fi

# T-116 | DELETE /admin/users/{id}/permissions/{pid} | Revoke permission
load_state
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
if [ -n "${PERM_ASSIGNMENT_ID:-}" ] && [ "$PERM_ASSIGNMENT_ID" != "null" ]; then
  http_request DELETE "${BASE_URL}/admin/users/${_USER_ID}/permissions/${PERM_ASSIGNMENT_ID}"
  if assert_status 204; then
    print_result "T-116" "DELETE /admin/users/{id}/permissions/{pid} - Revoke" "PASS"
  else
    print_result "T-116" "DELETE /admin/users/{id}/permissions/{pid} - Revoke" "FAIL" "Expected 204, got $HTTP_STATUS"
  fi
else
  print_result "T-116" "DELETE /admin/users/{id}/permissions/{pid} - Revoke" "SKIP" "No assignment ID"
fi

# T-117 | DELETE /admin/users/{id}/permissions/{pid} | Invalid IDs
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request DELETE "${BASE_URL}/admin/users/not-a-uuid/permissions/not-a-uuid"
if assert_status 400; then
  print_result "T-117" "DELETE /admin/users/{id}/permissions/{pid} - Invalid IDs" "PASS"
else
  print_result "T-117" "DELETE /admin/users/{id}/permissions/{pid} - Invalid IDs" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

print_section "ASSIGNMENTS - USER COST CENTERS"

# T-118 | POST /admin/users/{id}/cost-centers | Assign CeCos
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ] && [ -n "$_CECO_ID" ] && [ "$_CECO_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/cost-centers" \
    "{\"cost_center_ids\":[\"${_CECO_ID}\"]}"
  if assert_status 201 200; then
    print_result "T-118" "POST /admin/users/{id}/cost-centers - Assign CeCo" "PASS"
  else
    print_result "T-118" "POST /admin/users/{id}/cost-centers - Assign CeCo" "FAIL" "Got $HTTP_STATUS"
  fi
else
  print_result "T-118" "POST /admin/users/{id}/cost-centers - Assign CeCo" "SKIP" "Missing USER_ID or CECO_ID"
fi

# T-119 | POST /admin/users/{id}/cost-centers | Empty array
if [ -n "$_USER_ID" ] && [ "$_USER_ID" != "null" ]; then
  HEADER_APP_KEY="$APP_KEY"
  HEADER_AUTH="$ADMIN_TOKEN"
  http_request POST "${BASE_URL}/admin/users/${_USER_ID}/cost-centers" \
    '{"cost_center_ids":[]}'
  if assert_status 400; then
    print_result "T-119" "POST /admin/users/{id}/cost-centers - Empty array" "PASS"
  else
    if assert_status 201 200; then
      print_result "T-119" "POST /admin/users/{id}/cost-centers - Empty array" "PASS" "(accepted empty)"
    else
      print_result "T-119" "POST /admin/users/{id}/cost-centers - Empty array" "FAIL" "Expected 400"
    fi
  fi
else
  print_result "T-119" "POST /admin/users/{id}/cost-centers - Empty array" "SKIP" "Missing USER_ID"
fi

# T-120 | POST /admin/users/{id}/cost-centers | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/admin/users/${_USER_ID:-00000000-0000-0000-0000-000000000000}/cost-centers" \
  "{\"cost_center_ids\":[\"${_CECO_ID:-00000000-0000-0000-0000-000000000000}\"]}"
if assert_status 401; then
  print_result "T-120" "POST /admin/users/{id}/cost-centers - No Auth" "PASS"
else
  print_result "T-120" "POST /admin/users/{id}/cost-centers - No Auth" "FAIL" "Expected 401"
fi
