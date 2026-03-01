# Plan de Tests de API - Sentinel

**Fecha de creacion:** 2026-02-28
**Autor:** senior-analyst (coordinado por team-lead)
**Version:** 1.0

---

## 1. Objetivo

Verificar que todos los endpoints de la API de Sentinel funcionan correctamente segun la especificacion documentada en `docs/api/swagger.json` y los specs tecnicos en `docs/specs/`. Se cubriran escenarios happy path y casos de error para los ~40 endpoints de la API.

---

## 2. Inventario Completo de Endpoints

### 2.1 Sistema (sin autenticacion)

| # | Metodo | Ruta | Descripcion | Tags |
|---|--------|------|-------------|------|
| 1 | GET | `/health` | Estado del servicio y dependencias (PG, Redis) | Sistema |
| 2 | GET | `/.well-known/jwks.json` | Claves publicas RSA en formato JWKS | Sistema |

### 2.2 Autenticacion

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 3 | POST | `/auth/login` | Iniciar sesion (obtener tokens JWT) | AppKeyAuth |
| 4 | POST | `/auth/refresh` | Renovar tokens con refresh token | AppKeyAuth |
| 5 | POST | `/auth/logout` | Cerrar sesion (invalidar refresh token) | BearerAuth + AppKeyAuth |
| 6 | POST | `/auth/change-password` | Cambiar contrasena del usuario autenticado | BearerAuth + AppKeyAuth |

### 2.3 Autorizacion

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 7 | POST | `/authz/verify` | Verificar si el usuario tiene un permiso | BearerAuth + AppKeyAuth |
| 8 | GET | `/authz/me/permissions` | Obtener permisos del usuario autenticado | BearerAuth |
| 9 | GET | `/authz/permissions-map` | Mapa completo de permisos (firmado RSA-SHA256) | AppKeyAuth |
| 10 | GET | `/authz/permissions-map/version` | Version/hash del mapa de permisos | AppKeyAuth |

### 2.4 Administracion - Aplicaciones

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 11 | GET | `/admin/applications` | Listar aplicaciones (paginado) | BearerAuth + AppKeyAuth |
| 12 | POST | `/admin/applications` | Crear aplicacion | BearerAuth + AppKeyAuth |
| 13 | GET | `/admin/applications/{id}` | Obtener detalle de aplicacion | BearerAuth + AppKeyAuth |
| 14 | PUT | `/admin/applications/{id}` | Actualizar aplicacion | BearerAuth + AppKeyAuth |
| 15 | POST | `/admin/applications/{id}/rotate-key` | Rotar clave de aplicacion | BearerAuth + AppKeyAuth |

### 2.5 Administracion - Usuarios

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 16 | GET | `/admin/users` | Listar usuarios (paginado) | BearerAuth + AppKeyAuth |
| 17 | POST | `/admin/users` | Crear usuario | BearerAuth + AppKeyAuth |
| 18 | GET | `/admin/users/{id}` | Obtener detalle de usuario | BearerAuth + AppKeyAuth |
| 19 | PUT | `/admin/users/{id}` | Actualizar usuario | BearerAuth + AppKeyAuth |
| 20 | POST | `/admin/users/{id}/reset-password` | Restablecer contrasena (genera temporal) | BearerAuth + AppKeyAuth |
| 21 | POST | `/admin/users/{id}/unlock` | Desbloquear usuario | BearerAuth + AppKeyAuth |
| 22 | POST | `/admin/users/{id}/roles` | Asignar rol a usuario | BearerAuth + AppKeyAuth |
| 23 | DELETE | `/admin/users/{id}/roles/{rid}` | Revocar rol de usuario | BearerAuth + AppKeyAuth |
| 24 | POST | `/admin/users/{id}/permissions` | Asignar permiso directo a usuario | BearerAuth + AppKeyAuth |
| 25 | DELETE | `/admin/users/{id}/permissions/{pid}` | Revocar permiso de usuario | BearerAuth + AppKeyAuth |
| 26 | POST | `/admin/users/{id}/cost-centers` | Asignar centros de costo a usuario | BearerAuth + AppKeyAuth |

