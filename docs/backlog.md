# Backlog Oficial -- Proyecto Sentinel Auth Service

**Version:** 1.0.0
**Fecha:** 2026-02-21
**Estado:** Aprobado para desarrollo
**Referencia:** `docs/plan/auth-service-spec.md` v1.0.0

---

## Resumen de Epicas

| ID | Epica | Historias | Story Points |
|---|---|---|---|
| EP-01 | Infraestructura y Configuracion | 5 | 21 |
| EP-02 | Autenticacion | 6 | 29 |
| EP-03 | Gestion de Tokens (JWT / Refresh) | 4 | 21 |
| EP-04 | Autorizacion (RBAC + Permisos) | 6 | 31 |
| EP-05 | Administracion | 8 | 34 |
| EP-06 | Auditoria | 3 | 13 |
| EP-07 | Seguridad y Hardening | 5 | 21 |
| EP-08 | Calidad, Testing y Despliegue | 5 | 26 |
| **Total** | | **42** | **196** |

---

## EP-01: Infraestructura y Configuracion

### US-001 -- Inicializacion del proyecto Go

**Como** desarrollador backend, **quiero** tener el proyecto Go inicializado con la estructura de directorios definida, **para** comenzar el desarrollo con una base consistente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 1
**Especificacion:** `docs/specs/infrastructure.md`

**Criterios de Aceptacion:**
- [ ] Modulo Go inicializado (`go mod init`)
- [ ] Estructura de directorios creada: `cmd/server/`, `internal/{config,domain,repository,service,handler,middleware,token,bootstrap}/`, `migrations/`
- [ ] Linter `golangci-lint` configurado y ejecutable
- [ ] `Makefile` con targets: `build`, `run`, `test`, `lint`, `migrate`
- [ ] El proyecto compila sin errores con `go build ./...`

---

### US-002 -- Infraestructura local con Docker Compose

**Como** desarrollador, **quiero** levantar PostgreSQL, Redis y el servicio Go con un solo comando, **para** tener un entorno de desarrollo reproducible.

**Responsable:** devops
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1
**Especificacion:** `docs/specs/infrastructure.md`

**Criterios de Aceptacion:**
- [ ] `docker-compose.yml` define servicios: `auth-service`, `postgres` (15+), `redis` (7+)
- [ ] Volumenes persistentes configurados para PostgreSQL y Redis
- [ ] `Dockerfile` multi-stage produce un binario Go optimizado
- [ ] `docker-compose up` levanta los tres servicios sin errores
- [ ] El servicio Go se conecta exitosamente a PostgreSQL y Redis

---

### US-003 -- Sistema de configuracion

**Como** desarrollador backend, **quiero** cargar configuracion desde variables de entorno y archivo YAML, **para** parametrizar el servicio sin recompilar.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 1
**Especificacion:** `docs/specs/infrastructure.md`

**Criterios de Aceptacion:**
- [ ] Archivo `config.yaml` con estructura completa (server, database, redis, jwt, security, bootstrap, logging)
- [ ] Variables de entorno sobreescriben valores del YAML
- [ ] Validacion de configuracion al arranque: falla con mensaje claro si faltan campos obligatorios
- [ ] Secrets (passwords, keys) solo se aceptan via variable de entorno, nunca en YAML en texto plano

---

### US-004 -- Migraciones de base de datos

**Como** desarrollador backend, **quiero** que las migraciones de BD se ejecuten automaticamente al arrancar, **para** mantener el esquema sincronizado sin pasos manuales.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1
**Especificacion:** `docs/specs/data-model.md`, `docs/specs/infrastructure.md`

**Criterios de Aceptacion:**
- [ ] Herramienta de migraciones integrada (`golang-migrate` o `goose`)
- [ ] Migracion `001_initial_schema.sql` crea las 11 tablas: `applications`, `users`, `permissions`, `cost_centers`, `roles`, `role_permissions`, `user_roles`, `user_permissions`, `user_cost_centers`, `refresh_tokens`, `audit_logs`
- [ ] Indices de `audit_logs` creados (`user_id`, `actor_id`, `event_type`, `application_id`)
- [ ] Migracion `002_seed_permissions.sql` inserta permisos base de administracion
- [ ] Migraciones son idempotentes: ejecutar multiples veces no produce errores
- [ ] Las migraciones se ejecutan automaticamente al arrancar el servicio

