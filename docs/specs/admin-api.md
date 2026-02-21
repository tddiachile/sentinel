# Especificacion Tecnica: API de Administracion

**Referencia:** `docs/plan/auth-service-spec.md` seccion 4.4
**Historias relacionadas:** US-022 a US-030

---

## 1. Requisitos Generales

- Todos los endpoints requieren `Authorization: Bearer <access_token>` y `X-App-Key`
- Todos requieren el permiso `admin.system.manage`
- Todas las operaciones de escritura generan eventos de auditoria con `old_value` y `new_value`
- Las respuestas de listado deben soportar paginacion

### Formato de Paginacion

> **Decision de diseno (2026-02-21):** Se adopta paginacion por offset con formato estandar. Adecuada para el volumen esperado (~2.000 usuarios). `page_size` maximo: 100.

**Query params:**
| Param | Tipo | Default | Descripcion |
|---|---|---|---|
| `page` | int | 1 | Numero de pagina (base 1) |
| `page_size` | int | 20 | Registros por pagina (min 1, max 100) |

**Response wrapper:**
```json
{
  "data": [...],
  "page": 1,
  "page_size": 20,
  "total": 150,
  "total_pages": 8
}
```

**Reglas:**
- `page < 1` se normaliza a 1
- `page_size < 1` se normaliza a 20
- `page_size > 100` se normaliza a 100
- `total_pages = ceil(total / page_size)`
- Si `page > total_pages`, retorna `data: []` con los campos de paginacion correctos
- Todos los endpoints de listado (`GET /admin/users`, `GET /admin/roles`, `GET /admin/permissions`, `GET /admin/cost-centers`, `GET /admin/audit-logs`) usan este formato

---

## 2. Gestion de Usuarios

### GET /admin/users

**Descripcion:** Lista usuarios con paginacion.

**Query params adicionales:** `search` (busca en username y email), `is_active` (filtro booleano)

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "username": "jperez",
      "email": "jperez@sodexo.com",
      "is_active": true,
      "must_change_pwd": false,
      "last_login_at": "2026-02-18T10:30:00Z",
      "failed_attempts": 0,
      "locked_until": null,
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 50,
  "total_pages": 3
}
```

---

### POST /admin/users

**Descripcion:** Crea un nuevo usuario.

**Request:**
```json
{
  "username": "mgarcia",
  "email": "mgarcia@sodexo.com",
  "password": "T3mpP@ssw0rd!"
}
```

| Campo | Tipo | Requerido | Validacion |
|---|---|---|---|
| `username` | string | Si | 1-100 chars, unico |
| `email` | string | Si | Formato email, unico |
| `password` | string | Si | Cumplir politica de contrasena |

**Response 201:**
```json
{
  "id": "uuid",
  "username": "mgarcia",
  "email": "mgarcia@sodexo.com",
  "is_active": true,
  "must_change_pwd": true,
  "created_at": "2026-02-21T10:00:00Z"
}
```

**Nota:** `must_change_pwd` se establece automaticamente en `true` para nuevos usuarios.

**Evento:** `USER_CREATED`

---

### GET /admin/users/:id

**Response 200:**
```json
{
  "id": "uuid",
  "username": "jperez",
  "email": "jperez@sodexo.com",
  "is_active": true,
  "must_change_pwd": false,
  "last_login_at": "2026-02-18T10:30:00Z",
  "failed_attempts": 0,
  "locked_until": null,
  "roles": [
    {
      "id": "uuid",
      "name": "chef",
      "application": "hospitality-app",
      "valid_from": "2026-01-01T00:00:00Z",
      "valid_until": null,
      "is_active": true
    }
  ],
  "permissions": [
    {
      "id": "uuid",
      "code": "reports.special.export",
      "application": "hospitality-app",
      "valid_from": "2026-02-01T00:00:00Z",
      "valid_until": "2026-03-01T00:00:00Z"
    }
  ],
  "cost_centers": [
    {
      "id": "uuid",
      "code": "CC001",
      "name": "Casino Central",
      "application": "hospitality-app"
    }
  ],
  "created_at": "2026-01-15T08:00:00Z",
  "updated_at": "2026-02-18T10:30:00Z"
}
```

---

### PUT /admin/users/:id

**Request:**
```json
{
  "username": "jperez-updated",
  "email": "jperez.new@sodexo.com",
  "is_active": true
}
```

Todos los campos son opcionales. Solo se actualizan los campos enviados.

**Response 200:** Objeto usuario actualizado.

**Evento:** `USER_UPDATED` (con `old_value` y `new_value`)

Si `is_active` cambia de `true` a `false`: evento adicional `USER_DEACTIVATED`, revocar todos los refresh tokens.

---

### POST /admin/users/:id/roles

**Descripcion:** Asigna un rol al usuario.

**Request:**
```json
{
  "role_id": "uuid-del-rol",
  "valid_from": "2026-02-21T00:00:00Z",
  "valid_until": "2026-03-21T00:00:00Z"
}
```

| Campo | Tipo | Requerido | Descripcion |
|---|---|---|---|
| `role_id` | UUID | Si | ID del rol a asignar |
| `valid_from` | TIMESTAMPTZ | No | Default: `NOW()` |
| `valid_until` | TIMESTAMPTZ | No | `null` = sin expiracion |

**Response 201:**
```json
{
  "id": "uuid-de-asignacion",
  "user_id": "uuid",
  "role_id": "uuid",
  "role_name": "bodeguero-temporal",
  "valid_from": "2026-02-21T00:00:00Z",
  "valid_until": "2026-03-21T00:00:00Z",
  "granted_by": "uuid-del-admin"
}
```

**Evento:** `USER_ROLE_ASSIGNED`
**Cache:** Invalida contexto del usuario en Redis.

---

### DELETE /admin/users/:id/roles/:rid

**Descripcion:** Revoca un rol del usuario.

**Response 204:** Sin cuerpo.

Marca `user_roles.is_active = false` (no elimina fisicamente).

**Evento:** `USER_ROLE_REVOKED`
**Cache:** Invalida contexto del usuario en Redis.

---

### POST /admin/users/:id/permissions

**Descripcion:** Asigna permiso individual al usuario.

**Request:**
```json
{
  "permission_id": "uuid-del-permiso",
  "valid_from": "2026-02-21T00:00:00Z",
  "valid_until": "2026-03-21T00:00:00Z"
}
```

**Response 201:** Objeto de asignacion creado.

**Evento:** `USER_PERMISSION_ASSIGNED`

---

### DELETE /admin/users/:id/permissions/:pid

**Response 204:** Sin cuerpo.

**Evento:** `USER_PERMISSION_REVOKED`

---

### POST /admin/users/:id/cost-centers

**Descripcion:** Asigna CeCos al usuario.

**Request:**
```json
{
  "cost_center_ids": ["uuid-cc1", "uuid-cc2"],
  "valid_from": "2026-02-21T00:00:00Z",
  "valid_until": null
}
```

**Response 201:** Lista de asignaciones creadas.

**Evento:** `USER_COST_CENTER_ASSIGNED`

---

### POST /admin/users/:id/unlock

**Descripcion:** Desbloquea una cuenta bloqueada.

**Response 204:** Sin cuerpo.

**Logica:** Resetea `failed_attempts = 0` y `locked_until = NULL`.

**Evento:** `USER_UNLOCKED`

---

### POST /admin/users/:id/reset-password

**Descripcion:** Fuerza reset de contrasena del usuario.

**Response 200:**
```json
{
  "temporary_password": "Xt7$kL9mN2!p"
}
```

**Logica:**
1. Genera contrasena temporal aleatoria que cumple la politica
2. Hashea y almacena
3. Establece `must_change_pwd = true`
4. Revoca todos los refresh tokens del usuario

**Evento:** `AUTH_PASSWORD_RESET`

---

## 3. Gestion de Roles

### GET /admin/roles

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "chef",
      "description": "Rol de chef de cocina",
      "is_system": false,
      "is_active": true,
      "permissions_count": 5,
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "page": 1, "page_size": 20, "total": 50, "total_pages": 3
}
```

