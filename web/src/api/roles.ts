import { apiClient } from './client'
import type { Role, PaginatedResponse } from '@/types'

export interface ListRolesParams {
  page?: number
  page_size?: number
}

export interface CreateRolePayload {
  name: string
  description?: string
}

export interface UpdateRolePayload {
  name?: string
  description?: string
}

export interface AssignPermissionsPayload {
  permission_ids: string[]
}

export const rolesApi = {
  list: (params: ListRolesParams = {}) =>
    apiClient.get<PaginatedResponse<Role>>('/admin/roles', { params }).then((r) => r.data),

  get: (id: string) =>
    apiClient.get<Role>(`/admin/roles/${id}`).then((r) => r.data),

  create: (payload: CreateRolePayload) =>
    apiClient.post<Role>('/admin/roles', payload).then((r) => r.data),

  update: (id: string, payload: UpdateRolePayload) =>
    apiClient.put<Role>(`/admin/roles/${id}`, payload).then((r) => r.data),

  deactivate: (id: string) =>
    apiClient.delete(`/admin/roles/${id}`).then(() => undefined),

  assignPermissions: (id: string, payload: AssignPermissionsPayload) =>
    apiClient.post(`/admin/roles/${id}/permissions`, payload).then((r) => r.data),

  revokePermission: (id: string, pid: string) =>
    apiClient.delete(`/admin/roles/${id}/permissions/${pid}`).then(() => undefined),
}