---

### US-005 -- Modelos de dominio

**Como** desarrollador backend, **quiero** tener las estructuras Go del dominio definidas, **para** tipar correctamente todas las operaciones del servicio.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1
**Especificacion:** `docs/specs/data-model.md`

**Criterios de Aceptacion:**
- [ ] `domain/user.go`: struct `User` con todos los campos de la tabla `users`
- [ ] `domain/role.go`: structs `Role`, `UserRole` (con `valid_from`, `valid_until`)
- [ ] `domain/permission.go`: structs `Permission`, `UserPermission` (con vigencia)
- [ ] `domain/cost_center.go`: structs `CostCenter`, `UserCostCenter`
- [ ] `domain/audit.go`: struct `AuditLog`, constantes para todos los tipos de evento (seccion 6.1 del spec)
- [ ] `domain/application.go`: struct `Application`
- [ ] `domain/token.go`: structs `RefreshToken`, `JWTClaims`

---

## EP-02: Autenticacion

### US-006 -- Login de usuario

**Como** usuario, **quiero** autenticarme con mi nombre de usuario y contrasena, **para** obtener un token de acceso que me permita usar las aplicaciones.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `POST /auth/login` acepta `{"username", "password"}` con header `X-App-Key`
- [ ] Retorna 200 con `access_token` (JWT RS256), `refresh_token`, `token_type`, `expires_in`, `user`
- [ ] Valida credenciales con bcrypt (costo >= 12)
- [ ] Verifica que la cuenta este activa (`is_active = true`)
- [ ] Verifica que la cuenta no este bloqueada (`locked_until`)
- [ ] Verifica que la aplicacion exista y este activa via `X-App-Key`
- [ ] Actualiza `last_login_at` y resetea `failed_attempts` en login exitoso
- [ ] Registra evento `AUTH_LOGIN_SUCCESS` o `AUTH_LOGIN_FAILED` en auditoria
- [ ] Retorna error `INVALID_CREDENTIALS` (401) si usuario/contrasena son incorrectos
- [ ] Retorna error `ACCOUNT_LOCKED` (403) si la cuenta esta bloqueada
- [ ] Retorna error `ACCOUNT_INACTIVE` (403) si la cuenta esta inactiva
- [ ] Retorna error `APPLICATION_NOT_FOUND` (401) si el `X-App-Key` es invalido
- [ ] Incluye campo `must_change_password` en la respuesta del usuario

---

### US-007 -- Refresh de token

**Como** cliente (frontend/app), **quiero** renovar mi access token usando el refresh token, **para** mantener la sesion activa sin pedir credenciales nuevamente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `POST /auth/refresh` acepta `{"refresh_token"}` con header `X-App-Key`
- [ ] Retorna 200 con nuevos `access_token` y `refresh_token` (rotacion automatica)
- [ ] El refresh token anterior queda invalidado tras su uso
- [ ] Verifica hash del refresh token en Redis, luego PostgreSQL como fallback
- [ ] Retorna error `TOKEN_INVALID` (401) si el token no existe
- [ ] Retorna error `TOKEN_EXPIRED` (401) si el token ha expirado
- [ ] Retorna error `TOKEN_REVOKED` (401) si el token fue revocado
- [ ] Registra evento `AUTH_TOKEN_REFRESHED` en auditoria

---

### US-008 -- Logout

**Como** usuario, **quiero** cerrar mi sesion, **para** invalidar mi refresh token y prevenir uso no autorizado.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 2
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `POST /auth/logout` requiere header `Authorization: Bearer <access_token>`
- [ ] Retorna 204 sin cuerpo
- [ ] Revoca el refresh token activo del usuario para la aplicacion actual
- [ ] Registra evento `AUTH_LOGOUT` en auditoria
- [ ] Un refresh token revocado no puede ser usado para obtener nuevos tokens

---

### US-009 -- Cambio de contrasena

