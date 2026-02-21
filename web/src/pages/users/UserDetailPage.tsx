import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, Key, Building2, Clock, Plus, Trash2 } from 'lucide-react'
import { usersApi } from '@/api/users'
import { rolesApi } from '@/api/roles'
import { permissionsApi } from '@/api/permissions'
import { costCentersApi } from '@/api/costCenters'
import type { User, Role, Permission, CostCenter } from '@/types'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { toast } from '@/hooks/useToast'
import { formatDate } from '@/lib/utils'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

type TabType = 'roles' | 'permissions' | 'cost_centers' | 'activity'

export function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('roles')

  // Available resources for assignment
  const [availableRoles, setAvailableRoles] = useState<Role[]>([])
  const [availablePermissions, setAvailablePermissions] = useState<Permission[]>([])
  const [availableCostCenters, setAvailableCostCenters] = useState<CostCenter[]>([])

  // Assignment form state
  const [selectedRoleId, setSelectedRoleId] = useState('')
  const [selectedPermissionId, setSelectedPermissionId] = useState('')
  const [selectedCostCenterIds, setSelectedCostCenterIds] = useState<string[]>([])
  const [validFrom, setValidFrom] = useState('')
  const [validUntil, setValidUntil] = useState('')
  const [isAssigning, setIsAssigning] = useState(false)

  // Confirm revoke
  const [confirmRevoke, setConfirmRevoke] = useState<{
    type: 'role' | 'permission'
    id: string
    label: string
  } | null>(null)
  const [isRevoking, setIsRevoking] = useState(false)

  const loadUser = async () => {
    if (!id) return
    setIsLoading(true)
    try {
      const u = await usersApi.get(id)
      setUser(u)
    } catch {
      toast({ title: 'Error al cargar el usuario', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void loadUser()
    void rolesApi.list({ page_size: 100 }).then((r) => setAvailableRoles(r.data))
    void permissionsApi.list({ page_size: 100 }).then((r) => setAvailablePermissions(r.data))
    void costCentersApi.list({ page_size: 100 }).then((r) =>
      setAvailableCostCenters(r.data.filter((cc) => cc.is_active))
    )
  }, [id])

  const handleAssignRole = async () => {
    if (!id || !selectedRoleId) return
    setIsAssigning(true)
    try {
      await usersApi.assignRole(id, {
        role_id: selectedRoleId,
        valid_from: validFrom || undefined,
        valid_until: validUntil || null,
      })
      toast({ title: 'Rol asignado exitosamente' })
      setSelectedRoleId('')
      setValidFrom('')
      setValidUntil('')
      void loadUser()
    } catch {
      toast({ title: 'Error al asignar rol', variant: 'destructive' })
    } finally {
      setIsAssigning(false)
    }
  }

  const handleAssignPermission = async () => {
    if (!id || !selectedPermissionId) return
    setIsAssigning(true)
    try {
      await usersApi.assignPermission(id, {
        permission_id: selectedPermissionId,
        valid_from: validFrom || undefined,
        valid_until: validUntil || null,
      })
      toast({ title: 'Permiso asignado exitosamente' })
      setSelectedPermissionId('')
      setValidFrom('')
      setValidUntil('')
      void loadUser()
    } catch {
      toast({ title: 'Error al asignar permiso', variant: 'destructive' })
    } finally {
      setIsAssigning(false)
    }
  }

  const handleAssignCostCenters = async () => {
    if (!id || selectedCostCenterIds.length === 0) return
    setIsAssigning(true)
    try {
      await usersApi.assignCostCenters(id, {
        cost_center_ids: selectedCostCenterIds,
        valid_from: validFrom || undefined,
        valid_until: validUntil || null,
      })
      toast({ title: 'Centros de costo asignados' })
      setSelectedCostCenterIds([])
      void loadUser()
    } catch {
      toast({ title: 'Error al asignar centros de costo', variant: 'destructive' })
    } finally {
      setIsAssigning(false)
    }
  }

  const handleRevoke = async () => {
    if (!id || !confirmRevoke) return
    setIsRevoking(true)
    try {
      if (confirmRevoke.type === 'role') {
        await usersApi.revokeRole(id, confirmRevoke.id)
        toast({ title: 'Rol revocado' })
      } else {
        await usersApi.revokePermission(id, confirmRevoke.id)
        toast({ title: 'Permiso revocado' })
      }
      setConfirmRevoke(null)
      void loadUser()
    } catch {
      toast({ title: 'Error al revocar', variant: 'destructive' })
    } finally {
      setIsRevoking(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24 text-gray-400">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-600 border-t-transparent mr-3" />
        Cargando usuario...
      </div>
    )
  }

  if (!user) {
    return (
      <div className="text-center py-24 text-gray-400">
        <p>Usuario no encontrado.</p>
        <Button variant="outline" onClick={() => navigate('/users')} className="mt-4">
          Volver a usuarios
        </Button>
      </div>
    )
  }

  const tabs: { key: TabType; label: string; icon: typeof Shield }[] = [
    { key: 'roles', label: 'Roles', icon: Shield },
    { key: 'permissions', label: 'Permisos individuales', icon: Key },
    { key: 'cost_centers', label: 'Centros de costo', icon: Building2 },
    { key: 'activity', label: 'Actividad', icon: Clock },
  ]

  return (
    <div className="space-y-6">
      {/* Back + Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate('/users')} aria-label="Volver">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-xl font-bold text-gray-900">@{user.username}</h1>
          <p className="text-sm text-gray-500">{user.email}</p>
        </div>
      </div>

      {/* Info card */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">Informacion general</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
          <div>
            <p className="text-xs text-gray-400 mb-1">Estado</p>
            <StatusBadge active={user.is_active} />
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Cambio de contrasena</p>
            <Badge variant={user.must_change_pwd ? 'warning' : 'secondary'}>
              {user.must_change_pwd ? 'Requerido' : 'No requerido'}
            </Badge>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Ultimo login</p>
            <p className="text-gray-700">{formatDate(user.last_login_at)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Intentos fallidos</p>
            <p className={user.failed_attempts > 0 ? 'text-orange-600 font-semibold' : 'text-gray-700'}>
              {user.failed_attempts}
              {user.locked_until && (
                <span className="ml-2 text-xs text-red-600">(bloqueado hasta {formatDate(user.locked_until)})</span>
              )}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Creado</p>
            <p className="text-gray-700">{formatDate(user.created_at)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">Actualizado</p>
            <p className="text-gray-700">{formatDate(user.updated_at)}</p>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="border-b border-gray-100 flex" role="tablist">
          {tabs.map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              role="tab"
              aria-selected={activeTab === key}
              onClick={() => setActiveTab(key)}
              className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === key
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <Icon className="h-3.5 w-3.5" aria-hidden="true" />
              {label}
            </button>
          ))}
        </div>

        <div className="p-5">
          {/* Roles tab */}
          {activeTab === 'roles' && (
            <div className="space-y-4">
              <div className="space-y-2">
                {user.roles && user.roles.length > 0 ? (
                  user.roles.map((role) => (
                    <div key={role.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                      <div>
                        <p className="text-sm font-medium text-gray-900">{role.name}</p>
                        <p className="text-xs text-gray-500">
                          Desde {formatDate(role.valid_from)}
                          {role.valid_until ? ` hasta ${formatDate(role.valid_until)}` : ' (sin expirar)'}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <StatusBadge active={role.is_active} />
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setConfirmRevoke({ type: 'role', id: role.id, label: role.name })}
                          aria-label={`Revocar rol ${role.name}`}
                        >
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-gray-400 text-center py-6">Sin roles asignados.</p>
                )}
              </div>

              {/* Assign role form */}
              <div className="border-t pt-4 space-y-3">
                <p className="text-sm font-medium text-gray-700 flex items-center gap-2">
                  <Plus className="h-4 w-4" /> Asignar rol
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div>
                    <Label htmlFor="select-role">Rol</Label>
                    <Select value={selectedRoleId} onValueChange={setSelectedRoleId}>
                      <SelectTrigger id="select-role">
                        <SelectValue placeholder="Seleccionar rol" />
                      </SelectTrigger>
                      <SelectContent>
                        {availableRoles.map((r) => (
                          <SelectItem key={r.id} value={r.id}>{r.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label htmlFor="role-valid-from">Desde (opcional)</Label>
                    <Input id="role-valid-from" type="datetime-local" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
                  </div>
                  <div>
                    <Label htmlFor="role-valid-until">Hasta (opcional)</Label>
                    <Input id="role-valid-until" type="datetime-local" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
                  </div>
                </div>
                <Button onClick={handleAssignRole} disabled={!selectedRoleId || isAssigning} size="sm">
                  {isAssigning ? 'Asignando...' : 'Asignar rol'}
                </Button>
              </div>
            </div>
          )}

          {/* Permissions tab */}
          {activeTab === 'permissions' && (
            <div className="space-y-4">
              <div className="space-y-2">
                {user.permissions && user.permissions.length > 0 ? (
                  user.permissions.map((perm) => (
                    <div key={perm.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                      <div>
                        <p className="text-sm font-mono font-medium text-gray-900">{perm.code}</p>
                        <p className="text-xs text-gray-500">
                          {perm.valid_until ? `Expira: ${formatDate(perm.valid_until)}` : 'Sin expirar'}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setConfirmRevoke({ type: 'permission', id: perm.id, label: perm.code })}
                        aria-label={`Revocar permiso ${perm.code}`}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-gray-400 text-center py-6">Sin permisos individuales asignados.</p>
                )}
              </div>

              {/* Assign permission form */}
              <div className="border-t pt-4 space-y-3">
                <p className="text-sm font-medium text-gray-700 flex items-center gap-2">
                  <Plus className="h-4 w-4" /> Asignar permiso
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div>
                    <Label htmlFor="select-permission">Permiso</Label>
                    <Select value={selectedPermissionId} onValueChange={setSelectedPermissionId}>
                      <SelectTrigger id="select-permission">
                        <SelectValue placeholder="Seleccionar permiso" />
                      </SelectTrigger>
                      <SelectContent>
                        {availablePermissions.map((p) => (
                          <SelectItem key={p.id} value={p.id}>{p.code}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label htmlFor="perm-valid-from">Desde (opcional)</Label>
                    <Input id="perm-valid-from" type="datetime-local" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
                  </div>
                  <div>
                    <Label htmlFor="perm-valid-until">Hasta (opcional)</Label>
                    <Input id="perm-valid-until" type="datetime-local" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
                  </div>
                </div>
                <Button onClick={handleAssignPermission} disabled={!selectedPermissionId || isAssigning} size="sm">
                  {isAssigning ? 'Asignando...' : 'Asignar permiso'}
                </Button>
              </div>
            </div>
          )}

          {/* Cost Centers tab */}
          {activeTab === 'cost_centers' && (
            <div className="space-y-4">
              <div className="space-y-2">
                {user.cost_centers && user.cost_centers.length > 0 ? (
                  user.cost_centers.map((cc) => (
                    <div key={cc.id} className="p-3 bg-gray-50 rounded-md">
                      <p className="text-sm font-medium text-gray-900">{cc.code} - {cc.name}</p>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-gray-400 text-center py-6">Sin centros de costo asignados.</p>
                )}
              </div>

              {/* Assign cost centers */}
              <div className="border-t pt-4 space-y-3">
                <p className="text-sm font-medium text-gray-700 flex items-center gap-2">
                  <Plus className="h-4 w-4" /> Asignar centros de costo
                </p>
                <div className="space-y-2">
                  {availableCostCenters.map((cc) => (
                    <label key={cc.id} className="flex items-center gap-2 text-sm cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedCostCenterIds.includes(cc.id)}
                        onChange={(e) => {
                          setSelectedCostCenterIds((prev) =>
                            e.target.checked ? [...prev, cc.id] : prev.filter((id) => id !== cc.id)
                          )
                        }}
                        className="rounded border-gray-300"
                      />
                      <span className="font-mono">{cc.code}</span>
                      <span className="text-gray-600">{cc.name}</span>
                    </label>
                  ))}
                </div>
                <Button
                  onClick={handleAssignCostCenters}
                  disabled={selectedCostCenterIds.length === 0 || isAssigning}
                  size="sm"
                >
                  {isAssigning ? 'Asignando...' : `Asignar ${selectedCostCenterIds.length} centro(s)`}
                </Button>
              </div>
            </div>
          )}

          {/* Activity tab */}
          {activeTab === 'activity' && (
            <div className="text-center py-8 text-gray-400 text-sm">
              <Clock className="h-8 w-8 mx-auto mb-2 text-gray-300" />
              Para ver el historial de actividad del usuario, utilice la seccion de{' '}
              <a href={`/audit?user_id=${user.id}`} className="text-blue-600 hover:underline">
                Auditoria
              </a>{' '}
              filtrando por este usuario.
            </div>
          )}
        </div>
      </div>

      {confirmRevoke && (
        <ConfirmDialog
          open={true}
          onOpenChange={(open) => { if (!open) setConfirmRevoke(null) }}
          title={`Revocar ${confirmRevoke.type === 'role' ? 'rol' : 'permiso'}`}
          description={`Revocar "${confirmRevoke.label}" del usuario @${user.username}?`}
          confirmLabel="Revocar"
          variant="destructive"
          onConfirm={handleRevoke}
          isLoading={isRevoking}
        />
      )}
    </div>
  )
}
