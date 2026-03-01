# Especificaciones Detalladas de Tests de API - Sentinel

**Fecha de creacion:** 2026-02-28
**Autor:** senior-analyst (coordinado por team-lead)
**Basado en:** docs/plan/test-plan.md v1.0
**Total de tests:** 143 escenarios

---

## Convenciones

- **BASE_URL:** `http://localhost:8080`
- **APP_KEY:** Se obtiene del bootstrap (aplicacion sistema `sentinel`)
- **ADMIN_USER:** `admin`
- **ADMIN_PASS:** `Admin@Local1!` (fallback: `Admin@Sentinel2!`)
- **TEST_PASS:** `TestP@ssw0rd1!` (cumple politica: >= 10 chars, mayuscula, numero, simbolo)
- **NEW_PASS:** `NewP@ssw0rd2!`
- **Tokens:** Se truncan a 20 chars + "..." en reportes de salida
- **UUIDs de test:** Se usan UUIDs inexistentes como `00000000-0000-0000-0000-000000000000`

### Formato de Error Esperado

```json
{
  "error": {
    "code": "CODIGO_DE_ERROR",
    "message": "Descripcion legible",
    "details": null
  }
}
```

---

## SECCION 1: SISTEMA

---

### T-001 | GET /health | Servicio saludable

| Campo | Valor |
|-------|-------|
| **Test ID** | T-001 |
| **Endpoint** | `GET /health` |
| **Escenario** | Verificar que el servicio esta operativo con todas las dependencias |
| **Precondiciones** | Stack local corriendo (Go + PostgreSQL + Redis) |
| **Headers** | Ninguno requerido |
| **Request Body** | N/A |
| **Query Params** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"status":"healthy","version":"...","checks":{"postgresql":"ok","redis":"ok"}}` |
| **Criterio PASS** | HTTP 200 AND `.status == "healthy"` AND `.checks.postgresql == "ok"` AND `.checks.redis == "ok"` |
| **Criterio FAIL** | Status != 200 OR campos faltantes |

---

### T-002 | GET /.well-known/jwks.json | Claves publicas JWKS

| Campo | Valor |
|-------|-------|
| **Test ID** | T-002 |
| **Endpoint** | `GET /.well-known/jwks.json` |
| **Escenario** | Obtener claves publicas RSA para verificacion de JWT |
| **Precondiciones** | Servicio corriendo, claves RSA generadas en keys/ |
| **Headers** | Ninguno (endpoint publico) |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"keys":[{"kty":"RSA","alg":"RS256","use":"sig","kid":"...","n":"...","e":"AQAB"}]}` |
| **Criterio PASS** | HTTP 200 AND `.keys` es array no vacio AND `.keys[0].kty == "RSA"` AND `.keys[0].alg == "RS256"` AND `.keys[0].use == "sig"` |
| **Criterio FAIL** | Status != 200 OR `.keys` vacio OR `alg != "RS256"` |

---

### T-003 | GET /.well-known/jwks.json | Sin X-App-Key (debe funcionar)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-003 |
| **Endpoint** | `GET /.well-known/jwks.json` |
| **Escenario** | Endpoint publico: no requiere X-App-Key |
| **Precondiciones** | Servicio corriendo |
| **Headers** | Ninguno (deliberadamente sin X-App-Key) |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND respuesta contiene `.keys` |
| **Criterio FAIL** | Status 401 o cualquier error |

---

## SECCION 2: AUTENTICACION

---

### T-004 | POST /auth/login | Login exitoso (client_type=web)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-004 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Login con credenciales validas y client_type web |
| **Precondiciones** | Bootstrap completado, usuario admin existente |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"web"}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"access_token":"...","refresh_token":"...","token_type":"Bearer","expires_in":3600,"user":{"id":"...","username":"admin","email":"...","must_change_password":...}}` |
| **Criterio PASS** | HTTP 200 AND `.access_token` no vacio AND `.refresh_token` no vacio AND `.token_type == "Bearer"` AND `.expires_in > 0` AND `.user.username == "admin"` |
| **Criterio FAIL** | Status != 200 OR campos faltantes |

---

### T-005 | POST /auth/login | Login exitoso (client_type=mobile)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-005 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Login con client_type mobile (TTL 30 dias) |
| **Precondiciones** | Bootstrap completado |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"mobile"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.access_token` no vacio AND `.refresh_token` no vacio |
| **Criterio FAIL** | Status != 200 |

---

### T-006 | POST /auth/login | Login exitoso (client_type=desktop)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-006 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Login con client_type desktop (TTL 30 dias) |
| **Precondiciones** | Bootstrap completado |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"desktop"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.access_token` no vacio |
| **Criterio FAIL** | Status != 200 |

---

### T-007 | POST /auth/login | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-007 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Request sin header X-App-Key |
| **Precondiciones** | Servicio corriendo |
| **Headers** | `Content-Type: application/json` (sin X-App-Key) |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"web"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"APPLICATION_NOT_FOUND",...}}` |
| **Criterio PASS** | HTTP 401 AND `.error.code == "APPLICATION_NOT_FOUND"` |
| **Criterio FAIL** | Status != 401 |

---

### T-008 | POST /auth/login | X-App-Key invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-008 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | X-App-Key con valor falso |
| **Precondiciones** | Servicio corriendo |
| **Headers** | `X-App-Key: invalid-key-12345`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"web"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"APPLICATION_NOT_FOUND",...}}` |
| **Criterio PASS** | HTTP 401 AND `.error.code == "APPLICATION_NOT_FOUND"` |
| **Criterio FAIL** | Status != 401 |

---

### T-009 | POST /auth/login | Username incorrecto

| Campo | Valor |
|-------|-------|
| **Test ID** | T-009 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Username que no existe |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"nonexistent_user_xyz","password":"AnyP@ss123!","client_type":"web"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"INVALID_CREDENTIALS",...}}` |
| **Criterio PASS** | HTTP 401 AND `.error.code == "INVALID_CREDENTIALS"` |
| **Criterio FAIL** | Status != 401 |

---

### T-010 | POST /auth/login | Password incorrecto

| Campo | Valor |
|-------|-------|
| **Test ID** | T-010 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Password invalido para usuario existente |
| **Precondiciones** | APP_KEY valido, usuario admin existe |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"WrongP@ssw0rd!","client_type":"web"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"INVALID_CREDENTIALS",...}}` |
| **Criterio PASS** | HTTP 401 AND `.error.code == "INVALID_CREDENTIALS"` |
| **Criterio FAIL** | Status != 401 |

---

### T-011 | POST /auth/login | Body vacio

| Campo | Valor |
|-------|-------|
| **Test ID** | T-011 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Request sin body |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Body Esperado** | `{"error":{"code":"VALIDATION_ERROR",...}}` |
| **Criterio PASS** | HTTP 400 AND `.error.code` es `"VALIDATION_ERROR"` o similar |
| **Criterio FAIL** | Status != 400 |

---

### T-012 | POST /auth/login | client_type invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-012 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | client_type con valor no permitido |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}","client_type":"tablet"}` |
| **Respuesta Esperada** | Status: `400` |
| **Body Esperado** | `{"error":{"code":"INVALID_CLIENT_TYPE",...}}` |
| **Criterio PASS** | HTTP 400 AND `.error.code` contiene `"INVALID_CLIENT_TYPE"` o `"VALIDATION_ERROR"` |
| **Criterio FAIL** | Status != 400 |

