import { apiClient } from './client'
import type { LoginResponse } from '@/types'

export interface LoginCredentials {
  username: string
  password: string
  client_type: 'web' | 'mobile' | 'desktop'
}

export interface ChangePasswordPayload {
  current_password: string
  new_password: string
}

export const authApi = {
  login: (credentials: LoginCredentials) =>
    apiClient.post<LoginResponse>('/auth/login', credentials).then((r) => r.data),

  logout: () => apiClient.post('/auth/logout').then(() => undefined),

  changePassword: (payload: ChangePasswordPayload) =>
    apiClient.post('/auth/change-password', payload).then(() => undefined),
}
