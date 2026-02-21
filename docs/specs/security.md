# Especificacion Tecnica: Seguridad

**Referencia:** `docs/plan/auth-service-spec.md` secciones 7, 8, 10
**Historias relacionadas:** US-011, US-034, US-035, US-036, US-037, US-038

---

## 1. Hashing de Contrasenas

### Algoritmo: bcrypt

| Parametro | Valor |
|---|---|
| Algoritmo | bcrypt |
| Costo minimo | 12 |
| Libreria Go | `golang.org/x/crypto/bcrypt` |

**Reglas:**
- Toda contrasena se hashea con bcrypt costo >= 12 antes de almacenar
- Nunca se almacena una contrasena en texto plano
- Los refresh tokens tambien se almacenan como hash bcrypt

---

## 2. Politica de Contrasenas

### Requisitos

| Requisito | Valor |
|---|---|
| Longitud minima | 10 caracteres |
| Mayuscula | Al menos 1 |
| Numero | Al menos 1 |
| Simbolo | Al menos 1 caracter especial |
| Historial | No reutilizar las ultimas 5 |

### Caracteres Unicode

> **Decision de diseno (2026-02-21):** Se aceptan caracteres Unicode completos en contrasenas (acentos, caracteres no-ASCII, emojis). Antes de hashear con bcrypt, la contrasena se normaliza a **NFC (Canonical Decomposition followed by Canonical Composition)** para evitar ambiguedades entre representaciones equivalentes del mismo caracter. bcrypt opera sobre bytes, por lo que no hay impedimento tecnico.

**Procedimiento obligatorio:**
1. Recibir contrasena del request (UTF-8)
2. Normalizar a NFC: `password = unicode.NFC(password)` (en Go: `golang.org/x/text/unicode/norm`)
3. Aplicar validaciones de politica sobre la forma NFC
4. Hashear con bcrypt

Este procedimiento se aplica tanto al hashear como al comparar (login, cambio de contrasena, historial).

### Validacion (Regex sugerido)

```
Longitud:   len(password) >= 10  (contando code points, no bytes)
Mayuscula:  [A-Z]  (se evalua sobre la forma NFC)
Numero:     [0-9]
Simbolo:    cualquier caracter que no sea letra ni numero (incluye Unicode symbols)
```

### Historial de Contrasenas

- Al cambiar contrasena, el hash anterior se almacena en tabla `password_history`
- Al validar nueva contrasena, se comparan los ultimos 5 hashes con `bcrypt.CompareHashAndPassword`
- Si alguno coincide, se rechaza con error `PASSWORD_REUSED`

### Contextos de Aplicacion

La politica se aplica en:
1. `POST /auth/change-password` (cambio por el usuario)
2. `POST /admin/users` (creacion por admin)
3. `POST /admin/users/:id/reset-password` (la temporal generada debe cumplir la politica)

**Excepcion explicita:** La contrasena de bootstrap (`BOOTSTRAP_ADMIN_PASSWORD`) NO se valida contra la politica. Ver seccion 7.

---

## 3. Proteccion contra Fuerza Bruta

### Mecanismo de Bloqueo

```
Intento fallido:
  failed_attempts += 1

  Si failed_attempts >= 5:
    -- Reseteo diario automatico
    Si lockout_date != CURRENT_DATE:
      lockout_count = 0
      lockout_date = CURRENT_DATE

    lockout_count += 1

    Si lockout_count >= 3:
      locked_until = NULL  (bloqueo permanente, requiere admin)
      Registrar AUTH_ACCOUNT_LOCKED (permanente)
    Sino:
      locked_until = NOW() + 15 minutos
      Registrar AUTH_ACCOUNT_LOCKED

Login exitoso:
  failed_attempts = 0
  locked_until = NULL
  -- lockout_count y lockout_date NO se resetean en login exitoso
```

### Tabla de Comportamiento

