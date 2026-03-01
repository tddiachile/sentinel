#!/usr/bin/env bash
# 10_audit.sh - Audit log tests + pagination normalization (T-127 to T-135)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "ADMIN - AUDIT LOGS"

# T-127 | GET /admin/audit-logs | Paginated list
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/audit-logs"
if assert_status 200 && assert_json_is_array '.data'; then
  total=$(get_json_field '.total')
  if [ "$total" -ge 1 ] 2>/dev/null; then
    print_result "T-127" "GET /admin/audit-logs - Paginated list" "PASS"
  else
    print_result "T-127" "GET /admin/audit-logs - Paginated list" "FAIL" "total < 1"
  fi
else
  print_result "T-127" "GET /admin/audit-logs - Paginated list" "FAIL"
fi

# T-128 | GET /admin/audit-logs | Filter by event_type
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/audit-logs?event_type=LOGIN_SUCCESS"
if assert_status 200 && assert_json_is_array '.data'; then
  # Verify all returned items are LOGIN related
  mismatch=$(echo "$HTTP_BODY" | jq -r '[.data[] | select(.event_type | test("LOGIN") | not)] | length' 2>/dev/null)
  if [ "${mismatch:-0}" = "0" ]; then
    print_result "T-128" "GET /admin/audit-logs - Filter by event_type" "PASS"
  else
    print_result "T-128" "GET /admin/audit-logs - Filter by event_type" "FAIL" "Non-LOGIN events in results"
  fi
else
  print_result "T-128" "GET /admin/audit-logs - Filter by event_type" "FAIL"
fi

# T-129 | GET /admin/audit-logs | Filter by success
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/audit-logs?success=true"
if assert_status 200 && assert_json_is_array '.data'; then
  # Verify all returned items have success == true
  failed_count=$(echo "$HTTP_BODY" | jq -r '[.data[] | select(.success != true)] | length' 2>/dev/null)
  if [ "${failed_count:-0}" = "0" ]; then
    print_result "T-129" "GET /admin/audit-logs - Filter by success" "PASS"
  else
    print_result "T-129" "GET /admin/audit-logs - Filter by success" "FAIL" "Non-success events found"
  fi
else
  print_result "T-129" "GET /admin/audit-logs - Filter by success" "FAIL"
fi

# T-130 | GET /admin/audit-logs | Filter by date range
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/audit-logs?from_date=2026-01-01T00:00:00Z&to_date=2026-12-31T23:59:59Z"
if assert_status 200; then
  print_result "T-130" "GET /admin/audit-logs - Filter by date range" "PASS"
else
  print_result "T-130" "GET /admin/audit-logs - Filter by date range" "FAIL"
fi

# T-131 | GET /admin/audit-logs | Filter by user_id
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/audit-logs?user_id=${ADMIN_USER_ID}"
if assert_status 200; then
  print_result "T-131" "GET /admin/audit-logs - Filter by user_id" "PASS"
else
  print_result "T-131" "GET /admin/audit-logs - Filter by user_id" "FAIL"
fi

# T-132 | GET /admin/audit-logs | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request GET "${BASE_URL}/admin/audit-logs"
if assert_status 401; then
  print_result "T-132" "GET /admin/audit-logs - No Authorization" "PASS"
else
  print_result "T-132" "GET /admin/audit-logs - No Authorization" "FAIL" "Expected 401"
fi

print_section "PAGINATION NORMALIZATION"

# T-133 | GET /admin/users | page_size > 100 normalizes to 100
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/users?page_size=200"
if assert_status 200; then
  page_size=$(get_json_field '.page_size')
  if [ "$page_size" = "100" ]; then
    print_result "T-133" "GET /admin/users - page_size > 100 normalizes" "PASS"
  else
    print_result "T-133" "GET /admin/users - page_size > 100 normalizes" "FAIL" "page_size=$page_size, expected 100"
  fi
else
  print_result "T-133" "GET /admin/users - page_size > 100 normalizes" "FAIL"
fi

# T-134 | GET /admin/users | page=0 normalizes to 1
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/users?page=0"
if assert_status 200; then
  page=$(get_json_field '.page')
  if [ "$page" = "1" ]; then
    print_result "T-134" "GET /admin/users - page=0 normalizes to 1" "PASS"
  else
    print_result "T-134" "GET /admin/users - page=0 normalizes to 1" "FAIL" "page=$page, expected 1"
  fi
else
  print_result "T-134" "GET /admin/users - page=0 normalizes to 1" "FAIL"
fi

# T-135 | GET /admin/users | page > total_pages returns empty data
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/users?page=9999"
if assert_status 200; then
  data_len=$(echo "$HTTP_BODY" | jq -r '.data | length' 2>/dev/null)
  total=$(get_json_field '.total')
  if [ "$data_len" = "0" ] && [ "$total" -ge 0 ] 2>/dev/null; then
    print_result "T-135" "GET /admin/users - page > total_pages empty data" "PASS"
  else
    print_result "T-135" "GET /admin/users - page > total_pages empty data" "FAIL" "data length=$data_len"
  fi
else
  print_result "T-135" "GET /admin/users - page > total_pages empty data" "FAIL"
fi
