# Especificacion Tecnica: API de Autenticacion

**Referencia:** `docs/plan/auth-service-spec.md` secciones 4.2, 4.5
**Historias relacionadas:** US-006, US-007, US-008, US-009, US-010, US-011

---

## 1. Convenciones Generales

- **Base URL:** `https://auth.internal.sodexo.cl/api/v1`
- **Content-Type:** `application/json`
- **Autenticacion de aplicacion:** Header `X-App-Key: <secret_key>` en todos los endpoints excepto `/health` y `/.well-known/jwks.json`
- **Autenticacion de usuario:** Header `Authorization: Bearer <access_token>`

### Formato de Error Estandar

```json
{
  "error": {
    "code": "CODIGO_DE_ERROR",
    "message": "Descripcion legible del error",
    "details": null
  }
}
```

### Codigos de Error Globales

| Codigo HTTP | Error Code | Descripcion |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Datos de entrada invalidos |
| 400 | `INVALID_CLIENT_TYPE` | `client_type` no es un valor valido del enum |
| 401 | `INVALID_CREDENTIALS` | Usuario o contrasena incorrectos |
| 401 | `TOKEN_INVALID` | Token JWT o refresh token invalido |
| 401 | `TOKEN_EXPIRED` | Token expirado |
| 401 | `TOKEN_REVOKED` | Refresh token revocado |
| 401 | `APPLICATION_NOT_FOUND` | X-App-Key invalido o aplicacion inactiva |
| 403 | `ACCOUNT_LOCKED` | Cuenta bloqueada por intentos fallidos |
| 403 | `ACCOUNT_INACTIVE` | Cuenta desactivada |
| 403 | `FORBIDDEN` | Sin permiso para la operacion |
| 500 | `INTERNAL_ERROR` | Error interno del servidor |

---

## 2. POST /auth/login

**Descripcion:** Autentica al usuario con credenciales y retorna tokens de acceso.

### Request

**Headers:**
| Header | Requerido | Descripcion |
|---|---|---|
| `X-App-Key` | Si | Secret key de la aplicacion |
| `Content-Type` | Si | `application/json` |

**Body:**

> **Decision de diseno (2026-02-21):** Se agrega el campo obligatorio `client_type` al body del login para determinar el TTL del refresh token. Es un enum estricto con valores: `web`, `mobile`, `desktop`.

```json
{
  "username": "jperez",
  "password": "S3cur3P@ss!",
  "client_type": "web"
}
```

| Campo | Tipo | Requerido | Validacion |
|---|---|---|---|
| `username` | string | Si | 1-100 caracteres |
| `password` | string | Si | No vacio |
| `client_type` | string | Si | Enum estricto: `web`, `mobile`, `desktop` |

**TTL del refresh token segun `client_type`:**

| client_type | TTL del refresh token |
|---|---|
| `web` | 7 dias (168h) |
| `mobile` | 30 dias (720h) |
| `desktop` | 30 dias (720h) |

