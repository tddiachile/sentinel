import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Users, Shield, Key, ClipboardList, Building2, ArrowRight } from 'lucide-react'
import { usersApi } from '@/api/users'
import { rolesApi } from '@/api/roles'
import { permissionsApi } from '@/api/permissions'
import { auditApi } from '@/api/audit'
import type { AuditLog } from '@/types'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

interface Stats {
  totalUsers: number
  activeUsers: number
  totalRoles: number
  totalPermissions: number
}

const auditEventCategories: Record<string, 'success' | 'warning' | 'secondary' | 'destructive'> = {
  AUTH_LOGIN_SUCCESS: 'success',
  AUTH_LOGOUT: 'secondary',
  AUTH_LOGIN_FAILED: 'warning',
  AUTH_ACCOUNT_LOCKED: 'destructive',
  USER_CREATED: 'success',
  USER_UPDATED: 'secondary',
  USER_DEACTIVATED: 'warning',
  USER_UNLOCKED: 'success',
  ROLE_CREATED: 'success',
  ROLE_DELETED: 'destructive',
}

function getEventVariant(eventType: string) {
  return auditEventCategories[eventType] ?? 'secondary'
}

export function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [recentLogs, setRecentLogs] = useState<AuditLog[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      setIsLoading(true)
      try {
        const [usersAll, usersActive, roles, permissions, logs] = await Promise.allSettled([
          usersApi.list({ page: 1, page_size: 1 }),
          usersApi.list({ page: 1, page_size: 1, is_active: true }),
          rolesApi.list({ page: 1, page_size: 1 }),
          permissionsApi.list({ page: 1, page_size: 1 }),
          auditApi.list({ page: 1, page_size: 10 }),
        ])

        setStats({
          totalUsers: usersAll.status === 'fulfilled' ? usersAll.value.total : 0,
          activeUsers: usersActive.status === 'fulfilled' ? usersActive.value.total : 0,
          totalRoles: roles.status === 'fulfilled' ? roles.value.total : 0,
          totalPermissions: permissions.status === 'fulfilled' ? permissions.value.total : 0,
        })

        if (logs.status === 'fulfilled') {
          setRecentLogs(logs.value.data)
        }
      } finally {
        setIsLoading(false)
      }
    }
    void load()
  }, [])

  const statCards = [
    {
      label: 'Total Usuarios',
      value: stats?.totalUsers ?? '-',
      sub: `${stats?.activeUsers ?? '-'} activos`,
      icon: Users,
      color: 'text-blue-600',
      bg: 'bg-blue-50',
      href: '/users',
    },
    {
      label: 'Total Roles',
      value: stats?.totalRoles ?? '-',
      sub: 'Roles de acceso',
      icon: Shield,
      color: 'text-purple-600',
      bg: 'bg-purple-50',
      href: '/roles',
    },
    {
      label: 'Total Permisos',
      value: stats?.totalPermissions ?? '-',
      sub: 'Permisos del sistema',
      icon: Key,
      color: 'text-green-600',
      bg: 'bg-green-50',
      href: '/permissions',
    },
    {
      label: 'Auditoria',
      value: 'Activa',
      sub: 'Eventos registrados',
      icon: ClipboardList,
      color: 'text-orange-600',
      bg: 'bg-orange-50',
      href: '/audit',
    },
  ]

  const quickLinks = [
    { to: '/users', label: 'Gestionar usuarios', icon: Users },
    { to: '/roles', label: 'Gestionar roles', icon: Shield },
    { to: '/cost-centers', label: 'Centros de costo', icon: Building2 },
    { to: '/audit', label: 'Ver auditoria', icon: ClipboardList },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-sm text-gray-500 mt-1">Resumen del sistema Sentinel</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map(({ label, value, sub, icon: Icon, color, bg, href }) => (
          <Link
            key={label}
            to={href}
            className="bg-white rounded-lg border border-gray-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
          >
            <div className="flex items-start justify-between">
              <div>
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">{label}</p>
                <p className="mt-1 text-2xl font-bold text-gray-900">
                  {isLoading ? (
                    <span className="inline-block h-7 w-16 animate-pulse bg-gray-200 rounded" />
                  ) : (
                    value
                  )}
                </p>
                <p className="text-xs text-gray-400 mt-0.5">{sub}</p>
              </div>
              <div className={`${bg} rounded-lg p-2.5`}>
                <Icon className={`h-5 w-5 ${color}`} aria-hidden="true" />
              </div>
            </div>
          </Link>
        ))}
      </div>

      {/* Bottom grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent audit logs */}
        <div className="lg:col-span-2 bg-white rounded-lg border border-gray-200 overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
            <h2 className="font-semibold text-gray-900">Actividad reciente</h2>
            <Link to="/audit" className="text-sm text-blue-600 hover:text-blue-800 flex items-center gap-1">
              Ver todo <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="divide-y divide-gray-50">
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="px-5 py-3 flex items-center gap-3">
                  <div className="h-4 w-36 animate-pulse bg-gray-100 rounded" />
                  <div className="h-4 w-24 animate-pulse bg-gray-100 rounded ml-auto" />
                </div>
              ))
            ) : recentLogs.length === 0 ? (
              <div className="px-5 py-8 text-center text-sm text-gray-400">
                No hay eventos de auditoria aun.
              </div>
            ) : (
              recentLogs.map((log) => (
                <div key={log.id} className="px-5 py-3 flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <Badge variant={getEventVariant(log.event_type)} className="shrink-0 text-xs">
                      {log.event_type}
                    </Badge>
                    <span className="text-xs text-gray-400 truncate">
                      {log.ip_address ?? '-'}
                    </span>
                  </div>
                  <span className="text-xs text-gray-400 shrink-0">
                    {formatDate(log.created_at)}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Quick links */}
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <div className="px-5 py-4 border-b border-gray-100">
            <h2 className="font-semibold text-gray-900">Accesos rapidos</h2>
          </div>
          <div className="p-3 space-y-1">
            {quickLinks.map(({ to, label, icon: Icon }) => (
              <Link
                key={to}
                to={to}
                className="flex items-center gap-3 px-3 py-2.5 rounded-md text-sm text-gray-700 hover:bg-gray-50 hover:text-blue-600 transition-colors group"
              >
                <Icon className="h-4 w-4 text-gray-400 group-hover:text-blue-500" aria-hidden="true" />
                {label}
                <ArrowRight className="h-3 w-3 ml-auto text-gray-300 group-hover:text-blue-400" />
              </Link>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
