import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { Plus, Eye, Pencil, Trash2 } from 'lucide-react'
import { rolesApi } from '@/api/roles'
import type { Role } from '@/types'
import { DataTable } from '@/components/shared/DataTable'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { PageHeader } from '@/components/shared/PageHeader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { RoleFormDialog } from './RoleFormDialog'
import { toast } from '@/hooks/useToast'
import { usePagination } from '@/hooks/usePagination'
import { formatDate } from '@/lib/utils'

export function RolesPage() {
  const navigate = useNavigate()
  const { page, pageSize, goToPage } = usePagination()
  const [data, setData] = useState<{ items: Role[]; total: number; totalPages: number }>({
    items: [],
    total: 0,
    totalPages: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [editRole, setEditRole] = useState<Role | null>(null)
  const [deleteRole, setDeleteRole] = useState<Role | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const res = await rolesApi.list({ page, page_size: pageSize })
      setData({ items: res.data, total: res.total, totalPages: res.total_pages })
    } catch {
      toast({ title: 'Error al cargar roles', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    void load()
  }, [load])

  const handleDelete = async () => {
    if (!deleteRole) return
    setIsDeleting(true)
    try {
      await rolesApi.deactivate(deleteRole.id)
      toast({ title: 'Rol desactivado', description: `"${deleteRole.name}" desactivado.` })
      setDeleteRole(null)
      void load()
    } catch {
      toast({ title: 'Error al desactivar el rol', variant: 'destructive' })
    } finally {
      setIsDeleting(false)
    }
  }

  const columns: ColumnDef<Role>[] = [
    {
      accessorKey: 'name',
      header: 'Nombre',
      cell: ({ row }) => (
        <span className="font-medium text-gray-900">{row.original.name}</span>
      ),
    },
    {
      accessorKey: 'description',
      header: 'Descripcion',
      cell: ({ row }) => (
        <span className="text-gray-500 text-sm">{row.original.description ?? '-'}</span>
      ),
    },
    {
      accessorKey: 'is_system',
      header: 'Sistema',
      cell: ({ row }) =>
        row.original.is_system ? (
          <Badge variant="warning">Sistema</Badge>
        ) : (
          <Badge variant="secondary">Normal</Badge>
        ),
    },
    {
      accessorKey: 'is_active',
      header: 'Estado',
      cell: ({ row }) => <StatusBadge active={row.original.is_active} />,
    },
    {
      accessorKey: 'permissions_count',
      header: 'Permisos',
      cell: ({ row }) => (
        <span className="text-gray-700 font-mono text-sm">
          {row.original.permissions_count ?? 0}
        </span>
      ),
    },
    {
      accessorKey: 'created_at',
      header: 'Creado',
      cell: ({ row }) => (
        <span className="text-gray-400 text-xs">{formatDate(row.original.created_at)}</span>
      ),
    },
    {
      id: 'actions',
      header: 'Acciones',
      cell: ({ row }) => {
        const role = row.original
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              title="Ver detalle"
              onClick={() => navigate(`/roles/${role.id}`)}
              aria-label={`Ver detalle de ${role.name}`}
            >
              <Eye className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              title="Editar"
              onClick={() => setEditRole(role)}
              aria-label={`Editar ${role.name}`}
            >
              <Pencil className="h-4 w-4 text-blue-600" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              title={role.is_system ? 'No se puede desactivar un rol del sistema' : 'Desactivar'}
              disabled={role.is_system}
              onClick={() => setDeleteRole(role)}
              aria-label={`Desactivar ${role.name}`}
            >
              <Trash2 className={`h-4 w-4 ${role.is_system ? 'text-gray-300' : 'text-red-500'}`} />
            </Button>
          </div>
        )
      },
    },
  ]

  return (
    <div>
      <PageHeader
        title="Roles"
        description="Gestion de roles de acceso del sistema"
        actions={
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4" />
            Nuevo rol
          </Button>
        }
      />

      <DataTable
        data={data.items}
        columns={columns}
        page={page}
        pageSize={pageSize}
        total={data.total}
        totalPages={data.totalPages}
        onPageChange={goToPage}
        isLoading={isLoading}
        emptyMessage="No se encontraron roles."
      />

      <RoleFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={load}
      />

      <RoleFormDialog
        open={!!editRole}
        onOpenChange={(open) => { if (!open) setEditRole(null) }}
        onSuccess={load}
        role={editRole}
      />

      {deleteRole && (
        <ConfirmDialog
          open={true}
          onOpenChange={(open) => { if (!open) setDeleteRole(null) }}
          title="Desactivar rol"
          description={`Desactivar el rol "${deleteRole.name}"? Los usuarios con este rol perderan los accesos asociados.`}
          confirmLabel="Desactivar"
          variant="destructive"
          onConfirm={handleDelete}
          isLoading={isDeleting}
        />
      )}
    </div>
  )
}