| Intentos fallidos | Bloqueos hoy | Accion |
|---|---|---|
| 1-4 | 0 | Registrar `AUTH_LOGIN_FAILED`, continuar |
| 5 | 0 | Bloquear 15 min, registrar `AUTH_ACCOUNT_LOCKED` |
| 5 (tras desbloqueo) | 1 | Bloquear 15 min de nuevo |
| 5 (tras segundo desbloqueo) | 2 | Bloquear 15 min de nuevo |
| 5 (tercer bloqueo) | 3 | Bloqueo permanente |

### Desbloqueo

- **Automatico:** Despues de 15 minutos (`locked_until < NOW()`)
- **Manual:** Admin llama `POST /admin/users/:id/unlock`
- **Permanente:** Solo desbloqueo manual por admin

### Conteo de Bloqueos Diarios

> **Decision de diseno (2026-02-21):** El conteo de bloqueos diarios se almacena en columnas `lockout_count` (INTEGER DEFAULT 0) y `lockout_date` (DATE) en la tabla `users`. No se usa un job externo para el reseteo diario: la logica de negocio lo maneja en runtime.

**Columnas en `users`:**
- `lockout_count INT NOT NULL DEFAULT 0` -- numero de bloqueos en el dia
- `lockout_date DATE` -- fecha del ultimo bloqueo

**Logica de reseteo automatico en runtime:**

```
Al momento de evaluar un bloqueo (dentro del flujo de login fallido):

  Si lockout_date != CURRENT_DATE:
    lockout_count = 0
    lockout_date = CURRENT_DATE

  lockout_count += 1

  Si lockout_count >= 3:
    locked_until = NULL  (bloqueo permanente)
  Sino:
    locked_until = NOW() + 15 minutos
```

Esta logica se ejecuta **dentro del servicio de autenticacion** cada vez que `failed_attempts` alcanza el umbral de 5. No requiere un cron, job ni trigger de base de datos.

---

## 4. Headers de Seguridad HTTP

Middleware global que agrega a **todas las respuestas**:

| Header | Valor |
|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |

---

## 5. Autenticacion de Aplicaciones (X-App-Key)

### Flujo de Validacion

1. Extraer header `X-App-Key` del request
2. Buscar en tabla `applications` por `secret_key`
3. Verificar `is_active = true`
4. Si falla: retornar 401 `APPLICATION_NOT_FOUND`
5. Si exito: inyectar `application_id` en contexto del request

### Endpoints Exentos

| Endpoint | Razon |
|---|---|
| `GET /health` | Monitoreo sin autenticacion |
| `GET /.well-known/jwks.json` | Consumidores necesitan acceso libre |

---

## 6. Comunicacion

| Requisito | Detalle |
|---|---|
| Protocolo | HTTPS obligatorio |
| TLS minimo | 1.2 |
| Certificados | Gestionados en Azure (App Gateway / Front Door) |

**En desarrollo local:** HTTP permitido via Docker Compose. TLS no requerido.

---

## 7. Bootstrap del Sistema

### Condicion de Activacion

El bootstrap se ejecuta **una sola vez** cuando:
- No existen registros en tabla `applications`
- No existen registros en tabla `users`

### Contrasena del Bootstrap

> **Decision de diseno (2026-02-21):** La contrasena de bootstrap (`BOOTSTRAP_ADMIN_PASSWORD`) **no pasa por la validacion de politica de contrasenas**. La garantia de seguridad es el cambio forzado en el primer login (`must_change_pwd = true`), momento en el cual si se aplicara la politica completa. Esto permite al operador usar una contrasena simple para el primer arranque sin que el servicio falle por incumplimiento de politica.

### Procedimiento

1. Leer `BOOTSTRAP_ADMIN_USER` y `BOOTSTRAP_ADMIN_PASSWORD` de variables de entorno
2. Si no estan definidas: **fallo fatal** del servicio con mensaje claro
3. Crear aplicacion `system`:
   - `name`: "System"
   - `slug`: "system"
   - `secret_key`: generado aleatoriamente (mostrar en logs una sola vez)
4. Crear rol `admin`:
   - `name`: "admin"
   - `is_system`: true
   - Asignar todos los permisos de administracion (`admin.system.manage`, etc.)
