# Especificacion Tecnica: Autorizacion

**Referencia:** `docs/plan/auth-service-spec.md` secciones 4.3, 5.2, 5.3, 5.4
**Historias relacionadas:** US-016, US-017, US-018, US-019, US-020, US-021

---

## 1. Modelo de Autorizacion

Sentinel implementa un modelo hibrido:

1. **RBAC (Role-Based Access Control):** Roles agrupan permisos. Los usuarios reciben roles.
2. **Permisos individuales (extra_permissions):** Permisos asignados directamente al usuario, fuera de sus roles.
3. **Centros de Costo (CeCos):** Filtro adicional de acceso a datos por CeCo.
4. **Vigencia temporal:** Tanto roles como permisos individuales pueden tener fecha de inicio y expiracion.

**Permiso efectivo de un usuario = (permisos de sus roles vigentes) UNION (permisos individuales vigentes)**

---

## 2. Endpoints de Autorizacion

### 2.1 POST /authz/verify

**Descripcion:** Verificacion delegada. Los backends que no puedan verificar localmente llaman a este endpoint.

**Headers:**
| Header | Requerido |
|---|---|
| `Authorization: Bearer <access_token>` | Si |
| `X-App-Key: <secret>` | Si |

**Request:**
```json
{
  "permission": "inventory.stock.write",
  "cost_center_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Campo | Tipo | Requerido | Descripcion |
|---|---|---|---|
| `permission` | string | Si | Codigo del permiso a verificar |
| `cost_center_id` | UUID | No | CeCo a verificar (si aplica) |

**Response 200:**
```json
{
  "allowed": true,
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "jperez",
  "permission": "inventory.stock.write",
  "evaluated_at": "2026-02-18T10:30:00Z"
}
```

**Response 200 (denegado):**
```json
{
  "allowed": false,
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "jperez",
  "permission": "inventory.stock.write",
  "evaluated_at": "2026-02-18T10:30:00Z"
}
```

**Nota:** Siempre retorna 200. El campo `allowed` indica el resultado. Solo retorna errores HTTP (401, 403) si el token o la app key son invalidos.

**Latencia objetivo:** < 50 ms (p95)

---

### 2.2 GET /authz/me/permissions

**Descripcion:** Retorna el contexto completo de permisos del usuario autenticado para la aplicacion actual.

**Headers:**
| Header | Requerido |
|---|---|
| `Authorization: Bearer <access_token>` | Si |

**Response 200:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "application": "hospitality-app",
  "roles": ["chef", "bodeguero-temporal"],
  "permissions": [
    "inventory.stock.read",
    "inventory.stock.write",
    "reports.monthly.export"
  ],
  "cost_centers": ["CC001", "CC002"],
  "temporary_roles": [
    {
      "role": "bodeguero-temporal",
      "valid_until": "2026-03-01T00:00:00Z"
    }
  ]
}
```

| Campo | Tipo | Descripcion |
|---|---|---|
| `user_id` | UUID | ID del usuario |
| `application` | string | Slug de la aplicacion |
| `roles` | string[] | Nombres de roles vigentes |
| `permissions` | string[] | Union de permisos de roles + permisos individuales vigentes |
| `cost_centers` | string[] | Codigos de CeCos asignados y vigentes |
| `temporary_roles` | object[] | Roles con fecha de expiracion |

**Logica de calculo:**
1. Obtener roles vigentes del usuario para la app: `user_roles WHERE is_active = TRUE AND valid_from <= NOW() AND (valid_until IS NULL OR valid_until > NOW())`
2. Para cada rol vigente, obtener sus permisos via `role_permissions`
3. Obtener permisos individuales vigentes: `user_permissions WHERE is_active = TRUE AND valid_from <= NOW() AND (valid_until IS NULL OR valid_until > NOW())`
4. `permissions` = union de (2) y (3), sin duplicados
5. Obtener CeCos vigentes: `user_cost_centers WHERE valid_from <= NOW() AND (valid_until IS NULL OR valid_until > NOW())`
6. `temporary_roles` = roles donde `valid_until IS NOT NULL`

**Cache:** Resultado cacheado en Redis por `jti` con TTL de 60 min.

