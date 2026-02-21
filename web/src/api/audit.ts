import { apiClient } from './client'
import type { AuditLog, PaginatedResponse } from '@/types'

export interface ListAuditLogsParams {
  page?: number
  page_size?: number
  user_id?: string
  actor_id?: string
  event_type?: string
  from_date?: string
  to_date?: string
  application_id?: string
  success?: boolean
}

export const auditApi = {
  list: (params: ListAuditLogsParams = {}) =>
    apiClient.get<PaginatedResponse<AuditLog>>('/admin/audit-logs', { params }).then((r) => r.data),
}
