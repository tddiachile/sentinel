import { useEffect, useState, useCallback } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { ChevronDown, ChevronRight, Filter } from 'lucide-react'
import { auditApi } from '@/api/audit'
import type { ListAuditLogsParams } from '@/api/audit'
import type { AuditLog } from '@/types'
import { DataTable } from '@/components/shared/DataTable'
import { PageHeader } from '@/components/shared/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { toast } from '@/hooks/useToast'
import { usePagination } from '@/hooks/usePagination'
import { formatDate } from '@/lib/utils'

const EVENT_TYPES = [
  'AUTH_LOGIN_SUCCESS',
  'AUTH_LOGIN_FAILED',
  'AUTH_ACCOUNT_LOCKED',
  'AUTH_LOGOUT',
  'AUTH_TOKEN_REFRESHED',
  'AUTH_PASSWORD_CHANGED',
  'AUTH_PASSWORD_RESET',
  'USER_CREATED',
  'USER_UPDATED',
  'USER_DEACTIVATED',
  'USER_UNLOCKED',
  'USER_ROLE_ASSIGNED',
  'USER_ROLE_REVOKED',
  'USER_PERMISSION_ASSIGNED',
  'USER_PERMISSION_REVOKED',
  'USER_COST_CENTER_ASSIGNED',
  'ROLE_CREATED',
  'ROLE_UPDATED',
  'ROLE_DELETED',
  'ROLE_PERMISSION_ASSIGNED',
  'ROLE_PERMISSION_REVOKED',
  'PERMISSION_CREATED',
  'PERMISSION_DELETED',
]

const eventCategoryVariant = (eventType: string): 'default' | 'success' | 'warning' | 'destructive' | 'secondary' => {
  if (eventType.includes('FAILED') || eventType.includes('LOCKED') || eventType.includes('DELETED') || eventType.includes('REVOKED')) {
    return 'destructive'
  }
  if (eventType.includes('SUCCESS') || eventType.includes('CREATED') || eventType.includes('ASSIGNED') || eventType.includes('UNLOCKED')) {
    return 'success'
  }
  if (eventType.includes('UPDATED') || eventType.includes('DEACTIVATED') || eventType.includes('RESET') || eventType.includes('CHANGED')) {
    return 'warning'
  }
  if (eventType.includes('LOGOUT') || eventType.includes('REFRESH')) {
    return 'secondary'
  }
  return 'secondary'
}

interface ExpandedRowProps {
  log: AuditLog
}

function ExpandedRow({ log }: ExpandedRowProps) {
  return (
    <div className="px-4 py-3 bg-gray-50 border-t border-gray-100 text-xs font-mono space-y-2">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {log.old_value !== null && (
          <div>
            <p className="text-gray-500 mb-1 font-sans font-medium">Valor anterior:</p>
            <pre className="bg-white border border-gray-200 rounded p-2 overflow-auto max-h-40 text-gray-700">
              {JSON.stringify(log.old_value, null, 2)}
            </pre>
          </div>
        )}
        {log.new_value !== null && (
          <div>
            <p className="text-gray-500 mb-1 font-sans font-medium">Nuevo valor:</p>
            <pre className="bg-white border border-gray-200 rounded p-2 overflow-auto max-h-40 text-gray-700">
              {JSON.stringify(log.new_value, null, 2)}
            </pre>
          </div>
        )}
        {log.error_message && (
          <div className="sm:col-span-2">
            <p className="text-red-500 mb-1 font-sans font-medium">Error:</p>
            <p className="text-red-700 bg-red-50 border border-red-200 rounded p-2">{log.error_message}</p>
          </div>
        )}
        <div>
          <p className="text-gray-500 font-sans">User Agent: <span className="text-gray-700">{log.user_agent ?? '-'}</span></p>
        </div>
        {(log.old_value === null && log.new_value === null && !log.error_message) && (
          <div className="sm:col-span-2 text-gray-400 font-sans">Sin datos adicionales.</div>
        )}
      </div>
    </div>
  )
}