**Como** usuario autenticado, **quiero** cambiar mi contrasena, **para** mantener la seguridad de mi cuenta.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `POST /auth/change-password` requiere `Authorization: Bearer <access_token>`
- [ ] Acepta `{"current_password", "new_password"}`
- [ ] Retorna 204 sin cuerpo en caso exitoso
- [ ] Valida la contrasena actual antes de aceptar la nueva
- [ ] Aplica politica de contrasena: minimo 10 caracteres, 1 mayuscula, 1 numero, 1 simbolo
- [ ] Verifica historial de ultimas 5 contrasenas (no reutilizar)
- [ ] Hashea con bcrypt costo >= 12
- [ ] Actualiza `must_change_pwd = false` despues del cambio exitoso
- [ ] Registra evento `AUTH_PASSWORD_CHANGED` en auditoria

---

### US-010 -- Health check

**Como** operador del sistema, **quiero** un endpoint de health check, **para** monitorear la disponibilidad del servicio.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 1
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `GET /health` no requiere autenticacion
- [ ] Retorna 200 con estado del servicio (incluye conectividad a PostgreSQL y Redis)
- [ ] Retorna 503 si alguna dependencia critica no esta disponible

---

### US-011 -- Bootstrap del sistema

**Como** operador del sistema, **quiero** que el servicio se auto-configure al iniciar con una BD vacia, **para** tener un admin funcional sin pasos manuales.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] Detecta automaticamente que no existen aplicaciones ni usuarios en la BD
- [ ] Crea la aplicacion `system` con slug `system`
- [ ] Crea el rol `admin` con `is_system = true` y todos los permisos de gestion
- [ ] Crea el usuario administrador con credenciales de `BOOTSTRAP_ADMIN_USER` / `BOOTSTRAP_ADMIN_PASSWORD`
- [ ] Establece `must_change_pwd = true` en el usuario admin
- [ ] Registra evento `SYSTEM_BOOTSTRAP` en auditoria
- [ ] No se re-ejecuta en arranques subsiguientes (idempotente)
- [ ] Si las variables de entorno de bootstrap no estan definidas, el servicio falla con mensaje claro

---

## EP-03: Gestion de Tokens (JWT / Refresh)

### US-012 -- Generacion de JWT RS256

**Como** servicio de autenticacion, **quiero** emitir tokens JWT firmados con RS256, **para** que los backends puedan verificarlos localmente con la clave publica.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] JWT firmado con algoritmo RS256 usando clave privada RSA
- [ ] Header incluye: `alg: RS256`, `typ: JWT`, `kid: <identificador-de-clave>`
- [ ] Payload incluye: `sub`, `username`, `email`, `app`, `roles`, `iat`, `exp`, `jti`
- [ ] TTL del access token: 60 minutos (configurable)
- [ ] `jti` es un UUID v4 unico por token
- [ ] Token NO incluye `extra_permissions` ni `cost_centers` (solo roles)
- [ ] Token verificable con la clave publica expuesta en JWKS

---

### US-013 -- Endpoint JWKS

**Como** servicio backend consumidor, **quiero** obtener la clave publica RSA desde un endpoint estandar, **para** verificar JWT localmente sin compartir secretos.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `GET /.well-known/jwks.json` no requiere autenticacion
- [ ] Retorna la(s) clave(s) publica(s) RSA en formato JWKS estandar (RFC 7517)
- [ ] Cada clave incluye `kid`, `kty`, `alg`, `use`, `n`, `e`
- [ ] Soporta multiples claves para periodo de transicion durante rotacion
- [ ] Clave publica puede ser cacheada por los consumidores (TTL recomendado 60 min)

---

### US-014 -- Ciclo de vida del Refresh Token

**Como** servicio de autenticacion, **quiero** gestionar refresh tokens con rotacion automatica, **para** mantener sesiones seguras de larga duracion.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] Refresh token generado como UUID v4 aleatorio
- [ ] Almacenado como hash bcrypt en Redis (con TTL) y PostgreSQL
- [ ] TTL configurable: 7 dias (web), 30 dias (movil/escritorio)
- [ ] Cada uso genera un nuevo refresh token e invalida el anterior (rotacion)
- [ ] Almacena `device_info` (user-agent, IP, tipo de cliente) en JSONB
- [ ] Revocacion explicita via logout
- [ ] Clave Redis: `refresh:<token_hash>` con TTL nativo

