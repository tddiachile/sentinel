import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  Shield,
  Key,
  Building2,
  ClipboardList,
  X,
  ShieldCheck,
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface SidebarProps {
  open: boolean
  onClose: () => void
}

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/users', icon: Users, label: 'Usuarios' },
  { to: '/roles', icon: Shield, label: 'Roles' },
  { to: '/permissions', icon: Key, label: 'Permisos' },
  { to: '/cost-centers', icon: Building2, label: 'Centros de Costo' },
  { to: '/audit', icon: ClipboardList, label: 'Auditoria' },
]

export function Sidebar({ open, onClose }: SidebarProps) {
  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed left-0 top-0 z-50 h-full w-60 bg-gray-900 text-white flex flex-col transition-transform duration-300 lg:static lg:translate-x-0 lg:z-auto',
          open ? 'translate-x-0' : '-translate-x-full'
        )}
        aria-label="Navegacion principal"
      >
        {/* Logo */}
        <div className="flex items-center justify-between px-4 py-4 border-b border-gray-700">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-7 w-7 text-blue-400" aria-hidden="true" />
            <span className="text-lg font-bold tracking-tight">Sentinel</span>
          </div>
          <button
            className="lg:hidden text-gray-400 hover:text-white"
            onClick={onClose}
            aria-label="Cerrar menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-2 py-4 space-y-1" aria-label="Menu de navegacion">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                )
              }
              onClick={() => {
                if (window.innerWidth < 1024) onClose()
              }}
            >
              <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-gray-700 text-xs text-gray-500">
          Panel de Administracion v1.0
        </div>
      </aside>
    </>
  )
}
