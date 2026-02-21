import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { authApi, type LoginCredentials } from '@/api/auth'
import type { ApiError } from '@/types'
import type { AxiosError } from 'axios'

interface AuthUser {
  id: string
  username: string
  email: string
  must_change_password: boolean
}

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (credentials: LoginCredentials) => Promise<void>
  logout: () => Promise<void>
  setTokens: (accessToken: string, refreshToken: string) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (credentials: LoginCredentials) => {
        set({ isLoading: true })
        try {
          const data = await authApi.login(credentials)
          localStorage.setItem('sentinel_access_token', data.access_token)
          localStorage.setItem('sentinel_refresh_token', data.refresh_token)
          set({
            accessToken: data.access_token,
            refreshToken: data.refresh_token,
            user: {
              id: data.user.id,
              username: data.user.username,
              email: data.user.email,
              must_change_password: data.user.must_change_password,
            },
            isAuthenticated: true,
            isLoading: false,
          })
        } catch (error) {
          set({ isLoading: false })
          const axiosError = error as AxiosError<ApiError>
          const code = axiosError.response?.data?.error?.code ?? 'INTERNAL_ERROR'
          throw new Error(code)
        }
      },

      logout: async () => {
        try {
          await authApi.logout()
        } catch {
          // silently ignore logout errors
        } finally {
          get().clearAuth()
        }
      },

      setTokens: (accessToken: string, refreshToken: string) => {
        localStorage.setItem('sentinel_access_token', accessToken)
        localStorage.setItem('sentinel_refresh_token', refreshToken)
        set({ accessToken, refreshToken, isAuthenticated: true })
      },

      clearAuth: () => {
        localStorage.removeItem('sentinel_access_token')
        localStorage.removeItem('sentinel_refresh_token')
        set({
          accessToken: null,
          refreshToken: null,
          user: null,
          isAuthenticated: false,
        })
      },
    }),
    {
      name: 'sentinel_user',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