---

### US-015 -- Rotacion de claves RSA

**Como** operador de seguridad, **quiero** rotar las claves RSA sin downtime, **para** cumplir con politicas de seguridad sin afectar usuarios.

**Responsable:** devops / backend
**Prioridad:** Media
**Story Points:** 5
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Claves RSA leidas desde Azure Key Vault (o archivo configurable)
- [ ] JWKS expone multiples claves durante periodo de transicion
- [ ] JWT firmados con clave nueva; tokens existentes con clave anterior siguen siendo validos
- [ ] Mapas de permisos firmados con clave nueva; backends detectan fallo de firma y recargan clave publica
- [ ] Procedimiento operativo documentado

---

## EP-04: Autorizacion (RBAC + Permisos)

### US-016 -- Verificacion delegada de permisos

**Como** servicio backend, **quiero** verificar si un usuario tiene un permiso especifico via API, **para** autorizar operaciones cuando no puedo verificar localmente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] `POST /authz/verify` requiere `Authorization: Bearer` y `X-App-Key`
- [ ] Acepta `{"permission", "cost_center_id?"}`
- [ ] Retorna 200 con `{"allowed", "user_id", "username", "permission", "evaluated_at"}`
- [ ] Evalua permisos via roles + permisos individuales + vigencia temporal
- [ ] Si se especifica `cost_center_id`, valida que el usuario tenga acceso al CeCo
- [ ] Latencia objetivo < 50 ms (p95)
- [ ] Registra `AUTHZ_PERMISSION_GRANTED` o `AUTHZ_PERMISSION_DENIED` en auditoria

---

### US-017 -- Contexto de permisos del usuario

**Como** cliente (frontend/backend), **quiero** obtener todos los permisos efectivos del usuario autenticado, **para** controlar la interfaz y cachear el contexto localmente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] `GET /authz/me/permissions` requiere `Authorization: Bearer`
- [ ] Retorna roles vigentes, permisos efectivos (union de roles + individuales), CeCos asignados
- [ ] Incluye `temporary_roles` con fecha de expiracion para roles temporales
- [ ] Solo incluye roles/permisos donde `valid_from <= NOW()` y (`valid_until IS NULL` o `valid_until > NOW()`)
- [ ] Solo incluye asignaciones con `is_active = true`
- [ ] Respuesta cacheada en Redis por `jti` con TTL de 60 min

---

### US-018 -- Mapa global de permisos (firmado)

**Como** servicio backend consumidor, **quiero** descargar un mapa global de permisos firmado, **para** resolver autorizaciones localmente sin llamadas de red por request.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] `GET /authz/permissions-map` requiere `X-App-Key`
- [ ] Retorna JSON con: `application`, `generated_at`, `version`, `permissions`, `cost_centers`, `signature`
- [ ] `version` es un hash del estado actual del mapa
- [ ] `permissions` mapea cada codigo a sus roles habilitados y descripcion
- [ ] `cost_centers` lista CeCos activos con codigo y nombre
- [ ] `signature` es RSA-SHA256 del payload canonico (claves ordenadas, sin espacios)
- [ ] Firma verificable con la clave publica del endpoint JWKS
- [ ] El `version` hash se invalida al modificar cualquier permiso, rol o asignacion de CeCo

---

### US-019 -- Polling de version del mapa

**Como** servicio backend consumidor, **quiero** consultar la version del mapa sin descargar el payload completo, **para** detectar cambios de forma eficiente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 2
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] `GET /authz/permissions-map/version` requiere `X-App-Key`
- [ ] Retorna `{"application", "version", "generated_at"}`
- [ ] Si `version` difiere del valor cacheado por el backend, este descarga el mapa completo
- [ ] Latencia minima (sin calculo pesado; el hash se pre-calcula)

---

### US-020 -- Motor de evaluacion de permisos (HasPermission)

