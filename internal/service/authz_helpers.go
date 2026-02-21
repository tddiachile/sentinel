package service

// authz_helpers.go exposes internal authorization logic as exported functions
// so they can be tested from the service_test package.
// These functions mirror the private methods in authz_service.go.

import (
	"sort"

	redisrepo "github.com/enunezf/sentinel/internal/repository/redis"
)

// CheckHasPermission evaluates if a user context has a given permission code,
// optionally checking a cost-center constraint. This is the exported test-facing
// version of the private hasPermission + hasCostCenter combination in AuthzService.
func CheckHasPermission(uc *redisrepo.UserContext, permCode, costCenterCode string) bool {
	hasPermission := false
	for _, p := range uc.Permissions {
		if p == permCode {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return false
	}
	if costCenterCode == "" {
		return true
	}
	for _, cc := range uc.CostCenters {
		if cc == costCenterCode {
			return true
		}
	}
	return false
}

// MergePermissions returns the union of two permission slices without duplicates.
// Used in tests to verify the union logic that buildUserContext applies.
func MergePermissions(rolePerms, individualPerms []string) []string {
	set := make(map[string]struct{})
	for _, p := range rolePerms {
		set[p] = struct{}{}
	}
	for _, p := range individualPerms {
		set[p] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for p := range set {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
}

// CanonicalJSONPayload is the exported version of canonicalJSONPayload, used
// in authz_service_test.go to verify signature and version logic.
func CanonicalJSONPayload(application, generatedAt string, permissions map[string]PermissionMapEntry, costCenters map[string]CostCenterMapEntry) []byte {
	return canonicalJSONPayload(application, generatedAt, permissions, costCenters)
}
