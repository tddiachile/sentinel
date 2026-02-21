# Especificacion Tecnica: Sistema de Auditoria

**Referencia:** `docs/plan/auth-service-spec.md` seccion 6
**Historias relacionadas:** US-031, US-032, US-033

---

## 1. Principios

1. **Inmutabilidad:** Los registros de auditoria nunca se modifican ni eliminan. Solo INSERT y SELECT.
2. **Completitud:** Toda accion critica genera un evento de auditoria.
3. **Trazabilidad:** Cada evento registra quien (actor), sobre quien/que (recurso), cuando y el resultado.
4. **No-bloqueo:** El registro de auditoria no debe bloquear la respuesta al cliente.

---

## 2. Tipos de Eventos

### 2.1 Autenticacion

| Codigo | Descripcion | Trigger |
|---|---|---|
| `AUTH_LOGIN_SUCCESS` | Login exitoso | `POST /auth/login` (exito) |
| `AUTH_LOGIN_FAILED` | Login fallido | `POST /auth/login` (fallo) |
| `AUTH_LOGOUT` | Cierre de sesion | `POST /auth/logout` |
| `AUTH_TOKEN_REFRESHED` | Token renovado | `POST /auth/refresh` (exito) |
| `AUTH_PASSWORD_CHANGED` | Contrasena cambiada por usuario | `POST /auth/change-password` |
| `AUTH_PASSWORD_RESET` | Contrasena reseteada por admin | `POST /admin/users/:id/reset-password` |
| `AUTH_ACCOUNT_LOCKED` | Cuenta bloqueada | Tras 5 intentos fallidos |

### 2.2 Autorizacion

| Codigo | Descripcion | Trigger |
|---|---|---|
| `AUTHZ_PERMISSION_GRANTED` | Permiso concedido | `POST /authz/verify` (allowed=true) |
| `AUTHZ_PERMISSION_DENIED` | Permiso denegado | `POST /authz/verify` (allowed=false) |

### 2.3 Gestion de Usuarios

| Codigo | Descripcion | Trigger |
|---|---|---|
| `USER_CREATED` | Usuario creado | `POST /admin/users` |
| `USER_UPDATED` | Usuario actualizado | `PUT /admin/users/:id` |
| `USER_DEACTIVATED` | Usuario desactivado | `PUT /admin/users/:id` (is_active=false) |
| `USER_UNLOCKED` | Cuenta desbloqueada | `POST /admin/users/:id/unlock` |

### 2.4 Gestion de Roles

| Codigo | Descripcion | Trigger |
|---|---|---|
| `ROLE_CREATED` | Rol creado | `POST /admin/roles` |
| `ROLE_UPDATED` | Rol actualizado | `PUT /admin/roles/:id` |
| `ROLE_DELETED` | Rol desactivado | `DELETE /admin/roles/:id` |
| `ROLE_PERMISSION_ASSIGNED` | Permiso asignado a rol | `POST /admin/roles/:id/permissions` |
| `ROLE_PERMISSION_REVOKED` | Permiso removido de rol | `DELETE /admin/roles/:id/permissions/:pid` |

### 2.5 Asignaciones

| Codigo | Descripcion | Trigger |
|---|---|---|
| `USER_ROLE_ASSIGNED` | Rol asignado a usuario | `POST /admin/users/:id/roles` |
| `USER_ROLE_REVOKED` | Rol revocado de usuario | `DELETE /admin/users/:id/roles/:rid` |
| `USER_PERMISSION_ASSIGNED` | Permiso individual asignado | `POST /admin/users/:id/permissions` |
| `USER_PERMISSION_REVOKED` | Permiso individual revocado | `DELETE /admin/users/:id/permissions/:pid` |
| `USER_COST_CENTER_ASSIGNED` | CeCo asignado a usuario | `POST /admin/users/:id/cost-centers` |

### 2.6 Sistema

| Codigo | Descripcion | Trigger |
|---|---|---|
| `SYSTEM_BOOTSTRAP` | Bootstrap inicial | Primera inicializacion |

---

## 3. Esquema del Registro

Cada entrada en `audit_logs` contiene:

| Campo | Tipo | Obligatorio | Descripcion |
|---|---|---|---|
| `id` | UUID | Si | PK generado automaticamente |
| `event_type` | VARCHAR(50) | Si | Codigo del evento (ver seccion 2) |
| `application_id` | UUID | No | FK a applications (NULL en eventos de sistema) |
| `user_id` | UUID | No | Usuario afectado por la accion |
| `actor_id` | UUID | No | Quien realizo la accion |
| `resource_type` | VARCHAR(50) | No | Tipo de recurso: `user`, `role`, `permission`, `cost_center`, `refresh_token` |
| `resource_id` | UUID | No | ID del recurso afectado |
| `old_value` | JSONB | No | Estado anterior (para operaciones de update) |
| `new_value` | JSONB | No | Estado nuevo (para operaciones de create/update) |
| `ip_address` | INET | No | IP del cliente |
| `user_agent` | TEXT | No | User-Agent del cliente |
| `success` | BOOLEAN | Si | `true` si la operacion fue exitosa |
| `error_message` | TEXT | No | Mensaje de error si `success = false` |
| `created_at` | TIMESTAMPTZ | Si | Timestamp en UTC |