**Como** servicio de autorizacion, **quiero** implementar el algoritmo `HasPermission`, **para** resolver permisos combinando roles, permisos individuales y CeCos.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] Implementa el algoritmo de la seccion 5.3 del spec
- [ ] Paso 1: Verifica firma y expiracion del JWT
- [ ] Paso 2: Obtiene `UserContext` desde cache por `jti` (o lo solicita si no existe)
- [ ] Paso 3: Si el permiso esta en `extra_permissions` del usuario, concede acceso
- [ ] Paso 4: Busca en mapa global que roles otorgan el permiso; verifica si el usuario tiene alguno
- [ ] Paso 5: Si se especifica `cost_center`, valida que este en los CeCos del usuario
- [ ] Paso 6: Retorna PERMITIDO o DENEGADO
- [ ] Roles/permisos temporales expirados no cuentan
- [ ] Tests unitarios para todos los casos borde (ver spec `docs/specs/authorization.md`)

---

### US-021 -- Cache de autorizacion en Redis

**Como** servicio de autorizacion, **quiero** cachear el contexto de usuario y el mapa de permisos en Redis, **para** minimizar consultas a PostgreSQL en operaciones frecuentes.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 2

**Criterios de Aceptacion:**
- [ ] `UserContext` cacheado por `jti` con TTL de 60 min
- [ ] Cache de permisos efectivos del usuario con TTL de 5 min
- [ ] Cache invalidado cuando se modifican roles, permisos o CeCos del usuario
- [ ] Cache invalidado cuando se modifica la composicion de un rol (role_permissions)
- [ ] Mapa global de permisos pre-calculado y cacheado

---

## EP-05: Administracion

### US-022 -- CRUD de usuarios (admin)

**Como** administrador, **quiero** crear, consultar, actualizar y desactivar usuarios, **para** gestionar las cuentas del sistema.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `GET /admin/users` lista usuarios (paginado)
- [ ] `POST /admin/users` crea usuario con contrasena hasheada, `must_change_pwd = true`
- [ ] `GET /admin/users/:id` retorna detalle del usuario
- [ ] `PUT /admin/users/:id` actualiza datos del usuario (username, email, is_active)
- [ ] Todos requieren permiso `admin.system.manage`
- [ ] Registra eventos `USER_CREATED`, `USER_UPDATED`, `USER_DEACTIVATED` en auditoria
- [ ] Almacena `old_value` y `new_value` en auditoria para cambios

---

### US-023 -- Desbloqueo y reset de contrasena (admin)

**Como** administrador, **quiero** desbloquear cuentas y forzar reset de contrasena, **para** asistir a usuarios que pierden acceso.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `POST /admin/users/:id/unlock` resetea `failed_attempts = 0` y `locked_until = NULL`
- [ ] `POST /admin/users/:id/reset-password` genera contrasena temporal y fuerza `must_change_pwd = true`
- [ ] Requieren permiso `admin.system.manage`
- [ ] Registra eventos `USER_UNLOCKED` y `AUTH_PASSWORD_RESET` en auditoria

---

### US-024 -- CRUD de roles (admin)

**Como** administrador, **quiero** crear, consultar, actualizar y desactivar roles, **para** definir perfiles de acceso reutilizables.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `GET /admin/roles` lista roles de la aplicacion
- [ ] `POST /admin/roles` crea rol con nombre y descripcion
- [ ] `GET /admin/roles/:id` retorna detalle con sus permisos asignados
- [ ] `PUT /admin/roles/:id` actualiza nombre/descripcion
- [ ] `DELETE /admin/roles/:id` desactiva el rol (`is_active = false`), no lo elimina fisicamente
- [ ] No se puede eliminar/desactivar un rol con `is_system = true`
- [ ] Requieren permiso `admin.system.manage`
- [ ] Registra eventos `ROLE_CREATED`, `ROLE_UPDATED`, `ROLE_DELETED`

---

### US-025 -- Asignacion de permisos a roles (admin)

**Como** administrador, **quiero** asignar y revocar permisos de un rol, **para** definir que acciones permite cada perfil.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `POST /admin/roles/:id/permissions` asigna uno o mas permisos al rol
- [ ] `DELETE /admin/roles/:id/permissions/:pid` revoca un permiso del rol
- [ ] Invalida cache del mapa global de permisos (cambia `version`)
- [ ] Invalida cache de todos los usuarios que tienen el rol afectado
- [ ] Registra eventos `ROLE_PERMISSION_ASSIGNED`, `ROLE_PERMISSION_REVOKED`