5. Crear usuario administrador:
   - `username`: valor de `BOOTSTRAP_ADMIN_USER`
   - `password_hash`: bcrypt de `BOOTSTRAP_ADMIN_PASSWORD`
   - `must_change_pwd`: true
   - Asignar rol `admin` para aplicacion `system`
6. Registrar evento `SYSTEM_BOOTSTRAP` en auditoria
7. Marcar bootstrap como completado (flag en BD o en tabla de configuracion)

### Idempotencia

- Si ya existe al menos una aplicacion o usuario, el bootstrap **no se ejecuta**
- Multiples arranques del servicio no duplican datos

---

## 8. Graceful Shutdown

| Parametro | Valor |
|---|---|
| Timeout | 15 segundos (configurable via `server.graceful_shutdown_timeout`) |
| Senales | `SIGTERM`, `SIGINT` |

**Procedimiento:**
1. Recibir senal de shutdown
2. Dejar de aceptar nuevas conexiones
3. Esperar hasta 15 segundos para completar requests en vuelo
4. Cerrar pool de conexiones PostgreSQL
5. Cerrar conexion Redis
6. Salir con codigo 0

---

## 9. Proteccion de Inputs

| Amenaza | Mitigacion |
|---|---|
| SQL Injection | Queries parametrizadas (prepared statements). Nunca concatenar inputs en SQL. |
| XSS | Escapar datos de texto antes de almacenar. Encoding en respuestas JSON. |
| Secrets en logs | Nunca loguear passwords, tokens, secret keys. Sanitizar antes de escribir logs. |
| Secrets en codigo | Todos los secretos via variables de entorno. Nunca en archivos YAML committeados. |

---

## 10. Guia para el Tester

### Tests Obligatorios

1. **Bloqueo a 5 intentos:** 5 logins fallidos -> cuenta bloqueada 15 min
2. **Desbloqueo automatico:** Despues de 15 min, login exitoso
3. **Bloqueo permanente a 3 bloqueos diarios:** Tercer bloqueo no tiene `locked_until`
4. **Desbloqueo manual:** Admin desbloquea cuenta permanentemente bloqueada
5. **Politica de contrasena - longitud:** 9 chars -> rechazado; 10 chars -> aceptado
6. **Politica de contrasena - mayuscula:** sin mayuscula -> rechazado
7. **Politica de contrasena - numero:** sin numero -> rechazado
8. **Politica de contrasena - simbolo:** sin simbolo -> rechazado
9. **Historial de contrasenas:** reutilizar contrasena reciente -> rechazado
10. **Headers de seguridad:** Verificar presencia en todas las respuestas
11. **X-App-Key invalido:** Retorna 401
12. **X-App-Key de app inactiva:** Retorna 401
13. **Health sin X-App-Key:** Retorna 200 (exento)
14. **JWKS sin X-App-Key:** Retorna 200 (exento)
15. **Bootstrap BD vacia:** Crea app system + admin
16. **Bootstrap BD con datos:** No se ejecuta
17. **Bootstrap sin env vars:** Fallo fatal con mensaje claro
18. **Graceful shutdown:** Requests en vuelo se completan antes de cerrar
19. **SQL Injection:** Intentar inyeccion en campos de login, busqueda de usuarios
20. **Secrets en logs:** Verificar que passwords/tokens no aparecen en stdout/stderr

### Casos de Borde

- Login exactamente cuando `locked_until` expira (mismo segundo)
- Contrasena con exactamente 10 caracteres que cumple todos los requisitos
- Contrasena con caracteres Unicode (acentos, emojis): se aceptan, normalizar a NFC antes de hash/comparacion
- Contrasena con representacion Unicode no-NFC: debe producir mismo hash que la forma NFC equivalente
- Bootstrap con `BOOTSTRAP_ADMIN_PASSWORD` que no cumple politica de contrasena: se acepta (excepcion confirmada)
- Shutdown durante una transaccion de base de datos
- `X-App-Key` con caracteres especiales o muy largo

---

*Fin de especificacion security.md*