### Response 200 OK

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjIwMjYtMDIta2V5LTAxIn0...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jperez",
    "email": "jperez@sodexo.com",
    "must_change_password": false
  }
}
```

| Campo | Tipo | Descripcion |
|---|---|---|
| `access_token` | string | JWT RS256, TTL 60 min |
| `refresh_token` | string | Token opaco para renovacion |
| `token_type` | string | Siempre `"Bearer"` |
| `expires_in` | int | Segundos hasta expiracion del access token |
| `user.id` | UUID | ID del usuario |
| `user.username` | string | Nombre de usuario |
| `user.email` | string | Email del usuario |
| `user.must_change_password` | bool | `true` si debe cambiar contrasena |

### Errores

| Codigo | Error Code | Condicion |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Campos requeridos faltantes o invalidos |
| 400 | `INVALID_CLIENT_TYPE` | `client_type` no es `web`, `mobile` ni `desktop` |
| 401 | `INVALID_CREDENTIALS` | Username no existe o password incorrecto |
| 401 | `APPLICATION_NOT_FOUND` | `X-App-Key` invalido o aplicacion inactiva |
| 403 | `ACCOUNT_LOCKED` | Cuenta bloqueada (`locked_until > NOW()`) |
| 403 | `ACCOUNT_INACTIVE` | `is_active = false` |

### Logica de Negocio

1. Validar `X-App-Key` contra tabla `applications` (activa)
1b. Validar `client_type` sea uno de: `web`, `mobile`, `desktop`. Si no, retornar `INVALID_CLIENT_TYPE`
2. Buscar usuario por `username`
3. Si no existe: retornar `INVALID_CREDENTIALS`, registrar `AUTH_LOGIN_FAILED`
4. Si `is_active = false`: retornar `ACCOUNT_INACTIVE`
5. Si `locked_until > NOW()`: retornar `ACCOUNT_LOCKED`
6. Comparar password con `password_hash` usando bcrypt
7. Si falla:
   - Incrementar `failed_attempts`
   - Si `failed_attempts >= 5`: establecer `locked_until = NOW() + 15min`, registrar `AUTH_ACCOUNT_LOCKED`
   - Si 3 bloqueos en el mismo dia: bloqueo permanente (sin `locked_until` automatico)
   - Registrar `AUTH_LOGIN_FAILED`
   - Retornar `INVALID_CREDENTIALS`
8. Si exito:
   - Resetear `failed_attempts = 0`, `locked_until = NULL`
   - Actualizar `last_login_at = NOW()`
   - Obtener roles vigentes del usuario para la aplicacion
   - Generar access token (JWT RS256) y refresh token (UUID v4)
   - Determinar TTL del refresh token segun `client_type`: `web` = 7 dias, `mobile`/`desktop` = 30 dias
   - Persistir refresh token (hash bcrypt) en Redis y PostgreSQL con el TTL correspondiente
   - Registrar `AUTH_LOGIN_SUCCESS`
   - Retornar tokens + datos de usuario

### Eventos de Auditoria

- `AUTH_LOGIN_SUCCESS`: login exitoso
- `AUTH_LOGIN_FAILED`: credenciales incorrectas
- `AUTH_ACCOUNT_LOCKED`: cuenta bloqueada tras 5 intentos

---

## 3. POST /auth/refresh

**Descripcion:** Renueva el access token usando el refresh token. Implementa rotacion automatica.

### Request

**Headers:**
| Header | Requerido | Descripcion |
|---|---|---|
| `X-App-Key` | Si | Secret key de la aplicacion |

**Body:**
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

### Response 200 OK

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Nota:** A diferencia de `/auth/login`, no se incluye el objeto `user`.

### Errores

| Codigo | Error Code | Condicion |
|---|---|---|
| 401 | `TOKEN_INVALID` | Refresh token no encontrado en Redis/PostgreSQL |
| 401 | `TOKEN_EXPIRED` | `expires_at < NOW()` |
| 401 | `TOKEN_REVOKED` | `is_revoked = true` |

### Logica de Negocio

1. Calcular hash del refresh token recibido
2. Buscar en Redis (`refresh:<token_hash>`); si no existe, buscar en PostgreSQL
3. Si no encontrado: retornar `TOKEN_INVALID`
4. Si `is_revoked = true`: retornar `TOKEN_REVOKED`
5. Si `expires_at < NOW()`: retornar `TOKEN_EXPIRED`
6. Verificar que el usuario siga activo y no bloqueado
7. Invalidar refresh token actual (marcar `is_revoked = true`, eliminar de Redis)
8. Obtener roles vigentes del usuario para la aplicacion
9. Generar nuevo access token y nuevo refresh token
10. Persistir nuevo refresh token en Redis y PostgreSQL
11. Registrar `AUTH_TOKEN_REFRESHED`
12. Retornar nuevos tokens

### Eventos de Auditoria

- `AUTH_TOKEN_REFRESHED`

---

## 4. POST /auth/logout

**Descripcion:** Invalida el refresh token activo del usuario para la aplicacion actual.

### Request

**Headers:**
| Header | Requerido | Descripcion |
|---|---|---|
| `Authorization` | Si | `Bearer <access_token>` |
| `X-App-Key` | Si | Secret key de la aplicacion |

**Body:** Vacio o ninguno.

### Response 204 No Content

Sin cuerpo.

### Logica de Negocio

1. Extraer `user_id` y `app` del JWT
2. Buscar refresh tokens activos del usuario para la aplicacion
3. Marcar como revocados (`is_revoked = true`)
4. Eliminar de Redis
5. Registrar `AUTH_LOGOUT`

### Eventos de Auditoria

- `AUTH_LOGOUT`

---

## 5. POST /auth/change-password

**Descripcion:** Permite al usuario autenticado cambiar su contrasena.

### Request

**Headers:**
| Header | Requerido | Descripcion |
|---|---|---|
| `Authorization` | Si | `Bearer <access_token>` |
| `X-App-Key` | Si | Secret key de la aplicacion |

**Body:**
```json
{
  "current_password": "OldP@ssw0rd!",
  "new_password": "N3wS3cur3P@ss!"
}
```

| Campo | Tipo | Requerido | Validacion |
|---|---|---|---|
| `current_password` | string | Si | No vacio |
| `new_password` | string | Si | Politica de contrasena |

### Response 204 No Content

Sin cuerpo.

### Errores

| Codigo | Error Code | Condicion |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Contrasena no cumple politica |
| 400 | `PASSWORD_REUSED` | Contrasena en historial de ultimas 5 |
| 401 | `INVALID_CREDENTIALS` | Contrasena actual incorrecta |

### Politica de Contrasena

- Minimo 10 caracteres
- Al menos 1 letra mayuscula
- Al menos 1 numero
- Al menos 1 simbolo (caracteres especiales)
- No puede coincidir con las ultimas 5 contrasenas

### Logica de Negocio

1. Extraer `user_id` del JWT
2. Verificar contrasena actual con bcrypt
3. Si falla: retornar `INVALID_CREDENTIALS`
4. Validar nueva contrasena contra politica
5. Verificar historial de ultimas 5 contrasenas (comparar hash bcrypt)
6. Hashear nueva contrasena con bcrypt costo >= 12
7. Actualizar `password_hash`, agregar hash anterior al historial
8. Establecer `must_change_pwd = false`
9. Registrar `AUTH_PASSWORD_CHANGED`

### Eventos de Auditoria

- `AUTH_PASSWORD_CHANGED`

---

## 6. GET /health

**Descripcion:** Health check del servicio. No requiere autenticacion.

### Response 200 OK

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "checks": {
    "postgresql": "ok",
    "redis": "ok"
  }
}
```