### 2.6 Administracion - Roles

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 27 | GET | `/admin/roles` | Listar roles (paginado) | BearerAuth + AppKeyAuth |
| 28 | POST | `/admin/roles` | Crear rol | BearerAuth + AppKeyAuth |
| 29 | GET | `/admin/roles/{id}` | Obtener detalle de rol (con permisos) | BearerAuth + AppKeyAuth |
| 30 | PUT | `/admin/roles/{id}` | Actualizar rol | BearerAuth + AppKeyAuth |
| 31 | DELETE | `/admin/roles/{id}` | Eliminar (desactivar) rol | BearerAuth + AppKeyAuth |
| 32 | POST | `/admin/roles/{id}/permissions` | Agregar permisos a rol | BearerAuth + AppKeyAuth |
| 33 | DELETE | `/admin/roles/{id}/permissions/{pid}` | Remover permiso de rol | BearerAuth + AppKeyAuth |

### 2.7 Administracion - Permisos

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 34 | GET | `/admin/permissions` | Listar permisos (paginado) | BearerAuth + AppKeyAuth |
| 35 | POST | `/admin/permissions` | Crear permiso | BearerAuth + AppKeyAuth |
| 36 | DELETE | `/admin/permissions/{id}` | Eliminar permiso | BearerAuth + AppKeyAuth |

### 2.8 Administracion - Centros de Costo

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 37 | GET | `/admin/cost-centers` | Listar centros de costo (paginado) | BearerAuth + AppKeyAuth |
| 38 | POST | `/admin/cost-centers` | Crear centro de costo | BearerAuth + AppKeyAuth |
| 39 | PUT | `/admin/cost-centers/{id}` | Actualizar centro de costo | BearerAuth + AppKeyAuth |

### 2.9 Administracion - Auditoria

| # | Metodo | Ruta | Descripcion | Seguridad |
|---|--------|------|-------------|-----------|
| 40 | GET | `/admin/audit-logs` | Listar registros de auditoria (paginado, filtrable) | BearerAuth + AppKeyAuth |

**Total: 40 endpoints**

---

## 3. Escenarios de Test por Endpoint

### 3.1 Sistema

#### EP-01: GET /health
- **HAPPY-01:** Servicio saludable -> 200 con status "healthy", checks postgresql=ok, redis=ok
- **ERR-01:** (informativo) Si alguna dependencia falla -> 503 con status "degraded"

#### EP-02: GET /.well-known/jwks.json
- **HAPPY-02:** Retorna JWKS -> 200 con array "keys", cada key tiene kty=RSA, alg=RS256, use=sig
- **ERR-02:** Sin headers X-App-Key -> aun debe funcionar (endpoint publico)

### 3.2 Autenticacion

#### EP-03: POST /auth/login
- **HAPPY-03a:** Login exitoso con client_type=web -> 200 con access_token, refresh_token, token_type=Bearer, expires_in, user object
- **HAPPY-03b:** Login exitoso con client_type=mobile -> 200 (validar que funciona)
- **HAPPY-03c:** Login exitoso con client_type=desktop -> 200
- **ERR-03a:** Sin X-App-Key -> 401 APPLICATION_NOT_FOUND
- **ERR-03b:** X-App-Key invalido -> 401 APPLICATION_NOT_FOUND
- **ERR-03c:** Username incorrecto -> 401 INVALID_CREDENTIALS
- **ERR-03d:** Password incorrecto -> 401 INVALID_CREDENTIALS
- **ERR-03e:** Body vacio -> 400 VALIDATION_ERROR
- **ERR-03f:** client_type invalido (ej: "tablet") -> 400 INVALID_CLIENT_TYPE
- **ERR-03g:** client_type faltante -> 400 VALIDATION_ERROR
- **ERR-03h:** Cuenta inactiva -> 403 ACCOUNT_INACTIVE
- **ERR-03i:** Cuenta bloqueada -> 403 ACCOUNT_LOCKED

#### EP-04: POST /auth/refresh
- **HAPPY-04:** Refresh exitoso -> 200 con nuevos access_token y refresh_token
- **ERR-04a:** Refresh token invalido -> 401 TOKEN_INVALID
- **ERR-04b:** Refresh token vacio -> 400 VALIDATION_ERROR
- **ERR-04c:** Sin X-App-Key -> 401 APPLICATION_NOT_FOUND

#### EP-05: POST /auth/logout
- **HAPPY-05:** Logout exitoso -> 204 (refresh token invalidado)
- **ERR-05a:** Sin Authorization header -> 401
- **ERR-05b:** Token expirado -> 401 TOKEN_EXPIRED o TOKEN_INVALID