---

### T-013 | POST /auth/login | client_type faltante

| Campo | Valor |
|-------|-------|
| **Test ID** | T-013 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Body sin campo client_type |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"admin","password":"{ADMIN_PASS}"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status != 400 |

---

### T-014 | POST /auth/refresh | Refresh exitoso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-014 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Renovar tokens con refresh token valido |
| **Precondiciones** | Login exitoso previo, refresh_token disponible |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"refresh_token":"{REFRESH_TOKEN}"}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"access_token":"...","refresh_token":"...","token_type":"Bearer","expires_in":3600}` |
| **Criterio PASS** | HTTP 200 AND `.access_token` no vacio AND `.refresh_token` no vacio AND nuevo refresh_token != anterior |
| **Criterio FAIL** | Status != 200 OR tokens vacios |

---

### T-015 | POST /auth/refresh | Refresh token invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-015 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Refresh token que no existe |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"refresh_token":"totally-invalid-refresh-token-12345"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"TOKEN_INVALID",...}}` |
| **Criterio PASS** | HTTP 401 AND `.error.code` contiene `"TOKEN_INVALID"` o `"TOKEN_REVOKED"` |
| **Criterio FAIL** | Status != 401 |

---

### T-016 | POST /auth/refresh | Body vacio

| Campo | Valor |
|-------|-------|
| **Test ID** | T-016 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Request sin refresh_token |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` o `401` |
| **Criterio PASS** | HTTP 400 o 401 |
| **Criterio FAIL** | Status 200 |

---

### T-017 | POST /auth/refresh | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-017 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Request sin header X-App-Key |
| **Precondiciones** | Refresh token valido |
| **Headers** | `Content-Type: application/json` (sin X-App-Key) |
| **Request Body** | `{"refresh_token":"{REFRESH_TOKEN}"}` |
| **Respuesta Esperada** | Status: `401` |
| **Body Esperado** | `{"error":{"code":"APPLICATION_NOT_FOUND",...}}` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-018 | POST /auth/logout | Logout exitoso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-018 |
| **Endpoint** | `POST /auth/logout` |
| **Escenario** | Cerrar sesion con token valido |
| **Precondiciones** | Login exitoso, access_token disponible |
| **Headers** | `Authorization: Bearer {ACCESS_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `204` |
| **Body Esperado** | Vacio |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-019 | POST /auth/logout | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-019 |
| **Endpoint** | `POST /auth/logout` |
| **Escenario** | Request sin header Authorization |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}` (sin Authorization) |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-020 | POST /auth/logout | Token invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-020 |
| **Endpoint** | `POST /auth/logout` |
| **Escenario** | Token JWT malformado |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `Authorization: Bearer invalid.jwt.token`, `X-App-Key: {APP_KEY}` |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-021 | POST /auth/change-password | Cambio exitoso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-021 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | Cambio de contrasena con datos validos |
| **Precondiciones** | Usuario de test creado y logueado, token disponible. Password actual conocido. |
| **Headers** | `Authorization: Bearer {TEST_USER_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"{CURRENT_PASS}","new_password":"{NEW_PASS}"}` |
| **Respuesta Esperada** | Status: `204` |
| **Body Esperado** | Vacio |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |
| **Post-condicion** | Verificar que se puede hacer login con la nueva contrasena |

---

### T-022 | POST /auth/change-password | Contrasena actual incorrecta

| Campo | Valor |
|-------|-------|
| **Test ID** | T-022 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | current_password no coincide |
| **Precondiciones** | Usuario logueado |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"WrongCurrent1!","new_password":"{NEW_PASS}"}` |
| **Respuesta Esperada** | Status: `400` o `401` |
| **Criterio PASS** | HTTP 400 o 401 |
| **Criterio FAIL** | Status 204 (cambio no deberia ocurrir) |

---

### T-023 | POST /auth/change-password | Nueva contrasena muy corta

| Campo | Valor |
|-------|-------|
| **Test ID** | T-023 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | new_password < 10 caracteres |
| **Precondiciones** | Usuario logueado |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"{CURRENT}","new_password":"Short1!"}` |
| **Respuesta Esperada** | Status: `400` |
| **Body Esperado** | Error de politica de contrasena |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-024 | POST /auth/change-password | Sin mayuscula

| Campo | Valor |
|-------|-------|
| **Test ID** | T-024 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | new_password sin letras mayusculas |
| **Precondiciones** | Usuario logueado |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"{CURRENT}","new_password":"nouppercase1!@"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-025 | POST /auth/change-password | Sin numero

| Campo | Valor |
|-------|-------|
| **Test ID** | T-025 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | new_password sin digitos numericos |
| **Precondiciones** | Usuario logueado |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"{CURRENT}","new_password":"NoNumbers!!@@AB"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-026 | POST /auth/change-password | Sin simbolo

| Campo | Valor |
|-------|-------|
| **Test ID** | T-026 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | new_password sin caracteres especiales |
| **Precondiciones** | Usuario logueado |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"current_password":"{CURRENT}","new_password":"NoSymbols1234AB"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-027 | POST /auth/change-password | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-027 |
| **Endpoint** | `POST /auth/change-password` |
| **Escenario** | Request sin token de autenticacion |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` (sin Authorization) |
| **Request Body** | `{"current_password":"any","new_password":"AnyP@ss1234!"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 3: AUTORIZACION

---

### T-028 | POST /authz/verify | Permiso concedido (allowed=true)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-028 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Verificar permiso que el admin tiene (admin.system.manage) |
| **Precondiciones** | Admin logueado, tiene permiso admin.system.manage |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission":"admin.system.manage"}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"allowed":true,"user_id":"...","username":"admin","permission":"admin.system.manage","evaluated_at":"..."}` |
| **Criterio PASS** | HTTP 200 AND `.allowed == true` AND `.username == "admin"` |
| **Criterio FAIL** | Status != 200 OR `.allowed != true` |

---

### T-029 | POST /authz/verify | Permiso denegado (allowed=false)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-029 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Verificar permiso inexistente |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission":"nonexistent.permission.xyz"}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"allowed":false,...}` |
| **Criterio PASS** | HTTP 200 AND `.allowed == false` |
| **Criterio FAIL** | Status != 200 OR `.allowed != false` |