---

### POST /admin/roles

**Request:**
```json
{
  "name": "supervisor",
  "description": "Supervisor de planta"
}
```

**Response 201:** Objeto rol creado.

**Evento:** `ROLE_CREATED`

---

### GET /admin/roles/:id

**Response 200:**
```json
{
  "id": "uuid",
  "name": "chef",
  "description": "Rol de chef de cocina",
  "is_system": false,
  "is_active": true,
  "permissions": [
    { "id": "uuid", "code": "inventory.stock.read", "description": "Ver stock" },
    { "id": "uuid", "code": "inventory.stock.write", "description": "Modificar stock" }
  ],
  "users_count": 12,
  "created_at": "2026-01-15T08:00:00Z",
  "updated_at": "2026-02-18T10:00:00Z"
}
```

---

### PUT /admin/roles/:id

**Request:**
```json
{
  "name": "chef-senior",
  "description": "Chef senior con mas permisos"
}
```

**Response 200:** Objeto rol actualizado.

**Validacion:** No se puede modificar un rol con `is_system = true` (excepto descripcion).

**Evento:** `ROLE_UPDATED`

---

### DELETE /admin/roles/:id

**Descripcion:** Desactiva el rol (soft delete: `is_active = false`).

**Response 204:** Sin cuerpo.

**Validacion:** No se puede desactivar un rol con `is_system = true`.

**Evento:** `ROLE_DELETED`
**Cache:** Invalida mapa de permisos (cambia version).

---

### POST /admin/roles/:id/permissions

**Request:**
```json
{
  "permission_ids": ["uuid-perm-1", "uuid-perm-2"]
}
```

**Response 201:** Lista de asignaciones creadas.

**Evento:** `ROLE_PERMISSION_ASSIGNED`
**Cache:** Invalida mapa de permisos + contexto de usuarios con ese rol.