#### EP-06: POST /auth/change-password
- **HAPPY-06:** Cambio exitoso -> 204
- **ERR-06a:** Contrasena actual incorrecta -> 400 o 401
- **ERR-06b:** Nueva contrasena no cumple politica (< 10 chars) -> 400
- **ERR-06c:** Nueva contrasena sin mayuscula -> 400
- **ERR-06d:** Nueva contrasena sin numero -> 400
- **ERR-06e:** Nueva contrasena sin simbolo -> 400
- **ERR-06f:** Nueva contrasena reutilizada (historial) -> 400 PASSWORD_REUSED
- **ERR-06g:** Sin Authorization -> 401

### 3.3 Autorizacion

#### EP-07: POST /authz/verify
- **HAPPY-07a:** Permiso concedido -> 200 con allowed=true
- **HAPPY-07b:** Permiso denegado -> 200 con allowed=false
- **HAPPY-07c:** Verificacion con cost_center_id -> 200
- **ERR-07a:** Sin Authorization -> 401
- **ERR-07b:** Sin X-App-Key -> 401
- **ERR-07c:** Body vacio -> 400

#### EP-08: GET /authz/me/permissions
- **HAPPY-08:** Retorna permisos del usuario -> 200 con user_id, application, roles, permissions, cost_centers
- **ERR-08a:** Sin Authorization -> 401

#### EP-09: GET /authz/permissions-map
- **HAPPY-09:** Retorna mapa firmado -> 200 con permissions, signature, signed_at
- **ERR-09a:** Sin X-App-Key -> 401
- **ERR-09b:** X-App-Key invalido -> 401

#### EP-10: GET /authz/permissions-map/version
- **HAPPY-10:** Retorna version/hash -> 200 con version
- **ERR-10a:** Sin X-App-Key -> 401

### 3.4 Administracion - Aplicaciones

#### EP-11: GET /admin/applications
- **HAPPY-11a:** Lista con paginacion default -> 200 con data, page=1, page_size=20, total, total_pages
- **HAPPY-11b:** Con parametros de paginacion -> 200
- **HAPPY-11c:** Con filtro search -> 200
- **HAPPY-11d:** Con filtro is_active -> 200
- **ERR-11a:** Sin Authorization -> 401
- **ERR-11b:** Sin X-App-Key -> 401
- **ERR-11c:** Sin permiso admin -> 403

#### EP-12: POST /admin/applications
- **HAPPY-12:** Crear app exitoso -> 201 con id, name, slug, is_active=true
- **ERR-12a:** Slug duplicado -> 409
- **ERR-12b:** Datos faltantes -> 400
- **ERR-12c:** Sin Authorization -> 401
- **ERR-12d:** Sin permiso -> 403

#### EP-13: GET /admin/applications/{id}
- **HAPPY-13:** Obtener app existente -> 200
- **ERR-13a:** ID no encontrado -> 404
- **ERR-13b:** ID invalido (no UUID) -> 400
- **ERR-13c:** Sin Authorization -> 401

#### EP-14: PUT /admin/applications/{id}
- **HAPPY-14:** Actualizar nombre -> 200
- **ERR-14a:** App de sistema -> 403
- **ERR-14b:** ID no encontrado -> 404
- **ERR-14c:** Datos invalidos -> 400
- **ERR-14d:** Sin Authorization -> 401

#### EP-15: POST /admin/applications/{id}/rotate-key
- **HAPPY-15:** Rotar clave exitoso -> 200 con nueva clave
- **ERR-15a:** App de sistema -> 403
- **ERR-15b:** ID no encontrado -> 404
- **ERR-15c:** ID invalido -> 400
- **ERR-15d:** Sin Authorization -> 401

### 3.5 Administracion - Usuarios

#### EP-16: GET /admin/users
- **HAPPY-16a:** Lista con paginacion default -> 200
- **HAPPY-16b:** Con search -> 200
- **HAPPY-16c:** Con is_active -> 200
- **ERR-16a:** Sin Authorization -> 401
- **ERR-16b:** Sin permiso -> 403

#### EP-17: POST /admin/users
- **HAPPY-17:** Crear usuario exitoso -> 201
- **ERR-17a:** Username duplicado -> 400 o 409
- **ERR-17b:** Email duplicado -> 400 o 409
- **ERR-17c:** Contrasena no cumple politica -> 400
- **ERR-17d:** Datos faltantes -> 400
- **ERR-17e:** Sin Authorization -> 401

