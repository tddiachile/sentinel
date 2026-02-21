# Especificacion Tecnica: Gestion de Tokens

**Referencia:** `docs/plan/auth-service-spec.md` secciones 5.1, 5.2, 5.3, 5.4, 5.5, 7.4
**Historias relacionadas:** US-012, US-013, US-014, US-015

---

## 1. Access Token (JWT RS256)

### 1.1 Estructura del Token

**Header:**
```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "2026-02-key-01"
}
```

**Payload:**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "username": "jperez",
  "email": "jperez@sodexo.com",
  "app": "hospitality-app",
  "roles": ["chef", "bodeguero-temporal"],
  "iat": 1708250400,
  "exp": 1708254000,
  "jti": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
}
```

| Claim | Tipo | Descripcion |
|---|---|---|
| `sub` | UUID | ID del usuario |
| `username` | string | Nombre de usuario |
| `email` | string | Email del usuario |
| `app` | string | Slug de la aplicacion |
| `roles` | string[] | Roles vigentes del usuario en la aplicacion |
| `iat` | int (epoch) | Timestamp de emision |
| `exp` | int (epoch) | Timestamp de expiracion (iat + 3600) |
| `jti` | UUID | Identificador unico del token |

### 1.2 Decisiones de Diseno

El token lleva **unicamente los roles** del usuario. Los campos **NO incluidos** en el JWT son:

- `extra_permissions` (permisos individuales)
- `cost_centers` (CeCos autorizados)

Estos residen en el Auth Service y se obtienen via `GET /authz/me/permissions`, cacheados por `jti` en los backends consumidores.

**Razon:** Con mas de 50 permisos potenciales, incluirlos en el JWT incrementaria su tamano en 2-3 KB por request.

### 1.3 Configuracion

| Parametro | Valor | Configurable |
|---|---|---|
| Algoritmo | RS256 | No |
| TTL access token | 60 minutos | Si (`jwt.access_token_ttl`) |
| Clave privada | Archivo o Azure Key Vault | Si (`jwt.private_key_path`) |
| Clave publica | Archivo o Azure Key Vault | Si (`jwt.public_key_path`) |

---

## 2. Refresh Token

### 2.1 Generacion

- Generado como **UUID v4** aleatorio
- Almacenado como **hash bcrypt** (nunca en texto plano)
- Dual-storage: Redis (para velocidad) + PostgreSQL (para durabilidad)

### 2.2 Almacenamiento

**Redis:**
- Clave: `refresh:<bcrypt_hash>`
- Valor: JSON con `user_id`, `app_id`, `expires_at`, `device_info`
- TTL: nativo de Redis, coincide con la vigencia del token

**PostgreSQL (tabla `refresh_tokens`):**

| Campo | Tipo | Descripcion |
|---|---|---|
| `id` | UUID | PK |
| `user_id` | UUID | FK a users |
| `app_id` | UUID | FK a applications |
| `token_hash` | VARCHAR(255) | Hash bcrypt del token (UNIQUE) |
| `device_info` | JSONB | `{"user_agent": "...", "ip": "...", "client_type": "web"}` |
| `expires_at` | TIMESTAMPTZ | Fecha de expiracion |
| `used_at` | TIMESTAMPTZ | Ultima vez que se uso para refresh |
| `is_revoked` | BOOLEAN | `true` si fue revocado |
| `created_at` | TIMESTAMPTZ | Fecha de creacion |

### 2.3 TTL por Tipo de Cliente

> **Decision de diseno (2026-02-21):** El tipo de cliente se determina mediante un campo explicito `client_type` (enum: `web`, `mobile`, `desktop`) en el body del request `POST /auth/login`. Es un campo requerido, validado con enum estricto. No se infiere del User-Agent.

| Tipo de cliente (`client_type`) | TTL | Configuracion |
|---|---|---|
| `web` | 7 dias (168h) | `jwt.refresh_token_ttl_web` |
| `mobile` | 30 dias (720h) | `jwt.refresh_token_ttl_mobile` |
| `desktop` | 30 dias (720h) | `jwt.refresh_token_ttl_mobile` |

**Validacion:** Si `client_type` no es uno de los tres valores permitidos, el login retorna 400 `INVALID_CLIENT_TYPE`.

**Almacenamiento:** El valor de `client_type` se persiste en el campo `device_info.client_type` (JSONB) de la tabla `refresh_tokens` y en Redis.

### 2.4 Rotacion Automatica

Cada vez que se usa un refresh token:

1. Se invalida el token actual (`is_revoked = true` en PG, eliminado de Redis)
2. Se genera un nuevo refresh token
3. Se persiste el nuevo token en Redis y PostgreSQL
4. Se retorna el nuevo token junto con el nuevo access token

**Diagrama de rotacion:**
```
Cliente                    Auth Service               Redis          PostgreSQL
   |                            |                       |                |
   |-- POST /auth/refresh ----->|                       |                |
   |   {refresh_token: "A"}     |                       |                |
   |                            |-- GET refresh:hash(A)->|                |
   |                            |<-- {user_id, app_id}--|                |
   |                            |-- DEL refresh:hash(A)->|                |
   |                            |-- UPDATE is_revoked=T ----------------->|
   |                            |                       |                |
   |                            |-- SET refresh:hash(B)->|                |
   |                            |-- INSERT refresh_token ---------------->|
   |                            |                       |                |
   |<-- {access_token, --------|                       |                |
   |     refresh_token: "B"}    |                       |                |