---

### T-030 | POST /authz/verify | Con cost_center_id

| Campo | Valor |
|-------|-------|
| **Test ID** | T-030 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Verificacion con filtro de centro de costo |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission":"admin.system.manage","cost_center_id":"00000000-0000-0000-0000-000000000000"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND response contiene `.allowed` (true o false) |
| **Criterio FAIL** | Status != 200 |

---

### T-031 | POST /authz/verify | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-031 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Request sin token Bearer |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission":"admin.system.manage"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-032 | POST /authz/verify | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-032 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Request sin header X-App-Key |
| **Precondiciones** | Token valido |
| **Headers** | `Authorization: Bearer {TOKEN}`, `Content-Type: application/json` |
| **Request Body** | `{"permission":"admin.system.manage"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-033 | POST /authz/verify | Body vacio

| Campo | Valor |
|-------|-------|
| **Test ID** | T-033 |
| **Endpoint** | `POST /authz/verify` |
| **Escenario** | Request sin campo permission |
| **Precondiciones** | Token y APP_KEY validos |
| **Headers** | `Authorization: Bearer {TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

### T-034 | GET /authz/me/permissions | Permisos del usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-034 |
| **Endpoint** | `GET /authz/me/permissions` |
| **Escenario** | Obtener contexto de permisos del admin |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}` |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"user_id":"...","application":"...","roles":[...],"permissions":[...],"cost_centers":[...],"temporary_roles":[...]}` |
| **Criterio PASS** | HTTP 200 AND `.user_id` no vacio AND `.permissions` es array AND `.roles` es array |
| **Criterio FAIL** | Status != 200 OR campos faltantes |

---

### T-035 | GET /authz/me/permissions | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-035 |
| **Endpoint** | `GET /authz/me/permissions` |
| **Escenario** | Request sin token |
| **Precondiciones** | Ninguna |
| **Headers** | Ninguno |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-036 | GET /authz/permissions-map | Mapa de permisos firmado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-036 |
| **Endpoint** | `GET /authz/permissions-map` |
| **Escenario** | Obtener mapa completo firmado RSA-SHA256 |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | Response contiene `application`, `permissions`, `signature`, `generated_at`, `version` |
| **Criterio PASS** | HTTP 200 AND `.signature` no vacio AND `.permissions` es objeto AND `.version` no vacio |
| **Criterio FAIL** | Status != 200 OR `.signature` faltante |

---

### T-037 | GET /authz/permissions-map | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-037 |
| **Endpoint** | `GET /authz/permissions-map` |
| **Escenario** | Request sin X-App-Key |
| **Precondiciones** | Ninguna |
| **Headers** | Ninguno |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-038 | GET /authz/permissions-map | X-App-Key invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-038 |
| **Endpoint** | `GET /authz/permissions-map` |
| **Escenario** | X-App-Key con valor invalido |
| **Precondiciones** | Ninguna |
| **Headers** | `X-App-Key: fake-key-999` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-039 | GET /authz/permissions-map/version | Version del mapa

| Campo | Valor |
|-------|-------|
| **Test ID** | T-039 |
| **Endpoint** | `GET /authz/permissions-map/version` |
| **Escenario** | Obtener version/hash del mapa |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Request Body** | N/A |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"application":"...","version":"...","generated_at":"..."}` |
| **Criterio PASS** | HTTP 200 AND `.version` no vacio |
| **Criterio FAIL** | Status != 200 |

---

### T-040 | GET /authz/permissions-map/version | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-040 |
| **Endpoint** | `GET /authz/permissions-map/version` |
| **Escenario** | Request sin X-App-Key |
| **Precondiciones** | Ninguna |
| **Headers** | Ninguno |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 4: ADMIN - APLICACIONES

---

### T-041 | GET /admin/applications | Lista paginada default

| Campo | Valor |
|-------|-------|
| **Test ID** | T-041 |
| **Endpoint** | `GET /admin/applications` |
| **Escenario** | Listar aplicaciones sin parametros de paginacion |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Query Params** | Ninguno |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"data":[...],"page":1,"page_size":20,"total":...,"total_pages":...}` |
| **Criterio PASS** | HTTP 200 AND `.data` es array AND `.page == 1` AND `.page_size == 20` AND `.total >= 1` (al menos la app sistema) |
| **Criterio FAIL** | Status != 200 OR formato de paginacion incorrecto |

---

### T-042 | GET /admin/applications | Con paginacion custom

| Campo | Valor |
|-------|-------|
| **Test ID** | T-042 |
| **Endpoint** | `GET /admin/applications?page=1&page_size=5` |
| **Escenario** | Paginacion con parametros custom |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.page == 1` AND `.page_size == 5` |
| **Criterio FAIL** | `.page_size != 5` |

---

### T-043 | GET /admin/applications | Filtro search

| Campo | Valor |
|-------|-------|
| **Test ID** | T-043 |
| **Endpoint** | `GET /admin/applications?search=sentinel` |
| **Escenario** | Busqueda por nombre/slug |
| **Precondiciones** | Admin logueado, app sentinel existe |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.total >= 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-044 | GET /admin/applications | Filtro is_active

| Campo | Valor |
|-------|-------|
| **Test ID** | T-044 |
| **Endpoint** | `GET /admin/applications?is_active=true` |
| **Escenario** | Filtrar solo aplicaciones activas |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND todas las apps en `.data` tienen `.is_active == true` |
| **Criterio FAIL** | Alguna app con `.is_active == false` en los resultados |

---

### T-045 | GET /admin/applications | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-045 |
| **Endpoint** | `GET /admin/applications` |
| **Escenario** | Sin token de autenticacion |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-046 | GET /admin/applications | Sin X-App-Key

| Campo | Valor |
|-------|-------|
| **Test ID** | T-046 |
| **Endpoint** | `GET /admin/applications` |
| **Escenario** | Sin header X-App-Key |
| **Precondiciones** | Token valido |
| **Headers** | `Authorization: Bearer {TOKEN}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-047 | POST /admin/applications | Crear aplicacion

| Campo | Valor |
|-------|-------|
| **Test ID** | T-047 |
| **Endpoint** | `POST /admin/applications` |
| **Escenario** | Crear aplicacion con datos validos |
| **Precondiciones** | Admin logueado, slug `test-app-api` no existe |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Test App API","slug":"test-app-api"}` |
| **Respuesta Esperada** | Status: `201` |
| **Body Esperado** | `{"id":"...","name":"Test App API","slug":"test-app-api","is_active":true,"is_system":false,...}` |
| **Criterio PASS** | HTTP 201 AND `.slug == "test-app-api"` AND `.is_active == true` AND `.id` es UUID valido |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar ID para tests posteriores y para limpieza final |

