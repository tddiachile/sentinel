#!/usr/bin/env bash
# 04_applications.sh - Application admin tests (T-041 to T-057)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"
load_state

print_section "ADMIN - APPLICATIONS (LIST)"

# T-041 | GET /admin/applications | Paginated list
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications"
if assert_status 200 && assert_json_is_array '.data' && assert_json_field '.page' '1'; then
  total=$(get_json_field '.total')
  if [ "$total" -ge 1 ] 2>/dev/null; then
    print_result "T-041" "GET /admin/applications - Paginated list" "PASS"
  else
    print_result "T-041" "GET /admin/applications - Paginated list" "FAIL" "total < 1"
  fi
else
  print_result "T-041" "GET /admin/applications - Paginated list" "FAIL"
fi

# T-042 | GET /admin/applications | Custom pagination
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications?page=1&page_size=5"
if assert_status 200 && assert_json_field '.page_size' '5'; then
  print_result "T-042" "GET /admin/applications - Custom pagination" "PASS"
else
  print_result "T-042" "GET /admin/applications - Custom pagination" "FAIL"
fi

# T-043 | GET /admin/applications | Search filter
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications?search=system"
if assert_status 200; then
  total=$(get_json_field '.total')
  if [ "$total" -ge 1 ] 2>/dev/null; then
    print_result "T-043" "GET /admin/applications - Search filter" "PASS"
  else
    print_result "T-043" "GET /admin/applications - Search filter" "FAIL" "total=$total"
  fi
else
  print_result "T-043" "GET /admin/applications - Search filter" "FAIL"
fi

# T-044 | GET /admin/applications | is_active filter
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications?is_active=true"
if assert_status 200; then
  print_result "T-044" "GET /admin/applications - is_active filter" "PASS"
else
  print_result "T-044" "GET /admin/applications - is_active filter" "FAIL"
fi

# T-045 | GET /admin/applications | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request GET "${BASE_URL}/admin/applications"
if assert_status 401; then
  print_result "T-045" "GET /admin/applications - No Authorization" "PASS"
else
  print_result "T-045" "GET /admin/applications - No Authorization" "FAIL" "Expected 401"
fi

# T-046 | GET /admin/applications | No X-App-Key
HEADER_APP_KEY=""
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications"
if assert_status 401; then
  print_result "T-046" "GET /admin/applications - No X-App-Key" "PASS"
else
  print_result "T-046" "GET /admin/applications - No X-App-Key" "FAIL" "Expected 401"
fi

print_section "ADMIN - APPLICATIONS (CREATE)"

# Clean up if test app exists from a previous run
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications?search=test-app-api"
existing_app_id=$(echo "$HTTP_BODY" | jq -r '.data[] | select(.slug == "test-app-api") | .id' 2>/dev/null)

# T-047 | POST /admin/applications | Create application
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/applications" \
  '{"name":"Test App API","slug":"test-app-api"}'
if assert_status 201; then
  TEST_APP_ID=$(get_json_field '.id')
  save_state "TEST_APP_ID" "$TEST_APP_ID"
  if assert_json_field '.slug' 'test-app-api' && assert_json_field '.is_active' 'true'; then
    print_result "T-047" "POST /admin/applications - Create app" "PASS"
  else
    print_result "T-047" "POST /admin/applications - Create app" "FAIL" "Fields mismatch"
  fi
elif assert_status 409; then
  # Already exists from previous run, use existing ID
  if [ -n "$existing_app_id" ]; then
    TEST_APP_ID="$existing_app_id"
    save_state "TEST_APP_ID" "$TEST_APP_ID"
    print_result "T-047" "POST /admin/applications - Create app" "PASS" "(already existed)"
  else
    print_result "T-047" "POST /admin/applications - Create app" "FAIL" "409 but can't find ID"
  fi
else
  print_result "T-047" "POST /admin/applications - Create app" "FAIL"
fi

# T-048 | POST /admin/applications | Duplicate slug
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/applications" \
  '{"name":"Duplicate App","slug":"test-app-api"}'