### Ejemplo de Registro: Login Exitoso

```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "event_type": "AUTH_LOGIN_SUCCESS",
  "application_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "actor_id": "660e8400-e29b-41d4-a716-446655440001",
  "resource_type": "user",
  "resource_id": "660e8400-e29b-41d4-a716-446655440001",
  "old_value": null,
  "new_value": null,
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
  "success": true,
  "error_message": null,
  "created_at": "2026-02-18T10:30:00Z"
}
```

### Ejemplo de Registro: Actualizacion de Usuario

```json
{
  "id": "...",
  "event_type": "USER_UPDATED",
  "application_id": "...",
  "user_id": "...",
  "actor_id": "...(admin que hizo el cambio)",
  "resource_type": "user",
  "resource_id": "...(mismo que user_id)",
  "old_value": { "email": "old@sodexo.com", "is_active": true },
  "new_value": { "email": "new@sodexo.com", "is_active": true },
  "ip_address": "10.0.0.50",
  "user_agent": "SentinelAdmin/1.0",
  "success": true,
  "error_message": null,
  "created_at": "2026-02-18T11:00:00Z"
}
```

### Ejemplo de Registro: Login Fallido

```json
{
  "id": "...",
  "event_type": "AUTH_LOGIN_FAILED",
  "application_id": "...",
  "user_id": "...(usuario que intento loguearse)",
  "actor_id": null,
  "resource_type": "user",
  "resource_id": "...",
  "old_value": null,
  "new_value": { "failed_attempts": 3 },
  "ip_address": "203.0.113.50",
  "user_agent": "Mozilla/5.0...",
  "success": false,
  "error_message": "Invalid credentials",
  "created_at": "2026-02-18T10:31:00Z"
}
```

---

## 4. Middleware de Auditoria

### Responsabilidades

1. Capturar `ip_address` desde `X-Forwarded-For` (o IP remota como fallback)
2. Capturar `user_agent` desde header `User-Agent`
3. Capturar `actor_id` desde el JWT (si esta autenticado)
4. Capturar `application_id` desde el contexto del request (inyectado por app_key middleware)

### Implementacion

- El middleware **no reemplaza** el registro explicito de eventos en los servicios
- Su rol es capturar y propagar la informacion contextual (IP, user-agent, actor)
- Los servicios llaman al `AuditService.LogEvent(...)` con los datos especificos del evento

### No-Bloqueo

El registro de auditoria debe ejecutarse de forma asincrona:
- Opcion A: Goroutine con canal buffered
- Opcion B: Insert asincrono con retry

El fallo en el registro de auditoria **no debe** causar fallo en la operacion principal del usuario. Se debe loguear el error de auditoria en los logs del servicio.

---

## 5. Consulta de Logs

### Endpoint: GET /admin/audit-logs

Ver especificacion completa en `docs/specs/admin-api.md`, seccion 6.

### Indices de BD para Performance

```sql
CREATE INDEX idx_audit_user_id     ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_actor_id    ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_event_type  ON audit_logs (event_type, created_at DESC);
CREATE INDEX idx_audit_app_id      ON audit_logs (application_id, created_at DESC);
```

### Retencion de Datos

La spec no define politica de retencion. Para v1, los logs se retienen indefinidamente en PostgreSQL. En versiones futuras se puede implementar:
- Particionamiento por mes
- Exportacion a almacenamiento frio (Azure Blob Storage)
- Federacion a ELK Stack

---

## 6. Guia para el Tester

### Tests Obligatorios

1. **Login exitoso genera AUTH_LOGIN_SUCCESS** con IP y user-agent
2. **Login fallido genera AUTH_LOGIN_FAILED** con error message
3. **Bloqueo genera AUTH_ACCOUNT_LOCKED**
4. **Logout genera AUTH_LOGOUT**
5. **Refresh genera AUTH_TOKEN_REFRESHED**
6. **Cambio contrasena genera AUTH_PASSWORD_CHANGED**
7. **Reset contrasena genera AUTH_PASSWORD_RESET**
8. **Crear usuario genera USER_CREATED** con new_value
9. **Actualizar usuario genera USER_UPDATED** con old_value y new_value
10. **Asignar rol genera USER_ROLE_ASSIGNED**
11. **Revocar rol genera USER_ROLE_REVOKED**
12. **Verify permitido genera AUTHZ_PERMISSION_GRANTED**
13. **Verify denegado genera AUTHZ_PERMISSION_DENIED**
14. **Inmutabilidad:** No existe forma de modificar o eliminar logs via API
15. **Filtros:** Cada filtro de audit-logs retorna resultados correctos
16. **Paginacion:** Respuestas paginadas con total correcto
17. **Orden:** Siempre ordenado por `created_at DESC`

### Casos de Borde

- Evento con `application_id = NULL` (evento de sistema como SYSTEM_BOOTSTRAP)
- Evento con `actor_id = NULL` (login fallido de usuario no autenticado)
- Campo `old_value`/`new_value` con datos grandes (JSONB sin limite practico)
- Consulta de audit-logs con rango de fechas que no retorna resultados
- Alta concurrencia de escritura en audit_logs (no debe bloquear operaciones)

---

*Fin de especificacion audit.md*
