# Especificacion Tecnica: Modelo de Datos

**Referencia:** `docs/plan/auth-service-spec.md` seccion 3
**Historias relacionadas:** US-004, US-005

---

## 1. Diagrama Entidad-Relacion (Textual)

```
applications 1──N permissions
applications 1──N cost_centers
applications 1──N roles
applications 1──N user_roles
applications 1──N user_permissions
applications 1──N user_cost_centers
applications 1──N refresh_tokens
applications 1──N audit_logs

users 1──N user_roles
users 1──N user_permissions
users 1──N user_cost_centers
users 1──N refresh_tokens
users 1──N audit_logs (como user_id y actor_id)

roles 1──N role_permissions
roles 1──N user_roles

permissions 1──N role_permissions
permissions 1──N user_permissions

cost_centers 1──N user_cost_centers

users 1──N password_history
```

---

## 2. Tablas

### 2.1 applications

Representa cada sistema o aplicacion registrada. Actua como tenant logico.

```sql
CREATE TABLE applications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    slug        VARCHAR(50)  NOT NULL UNIQUE,
    secret_key  VARCHAR(255) NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

| Columna | Tipo | Nulo | Default | Descripcion |
|---|---|---|---|---|
| id | UUID | No | `gen_random_uuid()` | PK |
| name | VARCHAR(100) | No | - | Nombre legible, UNIQUE |
| slug | VARCHAR(50) | No | - | Identificador corto (ej: `hospitality-app`), UNIQUE |
| secret_key | VARCHAR(255) | No | - | Clave para autenticar la app via `X-App-Key` |
| is_active | BOOLEAN | No | `TRUE` | Estado de la aplicacion |
| created_at | TIMESTAMPTZ | No | `NOW()` | Fecha de creacion |
| updated_at | TIMESTAMPTZ | No | `NOW()` | Fecha de ultima actualizacion |

**Indices:**
- PK en `id`
- UNIQUE en `name`
- UNIQUE en `slug`

---

### 2.2 users

> **Decision de diseno (2026-02-21):** Se agregan las columnas `lockout_count` y `lockout_date` a la tabla `users` para implementar la regla de 3 bloqueos diarios = bloqueo permanente. Si `lockout_date != CURRENT_DATE`, el servicio resetea `lockout_count = 0` automaticamente sin requerir un job externo.

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(100) NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    must_change_pwd BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at   TIMESTAMPTZ,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    lockout_count   INT NOT NULL DEFAULT 0,
    lockout_date    DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

| Columna | Tipo | Nulo | Default | Descripcion |
|---|---|---|---|---|
| id | UUID | No | `gen_random_uuid()` | PK |
| username | VARCHAR(100) | No | - | Nombre de usuario, UNIQUE |
| email | VARCHAR(255) | No | - | Email, UNIQUE |
| password_hash | VARCHAR(255) | No | - | Hash bcrypt (costo >= 12) |
| is_active | BOOLEAN | No | `TRUE` | Cuenta activa |
| must_change_pwd | BOOLEAN | No | `FALSE` | Forzar cambio de contrasena |
| last_login_at | TIMESTAMPTZ | Si | `NULL` | Ultimo login exitoso |
| failed_attempts | INT | No | `0` | Intentos fallidos consecutivos |
| locked_until | TIMESTAMPTZ | Si | `NULL` | Bloqueo hasta esta fecha; `NULL` con `lockout_count >= 3` indica bloqueo permanente |
| lockout_count | INT | No | `0` | Numero de bloqueos en el dia indicado por `lockout_date` |
| lockout_date | DATE | Si | `NULL` | Fecha del ultimo bloqueo; si difiere de `CURRENT_DATE`, `lockout_count` se resetea a 0 en runtime |
| created_at | TIMESTAMPTZ | No | `NOW()` | Creacion |
| updated_at | TIMESTAMPTZ | No | `NOW()` | Ultima actualizacion |

**Indices:**
- PK en `id`
- UNIQUE en `username`
- UNIQUE en `email`

---

### 2.3 permissions

Catalogo de permisos por aplicacion. Convencion: `{modulo}.{recurso}.{accion}`.

```sql
CREATE TABLE permissions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id),
    code           VARCHAR(100) NOT NULL,
    description    TEXT,
    scope_type     VARCHAR(20) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id, code)
);
```

| Columna | Tipo | Valores permitidos para scope_type |
|---|---|---|
| scope_type | VARCHAR(20) | `global`, `module`, `resource`, `action` |

**Ejemplos de codigo:**
- `inventory.stock.read`
- `finance.ceco.write`
- `reports.monthly.export`
- `admin.system.manage`

---

### 2.4 cost_centers

```sql
CREATE TABLE cost_centers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id),
    code           VARCHAR(50) NOT NULL,
    name           VARCHAR(150) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id, code)
);
```

---

### 2.5 roles

```sql
CREATE TABLE roles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id),
    name           VARCHAR(100) NOT NULL,
    description    TEXT,
    is_system      BOOLEAN NOT NULL DEFAULT FALSE,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id, name)
);
```

**Nota:** `is_system = TRUE` solo para el rol `admin` creado en bootstrap. Los roles de sistema no pueden ser eliminados ni desactivados.

---

### 2.6 role_permissions

Relacion N:M entre roles y permisos.

```sql
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);
```

---

### 2.7 user_roles

Asignacion de roles a usuarios con soporte de vigencia temporal.

```sql
CREATE TABLE user_roles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES applications(id),
    granted_by     UUID NOT NULL REFERENCES users(id),
    valid_from     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until    TIMESTAMPTZ,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Logica de vigencia:**