---

### T-048 | POST /admin/applications | Slug duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-048 |
| **Endpoint** | `POST /admin/applications` |
| **Escenario** | Intentar crear app con slug que ya existe |
| **Precondiciones** | App `test-app-api` ya creada en T-047 |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Duplicate App","slug":"test-app-api"}` |
| **Respuesta Esperada** | Status: `409` |
| **Criterio PASS** | HTTP 409 |
| **Criterio FAIL** | Status 201 (no deberia permitir duplicados) |

---

### T-049 | POST /admin/applications | Datos faltantes

| Campo | Valor |
|-------|-------|
| **Test ID** | T-049 |
| **Endpoint** | `POST /admin/applications` |
| **Escenario** | Body sin campos requeridos |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-050 | POST /admin/applications | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-050 |
| **Endpoint** | `POST /admin/applications` |
| **Escenario** | Sin token |
| **Precondiciones** | APP_KEY valido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"No Auth App","slug":"no-auth-app"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-051 | GET /admin/applications/{id} | Obtener app existente

| Campo | Valor |
|-------|-------|
| **Test ID** | T-051 |
| **Endpoint** | `GET /admin/applications/{TEST_APP_ID}` |
| **Escenario** | Obtener detalle de app creada |
| **Precondiciones** | App test creada, ID conocido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.id == {TEST_APP_ID}` AND `.slug == "test-app-api"` |
| **Criterio FAIL** | Status != 200 |

---

### T-052 | GET /admin/applications/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-052 |
| **Endpoint** | `GET /admin/applications/00000000-0000-0000-0000-000000000000` |
| **Escenario** | UUID valido pero inexistente |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status != 404 |

---

### T-053 | GET /admin/applications/{id} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-053 |
| **Endpoint** | `GET /admin/applications/not-a-uuid` |
| **Escenario** | ID con formato invalido |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

### T-054 | PUT /admin/applications/{id} | Actualizar nombre

| Campo | Valor |
|-------|-------|
| **Test ID** | T-054 |
| **Endpoint** | `PUT /admin/applications/{TEST_APP_ID}` |
| **Escenario** | Actualizar nombre de la app de test |
| **Precondiciones** | App test creada |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Test App Updated"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.name == "Test App Updated"` |
| **Criterio FAIL** | Status != 200 |

---

### T-055 | PUT /admin/applications/{id} | App no encontrada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-055 |
| **Endpoint** | `PUT /admin/applications/00000000-0000-0000-0000-000000000000` |
| **Escenario** | Actualizar app inexistente |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Ghost App"}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status 200 |

---

### T-056 | POST /admin/applications/{id}/rotate-key | Rotar clave

| Campo | Valor |
|-------|-------|
| **Test ID** | T-056 |
| **Endpoint** | `POST /admin/applications/{TEST_APP_ID}/rotate-key` |
| **Escenario** | Rotar clave de app no-sistema |
| **Precondiciones** | App test creada (no es app de sistema) |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | Response contiene nueva clave secreta |
| **Criterio PASS** | HTTP 200 AND respuesta contiene campo de clave |
| **Criterio FAIL** | Status != 200 |

---

### T-057 | POST /admin/applications/{id}/rotate-key | App no encontrada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-057 |
| **Endpoint** | `POST /admin/applications/00000000-0000-0000-0000-000000000000/rotate-key` |
| **Escenario** | Rotar clave de app inexistente |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status 200 |

---

## SECCION 5: ADMIN - USUARIOS

---

### T-058 | GET /admin/users | Lista paginada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-058 |
| **Endpoint** | `GET /admin/users` |
| **Escenario** | Listar usuarios con paginacion default |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.data` es array AND `.page == 1` AND `.total >= 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-059 | GET /admin/users | Con search

| Campo | Valor |
|-------|-------|
| **Test ID** | T-059 |
| **Endpoint** | `GET /admin/users?search=admin` |
| **Escenario** | Busqueda por username |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.total >= 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-060 | GET /admin/users | Con is_active

| Campo | Valor |
|-------|-------|
| **Test ID** | T-060 |
| **Endpoint** | `GET /admin/users?is_active=true` |
| **Escenario** | Filtrar usuarios activos |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-061 | GET /admin/users | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-061 |
| **Endpoint** | `GET /admin/users` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-062 | POST /admin/users | Crear usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-062 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Crear usuario con datos validos |
| **Precondiciones** | Admin logueado, username `testuser_api` no existe |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"testuser_api","email":"testuser_api@test.com","password":"TestP@ssw0rd1!"}` |
| **Respuesta Esperada** | Status: `201` |
| **Body Esperado** | `{"id":"...","username":"testuser_api","email":"testuser_api@test.com","is_active":true,...}` |
| **Criterio PASS** | HTTP 201 AND `.username == "testuser_api"` AND `.is_active == true` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar ID del usuario creado |

---

### T-063 | POST /admin/users | Username duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-063 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Username ya existente |
| **Precondiciones** | testuser_api ya creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"testuser_api","email":"different@test.com","password":"TestP@ssw0rd1!"}` |
| **Respuesta Esperada** | Status: `400` o `409` |
| **Criterio PASS** | HTTP 400 o 409 |
| **Criterio FAIL** | Status 201 |

---

### T-064 | POST /admin/users | Email duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-064 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Email ya existente |
| **Precondiciones** | testuser_api ya creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"different_user","email":"testuser_api@test.com","password":"TestP@ssw0rd1!"}` |
| **Respuesta Esperada** | Status: `400` o `409` |
| **Criterio PASS** | HTTP 400 o 409 |
| **Criterio FAIL** | Status 201 |

---

### T-065 | POST /admin/users | Password no cumple politica

| Campo | Valor |
|-------|-------|
| **Test ID** | T-065 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Contrasena debil |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"weakpwd_user","email":"weak@test.com","password":"123"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-066 | POST /admin/users | Datos faltantes

| Campo | Valor |
|-------|-------|
| **Test ID** | T-066 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Body vacio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-067 | POST /admin/users | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-067 |
| **Endpoint** | `POST /admin/users` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"noauth","email":"noauth@test.com","password":"TestP@ssw0rd1!"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-068 | GET /admin/users/{id} | Obtener usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-068 |
| **Endpoint** | `GET /admin/users/{TEST_USER_ID}` |
| **Escenario** | Obtener detalle del usuario creado |
| **Precondiciones** | testuser_api creado, ID conocido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.username == "testuser_api"` |
| **Criterio FAIL** | Status != 200 |

---

### T-069 | GET /admin/users/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-069 |
| **Endpoint** | `GET /admin/users/00000000-0000-0000-0000-000000000000` |
| **Escenario** | UUID valido pero inexistente |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status != 404 |

---

### T-070 | GET /admin/users/{id} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-070 |
| **Endpoint** | `GET /admin/users/not-a-uuid` |
| **Escenario** | ID con formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

