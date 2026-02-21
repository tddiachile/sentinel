import { useAuthStore } from '@/store/authStore'

export function useAuth() {
  const { user, isAuthenticated, isLoading, login, logout, clearAuth } = useAuthStore()

  return {
    user,
    isAuthenticated,
    isLoading,
    login,
    logout,
    clearAuth,
  }
}
