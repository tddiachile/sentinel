import { Menu, LogOut, User } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/store/authStore'
import { toast } from '@/hooks/useToast'

interface HeaderProps {
  onMenuToggle: () => void
}

export function Header({ onMenuToggle }: HeaderProps) {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    toast({ title: 'Sesion cerrada', variant: 'default' })
    navigate('/login', { replace: true })
  }

  return (
    <header
      className="fixed top-0 right-0 left-0 lg:left-60 z-30 h-15 bg-white border-b border-gray-200 flex items-center px-4 gap-4"
      style={{ height: '60px' }}
    >
      <button
        className="lg:hidden text-gray-500 hover:text-gray-700 p-1 rounded"
        onClick={onMenuToggle}
        aria-label="Abrir menu de navegacion"
      >
        <Menu className="h-5 w-5" />
      </button>

      <div className="flex-1" />

      <div className="flex items-center gap-3">
        {user && (
          <div className="hidden sm:flex items-center gap-2 text-sm text-gray-700">
            <div className="h-7 w-7 rounded-full bg-blue-100 flex items-center justify-center">
              <User className="h-4 w-4 text-blue-600" aria-hidden="true" />
            </div>
            <span className="font-medium">{user.username}</span>
          </div>
        )}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          className="text-gray-500 hover:text-red-600"
          aria-label="Cerrar sesion"
        >
          <LogOut className="h-4 w-4" />
          <span className="hidden sm:inline ml-1">Salir</span>
        </Button>
      </div>
    </header>
  )
}