### T-071 | PUT /admin/users/{id} | Actualizar email

| Campo | Valor |
|-------|-------|
| **Test ID** | T-071 |
| **Endpoint** | `PUT /admin/users/{TEST_USER_ID}` |
| **Escenario** | Actualizar email del usuario |
| **Precondiciones** | testuser_api creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"email":"testuser_api_updated@test.com"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-072 | PUT /admin/users/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-072 |
| **Endpoint** | `PUT /admin/users/00000000-0000-0000-0000-000000000000` |
| **Escenario** | Actualizar usuario inexistente |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"email":"ghost@test.com"}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status 200 |

---

### T-073 | POST /admin/users/{id}/reset-password | Reset exitoso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-073 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/reset-password` |
| **Escenario** | Generar contrasena temporal |
| **Precondiciones** | testuser_api creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"temporary_password":"..."}` |
| **Criterio PASS** | HTTP 200 AND `.temporary_password` no vacio |
| **Criterio FAIL** | Status != 200 |

---

### T-074 | POST /admin/users/{id}/reset-password | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-074 |
| **Endpoint** | `POST /admin/users/not-a-uuid/reset-password` |
| **Escenario** | ID con formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

### T-075 | POST /admin/users/{id}/unlock | Desbloquear usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-075 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/unlock` |
| **Escenario** | Desbloquear cuenta (incluso si no esta bloqueada) |
| **Precondiciones** | testuser_api creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-076 | POST /admin/users/{id}/unlock | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-076 |
| **Endpoint** | `POST /admin/users/not-a-uuid/unlock` |
| **Escenario** | ID con formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

## SECCION 6: ADMIN - ROLES

---

### T-077 | GET /admin/roles | Lista paginada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-077 |
| **Endpoint** | `GET /admin/roles` |
| **Escenario** | Listar roles |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.data` es array AND `.page == 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-078 | GET /admin/roles | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-078 |
| **Endpoint** | `GET /admin/roles` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-079 | POST /admin/roles | Crear rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-079 |
| **Endpoint** | `POST /admin/roles` |
| **Escenario** | Crear rol de test |
| **Precondiciones** | Admin logueado, nombre `test-role-api` no existe |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"test-role-api","description":"Rol de prueba para tests de API"}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.name == "test-role-api"` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar ID del rol |

---

### T-080 | POST /admin/roles | Nombre duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-080 |
| **Endpoint** | `POST /admin/roles` |
| **Escenario** | Nombre que ya existe |
| **Precondiciones** | test-role-api ya creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"test-role-api","description":"Duplicado"}` |
| **Respuesta Esperada** | Status: `400` o `409` |
| **Criterio PASS** | HTTP 400 o 409 |
| **Criterio FAIL** | Status 201 |

---

### T-081 | POST /admin/roles | Datos faltantes

| Campo | Valor |
|-------|-------|
| **Test ID** | T-081 |
| **Endpoint** | `POST /admin/roles` |
| **Escenario** | Body vacio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-082 | POST /admin/roles | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-082 |
| **Endpoint** | `POST /admin/roles` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"no-auth-role"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-083 | GET /admin/roles/{id} | Obtener rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-083 |
| **Endpoint** | `GET /admin/roles/{TEST_ROLE_ID}` |
| **Escenario** | Obtener detalle del rol con permisos |
| **Precondiciones** | test-role-api creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.name == "test-role-api"` |
| **Criterio FAIL** | Status != 200 |

---

### T-084 | GET /admin/roles/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-084 |
| **Endpoint** | `GET /admin/roles/00000000-0000-0000-0000-000000000000` |
| **Escenario** | UUID inexistente |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status != 404 |

---

### T-085 | GET /admin/roles/{id} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-085 |
| **Endpoint** | `GET /admin/roles/not-a-uuid` |
| **Escenario** | Formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 200 |

---

### T-086 | PUT /admin/roles/{id} | Actualizar rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-086 |
| **Endpoint** | `PUT /admin/roles/{TEST_ROLE_ID}` |
| **Escenario** | Actualizar descripcion |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"description":"Descripcion actualizada para tests"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-087 | PUT /admin/roles/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-087 |
| **Endpoint** | `PUT /admin/roles/00000000-0000-0000-0000-000000000000` |
| **Escenario** | UUID inexistente |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"description":"Ghost"}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status 200 |

---

## SECCION 7: ADMIN - PERMISOS

---

### T-088 | GET /admin/permissions | Lista paginada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-088 |
| **Endpoint** | `GET /admin/permissions` |
| **Escenario** | Listar permisos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.data` es array AND `.page == 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-089 | GET /admin/permissions | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-089 |
| **Endpoint** | `GET /admin/permissions` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-090 | POST /admin/permissions | Crear permiso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-090 |
| **Endpoint** | `POST /admin/permissions` |
| **Escenario** | Crear permiso de test |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"test.api.read","description":"Permiso de prueba API","scope_type":"action"}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.code == "test.api.read"` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar ID del permiso |

---

### T-091 | POST /admin/permissions | Code duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-091 |
| **Endpoint** | `POST /admin/permissions` |
| **Escenario** | Code que ya existe |
| **Precondiciones** | test.api.read ya creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"test.api.read","description":"Duplicado","scope_type":"action"}` |
| **Respuesta Esperada** | Status: `400` o `409` |
| **Criterio PASS** | HTTP 400 o 409 |
| **Criterio FAIL** | Status 201 |

---

### T-092 | POST /admin/permissions | Datos faltantes

| Campo | Valor |
|-------|-------|
| **Test ID** | T-092 |
| **Endpoint** | `POST /admin/permissions` |
| **Escenario** | Body vacio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-093 | POST /admin/permissions | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-093 |
| **Endpoint** | `POST /admin/permissions` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"no.auth.perm","description":"x","scope_type":"action"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 8: ADMIN - CENTROS DE COSTO

---

### T-094 | GET /admin/cost-centers | Lista paginada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-094 |
| **Endpoint** | `GET /admin/cost-centers` |
| **Escenario** | Listar centros de costo |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.data` es array |
| **Criterio FAIL** | Status != 200 |

---

### T-095 | GET /admin/cost-centers | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-095 |
| **Endpoint** | `GET /admin/cost-centers` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-096 | POST /admin/cost-centers | Crear centro de costo

| Campo | Valor |
|-------|-------|
| **Test ID** | T-096 |
| **Endpoint** | `POST /admin/cost-centers` |
| **Escenario** | Crear CeCo de test |
| **Precondiciones** | Admin logueado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"TST-001","name":"Test CeCo API"}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.code == "TST-001"` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar ID |

---

### T-097 | POST /admin/cost-centers | Code duplicado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-097 |
| **Endpoint** | `POST /admin/cost-centers` |
| **Escenario** | Code que ya existe |
| **Precondiciones** | TST-001 ya creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"TST-001","name":"Duplicado"}` |
| **Respuesta Esperada** | Status: `400` o `409` |
| **Criterio PASS** | HTTP 400 o 409 |
| **Criterio FAIL** | Status 201 |