---

### 2.3 GET /authz/permissions-map

**Descripcion:** Mapa global de permisos firmado digitalmente. Los backends lo cachean para resolver autorizaciones localmente.

**Headers:**
| Header | Requerido |
|---|---|
| `X-App-Key: <secret>` | Si |

**Response 200:**
```json
{
  "application": "hospitality-app",
  "generated_at": "2026-02-18T10:00:00Z",
  "version": "a3f8c21d",
  "permissions": {
    "inventory.stock.read":   { "roles": ["chef", "bodeguero", "admin"], "description": "Ver stock" },
    "inventory.stock.write":  { "roles": ["bodeguero", "admin"],         "description": "Modificar stock" },
    "reports.monthly.export": { "roles": ["supervisor", "admin"],        "description": "Exportar reporte mensual" }
  },
  "cost_centers": {
    "CC001": { "code": "CC001", "name": "Casino Central",    "is_active": true },
    "CC002": { "code": "CC002", "name": "Comedor Ejecutivo", "is_active": true }
  },
  "signature": "base64url(RSA-SHA256(payload))"
}
```

### Firma del Mapa

**Que se firma:** Objeto canonico con los campos `application`, `generated_at`, `version`, `permissions`, `cost_centers` serializados en JSON con:
- Claves ordenadas lexicograficamente
- Sin espacios en blanco

```
payload_to_sign = canonicalJSON({application, cost_centers, generated_at, permissions, version})
signature = base64url(RSA-SHA256(payload_to_sign, privateKey))
```

**Verificacion en backends:**
1. Reconstruir payload canonico (mismos campos, mismo orden, sin `signature`)
2. Verificar: `RSA-SHA256-Verify(payload_canonico, base64url_decode(signature), publicKey)`
3. Si falla: descartar mapa, conservar cache anterior, loguear error
4. Si cache anterior tambien invalido: denegar todos los accesos y alertar

### Version Hash

- `version` es un hash del estado actual del mapa
- Se invalida al modificar cualquier permiso, rol, role_permission o CeCo
- Los backends usan el endpoint de polling para detectar cambios sin descargar el mapa completo

---

### 2.4 GET /authz/permissions-map/version

**Descripcion:** Endpoint liviano de polling para detectar cambios en el mapa.

**Headers:**
| Header | Requerido |
|---|---|
| `X-App-Key: <secret>` | Si |

**Response 200:**
```json
{
  "application": "hospitality-app",
  "version": "a3f8c21d",
  "generated_at": "2026-02-18T10:00:00Z"
}
```

---

## 3. Algoritmo HasPermission

```
HasPermission(jwt, userContext, permissionCode, costCenterCode?):

1. Verificar firma y expiracion del JWT (clave publica RSA cacheada).
2. Obtener userContext desde cache por jwt.jti.
   Si no existe -> llamar GET /authz/me/permissions y cachear.
3. Si permissionCode esta en userContext.extra_permissions -> paso 5.
4. Buscar en el mapa global: rolesConPermiso = permissionsMap[permissionCode].roles
   Si ninguno de jwt.roles esta en rolesConPermiso -> DENEGADO.
5. Si se especifico costCenterCode:
   Si costCenterCode NO esta en userContext.cost_centers -> DENEGADO.
6. -> PERMITIDO.
```

### Tabla de Decision

| extra_permission? | Rol con permiso? | CeCo requerido? | CeCo del usuario? | Resultado |
|---|---|---|---|---|
| Si | - | No | - | PERMITIDO |
| Si | - | Si | Si | PERMITIDO |
| Si | - | Si | No | DENEGADO |
| No | Si | No | - | PERMITIDO |
| No | Si | Si | Si | PERMITIDO |
| No | Si | Si | No | DENEGADO |
| No | No | - | - | DENEGADO |

---

## 4. Estrategia de Cache en Backends Consumidores

| Cache | Clave | Fuente | TTL |
|---|---|---|---|
| Mapa global de permisos | `app_slug` | `GET /authz/permissions-map` | 5 min (con polling de version) |
| Contexto por usuario | `jti` del token | `GET /authz/me/permissions` | 60 min (vida del access token) |
| Clave publica RSA | `kid` | `GET /.well-known/jwks.json` | 60 min |