if assert_status 409 400; then
  print_result "T-048" "POST /admin/applications - Duplicate slug" "PASS"
else
  print_result "T-048" "POST /admin/applications - Duplicate slug" "FAIL" "Expected 409 or 400"
fi

# T-049 | POST /admin/applications | Missing data
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/applications" '{}'
if assert_status 400; then
  print_result "T-049" "POST /admin/applications - Missing data" "PASS"
else
  print_result "T-049" "POST /admin/applications - Missing data" "FAIL" "Expected 400"
fi

# T-050 | POST /admin/applications | No Authorization
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH=""
http_request POST "${BASE_URL}/admin/applications" \
  '{"name":"No Auth App","slug":"no-auth-app"}'
if assert_status 401; then
  print_result "T-050" "POST /admin/applications - No Authorization" "PASS"
else
  print_result "T-050" "POST /admin/applications - No Authorization" "FAIL" "Expected 401"
fi

print_section "ADMIN - APPLICATIONS (GET/UPDATE/ROTATE)"

load_state

# T-051 | GET /admin/applications/{id} | Get existing app
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications/${TEST_APP_ID}"
if assert_status 200 && assert_json_field '.id' "$TEST_APP_ID"; then
  print_result "T-051" "GET /admin/applications/{id} - Existing app" "PASS"
else
  print_result "T-051" "GET /admin/applications/{id} - Existing app" "FAIL"
fi

# T-052 | GET /admin/applications/{id} | Not found
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications/00000000-0000-0000-0000-000000000000"
if assert_status 404; then
  print_result "T-052" "GET /admin/applications/{id} - Not found" "PASS"
else
  print_result "T-052" "GET /admin/applications/{id} - Not found" "FAIL" "Expected 404"
fi

# T-053 | GET /admin/applications/{id} | Invalid ID
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request GET "${BASE_URL}/admin/applications/not-a-uuid"
if assert_status 400; then
  print_result "T-053" "GET /admin/applications/{id} - Invalid ID" "PASS"
else
  print_result "T-053" "GET /admin/applications/{id} - Invalid ID" "FAIL" "Expected 400, got $HTTP_STATUS"
fi

# T-054 | PUT /admin/applications/{id} | Update name
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request PUT "${BASE_URL}/admin/applications/${TEST_APP_ID}" \
  '{"name":"Test App Updated"}'
if assert_status 200; then
  print_result "T-054" "PUT /admin/applications/{id} - Update name" "PASS"
else
  print_result "T-054" "PUT /admin/applications/{id} - Update name" "FAIL"
fi

# T-055 | PUT /admin/applications/{id} | Not found
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request PUT "${BASE_URL}/admin/applications/00000000-0000-0000-0000-000000000000" \
  '{"name":"Ghost App"}'
if assert_status 404; then
  print_result "T-055" "PUT /admin/applications/{id} - Not found" "PASS"
else
  print_result "T-055" "PUT /admin/applications/{id} - Not found" "FAIL" "Expected 404"
fi

# T-056 | POST /admin/applications/{id}/rotate-key | Rotate key
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/applications/${TEST_APP_ID}/rotate-key"
if assert_status 200; then
  print_result "T-056" "POST /admin/applications/{id}/rotate-key - Success" "PASS"
else
  print_result "T-056" "POST /admin/applications/{id}/rotate-key - Success" "FAIL"
fi

# T-057 | POST /admin/applications/{id}/rotate-key | Not found
HEADER_APP_KEY="$APP_KEY"
HEADER_AUTH="$ADMIN_TOKEN"
http_request POST "${BASE_URL}/admin/applications/00000000-0000-0000-0000-000000000000/rotate-key"
if assert_status 404; then
  print_result "T-057" "POST /admin/applications/{id}/rotate-key - Not found" "PASS"
else
  print_result "T-057" "POST /admin/applications/{id}/rotate-key - Not found" "FAIL" "Expected 404"
fi
