import { useEffect, useState, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import type { ColumnDef } from '@tanstack/react-table'
import { Plus, Trash2 } from 'lucide-react'
import { permissionsApi } from '@/api/permissions'
import type { Permission, ApiError } from '@/types'
import { DataTable } from '@/components/shared/DataTable'
import { PageHeader } from '@/components/shared/PageHeader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { toast } from '@/hooks/useToast'
import { usePagination } from '@/hooks/usePagination'
import { formatDate } from '@/lib/utils'
import type { AxiosError } from 'axios'

const permissionSchema = z.object({
  code: z
    .string()
    .min(1, 'Requerido')
    .regex(
      /^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$/,
      'Formato requerido: modulo.recurso.accion (ej: inventory.stock.read)'
    ),
  description: z.string().optional(),
  scope_type: z.enum(['global', 'module', 'resource', 'action'], {
    error: 'Seleccione un scope',
  }),
})

type PermissionFormData = z.infer<typeof permissionSchema>

const scopeBadgeVariant = (scope: string) => {
  const map: Record<string, 'default' | 'secondary' | 'success' | 'warning'> = {
    global: 'default',
    module: 'secondary',
    resource: 'warning',
    action: 'success',
  }
  return map[scope] ?? 'secondary'
}

export function PermissionsPage() {
  const { page, pageSize, goToPage } = usePagination()
  const [data, setData] = useState<{ items: Permission[]; total: number; totalPages: number }>({
    items: [],
    total: 0,
    totalPages: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const [deletePermission, setDeletePermission] = useState<Permission | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<PermissionFormData>({ resolver: zodResolver(permissionSchema) })

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const res = await permissionsApi.list({ page, page_size: pageSize })
      setData({ items: res.data, total: res.total, totalPages: res.total_pages })
    } catch {
      toast({ title: 'Error al cargar permisos', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    void load()
  }, [load])

  const onSubmit = async (data: PermissionFormData) => {
    try {
      await permissionsApi.create(data)
      toast({ title: 'Permiso creado', description: `"${data.code}" creado exitosamente.` })
      reset()
      setShowCreate(false)
      void load()
    } catch (err) {
      const axiosErr = err as AxiosError<ApiError>
      const msg = axiosErr.response?.data?.error?.message ?? 'No se pudo crear el permiso.'
      toast({ title: 'Error', description: msg, variant: 'destructive' })
    }
  }

  const handleDelete = async () => {
    if (!deletePermission) return
    setIsDeleting(true)
    try {
      await permissionsApi.delete(deletePermission.id)
      toast({ title: 'Permiso eliminado', description: `"${deletePermission.code}" eliminado.` })
      setDeletePermission(null)
      void load()
    } catch {
      toast({ title: 'Error al eliminar el permiso', variant: 'destructive' })
    } finally {
      setIsDeleting(false)
    }
  }

  const columns: ColumnDef<Permission>[] = [
    {
      accessorKey: 'code',
      header: 'Codigo',
      cell: ({ row }) => (
        <span className="font-mono text-sm font-medium text-gray-900">{row.original.code}</span>
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
      accessorKey: 'scope_type',
      header: 'Scope',
      cell: ({ row }) => (
        <Badge variant={scopeBadgeVariant(row.original.scope_type)} className="capitalize">
          {row.original.scope_type}
        </Badge>
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
      cell: ({ row }) => (
        <Button
          variant="ghost"
          size="icon"
          title="Eliminar permiso"
          onClick={() => setDeletePermission(row.original)}
          aria-label={`Eliminar permiso ${row.original.code}`}
        >
          <Trash2 className="h-4 w-4 text-red-500" />
        </Button>
      ),
    },
  ]

  return (
    <div>
      <PageHeader
        title="Permisos"
        description="Gestion de permisos del sistema"
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4" />
            Nuevo permiso
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
        emptyMessage="No se encontraron permisos."
      />

      {/* Create dialog */}
      <Dialog open={showCreate} onOpenChange={(open) => { if (!open) reset(); setShowCreate(open) }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Nuevo permiso</DialogTitle>
            <DialogDescription>
              El codigo debe seguir el formato <code className="bg-gray-100 px-1 rounded text-xs">modulo.recurso.accion</code>.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="perm-code">Codigo</Label>
              <Input
                id="perm-code"
                placeholder="inventory.stock.read"
                className="font-mono"
                aria-invalid={!!errors.code}
                {...register('code')}
              />
              {errors.code && (
                <p className="text-xs text-red-600" role="alert">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="perm-description">Descripcion (opcional)</Label>
              <Input id="perm-description" placeholder="Ver registros de stock..." {...register('description')} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="perm-scope">Scope</Label>
              <Select onValueChange={(val) => setValue('scope_type', val as PermissionFormData['scope_type'])}>
                <SelectTrigger id="perm-scope">
                  <SelectValue placeholder="Seleccionar scope" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="global">global</SelectItem>
                  <SelectItem value="module">module</SelectItem>
                  <SelectItem value="resource">resource</SelectItem>
                  <SelectItem value="action">action</SelectItem>
                </SelectContent>
              </Select>
              {errors.scope_type && (
                <p className="text-xs text-red-600" role="alert">{errors.scope_type.message}</p>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => { reset(); setShowCreate(false) }} disabled={isSubmitting}>
                Cancelar
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Creando...' : 'Crear permiso'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {deletePermission && (
        <ConfirmDialog
          open={true}
          onOpenChange={(open) => { if (!open) setDeletePermission(null) }}
          title="Eliminar permiso"
          description={`Eliminar el permiso "${deletePermission.code}"? Esta accion es irreversible y afectara todos los roles y usuarios que lo tengan asignado.`}
          confirmLabel="Eliminar"
          variant="destructive"
          onConfirm={handleDelete}
          isLoading={isDeleting}
        />
      )}
    </div>
  )
}