- Un rol es vigente si: `is_active = TRUE AND valid_from <= NOW() AND (valid_until IS NULL OR valid_until > NOW())`
- `valid_until = NULL` significa sin expiracion

---

### 2.8 user_permissions

Permisos individuales adicionales, fuera del rol.

```sql
CREATE TABLE user_permissions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id  UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES applications(id),
    granted_by     UUID NOT NULL REFERENCES users(id),
    valid_from     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until    TIMESTAMPTZ,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Misma logica de vigencia que `user_roles`.

---

### 2.9 user_cost_centers

```sql
CREATE TABLE user_cost_centers (
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cost_center_id UUID NOT NULL REFERENCES cost_centers(id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES applications(id),
    granted_by     UUID NOT NULL REFERENCES users(id),
    valid_from     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until    TIMESTAMPTZ,
    PRIMARY KEY (user_id, cost_center_id)
);
```

---

### 2.10 refresh_tokens

```sql
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id      UUID NOT NULL REFERENCES applications(id),
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    device_info JSONB,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    is_revoked  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Estructura de `device_info` (JSONB):**
```json
{
  "user_agent": "Mozilla/5.0 ...",
  "ip": "192.168.1.100",
  "client_type": "web"
}
```

---

### 2.11 audit_logs

```sql
CREATE TABLE audit_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type     VARCHAR(50) NOT NULL,
    application_id UUID REFERENCES applications(id),
    user_id        UUID REFERENCES users(id),
    actor_id       UUID REFERENCES users(id),
    resource_type  VARCHAR(50),
    resource_id    UUID,
    old_value      JSONB,
    new_value      JSONB,
    ip_address     INET,
    user_agent     TEXT,
    success        BOOLEAN NOT NULL,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user_id     ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_actor_id    ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_event_type  ON audit_logs (event_type, created_at DESC);
CREATE INDEX idx_audit_app_id      ON audit_logs (application_id, created_at DESC);
```

**Regla de inmutabilidad:** No se permiten `UPDATE` ni `DELETE` sobre esta tabla. Solo `INSERT` y `SELECT`.

---

## 3. Indices Adicionales Recomendados

| Tabla | Indice | Columnas | Justificacion |
|---|---|---|---|
| `user_roles` | `idx_user_roles_user_app` | `(user_id, application_id)` | Consulta de roles vigentes por usuario y app |
| `user_roles` | `idx_user_roles_role` | `(role_id)` | Buscar usuarios de un rol |
| `user_permissions` | `idx_user_perms_user_app` | `(user_id, application_id)` | Consulta de permisos individuales |
| `user_cost_centers` | `idx_user_cc_user_app` | `(user_id, application_id)` | Consulta de CeCos por usuario |
| `refresh_tokens` | `idx_refresh_user_app` | `(user_id, app_id)` | Buscar tokens activos por usuario |
| `refresh_tokens` | `idx_refresh_expires` | `(expires_at)` | Limpieza de tokens expirados |
| `permissions` | `idx_perms_app` | `(application_id)` | Listar permisos por app |
| `roles` | `idx_roles_app` | `(application_id)` | Listar roles por app |

---

## 4. Estructura Redis

| Clave | Valor | TTL | Descripcion |
|---|---|---|---|
| `refresh:<token_hash>` | JSON `{user_id, app_id, expires_at, device_info}` | TTL del refresh token | Lookup rapido de refresh tokens |
| `user_context:<jti>` | JSON (UserContext) | 60 min | Contexto de permisos del usuario |
| `permissions_cache:<user_id>:<app_id>` | JSON (permisos efectivos) | 5 min | Cache de permisos calculados |
| `permissions_map:<app_slug>` | JSON (mapa global) | 5 min | Mapa global de permisos pre-calculado |
| `permissions_map_version:<app_slug>` | string (hash) | 5 min | Version actual del mapa |

---

## 5. Tabla password_history

> **Decision de diseno (2026-02-21):** Se confirma la creacion de una tabla dedicada `password_history` (no usar `audit_logs`). Indexada por `user_id` para lookup eficiente de los ultimos 5 hashes en una sola query.

```sql
CREATE TABLE password_history (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user ON password_history (user_id, created_at DESC);
```

**Logica de uso:**
- Al cambiar contrasena, insertar el hash anterior en `password_history`
- Al validar nueva contrasena, obtener los ultimos 5 hashes en una sola query:
  ```sql
  SELECT password_hash FROM password_history
  WHERE user_id = $1
  ORDER BY created_at DESC
  LIMIT 5;
  ```
- Comparar cada hash con `bcrypt.CompareHashAndPassword`
- Si alguno coincide, rechazar con error `PASSWORD_REUSED`

---

## 6. Guia para el Tester

### Tests Obligatorios

1. **Migracion completa:** Las 12 tablas (11 originales + `password_history`) + indices se crean sin errores en BD limpia
2. **Idempotencia:** Ejecutar migraciones multiples veces sin error
3. **Constraints UNIQUE:** Insertar duplicados en `username`, `email`, `(application_id, code)` falla
4. **CASCADE:** Eliminar usuario elimina sus `user_roles`, `user_permissions`, `user_cost_centers`, `refresh_tokens`
5. **Vigencia temporal:** Query que filtra `valid_from <= NOW() AND (valid_until IS NULL OR valid_until > NOW())` retorna solo asignaciones vigentes
6. **audit_logs inmutabilidad:** Verificar que no se puede UPDATE ni DELETE (a nivel de aplicacion/politica)

---

*Fin de especificacion data-model.md*
