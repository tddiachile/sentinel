import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import { rolesApi } from '@/api/roles'
import { permissionsApi } from '@/api/permissions'
import type { Role, Permission } from '@/types'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Label } from '@/components/ui/label'
import { toast } from '@/hooks/useToast'
import { formatDate } from '@/lib/utils'

export function RoleDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [role, setRole] = useState<Role | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [availablePermissions, setAvailablePermissions] = useState<Permission[]>([])
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<string[]>([])
  const [isAssigning, setIsAssigning] = useState(false)
  const [confirmRevoke, setConfirmRevoke] = useState<{ id: string; code: string } | null>(null)
  const [isRevoking, setIsRevoking] = useState(false)

  const loadRole = async () => {
    if (!id) return
    setIsLoading(true)
    try {
      const r = await rolesApi.get(id)
      setRole(r)
    } catch {
      toast({ title: 'Error al cargar el rol', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void loadRole()
    void permissionsApi.list({ page_size: 100 }).then((r) => setAvailablePermissions(r.data))
  }, [id])

  const handleAssignPermissions = async () => {
    if (!id || selectedPermissionIds.length === 0) return
    setIsAssigning(true)
    try {
      await rolesApi.assignPermissions(id, { permission_ids: selectedPermissionIds })
      toast({ title: 'Permisos asignados', description: `${selectedPermissionIds.length} permiso(s) asignados al rol.` })
      setSelectedPermissionIds([])
      void loadRole()
    } catch {
      toast({ title: 'Error al asignar permisos', variant: 'destructive' })
    } finally {
      setIsAssigning(false)
    }
  }

  const handleRevokePermission = async () => {
    if (!id || !confirmRevoke) return
    setIsRevoking(true)
    try {
      await rolesApi.revokePermission(id, confirmRevoke.id)
      toast({ title: 'Permiso removido del rol' })
      setConfirmRevoke(null)
      void loadRole()
    } catch {
      toast({ title: 'Error al remover permiso', variant: 'destructive' })
    } finally {
      setIsRevoking(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24 text-gray-400">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-600 border-t-transparent mr-3" />
        Cargando rol...
      </div>
    )
  }

  if (!role) {
    return (
      <div className="text-center py-24 text-gray-400">
        <p>Rol no encontrado.</p>
        <Button variant="outline" onClick={() => navigate('/roles')} className="mt-4">
          Volver a roles
        </Button>
      </div>
    )
  }

  // Filter out already assigned permissions
  const assignedIds = new Set(role.permissions?.map((p) => p.id) ?? [])
  const unassignedPermissions = availablePermissions.filter((p) => !assignedIds.has(p.id))

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/roles')} aria-label="Volver">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-xl font-bold text-gray-900">{role.name}</h1>
          <p className="text-sm text-gray-500">{role.description ?? 'Sin descripcion'}</p>
        </div>
      </div>

      {/* Info */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
          <div>
            <p className="text-xs text-gray-400 mb-1">Estado</p>
            <StatusBadge active={role.is_active} />
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Tipo</p>
            <Badge variant={role.is_system ? 'warning' : 'secondary'}>
              {role.is_system ? 'Sistema' : 'Normal'}
            </Badge>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Usuarios asignados</p>
            <p className="font-semibold text-gray-900">{role.users_count ?? 0}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Creado</p>
            <p className="text-gray-700">{formatDate(role.created_at)}</p>
          </div>
        </div>
      </div>

      {/* Permissions */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100">
          <h2 className="font-semibold text-gray-900">Permisos asignados ({role.permissions?.length ?? 0})</h2>
        </div>
        <div className="p-5 space-y-4">
          {/* Assigned permissions list */}
          <div className="space-y-2">
            {role.permissions && role.permissions.length > 0 ? (
              role.permissions.map((perm) => (
                <div key={perm.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                  <div>
                    <p className="text-sm font-mono font-medium text-gray-900">{perm.code}</p>
                    {perm.description && (
                      <p className="text-xs text-gray-500">{perm.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-xs">{perm.scope_type}</Badge>
                    {!role.is_system && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setConfirmRevoke({ id: perm.id, code: perm.code })}
                        aria-label={`Remover permiso ${perm.code}`}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <p className="text-sm text-gray-400 text-center py-6">Sin permisos asignados.</p>
            )}
          </div>

          {/* Assign permissions */}
          {!role.is_system && (
            <div className="border-t pt-4 space-y-3">
              <p className="text-sm font-medium text-gray-700 flex items-center gap-2">
                <Plus className="h-4 w-4" /> Agregar permisos
              </p>
              <div className="space-y-2 max-h-48 overflow-y-auto">
                {unassignedPermissions.length === 0 ? (
                  <p className="text-xs text-gray-400">Todos los permisos ya estan asignados.</p>
                ) : (
                  unassignedPermissions.map((p) => (
                    <label key={p.id} className="flex items-center gap-2 text-sm cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedPermissionIds.includes(p.id)}
                        onChange={(e) => {
                          setSelectedPermissionIds((prev) =>
                            e.target.checked ? [...prev, p.id] : prev.filter((id) => id !== p.id)
                          )
                        }}
                        className="rounded border-gray-300"
                      />
                      <span className="font-mono">{p.code}</span>
                      <Badge variant="secondary" className="text-xs ml-auto">{p.scope_type}</Badge>
                    </label>
                  ))
                )}
              </div>
              <div className="flex items-center gap-2">
                <Label className="sr-only">Seleccion de permisos</Label>
                <Button
                  onClick={handleAssignPermissions}
                  disabled={selectedPermissionIds.length === 0 || isAssigning}
                  size="sm"
                >
                  {isAssigning ? 'Asignando...' : `Agregar ${selectedPermissionIds.length} permiso(s)`}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      {confirmRevoke && (
        <ConfirmDialog
          open={true}
          onOpenChange={(open) => { if (!open) setConfirmRevoke(null) }}
          title="Remover permiso del rol"
          description={`Remover el permiso "${confirmRevoke.code}" del rol "${role.name}"?`}
          confirmLabel="Remover"
          variant="destructive"
          onConfirm={handleRevokePermission}
          isLoading={isRevoking}
        />
      )}
    </div>
  )
}