---

### T-098 | POST /admin/cost-centers | Datos faltantes

| Campo | Valor |
|-------|-------|
| **Test ID** | T-098 |
| **Endpoint** | `POST /admin/cost-centers` |
| **Escenario** | Body vacio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-099 | POST /admin/cost-centers | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-099 |
| **Endpoint** | `POST /admin/cost-centers` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"code":"TST-002","name":"No Auth"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-100 | PUT /admin/cost-centers/{id} | Actualizar CeCo

| Campo | Valor |
|-------|-------|
| **Test ID** | T-100 |
| **Endpoint** | `PUT /admin/cost-centers/{TEST_CECO_ID}` |
| **Escenario** | Actualizar nombre |
| **Precondiciones** | TST-001 creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Test CeCo Updated"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-101 | PUT /admin/cost-centers/{id} | ID no encontrado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-101 |
| **Endpoint** | `PUT /admin/cost-centers/00000000-0000-0000-0000-000000000000` |
| **Escenario** | UUID inexistente |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"Ghost"}` |
| **Respuesta Esperada** | Status: `404` |
| **Criterio PASS** | HTTP 404 |
| **Criterio FAIL** | Status 200 |

---

### T-102 | PUT /admin/cost-centers/{id} | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-102 |
| **Endpoint** | `PUT /admin/cost-centers/{TEST_CECO_ID}` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"name":"No Auth"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 9: ASIGNACIONES (Roles, Permisos, CeCos a Usuarios)

---

### T-103 | POST /admin/roles/{id}/permissions | Agregar permiso a rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-103 |
| **Endpoint** | `POST /admin/roles/{TEST_ROLE_ID}/permissions` |
| **Escenario** | Asignar permiso de test al rol de test |
| **Precondiciones** | test-role-api y test.api.read creados |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_ids":["{TEST_PERM_ID}"]}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.assigned >= 1` |
| **Criterio FAIL** | Status != 201 |

---

### T-104 | POST /admin/roles/{id}/permissions | Datos invalidos

| Campo | Valor |
|-------|-------|
| **Test ID** | T-104 |
| **Endpoint** | `POST /admin/roles/{TEST_ROLE_ID}/permissions` |
| **Escenario** | Array vacio o IDs invalidos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_ids":[]}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-105 | POST /admin/roles/{id}/permissions | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-105 |
| **Endpoint** | `POST /admin/roles/{TEST_ROLE_ID}/permissions` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_ids":["{TEST_PERM_ID}"]}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-106 | DELETE /admin/roles/{id}/permissions/{pid} | Remover permiso de rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-106 |
| **Endpoint** | `DELETE /admin/roles/{TEST_ROLE_ID}/permissions/{TEST_PERM_ID}` |
| **Escenario** | Remover el permiso asignado en T-103 |
| **Precondiciones** | Permiso asignado al rol |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-107 | DELETE /admin/roles/{id}/permissions/{pid} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-107 |
| **Endpoint** | `DELETE /admin/roles/not-a-uuid/permissions/not-a-uuid` |
| **Escenario** | IDs invalidos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-108 | POST /admin/users/{id}/roles | Asignar rol a usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-108 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/roles` |
| **Escenario** | Asignar test-role-api al testuser_api |
| **Precondiciones** | Ambos creados |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"role_id":"{TEST_ROLE_ID}"}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.role_id == "{TEST_ROLE_ID}"` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar assignment ID (`.id`) para revocacion |

---

### T-109 | POST /admin/users/{id}/roles | Datos invalidos

| Campo | Valor |
|-------|-------|
| **Test ID** | T-109 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/roles` |
| **Escenario** | role_id invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"role_id":"not-a-valid-uuid"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-110 | POST /admin/users/{id}/roles | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-110 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/roles` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"role_id":"{TEST_ROLE_ID}"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-111 | DELETE /admin/users/{id}/roles/{rid} | Revocar rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-111 |
| **Endpoint** | `DELETE /admin/users/{TEST_USER_ID}/roles/{ROLE_ASSIGNMENT_ID}` |
| **Escenario** | Revocar el rol asignado en T-108 |
| **Precondiciones** | Rol asignado, assignment ID conocido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-112 | DELETE /admin/users/{id}/roles/{rid} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-112 |
| **Endpoint** | `DELETE /admin/users/not-a-uuid/roles/not-a-uuid` |
| **Escenario** | IDs invalidos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-113 | POST /admin/users/{id}/permissions | Asignar permiso a usuario

| Campo | Valor |
|-------|-------|
| **Test ID** | T-113 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/permissions` |
| **Escenario** | Asignar test.api.read directamente al usuario |
| **Precondiciones** | testuser_api y test.api.read creados |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_id":"{TEST_PERM_ID}"}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.permission_id == "{TEST_PERM_ID}"` |
| **Criterio FAIL** | Status != 201 |
| **Cleanup** | Guardar assignment ID |

---

### T-114 | POST /admin/users/{id}/permissions | Datos invalidos

| Campo | Valor |
|-------|-------|
| **Test ID** | T-114 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/permissions` |
| **Escenario** | permission_id invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_id":"not-a-uuid"}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-115 | POST /admin/users/{id}/permissions | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-115 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/permissions` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"permission_id":"{TEST_PERM_ID}"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-116 | DELETE /admin/users/{id}/permissions/{pid} | Revocar permiso

| Campo | Valor |
|-------|-------|
| **Test ID** | T-116 |
| **Endpoint** | `DELETE /admin/users/{TEST_USER_ID}/permissions/{PERM_ASSIGNMENT_ID}` |
| **Escenario** | Revocar permiso asignado en T-113 |
| **Precondiciones** | Permiso asignado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-117 | DELETE /admin/users/{id}/permissions/{pid} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-117 |
| **Endpoint** | `DELETE /admin/users/not-a-uuid/permissions/not-a-uuid` |
| **Escenario** | IDs invalidos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-118 | POST /admin/users/{id}/cost-centers | Asignar CeCos

| Campo | Valor |
|-------|-------|
| **Test ID** | T-118 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/cost-centers` |
| **Escenario** | Asignar TST-001 al usuario |
| **Precondiciones** | testuser_api y TST-001 creados |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"cost_center_ids":["{TEST_CECO_ID}"]}` |
| **Respuesta Esperada** | Status: `201` |
| **Criterio PASS** | HTTP 201 AND `.assigned >= 1` |
| **Criterio FAIL** | Status != 201 |

---