#### EP-18: GET /admin/users/{id}
- **HAPPY-18:** Obtener usuario existente -> 200
- **ERR-18a:** ID no encontrado -> 404
- **ERR-18b:** ID invalido -> 400
- **ERR-18c:** Sin Authorization -> 401

#### EP-19: PUT /admin/users/{id}
- **HAPPY-19:** Actualizar email -> 200
- **ERR-19a:** ID no encontrado -> 404
- **ERR-19b:** Datos invalidos -> 400
- **ERR-19c:** Sin Authorization -> 401

#### EP-20: POST /admin/users/{id}/reset-password
- **HAPPY-20:** Generar contrasena temporal -> 200 con temporary_password
- **ERR-20a:** ID no encontrado -> 400 o 404
- **ERR-20b:** Sin Authorization -> 401

#### EP-21: POST /admin/users/{id}/unlock
- **HAPPY-21:** Desbloquear usuario -> 204
- **ERR-21a:** ID invalido -> 400
- **ERR-21b:** Sin Authorization -> 401

#### EP-22: POST /admin/users/{id}/roles
- **HAPPY-22:** Asignar rol -> 201 con id, role_id, role_name, user_id
- **ERR-22a:** Datos invalidos -> 400
- **ERR-22b:** Sin Authorization -> 401

#### EP-23: DELETE /admin/users/{id}/roles/{rid}
- **HAPPY-23:** Revocar rol -> 204
- **ERR-23a:** ID invalido -> 400
- **ERR-23b:** Sin Authorization -> 401

#### EP-24: POST /admin/users/{id}/permissions
- **HAPPY-24:** Asignar permiso -> 201
- **ERR-24a:** Datos invalidos -> 400
- **ERR-24b:** Sin Authorization -> 401

#### EP-25: DELETE /admin/users/{id}/permissions/{pid}
- **HAPPY-25:** Revocar permiso -> 204
- **ERR-25a:** ID invalido -> 400
- **ERR-25b:** Sin Authorization -> 401

#### EP-26: POST /admin/users/{id}/cost-centers
- **HAPPY-26:** Asignar centros de costo -> 201 con assigned count
- **ERR-26a:** Datos invalidos -> 400
- **ERR-26b:** Sin Authorization -> 401

### 3.6 Administracion - Roles

#### EP-27: GET /admin/roles
- **HAPPY-27:** Lista paginada -> 200
- **ERR-27a:** Sin Authorization -> 401
- **ERR-27b:** Sin permiso -> 403

#### EP-28: POST /admin/roles
- **HAPPY-28:** Crear rol -> 201
- **ERR-28a:** Nombre duplicado -> 400 o 409
- **ERR-28b:** Datos faltantes -> 400
- **ERR-28c:** Sin Authorization -> 401

#### EP-29: GET /admin/roles/{id}
- **HAPPY-29:** Obtener rol con permisos -> 200
- **ERR-29a:** ID no encontrado -> 404
- **ERR-29b:** ID invalido -> 400
- **ERR-29c:** Sin Authorization -> 401

#### EP-30: PUT /admin/roles/{id}
- **HAPPY-30:** Actualizar nombre/descripcion -> 200
- **ERR-30a:** ID no encontrado -> 404
- **ERR-30b:** Sin Authorization -> 401

#### EP-31: DELETE /admin/roles/{id}
- **HAPPY-31:** Eliminar (desactivar) rol -> 204
- **ERR-31a:** ID invalido -> 400
- **ERR-31b:** Sin Authorization -> 401

#### EP-32: POST /admin/roles/{id}/permissions
- **HAPPY-32:** Agregar permisos a rol -> 201 con assigned count
- **ERR-32a:** Datos invalidos -> 400
- **ERR-32b:** Sin Authorization -> 401

#### EP-33: DELETE /admin/roles/{id}/permissions/{pid}
- **HAPPY-33:** Remover permiso de rol -> 204
- **ERR-33a:** ID invalido -> 400
- **ERR-33b:** Sin Authorization -> 401

### 3.7 Administracion - Permisos

#### EP-34: GET /admin/permissions
- **HAPPY-34:** Lista paginada -> 200
- **ERR-34a:** Sin Authorization -> 401
- **ERR-34b:** Sin permiso -> 403

#### EP-35: POST /admin/permissions
- **HAPPY-35:** Crear permiso -> 201
- **ERR-35a:** Code duplicado -> 400 o 409
- **ERR-35b:** Datos faltantes -> 400
- **ERR-35c:** Sin Authorization -> 401