---

### US-026 -- Asignacion de roles a usuarios (admin)

**Como** administrador, **quiero** asignar y revocar roles de un usuario con vigencia temporal opcional, **para** controlar el acceso de forma flexible.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `POST /admin/users/:id/roles` asigna rol con `valid_from`, `valid_until` (opcional) y `granted_by`
- [ ] `DELETE /admin/users/:id/roles/:rid` revoca el rol (marca `is_active = false`)
- [ ] Si `valid_until` se especifica, el rol expira automaticamente en la fecha indicada
- [ ] Invalida cache del contexto del usuario afectado
- [ ] Registra eventos `USER_ROLE_ASSIGNED`, `USER_ROLE_REVOKED`

---

### US-027 -- Asignacion de permisos individuales a usuarios (admin)

**Como** administrador, **quiero** asignar permisos individuales a un usuario por fuera de sus roles, **para** cubrir excepciones puntuales.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `POST /admin/users/:id/permissions` asigna permiso individual con vigencia temporal opcional
- [ ] `DELETE /admin/users/:id/permissions/:pid` revoca permiso individual
- [ ] Invalida cache del contexto del usuario afectado
- [ ] Registra eventos `USER_PERMISSION_ASSIGNED`, `USER_PERMISSION_REVOKED`

---

### US-028 -- Gestion de permisos (admin)

**Como** administrador, **quiero** registrar y eliminar permisos del catalogo, **para** definir las acciones disponibles en cada aplicacion.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `GET /admin/permissions` lista permisos de la aplicacion
- [ ] `POST /admin/permissions` crea permiso con `code` (formato `modulo.recurso.accion`), `description`, `scope_type`
- [ ] `DELETE /admin/permissions/:id` elimina permiso (CASCADE en role_permissions y user_permissions)
- [ ] Valida formato del codigo de permiso: `{modulo}.{recurso}.{accion}`
- [ ] Valida `scope_type` sea uno de: `global`, `module`, `resource`, `action`
- [ ] Unique constraint: `(application_id, code)`

---

### US-029 -- Gestion de Centros de Costo (admin)

**Como** administrador, **quiero** crear, actualizar CeCos y asignarlos a usuarios, **para** controlar acceso a datos por centro de costo.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `GET /admin/cost-centers` lista CeCos de la aplicacion
- [ ] `POST /admin/cost-centers` crea CeCo con `code` y `name`
- [ ] `PUT /admin/cost-centers/:id` actualiza nombre y estado
- [ ] `POST /admin/users/:id/cost-centers` asigna CeCos al usuario con vigencia
- [ ] Invalida cache del mapa global y del contexto del usuario
- [ ] Registra evento `USER_COST_CENTER_ASSIGNED`
- [ ] Unique constraint: `(application_id, code)`

---

### US-030 -- Middleware de permisos para rutas admin

**Como** servicio de autenticacion, **quiero** proteger todas las rutas `/admin/*` con verificacion de permiso `admin.system.manage`, **para** garantizar que solo administradores accedan.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 2
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] Middleware declarativo que valida permiso requerido antes de ejecutar el handler
- [ ] Retorna 403 si el usuario no tiene el permiso `admin.system.manage`
- [ ] Funciona en combinacion con el JWT middleware (requiere token valido)

---

## EP-06: Auditoria

### US-031 -- Registro inmutable de eventos de auditoria

**Como** oficial de seguridad, **quiero** que todas las acciones criticas se registren de forma inmutable, **para** tener trazabilidad completa.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] Cada registro incluye: `event_type`, `application_id`, `user_id`, `actor_id`, `resource_type`, `resource_id`, `old_value`, `new_value`, `ip_address`, `user_agent`, `success`, `error_message`, `created_at`
- [ ] Todos los tipos de evento de la seccion 6.1 son registrados en los flujos correspondientes
- [ ] No existen endpoints ni metodos de UPDATE o DELETE sobre `audit_logs`
- [ ] `created_at` siempre en UTC con zona horaria

---

### US-032 -- Middleware de auditoria automatica

