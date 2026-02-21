import { apiClient } from './client'
import type { User, PaginatedResponse, RoleAssignment, ResetPasswordResponse } from '@/types'

export interface ListUsersParams {
  page?: number
  page_size?: number
  search?: string
  is_active?: boolean
}

export interface CreateUserPayload {
  username: string
  email: string
  password: string
}

export interface UpdateUserPayload {
  username?: string
  email?: string
  is_active?: boolean
}

export interface AssignRolePayload {
  role_id: string
  valid_from?: string
  valid_until?: string | null
}

export interface AssignPermissionPayload {
  permission_id: string
  valid_from?: string
  valid_until?: string | null
}

export interface AssignCostCentersPayload {
  cost_center_ids: string[]
  valid_from?: string
  valid_until?: string | null
}

export const usersApi = {
  list: (params: ListUsersParams = {}) =>
    apiClient.get<PaginatedResponse<User>>('/admin/users', { params }).then((r) => r.data),

  get: (id: string) =>
    apiClient.get<User>(`/admin/users/${id}`).then((r) => r.data),

  create: (payload: CreateUserPayload) =>
    apiClient.post<User>('/admin/users', payload).then((r) => r.data),

  update: (id: string, payload: UpdateUserPayload) =>
    apiClient.put<User>(`/admin/users/${id}`, payload).then((r) => r.data),

  unlock: (id: string) =>
    apiClient.post(`/admin/users/${id}/unlock`).then(() => undefined),

  resetPassword: (id: string) =>
    apiClient.post<ResetPasswordResponse>(`/admin/users/${id}/reset-password`).then((r) => r.data),

  assignRole: (id: string, payload: AssignRolePayload) =>
    apiClient.post<RoleAssignment>(`/admin/users/${id}/roles`, payload).then((r) => r.data),

  revokeRole: (id: string, rid: string) =>
    apiClient.delete(`/admin/users/${id}/roles/${rid}`).then(() => undefined),

  assignPermission: (id: string, payload: AssignPermissionPayload) =>
    apiClient.post(`/admin/users/${id}/permissions`, payload).then((r) => r.data),

  revokePermission: (id: string, pid: string) =>
    apiClient.delete(`/admin/users/${id}/permissions/${pid}`).then(() => undefined),

  assignCostCenters: (id: string, payload: AssignCostCentersPayload) =>
    apiClient.post(`/admin/users/${id}/cost-centers`, payload).then((r) => r.data),
}