```

### 2.5 Revocacion

Un refresh token se revoca en los siguientes casos:

1. **Uso normal (rotacion):** El token usado queda revocado al generar uno nuevo
2. **Logout explicito:** El usuario cierra sesion
3. **Reset de contrasena por admin:** Se revocan todos los refresh tokens del usuario
4. **Desactivacion de cuenta:** Se revocan todos los refresh tokens del usuario

---

## 3. Endpoint JWKS

### GET /.well-known/jwks.json

**Descripcion:** Expone las claves publicas RSA en formato JWKS estandar (RFC 7517) para que los backends consumidores verifiquen JWT localmente.

**Autenticacion:** Ninguna requerida.

**Response 200:**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "2026-02-key-01",
      "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmb...",
      "e": "AQAB"
    }
  ]
}
```

| Campo | Descripcion |
|---|---|
| `kty` | Tipo de clave: `"RSA"` |
| `alg` | Algoritmo: `"RS256"` |
| `use` | Uso: `"sig"` (firma) |
| `kid` | Identificador de la clave (coincide con el `kid` del JWT header) |
| `n` | Modulus RSA (base64url) |
| `e` | Exponente RSA (base64url) |

**Cache recomendado:** 60 minutos.

### Rotacion de Claves

Durante la rotacion, el array `keys` contiene **dos claves** (anterior y nueva):

```json
{
  "keys": [
    { "kid": "2026-04-key-02", ... },
    { "kid": "2026-02-key-01", ... }
  ]
}
```

La clave nueva se lista primero. Los nuevos JWT se firman con la clave nueva. Los JWT existentes firmados con la clave anterior siguen siendo validos hasta su expiracion.

---

## 4. Rotacion de Claves RSA

### 4.1 Almacenamiento

- Claves RSA almacenadas como secretos en **Azure Key Vault**
- En desarrollo local: archivos PEM configurables via `jwt.private_key_path` y `jwt.public_key_path`

### 4.2 Procedimiento de Rotacion

1. Generar nuevo par de claves RSA (minimo 2048 bits, recomendado 4096)
2. Subir nueva clave privada a Azure Key Vault con nuevo `kid`
3. Configurar Auth Service para leer la nueva clave
4. Auth Service expone ambas claves publicas en JWKS
5. Nuevos JWT se firman con la clave nueva
6. Los backends detectan el nuevo `kid` y descargan JWKS actualizado
7. Despues de que todos los JWT firmados con la clave anterior hayan expirado (60 min), se puede retirar la clave anterior del JWKS

### 4.3 Impacto en el Mapa de Permisos

La rotacion de claves invalida automaticamente los mapas de permisos firmados con la clave anterior:

1. Backends intentan verificar firma del mapa con clave anterior -> falla
2. Backends descargan JWKS actualizado
3. Backends descargan mapa completo nuevo (firmado con clave nueva)
4. Si ambos fallan -> conservar cache anterior; si cache anterior tambien es invalido -> denegar todos los accesos y alertar

---

## 5. Guia para el Tester

### Tests Unitarios Obligatorios

1. **Generacion JWT RS256:** Token generado con estructura correcta (header, payload, firma)
2. **Validacion JWT exitosa:** Token verificable con clave publica correspondiente
3. **Validacion JWT expirado:** Token con `exp` pasado retorna error
4. **Validacion JWT firma invalida:** Token firmado con clave diferente retorna error
5. **Validacion JWT kid desconocido:** Token con `kid` no presente en JWKS retorna error
6. **Claims completos:** Verificar que `sub`, `username`, `email`, `app`, `roles`, `iat`, `exp`, `jti` estan presentes
7. **jti unico:** Dos tokens generados consecutivamente tienen `jti` diferentes
8. **JWKS formato correcto:** Respuesta cumple RFC 7517
9. **JWKS multiples claves:** Durante rotacion, ambas claves estan presentes
10. **Refresh token hash:** Token almacenado nunca en texto plano
11. **Refresh token rotacion:** Token usado queda revocado, nuevo token es diferente
12. **Refresh token TTL web:** `client_type = "web"` -> expira a los 7 dias
13. **Refresh token TTL movil:** `client_type = "mobile"` -> expira a los 30 dias
14. **Refresh token TTL desktop:** `client_type = "desktop"` -> expira a los 30 dias
15. **Refresh token doble uso:** Segundo uso del mismo token falla
16. **Login con client_type invalido:** Retorna 400 `INVALID_CLIENT_TYPE`

### Casos de Borde

- Token generado exactamente en el limite de expiracion (borde de segundo)
- Rotacion de claves: JWT firmado con clave anterior sigue siendo valido
- Rotacion de claves: nuevo JWT firmado con clave nueva es valido
- Refresh token usado simultaneamente desde dos clientes (race condition)
- JWKS con clave anterior retirada: JWT firmado con ella ya no es valido
- Refresh token con `expires_at` exactamente en `NOW()` (borde de expiracion)

---

*Fin de especificacion token-management.md*