### T-119 | POST /admin/users/{id}/cost-centers | Datos invalidos

| Campo | Valor |
|-------|-------|
| **Test ID** | T-119 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/cost-centers` |
| **Escenario** | Array vacio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"cost_center_ids":[]}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 201 |

---

### T-120 | POST /admin/users/{id}/cost-centers | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-120 |
| **Endpoint** | `POST /admin/users/{TEST_USER_ID}/cost-centers` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"cost_center_ids":["{TEST_CECO_ID}"]}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 10: ELIMINACIONES Y DESACTIVACIONES

---

### T-121 | DELETE /admin/roles/{id} | Eliminar (desactivar) rol

| Campo | Valor |
|-------|-------|
| **Test ID** | T-121 |
| **Endpoint** | `DELETE /admin/roles/{TEST_ROLE_ID}` |
| **Escenario** | Soft-delete del rol de test |
| **Precondiciones** | test-role-api creado, sin asignaciones activas |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-122 | DELETE /admin/roles/{id} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-122 |
| **Endpoint** | `DELETE /admin/roles/not-a-uuid` |
| **Escenario** | Formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-123 | DELETE /admin/roles/{id} | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-123 |
| **Endpoint** | `DELETE /admin/roles/{TEST_ROLE_ID}` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

### T-124 | DELETE /admin/permissions/{id} | Eliminar permiso (sin asignar)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-124 |
| **Endpoint** | `DELETE /admin/permissions/{TEST_PERM_ID}` |
| **Escenario** | Eliminar permiso no asignado a roles/usuarios |
| **Precondiciones** | test.api.read creado, todas las asignaciones revocadas |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` |
| **Criterio PASS** | HTTP 204 |
| **Criterio FAIL** | Status != 204 |

---

### T-125 | DELETE /admin/permissions/{id} | ID invalido

| Campo | Valor |
|-------|-------|
| **Test ID** | T-125 |
| **Endpoint** | `DELETE /admin/permissions/not-a-uuid` |
| **Escenario** | Formato invalido |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `400` |
| **Criterio PASS** | HTTP 400 |
| **Criterio FAIL** | Status 204 |

---

### T-126 | DELETE /admin/permissions/{id} | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-126 |
| **Endpoint** | `DELETE /admin/permissions/{TEST_PERM_ID}` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 11: AUDITORIA

---

### T-127 | GET /admin/audit-logs | Lista paginada

| Campo | Valor |
|-------|-------|
| **Test ID** | T-127 |
| **Endpoint** | `GET /admin/audit-logs` |
| **Escenario** | Listar logs de auditoria |
| **Precondiciones** | Admin logueado, al menos 1 evento generado (login) |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `{"data":[...],"page":1,"page_size":20,"total":...,"total_pages":...}` |
| **Criterio PASS** | HTTP 200 AND `.data` es array AND `.total >= 1` |
| **Criterio FAIL** | Status != 200 |

---

### T-128 | GET /admin/audit-logs | Filtro por event_type

| Campo | Valor |
|-------|-------|
| **Test ID** | T-128 |
| **Endpoint** | `GET /admin/audit-logs?event_type=LOGIN_SUCCESS` |
| **Escenario** | Filtrar por tipo LOGIN_SUCCESS |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND todos los items en `.data` tienen `.event_type` que contiene "LOGIN" |
| **Criterio FAIL** | Status != 200 |

---

### T-129 | GET /admin/audit-logs | Filtro por success

| Campo | Valor |
|-------|-------|
| **Test ID** | T-129 |
| **Endpoint** | `GET /admin/audit-logs?success=true` |
| **Escenario** | Filtrar eventos exitosos |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND todos los items en `.data` tienen `.success == true` |
| **Criterio FAIL** | Status != 200 |

---

### T-130 | GET /admin/audit-logs | Filtro por fechas

| Campo | Valor |
|-------|-------|
| **Test ID** | T-130 |
| **Endpoint** | `GET /admin/audit-logs?from_date=2026-01-01T00:00:00Z&to_date=2026-12-31T23:59:59Z` |
| **Escenario** | Filtrar por rango de fechas amplio |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-131 | GET /admin/audit-logs | Filtro por user_id

| Campo | Valor |
|-------|-------|
| **Test ID** | T-131 |
| **Endpoint** | `GET /admin/audit-logs?user_id={ADMIN_USER_ID}` |
| **Escenario** | Filtrar por ID del admin |
| **Precondiciones** | ID del admin obtenido del login |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 |
| **Criterio FAIL** | Status != 200 |

---

### T-132 | GET /admin/audit-logs | Sin Authorization

| Campo | Valor |
|-------|-------|
| **Test ID** | T-132 |
| **Endpoint** | `GET /admin/audit-logs` |
| **Escenario** | Sin token |
| **Headers** | `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status != 401 |

---

## SECCION 12: TESTS DE PAGINACION (transversales)

---

### T-133 | GET /admin/users | page_size > 100 se normaliza

| Campo | Valor |
|-------|-------|
| **Test ID** | T-133 |
| **Endpoint** | `GET /admin/users?page_size=200` |
| **Escenario** | page_size excede maximo, debe normalizarse a 100 |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.page_size == 100` |
| **Criterio FAIL** | `.page_size != 100` |

---

### T-134 | GET /admin/users | page=0 se normaliza a 1

| Campo | Valor |
|-------|-------|
| **Test ID** | T-134 |
| **Endpoint** | `GET /admin/users?page=0` |
| **Escenario** | page < 1 se normaliza |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.page == 1` |
| **Criterio FAIL** | `.page != 1` |

---

### T-135 | GET /admin/users | page > total_pages retorna data vacio

| Campo | Valor |
|-------|-------|
| **Test ID** | T-135 |
| **Endpoint** | `GET /admin/users?page=9999` |
| **Escenario** | Pagina fuera de rango |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.data` es array vacio AND `.total` es numerico |
| **Criterio FAIL** | Status != 200 OR `.data` no es array vacio |

---

## SECCION 13: TESTS DE REFRESH TOKEN USADO (rotacion)

---

### T-136 | POST /auth/refresh | Token ya usado (rotacion)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-136 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Usar refresh token que ya fue rotado en T-014 |
| **Precondiciones** | T-014 ya ejecutado, el refresh token original fue invalidado |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"refresh_token":"{OLD_REFRESH_TOKEN}"}` (el token usado en T-014) |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 AND `.error.code` contiene `"TOKEN_INVALID"` o `"TOKEN_REVOKED"` |
| **Criterio FAIL** | Status 200 (la rotacion no funciono) |

---

## SECCION 14: TESTS DE LOGOUT + POST-LOGOUT

---

### T-137 | POST /auth/refresh | Refresh despues de logout

| Campo | Valor |
|-------|-------|
| **Test ID** | T-137 |
| **Endpoint** | `POST /auth/refresh` |
| **Escenario** | Intentar refresh despues de logout (token revocado) |
| **Precondiciones** | Logout ejecutado en T-018, refresh token invalidado |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"refresh_token":"{LOGGED_OUT_REFRESH_TOKEN}"}` |
| **Respuesta Esperada** | Status: `401` |
| **Criterio PASS** | HTTP 401 |
| **Criterio FAIL** | Status 200 |