#### EP-36: DELETE /admin/permissions/{id}
- **HAPPY-36:** Eliminar permiso (no asignado) -> 204
- **ERR-36a:** Permiso en uso (asignado a rol/usuario) -> 400 o 409
- **ERR-36b:** ID invalido -> 400
- **ERR-36c:** Sin Authorization -> 401

### 3.8 Administracion - Centros de Costo

#### EP-37: GET /admin/cost-centers
- **HAPPY-37:** Lista paginada -> 200
- **ERR-37a:** Sin Authorization -> 401
- **ERR-37b:** Sin permiso -> 403

#### EP-38: POST /admin/cost-centers
- **HAPPY-38:** Crear centro de costo -> 201
- **ERR-38a:** Code duplicado -> 400 o 409
- **ERR-38b:** Datos faltantes -> 400
- **ERR-38c:** Sin Authorization -> 401

#### EP-39: PUT /admin/cost-centers/{id}
- **HAPPY-39:** Actualizar nombre/estado -> 200
- **ERR-39a:** ID no encontrado -> 404
- **ERR-39b:** Datos invalidos -> 400
- **ERR-39c:** Sin Authorization -> 401

### 3.9 Auditoria

#### EP-40: GET /admin/audit-logs
- **HAPPY-40a:** Lista paginada -> 200
- **HAPPY-40b:** Filtro por event_type -> 200
- **HAPPY-40c:** Filtro por success -> 200
- **HAPPY-40d:** Filtro por fecha (from_date, to_date) -> 200
- **HAPPY-40e:** Filtro por user_id -> 200
- **ERR-40a:** Sin Authorization -> 401
- **ERR-40b:** Sin permiso -> 403

---

## 4. Resumen de Escenarios

| Categoria | Endpoints | Happy Path | Error Cases | Total Escenarios |
|-----------|-----------|------------|-------------|------------------|
| Sistema | 2 | 2 | 2 | 4 |
| Autenticacion | 4 | 6 | 18 | 24 |
| Autorizacion | 4 | 6 | 6 | 12 |
| Admin - Aplicaciones | 5 | 7 | 15 | 22 |
| Admin - Usuarios | 11 | 11 | 22 | 33 |
| Admin - Roles | 7 | 7 | 14 | 21 |
| Admin - Permisos | 3 | 3 | 7 | 10 |
| Admin - CeCos | 3 | 3 | 7 | 10 |
| Auditoria | 1 | 5 | 2 | 7 |
| **TOTAL** | **40** | **50** | **93** | **143** |

---

## 5. Herramienta Recomendada

### Decision: Bash script con curl + jq

**Justificacion:**
1. **Sin dependencias adicionales:** curl y jq ya estan disponibles en el entorno Linux.
2. **Rapida ejecucion:** No requiere compilacion ni instalacion de runtime extra.
3. **Alineado con el stack:** El proyecto ya usa bash para Makefile y scripts de deploy.
4. **Salida legible:** jq permite parsear y validar JSON facilmente.
5. **Idempotencia:** Cada test puede hacer setup/teardown con llamadas curl adicionales.
6. **Portabilidad:** Funciona en cualquier entorno UNIX sin Node, Go ni k6.

### Alternativa considerada: k6

k6 ya existe en `tests/load/` pero es mas adecuado para tests de carga, no para validacion funcional detallada donde necesitamos inspeccionar cada campo de respuesta.

---

## 6. Estructura de Archivos de Test

```
tests/api/
  run_all.sh              # Script principal que ejecuta todos los tests
  lib/
    common.sh             # Funciones compartidas (login, assert_status, assert_json, cleanup)
    config.sh             # Variables de configuracion (BASE_URL, APP_KEY, credenciales)
  01_system.sh            # Tests de /health y /.well-known/jwks.json
  02_auth_login.sh        # Tests de POST /auth/login
  03_auth_refresh.sh      # Tests de POST /auth/refresh
  04_auth_logout.sh       # Tests de POST /auth/logout
  05_auth_change_pwd.sh   # Tests de POST /auth/change-password
  06_authz_verify.sh      # Tests de POST /authz/verify
  07_authz_me.sh          # Tests de GET /authz/me/permissions
  08_authz_map.sh         # Tests de GET /authz/permissions-map y /version
  09_admin_apps.sh        # Tests CRUD de /admin/applications
  10_admin_users.sh       # Tests CRUD de /admin/users
  11_admin_user_roles.sh  # Tests de asignacion/revocacion de roles a usuarios
  12_admin_user_perms.sh  # Tests de asignacion/revocacion de permisos a usuarios
  13_admin_user_cecos.sh  # Tests de asignacion de centros de costo a usuarios
  14_admin_roles.sh       # Tests CRUD de /admin/roles
  15_admin_role_perms.sh  # Tests de permisos en roles
  16_admin_permissions.sh # Tests CRUD de /admin/permissions
  17_admin_cecos.sh       # Tests CRUD de /admin/cost-centers
  18_admin_audit.sh       # Tests de /admin/audit-logs
  results/                # Directorio para resultados de ejecucion
    test-output.log       # Log completo de la ejecucion
```