### Response 503 Service Unavailable

```json
{
  "status": "unhealthy",
  "version": "1.0.0",
  "checks": {
    "postgresql": "ok",
    "redis": "error: connection refused"
  }
}
```

---

## 7. Guia para el Tester

### Tests Unitarios Obligatorios

1. **Login exitoso:** credenciales validas, retorna tokens y datos de usuario
2. **Login credenciales incorrectas:** retorna 401 `INVALID_CREDENTIALS`
3. **Login cuenta bloqueada:** retorna 403 `ACCOUNT_LOCKED`
4. **Login cuenta inactiva:** retorna 403 `ACCOUNT_INACTIVE`
5. **Login X-App-Key invalido:** retorna 401 `APPLICATION_NOT_FOUND`
6. **Login incremento de failed_attempts:** verifica que se incrementa en cada fallo
7. **Login bloqueo a los 5 intentos:** verifica `locked_until` se establece
8. **Login bloqueo permanente tras 3 bloqueos diarios:** requiere desbloqueo admin
9. **Refresh exitoso:** retorna nuevos tokens, invalida token anterior
10. **Refresh token invalido:** retorna 401 `TOKEN_INVALID`
11. **Refresh token expirado:** retorna 401 `TOKEN_EXPIRED`
12. **Refresh token revocado:** retorna 401 `TOKEN_REVOKED`
13. **Logout exitoso:** retorna 204, refresh token queda revocado
14. **Cambio de contrasena exitoso:** retorna 204, hash actualizado
15. **Cambio de contrasena actual incorrecta:** retorna 401
16. **Cambio de contrasena no cumple politica:** retorna 400 con detalle
17. **Cambio de contrasena reutilizada:** retorna 400 `PASSWORD_REUSED`
18. **Health check con dependencias OK:** retorna 200
19. **Health check con dependencia caida:** retorna 503

### Casos de Borde

- Login con username que contiene caracteres especiales
- Login inmediatamente despues de que `locked_until` expire (deberia permitir)
- Refresh token usado dos veces (segunda vez debe fallar: rotacion)
- Cambio de contrasena exactamente igual a la 5ta contrasena anterior
- Cambio de contrasena con exactamente 10 caracteres (limite minimo)
- Login con `must_change_pwd = true`: login exitoso, frontend debe forzar cambio
- Login con `client_type = "web"`: refresh token TTL = 7 dias
- Login con `client_type = "mobile"`: refresh token TTL = 30 dias
- Login con `client_type = "desktop"`: refresh token TTL = 30 dias
- Login con `client_type = "tablet"` (valor invalido): retorna 400 `INVALID_CLIENT_TYPE`
- Login sin campo `client_type`: retorna 400 `VALIDATION_ERROR`

---

*Fin de especificacion auth-api.md*
