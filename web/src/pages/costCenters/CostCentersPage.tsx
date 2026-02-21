import { useEffect, useState, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import type { ColumnDef } from '@tanstack/react-table'
import { Plus, Pencil, ToggleLeft, ToggleRight } from 'lucide-react'
import { costCentersApi } from '@/api/costCenters'
import type { CostCenter, ApiError } from '@/types'
import { DataTable } from '@/components/shared/DataTable'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { PageHeader } from '@/components/shared/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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

const createCostCenterSchema = z.object({
  code: z.string().min(1, 'Requerido').max(50),
  name: z.string().min(1, 'Requerido').max(200),
})

const editCostCenterSchema = z.object({
  name: z.string().min(1, 'Requerido').max(200),
})

type CreateFormData = z.infer<typeof createCostCenterSchema>
type EditFormData = z.infer<typeof editCostCenterSchema>

export function CostCentersPage() {
  const { page, pageSize, goToPage } = usePagination()
  const [data, setData] = useState<{ items: CostCenter[]; total: number; totalPages: number }>({
    items: [],
    total: 0,
    totalPages: 0,
  })
  const [isLoading, setIsLoading] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const [editCC, setEditCC] = useState<CostCenter | null>(null)
  const [isTogglingId, setIsTogglingId] = useState<string | null>(null)

  const createForm = useForm<CreateFormData>({ resolver: zodResolver(createCostCenterSchema) })
  const editForm = useForm<EditFormData>({ resolver: zodResolver(editCostCenterSchema) })

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const res = await costCentersApi.list({ page, page_size: pageSize })
      setData({ items: res.data, total: res.total, totalPages: res.total_pages })
    } catch {
      toast({ title: 'Error al cargar centros de costo', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    void load()
  }, [load])

  const onCreateSubmit = async (formData: CreateFormData) => {
    try {
      await costCentersApi.create(formData)
      toast({ title: 'Centro de costo creado', description: `"${formData.code} - ${formData.name}" creado.` })
      createForm.reset()
      setShowCreate(false)
      void load()
    } catch (err) {
      const axiosErr = err as AxiosError<ApiError>
      const msg = axiosErr.response?.data?.error?.message ?? 'No se pudo crear el centro de costo.'
      toast({ title: 'Error', description: msg, variant: 'destructive' })
    }
  }

  const onEditSubmit = async (formData: EditFormData) => {
    if (!editCC) return
    try {
      await costCentersApi.update(editCC.id, { name: formData.name })
      toast({ title: 'Centro de costo actualizado' })
      editForm.reset()
      setEditCC(null)
      void load()
    } catch {
      toast({ title: 'Error al actualizar', variant: 'destructive' })
    }
  }

  const handleToggleActive = async (cc: CostCenter) => {
    setIsTogglingId(cc.id)
    try {
      await costCentersApi.update(cc.id, { is_active: !cc.is_active })
      toast({
        title: cc.is_active ? 'Centro de costo desactivado' : 'Centro de costo activado',
        description: `"${cc.code} - ${cc.name}"`,
      })
      void load()
    } catch {
      toast({ title: 'Error al cambiar el estado', variant: 'destructive' })
    } finally {
      setIsTogglingId(null)
    }
  }

  const openEdit = (cc: CostCenter) => {
    setEditCC(cc)
    editForm.reset({ name: cc.name })
  }

  const columns: ColumnDef<CostCenter>[] = [
    {
      accessorKey: 'code',
      header: 'Codigo',
      cell: ({ row }) => (
        <span className="font-mono font-semibold text-gray-900">{row.original.code}</span>
      ),
    },
    {
      accessorKey: 'name',
      header: 'Nombre',
      cell: ({ row }) => <span className="text-gray-700">{row.original.name}</span>,
    },
    {
      accessorKey: 'is_active',
      header: 'Estado',
      cell: ({ row }) => <StatusBadge active={row.original.is_active} />,
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
        const cc = row.original
        const isToggling = isTogglingId === cc.id
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              title="Editar"
              onClick={() => openEdit(cc)}
              aria-label={`Editar ${cc.name}`}
            >
              <Pencil className="h-4 w-4 text-blue-600" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              title={cc.is_active ? 'Desactivar' : 'Activar'}
              onClick={() => handleToggleActive(cc)}
              disabled={isToggling}
              aria-label={cc.is_active ? `Desactivar ${cc.name}` : `Activar ${cc.name}`}
            >
              {cc.is_active ? (
                <ToggleRight className="h-4 w-4 text-green-600" />
              ) : (
                <ToggleLeft className="h-4 w-4 text-gray-400" />
              )}
            </Button>
          </div>
        )
      },
    },
  ]

  return (
    <div>
      <PageHeader
        title="Centros de Costo"
        description="Gestion de centros de costo del sistema"
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4" />
            Nuevo CeCo
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
        emptyMessage="No se encontraron centros de costo."
      />

      {/* Create dialog */}
      <Dialog open={showCreate} onOpenChange={(open) => { if (!open) createForm.reset(); setShowCreate(open) }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Nuevo centro de costo</DialogTitle>
            <DialogDescription>Ingrese el codigo y nombre del nuevo centro de costo.</DialogDescription>
          </DialogHeader>
          <form onSubmit={createForm.handleSubmit(onCreateSubmit)} noValidate className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="cc-code">Codigo</Label>
              <Input
                id="cc-code"
                placeholder="CC001"
                className="font-mono"
                aria-invalid={!!createForm.formState.errors.code}
                {...createForm.register('code')}
              />
              {createForm.formState.errors.code && (
                <p className="text-xs text-red-600" role="alert">{createForm.formState.errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="cc-name">Nombre</Label>
              <Input
                id="cc-name"
                placeholder="Casino Central"
                aria-invalid={!!createForm.formState.errors.name}
                {...createForm.register('name')}
              />
              {createForm.formState.errors.name && (
                <p className="text-xs text-red-600" role="alert">{createForm.formState.errors.name.message}</p>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => { createForm.reset(); setShowCreate(false) }} disabled={createForm.formState.isSubmitting}>
                Cancelar
              </Button>
              <Button type="submit" disabled={createForm.formState.isSubmitting}>
                {createForm.formState.isSubmitting ? 'Creando...' : 'Crear centro de costo'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog open={!!editCC} onOpenChange={(open) => { if (!open) setEditCC(null) }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Editar centro de costo</DialogTitle>
            <DialogDescription>
              Codigo: <span className="font-mono font-semibold">{editCC?.code}</span>
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={editForm.handleSubmit(onEditSubmit)} noValidate className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="edit-cc-name">Nombre</Label>
              <Input
                id="edit-cc-name"
                aria-invalid={!!editForm.formState.errors.name}
                {...editForm.register('name')}
              />
              {editForm.formState.errors.name && (
                <p className="text-xs text-red-600" role="alert">{editForm.formState.errors.name.message}</p>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditCC(null)} disabled={editForm.formState.isSubmitting}>
                Cancelar
              </Button>
              <Button type="submit" disabled={editForm.formState.isSubmitting}>
                {editForm.formState.isSubmitting ? 'Guardando...' : 'Guardar cambios'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