---

## SECCION 15: TESTS DE CAMBIO DE PASSWORD (con usuario de test)

---

### T-138 | POST /auth/login | Login con password cambiado

| Campo | Valor |
|-------|-------|
| **Test ID** | T-138 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Verificar que login funciona con la nueva contrasena despues del cambio |
| **Precondiciones** | T-021 ejecutado (cambio de contrasena exitoso) |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"testuser_api","password":"{NEW_PASS}","client_type":"web"}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.access_token` no vacio |
| **Criterio FAIL** | Status != 200 |

---

## SECCION 16: TESTS DE RESET PASSWORD

---

### T-139 | POST /auth/login | Login con password temporal

| Campo | Valor |
|-------|-------|
| **Test ID** | T-139 |
| **Endpoint** | `POST /auth/login` |
| **Escenario** | Login con la contrasena temporal generada por reset |
| **Precondiciones** | T-073 ejecutado, temporary_password conocido |
| **Headers** | `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"username":"testuser_api","password":"{TEMP_PASSWORD}","client_type":"web"}` |
| **Respuesta Esperada** | Status: `200` |
| **Body Esperado** | `.user.must_change_password == true` |
| **Criterio PASS** | HTTP 200 AND `.user.must_change_password == true` |
| **Criterio FAIL** | Status != 200 OR `.user.must_change_password != true` |

---

## SECCION 17: CLEANUP Y TESTS FINALES

---

### T-140 | DELETE /admin/permissions/{id} | Eliminar permiso con cleanup

| Campo | Valor |
|-------|-------|
| **Test ID** | T-140 |
| **Endpoint** | `DELETE /admin/permissions/{CLEANUP_PERM_ID}` |
| **Escenario** | Limpiar permiso creado durante tests (si no fue eliminado en T-124) |
| **Precondiciones** | Permiso sin asignaciones activas |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` o `404` (si ya fue eliminado) |
| **Criterio PASS** | HTTP 204 o 404 |
| **Criterio FAIL** | Status 500 |

---

### T-141 | DELETE /admin/roles/{id} | Eliminar rol con cleanup

| Campo | Valor |
|-------|-------|
| **Test ID** | T-141 |
| **Endpoint** | `DELETE /admin/roles/{CLEANUP_ROLE_ID}` |
| **Escenario** | Limpiar rol creado durante tests (si no fue eliminado en T-121) |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}` |
| **Respuesta Esperada** | Status: `204` o `404` |
| **Criterio PASS** | HTTP 204 o 404 |
| **Criterio FAIL** | Status 500 |

---

### T-142 | PUT /admin/users/{id} | Desactivar usuario de test (cleanup)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-142 |
| **Endpoint** | `PUT /admin/users/{TEST_USER_ID}` |
| **Escenario** | Desactivar el usuario de test creado |
| **Headers** | `Authorization: Bearer {ADMIN_TOKEN}`, `X-App-Key: {APP_KEY}`, `Content-Type: application/json` |
| **Request Body** | `{"is_active":false}` |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.is_active == false` |
| **Criterio FAIL** | Status != 200 |

---

### T-143 | GET /health | Health check final (post-tests)

| Campo | Valor |
|-------|-------|
| **Test ID** | T-143 |
| **Endpoint** | `GET /health` |
| **Escenario** | Verificar que el servicio sigue saludable despues de todos los tests |
| **Headers** | Ninguno |
| **Respuesta Esperada** | Status: `200` |
| **Criterio PASS** | HTTP 200 AND `.status == "healthy"` |
| **Criterio FAIL** | Status != 200 OR `.status != "healthy"` |

---

## Resumen Final

| Seccion | Tests | IDs |
|---------|-------|-----|
| Sistema | 3 | T-001 a T-003 |
| Autenticacion - Login | 10 | T-004 a T-013 |
| Autenticacion - Refresh | 4 | T-014 a T-017 |
| Autenticacion - Logout | 3 | T-018 a T-020 |
| Autenticacion - Change Password | 7 | T-021 a T-027 |
| Autorizacion | 13 | T-028 a T-040 |
| Admin - Aplicaciones | 17 | T-041 a T-057 |
| Admin - Usuarios | 15 | T-058 a T-072 |
| Admin - User Reset/Unlock | 4 | T-073 a T-076 |
| Admin - Roles | 11 | T-077 a T-087 |
| Admin - Permisos | 6 | T-088 a T-093 |
| Admin - CeCos | 9 | T-094 a T-102 |
| Asignaciones | 18 | T-103 a T-120 |
| Eliminaciones | 6 | T-121 a T-126 |
| Auditoria | 6 | T-127 a T-132 |
| Paginacion | 3 | T-133 a T-135 |
| Rotacion refresh | 1 | T-136 |
| Post-logout | 1 | T-137 |
| Post-change-pwd | 1 | T-138 |
| Post-reset-pwd | 1 | T-139 |
| Cleanup | 3 | T-140 a T-142 |
| Health final | 1 | T-143 |
| **TOTAL** | **143** | T-001 a T-143 |

---

## Orden de Ejecucion Recomendado

1. **Setup:** Obtener APP_KEY via bootstrap login
2. **T-001 a T-003:** Sistema
3. **T-004 a T-013:** Login
4. **T-014 a T-017:** Refresh
5. **T-018 a T-020:** Logout
6. **T-136:** Refresh token ya usado (rotacion)
7. **T-137:** Refresh post-logout
8. **T-088 a T-093:** Permisos CRUD (crear primero para asignar despues)
9. **T-077 a T-087:** Roles CRUD
10. **T-094 a T-102:** CeCos CRUD
11. **T-041 a T-057:** Aplicaciones CRUD
12. **T-058 a T-076:** Usuarios CRUD + reset/unlock
13. **T-103 a T-120:** Asignaciones (roles->permisos, usuarios->roles/permisos/cecos)
14. **T-028 a T-040:** Autorizacion (verify, me/permissions, map)
15. **T-021 a T-027:** Change password (con usuario de test)
16. **T-138:** Login post-change-password
17. **T-139:** Login post-reset-password
18. **T-127 a T-135:** Auditoria + paginacion
19. **T-116, T-111, T-106:** Revocaciones (limpiar asignaciones)
20. **T-121 a T-126:** Eliminaciones de roles y permisos
21. **T-140 a T-143:** Cleanup + health final

---

*Fin de especificaciones detalladas*