---

## 7. Datos de Prueba Necesarios

### 7.1 Datos del Bootstrap (ya existentes)

| Dato | Valor |
|------|-------|
| Usuario admin | `admin` |
| Password admin | `Admin@Local1!` (o `Admin@Sentinel2!` si ya fue cambiado) |
| Aplicacion sistema | `sentinel` (slug) |
| X-App-Key | Se obtiene via login + listar apps |

### 7.2 Datos a Crear Durante los Tests

| Recurso | Nombre/Identificador | Proposito |
|---------|---------------------|-----------|
| Usuario de test | `testuser_api` / `testuser_api@test.com` | CRUD de usuarios, asignacion de roles/permisos |
| Aplicacion de test | `test-app-api` (slug) | CRUD de aplicaciones |
| Rol de test | `test-role-api` | CRUD de roles, asignacion a usuarios |
| Permiso de test | `test.api.read` | CRUD de permisos, asignacion a roles/usuarios |
| Centro de costo | `TST-001` / `Test CeCo API` | CRUD de centros de costo |

### 7.3 Password de Test (cumple politica)

- Para creacion de usuario: `TestP@ssw0rd1!` (>= 10 chars, mayuscula, numero, simbolo)
- Para cambio de password: `NewP@ssw0rd2!`

### 7.4 Orden de Ejecucion y Dependencias

1. **Sistema** (01) - sin dependencias
2. **Auth Login** (02) - obtiene tokens y APP_KEY para el resto
3. **Auth Refresh** (03) - depende del login
4. **Auth Logout** (04) - depende del login
5. **Auth Change Password** (05) - se ejecuta con usuario de test
6. **Admin Permisos** (16) - se crean primero para poder asignar a roles
7. **Admin Roles** (14) - se crean despues de permisos
8. **Admin Role Perms** (15) - depende de roles y permisos
9. **Admin CeCos** (17) - independiente
10. **Admin Apps** (09) - independiente
11. **Admin Users** (10) - depende de tener token admin
12. **Admin User Roles** (11) - depende de usuarios y roles
13. **Admin User Perms** (12) - depende de usuarios y permisos
14. **Admin User CeCos** (13) - depende de usuarios y CeCos
15. **Authz** (06, 07, 08) - depende de tener permisos configurados
16. **Audit** (18) - se ejecuta al final para capturar eventos generados

---

## 8. Criterios de Exito

- **Test PASS:** El status code HTTP coincide con el esperado Y la estructura JSON de respuesta contiene los campos requeridos con valores validos.
- **Test FAIL:** El status code HTTP no coincide O faltan campos requeridos O los valores son incorrectos.
- **Cobertura minima:** Todos los 40 endpoints deben tener al menos 1 happy path y 1 error case ejecutado.
- **Idempotencia:** Los tests deben limpiar los recursos creados al finalizar (o al inicio, en caso de ejecuciones previas fallidas).

---

## 9. Restricciones de Seguridad

- Nunca loguear passwords en texto plano en los reportes de salida.
- Nunca loguear tokens JWT completos en los reportes (truncar a primeros 20 chars + "...").
- Los scripts de test NO deben contener passwords hardcodeados en el codigo; se leen de config.sh o variables de entorno.
- Todos los datos de test se crean con un prefijo identificable (`test_api_*`, `TST-*`) para facilitar limpieza.

---

## 10. Ambiente de Ejecucion

| Componente | URL |
|-----------|-----|
| Backend API | http://localhost:8080 |
| Frontend | http://localhost:8090 |
| Swagger UI | http://localhost:8080/swagger/ |
| Stack local | `deploy/local/` (docker compose) |

**Prerequisito:** El stack local debe estar corriendo (`make docker-up` o `docker compose -f deploy/local/docker-compose.yml up -d`).
