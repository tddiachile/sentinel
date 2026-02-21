import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { Plus, Search, Eye, Unlock, RotateCcw, UserCheck, UserX } from 'lucide-react'
import { usersApi } from '@/api/users'
import type { User } from '@/types'
import { DataTable } from '@/components/shared/DataTable'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { PageHeader } from '@/components/shared/PageHeader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { UserFormDialog } from './UserFormDialog'
import { toast } from '@/hooks/useToast'
import { usePagination } from '@/hooks/usePagination'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

export function UsersPage() {
  const navigate = useNavigate()
  const { page, pageSize, goToPage, reset } = usePagination()
  const [data, setData] = useState<{ items: User[]; total: number; totalPages: number }>({
    items: [],
    total: 0,
    totalPages: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [filterActive, setFilterActive] = useState<boolean | undefined>(undefined)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [confirmAction, setConfirmAction] = useState<{
    type: 'unlock' | 'reset' | 'activate' | 'deactivate'
    user: User
  } | null>(null)
  const [isActionLoading, setIsActionLoading] = useState(false)

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const params: { page: number; page_size: number; search?: string; is_active?: boolean } = {
        page,
        page_size: pageSize,
      }
      if (search) params.search = search
      if (filterActive !== undefined) params.is_active = filterActive

      const res = await usersApi.list(params)
      setData({ items: res.data, total: res.total, totalPages: res.total_pages })
    } catch {
      toast({ title: 'Error al cargar usuarios', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize, search, filterActive])

  useEffect(() => {
    void load()
  }, [load])

  const handleSearch = (value: string) => {
    setSearch(value)
    reset()
  }

  const handleConfirmAction = async () => {
    if (!confirmAction) return
    setIsActionLoading(true)
    const { type, user } = confirmAction
    try {
      if (type === 'unlock') {
        await usersApi.unlock(user.id)
        toast({ title: 'Usuario desbloqueado', description: `@${user.username} ha sido desbloqueado.` })
      } else if (type === 'reset') {
        const res = await usersApi.resetPassword(user.id)
        toast({
          title: 'Contrasena reseteada',
          description: `Contrasena temporal: ${res.temporary_password}`,
        })
      } else if (type === 'activate') {
        await usersApi.update(user.id, { is_active: true })
        toast({ title: 'Usuario activado', description: `@${user.username} activado.` })
      } else if (type === 'deactivate') {
        await usersApi.update(user.id, { is_active: false })
        toast({ title: 'Usuario desactivado', description: `@${user.username} desactivado.` })
      }
      setConfirmAction(null)
      void load()
    } catch {
      toast({ title: 'Error al ejecutar la accion', variant: 'destructive' })
    } finally {
      setIsActionLoading(false)
    }
  }

  const columns: ColumnDef<User>[] = [
    {
      accessorKey: 'username',
      header: 'Usuario',
      cell: ({ row }) => (
        <span className="font-medium text-gray-900">@{row.original.username}</span>
      ),
    },
    {
      accessorKey: 'email',
      header: 'Email',
      cell: ({ row }) => <span className="text-gray-600">{row.original.email}</span>,
    },
    {
      accessorKey: 'is_active',
      header: 'Estado',
      cell: ({ row }) => <StatusBadge active={row.original.is_active} />,
    },
    {
      accessorKey: 'last_login_at',
      header: 'Ultimo login',
      cell: ({ row }) => (
        <span className="text-gray-500 text-xs">{formatDate(row.original.last_login_at)}</span>
      ),
    },
    {
      accessorKey: 'failed_attempts',
      header: 'Int. fallidos',
      cell: ({ row }) => {
        const attempts = row.original.failed_attempts
        const locked = row.original.locked_until
        return (
          <div className="flex items-center gap-2">
            <span className={attempts > 0 ? 'text-orange-600 font-semibold' : 'text-gray-400'}>
              {attempts}
            </span>
            {locked && (
              <Badge variant="destructive" className="text-xs">Bloqueado</Badge>
            )}
          </div>
        )
      },
    },
    {
      id: 'actions',
      header: 'Acciones',
      cell: ({ row }) => {
        const user = row.original
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              title="Ver detalle"
              onClick={() => navigate(`/users/${user.id}`)}
              aria-label={`Ver detalle de ${user.username}`}
            >
              <Eye className="h-4 w-4" />
            </Button>
            {user.locked_until && (
              <Button
                variant="ghost"
                size="icon"
                title="Desbloquear"
                onClick={() => setConfirmAction({ type: 'unlock', user })}
                aria-label={`Desbloquear ${user.username}`}
              >
                <Unlock className="h-4 w-4 text-orange-600" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              title="Resetear contrasena"
              onClick={() => setConfirmAction({ type: 'reset', user })}
              aria-label={`Resetear contrasena de ${user.username}`}
            >
              <RotateCcw className="h-4 w-4 text-blue-600" />
            </Button>
            {user.is_active ? (
              <Button
                variant="ghost"
                size="icon"
                title="Desactivar usuario"
                onClick={() => setConfirmAction({ type: 'deactivate', user })}
                aria-label={`Desactivar ${user.username}`}
              >
                <UserX className="h-4 w-4 text-red-500" />
              </Button>
            ) : (
              <Button
                variant="ghost"
                size="icon"
                title="Activar usuario"
                onClick={() => setConfirmAction({ type: 'activate', user })}
                aria-label={`Activar ${user.username}`}
              >
                <UserCheck className="h-4 w-4 text-green-600" />
              </Button>
            )}
          </div>
        )
      },
    },
  ]

  const confirmMessages = {
    unlock: { title: 'Desbloquear usuario', description: `Desbloquear la cuenta de @${confirmAction?.user.username}?` },
    reset: { title: 'Resetear contrasena', description: `Se generara una contrasena temporal para @${confirmAction?.user.username}. El usuario debera cambiarla al siguiente login.` },
    activate: { title: 'Activar usuario', description: `Activar la cuenta de @${confirmAction?.user.username}?` },
    deactivate: { title: 'Desactivar usuario', description: `Desactivar la cuenta de @${confirmAction?.user.username}? Se revocan todos sus tokens activos.` },
  }

  return (
    <div>
      <PageHeader
        title="Usuarios"
        description="Gestion de cuentas de usuario del sistema"
        actions={
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4" />
            Nuevo usuario
          </Button>
        }
      />

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" aria-hidden="true" />
          <Input
            placeholder="Buscar por usuario o email..."
            value={search}
            onChange={(e) => handleSearch(e.target.value)}
            className="pl-9"
            aria-label="Buscar usuarios"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-500">Estado:</span>
          {[
            { label: 'Todos', value: undefined },
            { label: 'Activos', value: true },
            { label: 'Inactivos', value: false },
          ].map(({ label, value }) => (
            <button
              key={label}
              onClick={() => { setFilterActive(value); reset() }}
              className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
                filterActive === value
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      <DataTable
        data={data.items}
        columns={columns}
        page={page}
        pageSize={pageSize}
        total={data.total}
        totalPages={data.totalPages}
        onPageChange={goToPage}
        isLoading={isLoading}
        emptyMessage="No se encontraron usuarios."
      />

      <UserFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={load}
      />

      {confirmAction && (
        <ConfirmDialog
          open={true}
          onOpenChange={(open) => { if (!open) setConfirmAction(null) }}
          title={confirmMessages[confirmAction.type].title}
          description={confirmMessages[confirmAction.type].description}
          confirmLabel="Confirmar"
          variant={confirmAction.type === 'deactivate' || confirmAction.type === 'reset' ? 'destructive' : 'default'}
          onConfirm={handleConfirmAction}
          isLoading={isActionLoading}
        />
      )}
    </div>
  )
}