---

### DELETE /admin/roles/:id/permissions/:pid

**Response 204:** Sin cuerpo.

**Evento:** `ROLE_PERMISSION_REVOKED`
**Cache:** Invalida mapa de permisos + contexto de usuarios con ese rol.

---

## 4. Gestion de Permisos

### GET /admin/permissions

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "code": "inventory.stock.read",
      "description": "Ver stock",
      "scope_type": "action",
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "page": 1, "page_size": 20, "total": 50, "total_pages": 3
}
```

---

### POST /admin/permissions

**Request:**
```json
{
  "code": "inventory.stock.delete",
  "description": "Eliminar registros de stock",
  "scope_type": "action"
}
```

| Campo | Tipo | Requerido | Validacion |
|---|---|---|---|
| `code` | string | Si | Formato `modulo.recurso.accion`, UNIQUE por app |
| `description` | string | No | Texto libre |
| `scope_type` | string | Si | `global`, `module`, `resource`, `action` |

**Response 201:** Objeto permiso creado.

**Cache:** Invalida mapa de permisos.

---

### DELETE /admin/permissions/:id

**Response 204:** Sin cuerpo.

**Nota:** CASCADE elimina entradas en `role_permissions` y `user_permissions`.

**Cache:** Invalida mapa de permisos.

---

## 5. Gestion de Centros de Costo

### GET /admin/cost-centers

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "code": "CC001",
      "name": "Casino Central",
      "is_active": true,
      "created_at": "2026-01-15T08:00:00Z"
    }
  ],
  "page": 1, "page_size": 20, "total": 50, "total_pages": 3
}
```

---

### POST /admin/cost-centers

**Request:**
```json
{
  "code": "CC003",
  "name": "Comedor Norte"
}
```

**Response 201:** Objeto CeCo creado.

**Cache:** Invalida mapa de permisos.

---

### PUT /admin/cost-centers/:id

**Request:**
```json
{
  "name": "Casino Central - Renovado",
  "is_active": false
}
```

**Response 200:** Objeto CeCo actualizado.

**Cache:** Invalida mapa de permisos.

---

## 6. Consulta de Auditoria

### GET /admin/audit-logs

**Query params:**

| Param | Tipo | Descripcion |
|---|---|---|
| `user_id` | UUID | Filtrar por usuario afectado |
| `actor_id` | UUID | Filtrar por quien realizo la accion |
| `event_type` | string | Filtrar por tipo de evento |
| `from_date` | ISO 8601 | Fecha inicio |
| `to_date` | ISO 8601 | Fecha fin |
| `application_id` | UUID | Filtrar por aplicacion |
| `success` | boolean | Filtrar por resultado |
| `page` | int | Pagina |
| `page_size` | int | Tamano de pagina |

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "event_type": "AUTH_LOGIN_SUCCESS",
      "application_id": "uuid",
      "user_id": "uuid",
      "actor_id": "uuid",
      "resource_type": "user",
      "resource_id": "uuid",
      "old_value": null,
      "new_value": null,
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0...",
      "success": true,
      "error_message": null,
      "created_at": "2026-02-18T10:30:00Z"
    }
  ],
  "page": 1, "page_size": 20, "total": 50, "total_pages": 3
}
```

**Ordenamiento:** `created_at DESC` (siempre).

---

## 7. Guia para el Tester

### Tests Obligatorios

**Usuarios:**
1. Crear usuario con datos validos -> 201, `must_change_pwd = true`
2. Crear usuario con username duplicado -> 400/409
3. Crear usuario con email duplicado -> 400/409
4. Crear usuario con contrasena que no cumple politica -> 400
5. Actualizar usuario -> 200, evento `USER_UPDATED` con old/new values
6. Desactivar usuario -> revocar refresh tokens, evento `USER_DEACTIVATED`
7. Desbloquear usuario -> `failed_attempts = 0`, `locked_until = NULL`
8. Reset password -> genera temporal, `must_change_pwd = true`

**Roles:**
9. Crear rol -> 201
10. Crear rol con nombre duplicado en misma app -> 400/409
11. Desactivar rol `is_system = true` -> 403/400
12. Asignar permiso a rol -> invalida mapa de permisos
13. Revocar permiso de rol -> invalida mapa de permisos

**Permisos:**
14. Crear permiso con formato invalido de codigo -> 400
15. Crear permiso con `scope_type` invalido -> 400
16. Eliminar permiso -> CASCADE funciona en role_permissions

**Paginacion:**
17. Listado sin params de paginacion -> `page=1`, `page_size=20` por defecto
18. `page_size=200` -> normalizado a 100
19. `page=0` -> normalizado a 1
20. `page > total_pages` -> `data: []` con `total` y `total_pages` correctos
21. Verificar `total_pages = ceil(total / page_size)`

**Acceso:**
22. Request sin permiso `admin.system.manage` -> 403
23. Request sin token -> 401
24. Request con token expirado -> 401

---

*Fin de especificacion admin-api.md*
