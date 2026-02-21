import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ShieldCheck, Eye, EyeOff } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useAuthStore } from '@/store/authStore'
import { getErrorMessage } from '@/lib/utils'

const loginSchema = z.object({
  username: z.string().min(1, 'El usuario es requerido').max(100),
  password: z.string().min(1, 'La contrasena es requerida'),
  client_type: z.enum(['web', 'mobile', 'desktop'], {
    error: 'Seleccione el tipo de cliente',
  }),
})

type LoginFormData = z.infer<typeof loginSchema>

export function LoginPage() {
  const navigate = useNavigate()
  const { login, isAuthenticated, isLoading } = useAuthStore()
  const [showPassword, setShowPassword] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: { client_type: 'web' },
  })

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true })
    }
  }, [isAuthenticated, navigate])

  const onSubmit = async (data: LoginFormData) => {
    setErrorMessage(null)
    try {
      await login(data)
      const user = useAuthStore.getState().user
      if (user?.must_change_password) {
        navigate('/change-password', { replace: true })
      } else {
        navigate('/dashboard', { replace: true })
      }
    } catch (err) {
      const code = err instanceof Error ? err.message : 'INTERNAL_ERROR'
      setErrorMessage(getErrorMessage(code))
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center h-16 w-16 rounded-full bg-blue-600 mb-4">
            <ShieldCheck className="h-9 w-9 text-white" aria-hidden="true" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Sentinel</h1>
          <p className="text-sm text-gray-500 mt-1">Panel de Administracion</p>
        </div>

        {/* Card */}
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-6">Iniciar sesion</h2>

          <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
            {/* Error global */}
            {errorMessage && (
              <div
                className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-800"
                role="alert"
              >
                {errorMessage}
              </div>
            )}

            {/* Username */}
            <div className="space-y-1.5">
              <Label htmlFor="username">Usuario</Label>
              <Input
                id="username"
                type="text"
                autoComplete="username"
                placeholder="jperez"
                aria-invalid={!!errors.username}
                aria-describedby={errors.username ? 'username-error' : undefined}
                {...register('username')}
              />
              {errors.username && (
                <p id="username-error" className="text-xs text-red-600" role="alert">
                  {errors.username.message}
                </p>
              )}
            </div>

            {/* Password */}
            <div className="space-y-1.5">
              <Label htmlFor="password">Contrasena</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  placeholder="••••••••••"
                  aria-invalid={!!errors.password}
                  aria-describedby={errors.password ? 'password-error' : undefined}
                  {...register('password')}
                />
                <button
                  type="button"
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? 'Ocultar contrasena' : 'Mostrar contrasena'}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && (
                <p id="password-error" className="text-xs text-red-600" role="alert">
                  {errors.password.message}
                </p>
              )}
            </div>

            {/* Client Type */}
            <div className="space-y-1.5">
              <Label htmlFor="client_type">Tipo de cliente</Label>
              <Select
                defaultValue="web"
                onValueChange={(val) =>
                  setValue('client_type', val as 'web' | 'mobile' | 'desktop')
                }
              >
                <SelectTrigger id="client_type" aria-label="Tipo de cliente">
                  <SelectValue placeholder="Seleccionar tipo" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="web">Web</SelectItem>
                  <SelectItem value="mobile">Mobile</SelectItem>
                  <SelectItem value="desktop">Desktop</SelectItem>
                </SelectContent>
              </Select>
              {errors.client_type && (
                <p className="text-xs text-red-600" role="alert">
                  {errors.client_type.message}
                </p>
              )}
            </div>

            <Button type="submit" className="w-full mt-6" disabled={isLoading}>
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Iniciando sesion...
                </span>
              ) : (
                'Iniciar sesion'
              )}
            </Button>
          </form>
        </div>
      </div>
    </div>
  )
}