**Polling de version:** Cada 2 minutos, el backend llama a `GET /authz/permissions-map/version`. Si `version` difiere del valor cacheado, descarga el mapa completo y verifica la firma antes de reemplazar el cache.

**Almacenamiento:** En memoria del proceso (no requiere Redis en el backend consumidor).

---

## 5. Headers de Propagacion para Backends

Al validar el JWT, los backends deben propagar los siguientes headers hacia servicios internos:

```
X-User-Id:      <uuid>
X-Username:     <username>
X-App:          <app-slug>
X-Roles:        <rol1,rol2>
X-Cost-Centers: <CC001,CC002>
```

---

## 6. Flujo Completo de Verificacion Local (Recomendado)

```
Arranque del backend:
  1. Descargar clave publica RSA desde /.well-known/jwks.json   (cachear 60 min)
  2. Descargar mapa global desde /authz/permissions-map          (cachear 5 min)
  3. Iniciar polling liviano a /authz/permissions-map/version    (cada 2 min)

Por cada request (primera vez con ese token):
  4. Extraer Bearer token del header Authorization
  5. Verificar firma RS256 y expiracion del JWT con clave publica cacheada
  6. Buscar contexto de usuario en cache por jwt.jti
     -> Si no existe: llamar GET /authz/me/permissions y cachear con TTL = 60 min
  7. Evaluar HasPermission(claims, userContext, permMap, required, costCenter?)

Por cada request (token ya visto):
  4-5. Verificar JWT localmente
  6. userContext ya esta en cache -> verificacion 100% en memoria
```

---

## 7. Guia para el Tester

### Tests Unitarios Obligatorios

1. **HasPermission - permiso por rol:** Usuario con rol "chef", mapa dice "inventory.stock.read" requiere ["chef"] -> PERMITIDO
2. **HasPermission - sin rol:** Usuario con rol "chef", mapa dice "finance.ceco.write" requiere ["admin"] -> DENEGADO
3. **HasPermission - extra_permission:** Usuario sin rol pero con extra_permission "reports.special.export" -> PERMITIDO
4. **HasPermission - CeCo valido:** Permiso OK + CeCo "CC001" en usuario -> PERMITIDO
5. **HasPermission - CeCo invalido:** Permiso OK + CeCo "CC999" no en usuario -> DENEGADO
6. **HasPermission - CeCo no requerido:** Permiso OK, sin CeCo especificado -> PERMITIDO
7. **HasPermission - rol temporal vigente:** `valid_until = futuro` -> PERMITIDO
8. **HasPermission - rol temporal expirado:** `valid_until = pasado` -> DENEGADO (rol no en lista)
9. **HasPermission - permiso individual expirado:** `valid_until = pasado` -> DENEGADO
10. **HasPermission - permiso inexistente en mapa:** Permiso "xxx.yyy.zzz" no existe -> DENEGADO
11. **Verify endpoint:** Retorna `allowed: true` para usuario con permiso
12. **Verify endpoint:** Retorna `allowed: false` para usuario sin permiso
13. **Me/permissions:** Retorna union correcta de permisos de roles + individuales
14. **Me/permissions:** Excluye roles/permisos con `is_active = false`
15. **Me/permissions:** Excluye roles/permisos expirados
16. **Permissions-map:** Firma verificable con clave publica del JWKS
17. **Permissions-map:** Firma invalida si se modifica el payload
18. **Permissions-map/version:** Version cambia al modificar un rol o permiso
19. **canonicalJSON:** Serializacion determinista con claves ordenadas

### Casos de Borde

- Usuario con mismo permiso otorgado por rol Y como extra_permission: no duplicar en la lista
- Rol con `valid_from` en el futuro: no vigente aun
- Permiso asignado a rol que ya fue desactivado (`roles.is_active = false`): no cuenta
- Mapa de permisos para aplicacion sin permisos definidos: retorna `permissions: {}, cost_centers: {}`
- CeCo con `is_active = false` en el mapa: se incluye pero marcado como inactivo
- Polling de version concurrente desde multiples backends

---

*Fin de especificacion authorization.md*
