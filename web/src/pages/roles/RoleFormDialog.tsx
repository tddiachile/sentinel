import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { rolesApi } from '@/api/roles'
import { toast } from '@/hooks/useToast'
import type { Role, ApiError } from '@/types'
import type { AxiosError } from 'axios'

const roleSchema = z.object({
  name: z.string().min(1, 'El nombre es requerido').max(100),
  description: z.string().optional(),
})

type RoleFormData = z.infer<typeof roleSchema>

interface RoleFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
  role?: Role | null
}

export function RoleFormDialog({ open, onOpenChange, onSuccess, role }: RoleFormDialogProps) {
  const isEditing = !!role

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<RoleFormData>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      name: role?.name ?? '',
      description: role?.description ?? '',
    },
  })

  useEffect(() => {
    if (open) {
      reset({
        name: role?.name ?? '',
        description: role?.description ?? '',
      })
    }
  }, [open, role, reset])

  const onSubmit = async (data: RoleFormData) => {
    try {
      if (isEditing && role) {
        await rolesApi.update(role.id, data)
        toast({ title: 'Rol actualizado', description: `"${data.name}" actualizado.` })
      } else {
        await rolesApi.create(data)
        toast({ title: 'Rol creado', description: `"${data.name}" creado exitosamente.` })
      }
      reset()
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      const axiosErr = err as AxiosError<ApiError>
      const msg = axiosErr.response?.data?.error?.message ?? 'No se pudo guardar el rol.'
      toast({ title: 'Error', description: msg, variant: 'destructive' })
    }
  }

  const handleOpenChange = (open: boolean) => {
    if (!open) reset()
    onOpenChange(open)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Editar rol' : 'Nuevo rol'}</DialogTitle>
          <DialogDescription>
            {isEditing ? 'Modifique los datos del rol.' : 'Complete los datos para crear un nuevo rol de acceso.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="role-name">Nombre</Label>
            <Input
              id="role-name"
              placeholder="supervisor"
              aria-invalid={!!errors.name}
              disabled={isEditing && role?.is_system}
              {...register('name')}
            />
            {errors.name && (
              <p className="text-xs text-red-600" role="alert">{errors.name.message}</p>
            )}
            {isEditing && role?.is_system && (
              <p className="text-xs text-orange-600">Los roles del sistema no pueden cambiar nombre.</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="role-description">Descripcion (opcional)</Label>
            <Input
              id="role-description"
              placeholder="Descripcion del rol..."
              {...register('description')}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting}>
              Cancelar
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Guardando...' : isEditing ? 'Guardar cambios' : 'Crear rol'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