**Como** desarrollador backend, **quiero** un middleware que registre automaticamente eventos en endpoints criticos, **para** no depender de registro manual en cada handler.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] Middleware captura IP (`X-Forwarded-For` o remota), user-agent, usuario autenticado
- [ ] Se ejecuta despues de los handlers de autenticacion, autorizacion y administracion
- [ ] Registra exito o fallo segun el status code de la respuesta
- [ ] No bloquea la respuesta al cliente (registro asincrono o con goroutine)

---

### US-033 -- Consulta de logs de auditoria (admin)

**Como** administrador, **quiero** consultar los logs de auditoria con filtros, **para** investigar incidentes de seguridad.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] `GET /admin/audit-logs` requiere permiso `admin.system.manage`
- [ ] Filtros soportados: `user_id`, `actor_id`, `event_type`, `from_date`, `to_date`, `application_id`, `success`
- [ ] Resultados paginados y ordenados por `created_at DESC`
- [ ] Indices de BD optimizan las consultas filtradas

---

## EP-07: Seguridad y Hardening

### US-034 -- Proteccion contra fuerza bruta

**Como** servicio de autenticacion, **quiero** bloquear cuentas tras intentos fallidos repetidos, **para** prevenir ataques de fuerza bruta.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] Despues de 5 intentos fallidos consecutivos, la cuenta se bloquea por 15 minutos
- [ ] Despues de 3 bloqueos en el mismo dia, la cuenta requiere desbloqueo manual por admin
- [ ] `failed_attempts` se incrementa en cada login fallido y se resetea en login exitoso
- [ ] `locked_until` se establece al momento del bloqueo
- [ ] Se registra `AUTH_LOGIN_FAILED` por cada intento fallido
- [ ] Se registra `AUTH_ACCOUNT_LOCKED` al activar el bloqueo

---

### US-035 -- Politica de contrasenas

**Como** servicio de autenticacion, **quiero** validar politicas de contrasena en cada creacion o cambio, **para** garantizar contrasenas seguras.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 3

**Criterios de Aceptacion:**
- [ ] Minimo 10 caracteres
- [ ] Al menos 1 mayuscula, 1 numero, 1 simbolo
- [ ] Historial de ultimas 5 contrasenas (almacenar hashes, comparar con bcrypt)
- [ ] Se aplica en: creacion de usuario, cambio de contrasena, reset de contrasena
- [ ] Mensaje de error claro indicando que requisito no se cumple

---

### US-036 -- Headers de seguridad HTTP

**Como** servicio de autenticacion, **quiero** agregar headers de seguridad en todas las respuestas, **para** proteger contra ataques comunes.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 2
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] `Strict-Transport-Security: max-age=31536000; includeSubDomains` en todas las respuestas
- [ ] `X-Content-Type-Options: nosniff` en todas las respuestas
- [ ] `X-Frame-Options: DENY` en todas las respuestas
- [ ] Headers aplicados via middleware global

---

### US-037 -- Middleware de autenticacion de aplicacion

**Como** servicio de autenticacion, **quiero** validar el `X-App-Key` en cada request (excepto `/health`), **para** asegurar que solo aplicaciones registradas consuman la API.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 3
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] Valida `X-App-Key` contra `applications.secret_key` en BD
- [ ] Verifica que la aplicacion este activa (`is_active = true`)
- [ ] Excluye `GET /health` y `GET /.well-known/jwks.json` de la validacion
- [ ] Retorna 401 con error `APPLICATION_NOT_FOUND` si la clave es invalida
- [ ] Inyecta `application_id` en el contexto del request para uso en handlers

---

### US-038 -- Servidor HTTP con graceful shutdown

**Como** operador del sistema, **quiero** que el servicio cierre conexiones de forma ordenada, **para** evitar perdida de datos en despliegues.

**Responsable:** backend
**Prioridad:** Media
**Story Points:** 3
**Fase:** 1

**Criterios de Aceptacion:**
- [ ] Graceful shutdown con timeout de 15 segundos (configurable)
- [ ] Completa requests en vuelo antes de cerrar
- [ ] Cierra conexiones a PostgreSQL y Redis de forma ordenada
- [ ] Responde a senales SIGTERM y SIGINT

---

## EP-08: Calidad, Testing y Despliegue

