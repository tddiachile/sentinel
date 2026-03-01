#!/usr/bin/env bash
# run_all.sh - Orchestrator for all Sentinel API tests
# Executes test scripts in sequence, aggregates results.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/common.sh"

echo ""
echo -e "${CYAN}============================================${NC}"
echo -e "${CYAN}  SENTINEL API TEST SUITE${NC}"
echo -e "${CYAN}  $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "${CYAN}  Target: ${BASE_URL}${NC}"
echo -e "${CYAN}============================================${NC}"

# ---- Pre-flight checks ----

print_section "PRE-FLIGHT CHECKS"

# Check that jq is available
if ! command -v jq &> /dev/null; then
  echo -e "${RED}ERROR: jq is not installed. Install with: sudo apt install jq${NC}"
  exit 1
fi
echo -e "  ${GREEN}OK${NC} | jq is available"

# Check that curl is available
if ! command -v curl &> /dev/null; then
  echo -e "${RED}ERROR: curl is not installed.${NC}"
  exit 1
fi
echo -e "  ${GREEN}OK${NC} | curl is available"

# Check health endpoint
HEADER_APP_KEY=""
HEADER_AUTH=""
http_request GET "${BASE_URL}/health"
if [ "$HTTP_STATUS" = "200" ]; then
  echo -e "  ${GREEN}OK${NC} | Service is healthy at ${BASE_URL}"
else
  echo -e "${RED}ERROR: Service not reachable at ${BASE_URL} (HTTP $HTTP_STATUS)${NC}"
  echo -e "${RED}Make sure the stack is running: make docker-up${NC}"
  exit 1
fi

# ---- Initialize state ----
init_state

# ---- Obtain APP_KEY ----
echo ""
echo -e "  Obtaining APP_KEY..."
if obtain_app_key; then
  echo -e "  ${GREEN}OK${NC} | APP_KEY obtained: $(truncate_token "$APP_KEY")"
else
  echo -e "${RED}ERROR: Could not obtain APP_KEY${NC}"
  exit 1
fi

# ---- Admin Login ----
echo -e "  Logging in as admin..."
if do_admin_login; then
  echo -e "  ${GREEN}OK${NC} | Admin login successful (token: $(truncate_token "$ADMIN_TOKEN"))"
else
  echo -e "${RED}ERROR: Admin login failed${NC}"
  exit 1
fi

# Save results file path
RESULTS_DIR="$SCRIPT_DIR/results"
mkdir -p "$RESULTS_DIR"
RESULTS_FILE="$RESULTS_DIR/test_results_$(date '+%Y%m%d_%H%M%S').log"

# ---- Execute test scripts in order ----

# Array of test scripts in execution order
TEST_SCRIPTS=(
  "01_system.sh"
  "02_auth.sh"
  "03_authz.sh"
  "04_applications.sh"
  "05_users.sh"
  "06_roles.sh"
  "07_permissions.sh"
  "08_cost_centers.sh"
  "09_assignments.sh"
  "10_audit.sh"
  "11_change_password.sh"
  "12_cleanup.sh"
)

# Track aggregated counters
AGG_TOTAL=0
AGG_PASSED=0
AGG_FAILED=0
AGG_SKIPPED=0
SCRIPT_ERRORS=()

for script in "${TEST_SCRIPTS[@]}"; do
  script_path="$SCRIPT_DIR/$script"
  if [ ! -f "$script_path" ]; then
    echo -e "${RED}WARNING: Script not found: $script_path${NC}"
    continue
  fi

  # Execute script and capture output
  # Each script sources config.sh and common.sh which reset counters
  # We capture the output and parse the results
  echo ""
  echo -e "${CYAN}--- Running: ${script} ---${NC}"

  script_output=""
  script_exit=0
  script_output=$(bash "$script_path" 2>&1) || script_exit=$?

  echo "$script_output"

  if [ $script_exit -ne 0 ]; then
    echo -e "  ${RED}Script ${script} exited with error code ${script_exit}${NC}"
    SCRIPT_ERRORS+=("$script (exit $script_exit)")
  fi

  # Count PASS/FAIL/SKIP from output (strip ANSI codes first)
  clean_output=$(echo "$script_output" | sed 's/\x1b\[[0-9;]*m//g')
  pass_count=$(echo "$clean_output" | grep -c "PASS |" || true)
  fail_count=$(echo "$clean_output" | grep -c "FAIL |" || true)
  skip_count=$(echo "$clean_output" | grep -c "SKIP |" || true)

  AGG_TOTAL=$((AGG_TOTAL + pass_count + fail_count + skip_count))
  AGG_PASSED=$((AGG_PASSED + pass_count))
  AGG_FAILED=$((AGG_FAILED + fail_count))
  AGG_SKIPPED=$((AGG_SKIPPED + skip_count))
done

# ---- Print aggregated summary ----
echo ""
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  AGGREGATED TEST SUMMARY${NC}"
echo -e "${CYAN}========================================${NC}"
echo -e "  Total:   ${AGG_TOTAL}"
echo -e "  ${GREEN}Passed:  ${AGG_PASSED}${NC}"
echo -e "  ${RED}Failed:  ${AGG_FAILED}${NC}"
echo -e "  ${YELLOW}Skipped: ${AGG_SKIPPED}${NC}"

if [ ${#SCRIPT_ERRORS[@]} -gt 0 ]; then
  echo ""
  echo -e "  ${RED}Scripts with errors:${NC}"
  for err in "${SCRIPT_ERRORS[@]}"; do
    echo -e "    ${RED}- ${err}${NC}"
  done
fi

echo -e "${CYAN}========================================${NC}"
echo -e "  Execution completed: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "${CYAN}========================================${NC}"

# ---- Save results ----
{
  echo "SENTINEL API TEST RESULTS"
  echo "========================="
  echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
  echo "Target: ${BASE_URL}"
  echo ""
  echo "Total:   ${AGG_TOTAL}"
  echo "Passed:  ${AGG_PASSED}"
  echo "Failed:  ${AGG_FAILED}"
  echo "Skipped: ${AGG_SKIPPED}"
  echo ""
  if [ ${#SCRIPT_ERRORS[@]} -gt 0 ]; then
    echo "Scripts with errors:"
    for err in "${SCRIPT_ERRORS[@]}"; do
      echo "  - ${err}"
    done
  fi
} > "$RESULTS_FILE"

echo ""
echo -e "Results saved to: ${RESULTS_FILE}"

# Exit with error if any tests failed
if [ "$AGG_FAILED" -gt 0 ]; then
  exit 1
fi