export function AuditLogsPage() {
  const { page, pageSize, goToPage, reset } = usePagination()
  const [data, setData] = useState<{ items: AuditLog[]; total: number; totalPages: number }>({
    items: [],
    total: 0,
    totalPages: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Filters
  const [filters, setFilters] = useState<{
    userId: string
    actorId: string
    eventType: string
    fromDate: string
    toDate: string
    success: string
  }>({
    userId: '',
    actorId: '',
    eventType: '',
    fromDate: '',
    toDate: '',
    success: '',
  })

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const params: ListAuditLogsParams = { page, page_size: pageSize }
      if (filters.userId) params.user_id = filters.userId
      if (filters.actorId) params.actor_id = filters.actorId
      if (filters.eventType) params.event_type = filters.eventType
      if (filters.fromDate) params.from_date = new Date(filters.fromDate).toISOString()
      if (filters.toDate) params.to_date = new Date(filters.toDate).toISOString()
      if (filters.success === 'true') params.success = true
      if (filters.success === 'false') params.success = false

      const res = await auditApi.list(params)
      setData({ items: res.data, total: res.total, totalPages: res.total_pages })
    } catch {
      toast({ title: 'Error al cargar logs de auditoria', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize, filters])

  useEffect(() => {
    void load()
  }, [load])

  const handleFilterChange = (key: keyof typeof filters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }))
    reset()
  }

  const clearFilters = () => {
    setFilters({ userId: '', actorId: '', eventType: '', fromDate: '', toDate: '', success: '' })
    reset()
  }

  const columns: ColumnDef<AuditLog>[] = [
    {
      id: 'expand',
      header: '',
      cell: ({ row }) => (
        <button
          onClick={() => setExpandedId((prev) => (prev === row.original.id ? null : row.original.id))}
          className="text-gray-400 hover:text-gray-600"
          aria-label="Expandir fila"
        >
          {expandedId === row.original.id ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>
      ),
    },
    {
      accessorKey: 'created_at',
      header: 'Timestamp',
      cell: ({ row }) => (
        <span className="text-gray-500 text-xs whitespace-nowrap">{formatDate(row.original.created_at)}</span>
      ),
    },
    {
      accessorKey: 'event_type',
      header: 'Evento',
      cell: ({ row }) => (
        <Badge variant={eventCategoryVariant(row.original.event_type)} className="text-xs whitespace-nowrap">
          {row.original.event_type}
        </Badge>
      ),
    },
    {
      accessorKey: 'user_id',
      header: 'Usuario',
      cell: ({ row }) => (
        <span className="text-gray-600 text-xs font-mono">
          {row.original.user_id ? row.original.user_id.slice(0, 8) + '...' : '-'}
        </span>
      ),
    },
    {
      accessorKey: 'actor_id',
      header: 'Actor',
      cell: ({ row }) => (
        <span className="text-gray-600 text-xs font-mono">
          {row.original.actor_id ? row.original.actor_id.slice(0, 8) + '...' : '-'}
        </span>
      ),
    },
    {
      accessorKey: 'ip_address',
      header: 'IP',
      cell: ({ row }) => (
        <span className="text-gray-500 text-xs font-mono">{row.original.ip_address ?? '-'}</span>
      ),
    },
    {
      accessorKey: 'success',
      header: 'Resultado',
      cell: ({ row }) => (
        <Badge variant={row.original.success ? 'success' : 'destructive'} className="text-xs">
          {row.original.success ? 'Exito' : 'Fallo'}
        </Badge>
      ),
    },
  ]

  // Custom render that supports expanded rows
  const dataWithExpanded = data.items

  return (
    <div>
      <PageHeader
        title="Auditoria"
        description="Registro de eventos del sistema (solo lectura)"
      />

      {/* Filters */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 mb-4 space-y-3">
        <div className="flex items-center gap-2 text-sm font-medium text-gray-700">
          <Filter className="h-4 w-4 text-gray-400" aria-hidden="true" />
          Filtros
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div className="space-y-1">
            <Label htmlFor="filter-user-id">User ID</Label>
            <Input
              id="filter-user-id"
              placeholder="UUID del usuario"
              value={filters.userId}
              onChange={(e) => handleFilterChange('userId', e.target.value)}
              className="text-xs font-mono"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="filter-actor-id">Actor ID</Label>
            <Input
              id="filter-actor-id"
              placeholder="UUID del actor"
              value={filters.actorId}
              onChange={(e) => handleFilterChange('actorId', e.target.value)}
              className="text-xs font-mono"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="filter-event-type">Tipo de evento</Label>
            <Select
              value={filters.eventType || '_all'}
              onValueChange={(val) => handleFilterChange('eventType', val === '_all' ? '' : val)}
            >
              <SelectTrigger id="filter-event-type">
                <SelectValue placeholder="Todos los eventos" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="_all">Todos los eventos</SelectItem>
                {EVENT_TYPES.map((et) => (
                  <SelectItem key={et} value={et}>{et}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="filter-from-date">Desde</Label>
            <Input
              id="filter-from-date"
              type="datetime-local"
              value={filters.fromDate}
              onChange={(e) => handleFilterChange('fromDate', e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="filter-to-date">Hasta</Label>
            <Input
              id="filter-to-date"
              type="datetime-local"
              value={filters.toDate}
              onChange={(e) => handleFilterChange('toDate', e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="filter-success">Resultado</Label>
            <Select
              value={filters.success || '_all'}
              onValueChange={(val) => handleFilterChange('success', val === '_all' ? '' : val)}
            >
              <SelectTrigger id="filter-success">
                <SelectValue placeholder="Todos" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="_all">Todos</SelectItem>
                <SelectItem value="true">Solo exitos</SelectItem>
                <SelectItem value="false">Solo fallos</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="flex justify-end">
          <Button variant="outline" size="sm" onClick={clearFilters}>
            Limpiar filtros
          </Button>
        </div>
      </div>

      {/* Table with expandable rows */}
      <div className="space-y-4">
        <div className="rounded-md border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-4 py-3 w-8" />
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">Timestamp</th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">Evento</th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">Usuario</th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">Actor</th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">IP</th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">Resultado</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-100">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-gray-500">
                    <div className="flex items-center justify-center gap-2">
                      <div className="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
                      Cargando...
                    </div>
                  </td>
                </tr>
              ) : dataWithExpanded.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-gray-400">
                    No se encontraron eventos de auditoria.
                  </td>
                </tr>
              ) : (
                dataWithExpanded.map((log) => (
                  <>
                    <tr key={log.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3">
                        <button
                          onClick={() => setExpandedId((prev) => (prev === log.id ? null : log.id))}
                          className="text-gray-400 hover:text-gray-600"
                          aria-label="Expandir detalles"
                        >
                          {expandedId === log.id ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                        </button>
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-xs whitespace-nowrap">
                        {formatDate(log.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={eventCategoryVariant(log.event_type)} className="text-xs whitespace-nowrap">
                          {log.event_type}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-xs font-mono">
                        {log.user_id ? log.user_id.slice(0, 8) + '...' : '-'}
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-xs font-mono">
                        {log.actor_id ? log.actor_id.slice(0, 8) + '...' : '-'}
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-xs font-mono">
                        {log.ip_address ?? '-'}
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={log.success ? 'success' : 'destructive'} className="text-xs">
                          {log.success ? 'Exito' : 'Fallo'}
                        </Badge>
                      </td>
                    </tr>
                    {expandedId === log.id && (
                      <tr key={`${log.id}-expanded`}>
                        <td colSpan={7} className="p-0">
                          <ExpandedRow log={log} />
                        </td>
                      </tr>
                    )}
                  </>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between text-sm text-gray-500">
          <span>
            {data.total === 0
              ? 'Sin resultados'
              : `Mostrando ${data.total === 0 ? 0 : (page - 1) * pageSize + 1}–${Math.min(page * pageSize, data.total)} de ${data.total} eventos`}
          </span>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => goToPage(page - 1)} disabled={page <= 1 || isLoading}>
              Anterior
            </Button>
            <span className="px-2 font-medium text-gray-700">
              Pagina {page} de {Math.max(1, data.totalPages)}
            </span>
            <Button variant="outline" size="sm" onClick={() => goToPage(page + 1)} disabled={page >= data.totalPages || isLoading}>
              Siguiente
            </Button>
          </div>
        </div>
      </div>

      {/* Hidden DataTable for columns reference */}
      <div className="hidden">
        <DataTable
          data={[]}
          columns={columns}
          page={1}
          pageSize={20}
          total={0}
          totalPages={0}
          onPageChange={() => undefined}
        />
      </div>
    </div>
  )
}
