import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import type { ApiError, RefreshResponse } from '@/types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080/api/v1'
const APP_KEY = import.meta.env.VITE_APP_KEY ?? ''

export const apiClient = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Content-Type': 'application/json',
    'X-App-Key': APP_KEY,
  },
})

// Request interceptor: inject Authorization header from localStorage
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('sentinel_access_token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let pendingQueue: Array<{
  resolve: (value: string) => void
  reject: (reason: unknown) => void
}> = []

function processQueue(error: unknown, token: string | null) {
  pendingQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error)
    } else if (token) {
      resolve(token)
    }
  })
  pendingQueue = []
}

// Response interceptor: handle 401 with token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    if (error.response?.status === 401 && !originalRequest._retry) {
      const errorCode = error.response.data?.error?.code
      // Only refresh if it's a token expiry, not invalid credentials
      if (errorCode === 'TOKEN_EXPIRED' || errorCode === 'TOKEN_INVALID') {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            pendingQueue.push({ resolve, reject })
          }).then((token) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`
            }
            return apiClient(originalRequest)
          })
        }

        originalRequest._retry = true
        isRefreshing = true

        const refreshToken = localStorage.getItem('sentinel_refresh_token')

        if (!refreshToken) {
          processQueue(new Error('No refresh token'), null)
          isRefreshing = false
          forceLogout()
          return Promise.reject(error)
        }

        try {
          const { data } = await axios.post<RefreshResponse>(
            `${BASE_URL}/auth/refresh`,
            { refresh_token: refreshToken },
            { headers: { 'X-App-Key': APP_KEY, 'Content-Type': 'application/json' } }
          )

          localStorage.setItem('sentinel_access_token', data.access_token)
          localStorage.setItem('sentinel_refresh_token', data.refresh_token)

          processQueue(null, data.access_token)

          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${data.access_token}`
          }

          return apiClient(originalRequest)
        } catch (refreshError) {
          processQueue(refreshError, null)
          forceLogout()
          return Promise.reject(refreshError)
        } finally {
          isRefreshing = false
        }
      }
    }

    return Promise.reject(error)
  }
)

function forceLogout() {
  localStorage.removeItem('sentinel_access_token')
  localStorage.removeItem('sentinel_refresh_token')
  localStorage.removeItem('sentinel_user')
  window.location.href = '/login'
}
