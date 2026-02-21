import { apiClient } from './client'
import type { Permission, PaginatedResponse } from '@/types'

export interface ListPermissionsParams {
  page?: number
  page_size?: number
}

export interface CreatePermissionPayload {
  code: string
  description?: string
  scope_type: 'global' | 'module' | 'resource' | 'action'
}

export const permissionsApi = {
  list: (params: ListPermissionsParams = {}) =>
    apiClient.get<PaginatedResponse<Permission>>('/admin/permissions', { params }).then((r) => r.data),

  create: (payload: CreatePermissionPayload) =>
    apiClient.post<Permission>('/admin/permissions', payload).then((r) => r.data),

  delete: (id: string) =>
    apiClient.delete(`/admin/permissions/${id}`).then(() => undefined),
}
