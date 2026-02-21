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
import { usersApi } from '@/api/users'
import { toast } from '@/hooks/useToast'
import type { AxiosError } from 'axios'
import type { ApiError } from '@/types'

const createUserSchema = z.object({
  username: z.string().min(1, 'Requerido').max(100, 'Max 100 caracteres'),
  email: z.string().email('Email invalido').min(1, 'Requerido'),
  password: z
    .string()
    .min(10, 'Minimo 10 caracteres')
    .regex(/[A-Z]/, 'Requiere al menos una mayuscula')
    .regex(/[0-9]/, 'Requiere al menos un numero')
    .regex(/[^a-zA-Z0-9]/, 'Requiere al menos un simbolo'),
})

type CreateUserFormData = z.infer<typeof createUserSchema>

interface UserFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function UserFormDialog({ open, onOpenChange, onSuccess }: UserFormDialogProps) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateUserFormData>({
    resolver: zodResolver(createUserSchema),
  })

  const onSubmit = async (data: CreateUserFormData) => {
    try {
      await usersApi.create(data)
      toast({ title: 'Usuario creado', description: `@${data.username} creado exitosamente.`, variant: 'default' })
      reset()
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      const axiosErr = err as AxiosError<ApiError>
      const msg = axiosErr.response?.data?.error?.message ?? 'No se pudo crear el usuario.'
      toast({ title: 'Error al crear usuario', description: msg, variant: 'destructive' })
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
          <DialogTitle>Nuevo usuario</DialogTitle>
          <DialogDescription>
            Complete los datos para crear un nuevo usuario. La contrasena debe cumplir la politica de seguridad.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="new-username">Usuario</Label>
            <Input
              id="new-username"
              placeholder="jperez"
              aria-invalid={!!errors.username}
              {...register('username')}
            />
            {errors.username && (
              <p className="text-xs text-red-600" role="alert">{errors.username.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="new-email">Email</Label>
            <Input
              id="new-email"
              type="email"
              placeholder="jperez@empresa.com"
              aria-invalid={!!errors.email}
              {...register('email')}
            />
            {errors.email && (
              <p className="text-xs text-red-600" role="alert">{errors.email.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="new-password">Contrasena inicial</Label>
            <Input
              id="new-password"
              type="password"
              placeholder="••••••••••"
              aria-invalid={!!errors.password}
              {...register('password')}
            />
            {errors.password && (
              <p className="text-xs text-red-600" role="alert">{errors.password.message}</p>
            )}
            <p className="text-xs text-gray-400">
              Min. 10 caracteres, 1 mayuscula, 1 numero, 1 simbolo. El usuario debera cambiarla al ingresar.
            </p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting}>
              Cancelar
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Creando...' : 'Crear usuario'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