### US-039 -- Tests de integracion

**Como** equipo de desarrollo, **quiero** una suite de tests de integracion con cobertura >= 80%, **para** garantizar la calidad del servicio.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 8
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Entorno de test con PostgreSQL y Redis en Docker (Testcontainers o `docker-compose.test.yml`)
- [ ] Tests de autenticacion: login exitoso/fallido, cuenta bloqueada/inactiva, refresh valido/expirado/revocado, logout, cambio de contrasena
- [ ] Tests de autorizacion: HasPermission por rol, por extra_permission, con CeCo, roles temporales expirados, firma del mapa
- [ ] Tests de administracion: CRUD usuarios/roles/permisos, audit logs
- [ ] Tests de bootstrap: idempotencia
- [ ] Cobertura >= 80% (`go test ./... -cover`)

---

### US-040 -- Tests unitarios

**Como** equipo de desarrollo, **quiero** tests unitarios para la logica de negocio critica, **para** detectar regresiones rapidamente.

**Responsable:** backend
**Prioridad:** Alta
**Story Points:** 5
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Tests para `token/jwt.go`: generacion y validacion RS256
- [ ] Tests para `service/authz_service.go`: algoritmo HasPermission con todos los casos borde
- [ ] Tests para `bootstrap/initializer.go`: idempotencia
- [ ] Tests para `canonicalJSON`: serializacion determinista
- [ ] Tests para `VerifyPermissionsMap`: firma valida e invalida

---

### US-041 -- Documentacion OpenAPI

**Como** equipo de desarrollo y consumidores de la API, **quiero** documentacion OpenAPI completa, **para** entender y consumir la API correctamente.

**Responsable:** backend
**Prioridad:** Media
**Story Points:** 5
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Especificacion OpenAPI 3.0 completa (`swagger.yaml`)
- [ ] Todos los endpoints documentados con request, response y errores
- [ ] Ejemplos de payload para cada endpoint
- [ ] Swagger UI disponible en `/docs` (solo en entornos non-production)

---

### US-042 -- Load testing y benchmarks

**Como** equipo de desarrollo, **quiero** ejecutar load tests contra los endpoints criticos, **para** verificar que se cumplen los SLAs de rendimiento.

**Responsable:** backend
**Prioridad:** Media
**Story Points:** 3
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Escenarios con Grafana k6 o hey
- [ ] `POST /auth/login` < 200 ms p95 bajo carga sostenida
- [ ] `POST /authz/verify` < 50 ms p95
- [ ] Resultados documentados con ajustes realizados

---

### US-043 -- Despliegue en Azure staging

**Como** equipo de operaciones, **quiero** desplegar el servicio en Azure staging, **para** validar el funcionamiento en un entorno similar a produccion.

**Responsable:** devops
**Prioridad:** Media
**Story Points:** 5
**Fase:** 4

**Criterios de Aceptacion:**
- [ ] Dockerfile de produccion (multi-stage, imagen minima)
- [ ] Variables de entorno y secretos configurados en Azure Key Vault
- [ ] Manifiestos de despliegue para Azure Container Apps o AKS
- [ ] Health check configurado en el orquestador (`GET /health`)
- [ ] Smoke tests pasando en staging
- [ ] Logging estructurado en JSON
- [ ] Metricas basicas de latencia por endpoint

---

## Matriz de Dependencias entre Historias

| Historia | Depende de |
|---|---|
| US-006 (Login) | US-001, US-002, US-003, US-004, US-005, US-012, US-014, US-037 |
| US-007 (Refresh) | US-006, US-014 |
| US-008 (Logout) | US-006 |
| US-009 (Cambio pwd) | US-006, US-035 |
| US-011 (Bootstrap) | US-004, US-005 |
| US-016 (Verify) | US-012, US-020 |
| US-017 (Me/Permissions) | US-012, US-021 |
| US-018 (Permissions Map) | US-012, US-013 |
| US-020 (HasPermission) | US-017, US-018 |
| US-022-029 (Admin) | US-030, US-031 |
| US-039 (Tests integracion) | US-001 a US-035 |
| US-043 (Deploy) | US-039 |

---

*Fin del Backlog v1.0.0 -- Proyecto Sentinel*
