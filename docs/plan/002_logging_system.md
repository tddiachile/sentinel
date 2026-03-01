# Plan de Trabajo — Sistema de Logging Estructurado
## Sentinel Auth Service

**Version:** 1.0.0
**Fecha:** Marzo 2026
**Autor:** Equipo de Ingenieria
**Estado:** Completado

---

## 1. Objetivo

Reemplazar todas las llamadas a `log` (stdlib) dispersas en el backend de Sentinel por un sistema de logging estructurado que:

- Emita logs en formato JSON en produccion y texto legible en desarrollo.
- Soporte niveles configurables (DEBUG, INFO, WARN, ERROR).
- Propague un **request ID** a traves de toda la cadena de una peticion HTTP (correlation ID).
- Sea inyectable como dependencia (no singleton global) para facilitar testing.
- Se integre nativamente con Fiber v2 como middleware de logging de requests.
- No altere el comportamiento del **audit log de negocio** existente (`audit_service.go`), que es un sistema separado.

---

## 2. Diagnostico del Estado Actual

### 2.1 Uso actual de `log` (stdlib)

El proyecto tiene **25 llamadas** a `log.Printf`, `log.Println`, `log.Fatalf` y `log.Fatal` distribuidas en 3 archivos:

| Archivo | Llamadas | Tipos |
|---|---|---|
| `cmd/server/main.go` | 17 | `Fatalf` (8), `Println` (4), `Printf` (5) |
| `internal/bootstrap/initializer.go` | 6 | `Println` (3), `Printf` (3) |
| `internal/service/audit_service.go` | 2 | `Printf` (2) |

### 2.2 Middleware Fiber actual

En `cmd/server/main.go` linea 201-203, se usa el middleware `logger` de Fiber con un formato JSON hardcodeado:

```go
app.Use(logger.New(logger.Config{
    Format: `{"time":"${time}","status":${status},"latency":"${latency}","method":"${method}","path":"${path}"}` + "\n",
}))
```

Este middleware no:
- Genera ni propaga un request ID.
- Se conecta con un logger estructurado.
- Incluye campos como `user_id`, `app_id` o `error`.

### 2.3 Configuracion existente

Ya existe un `LoggingConfig` en `internal/config/config.go` (lineas 84-87):

```go
type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
}
```

Y en `config.yaml` (lineas 39-41):

```yaml
logging:
  level: info
  format: json
```

Con defaults aplicados en `Load()` (lineas 166-171). **Esta configuracion no se usa actualmente en ningun lugar del codigo**.

### 2.4 Archivos que NO usan `log` pero deberian tener logging

Los siguientes componentes manejan errores silenciosamente (retornan `error` pero no registran nada):

- `internal/handler/auth_handler.go` -- errores mapeados a HTTP sin registrar el error original.
- `internal/handler/admin_handler.go` -- ~40 endpoints que usan `respondError()` sin logging.
- `internal/middleware/permission_middleware.go` -- fallos de autorizacion silenciosos.
- `internal/middleware/app_key_middleware.go` -- app key invalida sin logging.
- `internal/middleware/jwt_middleware.go` -- tokens invalidos/expirados sin logging.
- `internal/repository/postgres/*.go` (13 archivos) -- queries que fallan sin contexto.
- `internal/repository/redis/*.go` (2 archivos) -- operaciones Redis silenciosas.
- `internal/token/jwt.go` -- errores de firma/validacion sin logging.

---

## 3. Decisiones Tecnicas

### 3.1 Libreria: `log/slog` (stdlib Go 1.21+)

**Decision:** Usar `log/slog` de la biblioteca estandar.

**Justificacion:**

| Criterio | `log/slog` | `zap` | `zerolog` |
|---|---|---|---|
| Dependencias externas | 0 (stdlib) | +1 | +1 |
| Rendimiento | Excelente (zero-alloc para niveles deshabilitados) | Excelente | Excelente |
| API estandar Go | Si (desde Go 1.21) | No | No |
| Madurez | Estandar del lenguaje | Madura | Madura |
| Soporte JSON/Text | Si (handlers intercambiables) | Si | Si (JSON nativo) |
| Compatibilidad futura | Garantizada (stdlib) | Depende del mantenedor | Depende del mantenedor |
| Go 1.24 (version del proyecto) | Soportado nativamente | Si | Si |

El proyecto usa **Go 1.24.0** (segun `go.mod`), por lo tanto `log/slog` esta disponible sin dependencias adicionales. Dado que Sentinel no tiene requisitos de logging de ultra-alto rendimiento (hasta 2.000 usuarios activos), las tres opciones son equivalentes en performance. Se elige `slog` por: cero dependencias, API estandar, y garantia de mantenimiento a largo plazo.

### 3.2 Formato de salida

| Entorno | Handler | Formato |
|---|---|---|
| Produccion (`format: json`) | `slog.NewJSONHandler` | JSON una linea por evento |
| Desarrollo (`format: text`) | `slog.NewTextHandler` | Texto legible con colores (key=value) |

El handler se selecciona segun `cfg.Logging.Format` al inicializar el logger.

### 3.3 Niveles de log

| Nivel slog | Uso en Sentinel |
|---|---|
| `DEBUG` | Detalle de queries, cache hits/misses, flujos internos |
| `INFO` | Startup, shutdown, conexiones establecidas, bootstrap |
| `WARN` | Canal audit lleno, timeouts de Redis, degradaciones |
| `ERROR` | Fallos de DB, errores de firma JWT, panics recuperados |

El nivel se configura via `cfg.Logging.Level` (ya soporta "debug", "info", "warn", "error").

### 3.4 Arquitectura del Logger

```
                   +------------------+
                   |   config.yaml    |
                   | logging.level    |
                   | logging.format   |
                   +--------+---------+
                            |
                   +--------v---------+
                   |  internal/logger  |
                   |   logger.go       |
                   |                   |
                   |  New(cfg) *slog   |
                   |  FromContext(ctx) |
                   |  WithRequestID() |
                   +--------+---------+
                            |
              +-------------+-------------+
              |             |             |
     +--------v---+  +-----v------+  +---v----------+
     | middleware/ |  |  service/  |  | repository/  |
     | request_   |  | (recibe    |  | (recibe      |
     | logger.go  |  |  *slog via |  |  *slog via   |
     |            |  |  struct)   |  |  struct)      |
     +------------+  +------------+  +--------------+
```

**Patron de inyeccion:** El `*slog.Logger` se crea una vez en `main.go` y se pasa como dependencia a servicios y repositorios que lo necesiten. Para la capa HTTP, se almacena en `fiber.Ctx.Locals()` enriquecido con el request ID.

### 3.5 Request ID (Correlation ID)

- Generacion: UUID v4 via `github.com/google/uuid` (ya es dependencia del proyecto).
- Header: `X-Request-ID` (estandar de facto).
- Propagacion: Se genera en el middleware `RequestID`, se inyecta en `c.Locals("request_id", ...)` y se agrega al response header.
- El logger contextual incluye el campo `request_id` automaticamente.

### 3.6 Campos estructurados estandar

Cada log entry debe incluir los campos que correspondan de la siguiente tabla:

| Campo | Tipo | Presente en | Descripcion |
|---|---|---|---|
| `time` | RFC3339 | todos | Timestamp del evento |
| `level` | string | todos | DEBUG/INFO/WARN/ERROR |
| `msg` | string | todos | Mensaje del evento |
| `request_id` | string | HTTP | UUID de correlacion |
| `method` | string | HTTP | GET, POST, etc. |
| `path` | string | HTTP | Ruta de la peticion |
| `status` | int | HTTP response | Codigo HTTP de respuesta |
| `latency_ms` | float64 | HTTP response | Duracion en milisegundos |
| `ip` | string | HTTP | IP del cliente |
| `user_id` | string | HTTP autenticado | UUID del usuario (si hay JWT) |
| `app_id` | string | HTTP con X-App-Key | UUID de la aplicacion |
| `component` | string | todos | "server", "bootstrap", "auth", etc. |
| `error` | string | solo en errores | Mensaje de error |

---

## 4. Tareas Tecnicas

### Tarea 1: Crear paquete `internal/logger`

**Descripcion:** Crear el paquete central de logging que inicializa y configura `slog`.

**Archivos a crear:**
- `internal/logger/logger.go`

**Contenido esperado:**

```go
package logger

// New crea un *slog.Logger configurado segun LoggingConfig.
// - format "json" -> slog.NewJSONHandler(os.Stdout, opts)
// - format "text" -> slog.NewTextHandler(os.Stdout, opts)
// - level se mapea a slog.Level (debug=-4, info=0, warn=4, error=8).
func New(cfg config.LoggingConfig) *slog.Logger

// ParseLevel convierte un string "debug"/"info"/"warn"/"error" a slog.Level.
func ParseLevel(level string) slog.Level

// WithComponent retorna un logger hijo con el campo "component" agregado.
func WithComponent(logger *slog.Logger, component string) *slog.Logger
```

**Criterios de aceptacion:**
- `New()` retorna un `*slog.Logger` funcional con el handler correcto segun `format`.
- El nivel de log filtra correctamente (ej: nivel "warn" no emite "info").
- `WithComponent("auth")` agrega `"component":"auth"` a todos los logs del logger hijo.
- Tests unitarios para: `ParseLevel` con todos los niveles validos + fallback a INFO para valores desconocidos.
- Tests unitarios para: `New` con format "json" y "text".

**Estimacion:** 2 horas

---

### Tarea 2: Crear middleware `RequestID`

**Descripcion:** Middleware Fiber que genera o propaga un request ID en cada peticion.

**Archivos a crear:**
- `internal/middleware/request_id_middleware.go`

**Comportamiento:**
1. Leer header `X-Request-ID` de la peticion entrante.
2. Si esta vacio, generar un UUID v4 con `uuid.New().String()`.
3. Almacenar en `c.Locals("request_id", requestID)`.
4. Agregar header `X-Request-ID` a la respuesta.
5. Continuar con `c.Next()`.

**Constante a agregar en `audit_middleware.go`:**
```go
const LocalRequestID = "request_id"
```

**Criterios de aceptacion:**
- Si el request trae `X-Request-ID`, se reutiliza (no se genera uno nuevo).
- Si no trae header, se genera un UUID v4 valido.
- El response siempre incluye el header `X-Request-ID`.
- El valor esta disponible en `c.Locals("request_id")` para otros middlewares.
- Test unitario con y sin header entrante.

**Estimacion:** 1 hora

---

### Tarea 3: Crear middleware `RequestLogger` (reemplazo del logger Fiber)

**Descripcion:** Reemplazar el middleware `logger.New()` de Fiber por un middleware custom que use `slog`.

**Archivos a crear:**
- `internal/middleware/request_logger_middleware.go`

**Archivos a modificar:**
- `cmd/server/main.go` -- reemplazar `app.Use(logger.New(...))` por `app.Use(middleware.RequestLogger(log))`

**Comportamiento:**
1. Capturar `time.Now()` al inicio.
2. Ejecutar `c.Next()`.
3. Calcular latencia.
4. Extraer: method, path, status, ip, request_id, user_id (si existe), app_id (si existe).
5. Elegir nivel: status >= 500 -> ERROR, status >= 400 -> WARN, default -> INFO.
6. Emitir un log estructurado con todos los campos.

**Formato de salida ejemplo (JSON):**
```json
{
  "time": "2026-03-01T12:00:00Z",
  "level": "INFO",
  "msg": "HTTP request",
  "request_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "method": "POST",
  "path": "/auth/login",
  "status": 200,
  "latency_ms": 42.5,
  "ip": "192.168.1.100",
  "component": "http"
}
```

**Criterios de aceptacion:**
- Todos los campos de la tabla 3.6 relevantes a HTTP estan presentes.
- Rutas `/health` y `/swagger/*` se loguean a nivel DEBUG (para evitar ruido).
- Status >= 500 se loguea como ERROR.
- Status >= 400 se loguea como WARN.
- El campo `user_id` solo aparece si hay un JWT valido en Locals.
- El campo `app_id` solo aparece si hay un X-App-Key valido en Locals.
- Test unitario que verifica los campos del log emitido.

**Estimacion:** 3 horas

---

### Tarea 4: Migrar `cmd/server/main.go`

**Descripcion:** Reemplazar todas las llamadas a `log` (stdlib) en main.go por el logger estructurado.

**Archivos a modificar:**
- `cmd/server/main.go`

**Cambios especificos:**

1. **Crear logger** justo despues de cargar la configuracion:
   ```go
   appLogger := logger.New(cfg.Logging)
   ```

2. **Redirigir slog default** (para capturar libs que usen `slog.Default()`):
   ```go
   slog.SetDefault(appLogger)
   ```

3. **Reemplazar 17 llamadas a `log.*`:**

   | Linea actual | Cambio |
   |---|---|
   | `log.Fatalf("FATAL: %v", err)` | `appLogger.Error("config load failed", "error", err); os.Exit(1)` |
   | `log.Println("INFO: PostgreSQL connection pool established")` | `appLogger.Info("PostgreSQL connection pool established", "component", "database")` |
   | `log.Fatalf("FATAL: cannot connect to PostgreSQL: %v", err)` | `appLogger.Error("cannot connect to PostgreSQL", "error", err, "component", "database"); os.Exit(1)` |
   | ... (aplicar patron equivalente a las 17 llamadas) | |

4. **Reemplazar middleware de Fiber:**
   ```go
   // ANTES:
   app.Use(logger.New(logger.Config{...}))

   // DESPUES:
   app.Use(middleware.RequestID())
   app.Use(middleware.RequestLogger(appLogger))
   ```

5. **Remover import** de `"log"` y del middleware `"github.com/gofiber/fiber/v2/middleware/logger"`.

6. **Pasar logger a servicios** que lo requieran (ver Tarea 6).

**Criterios de aceptacion:**
- Cero imports de `"log"` en `cmd/server/main.go`.
- Cero imports de `"github.com/gofiber/fiber/v2/middleware/logger"`.
- El servidor arranca y emite logs estructurados en formato JSON (o text segun config).
- Los mensajes fatales (conexion a DB fallida, etc.) terminan el proceso con `os.Exit(1)`.
- El shutdown graceful emite logs correctos.
- Todas las rutas HTTP generan un log con request_id.

**Estimacion:** 3 horas

---

### Tarea 5: Migrar `internal/bootstrap/initializer.go`

**Descripcion:** Reemplazar las 6 llamadas a `log.*` en el initializer por `slog`.

**Archivos a modificar:**
- `internal/bootstrap/initializer.go`

**Cambios:**
1. Agregar `logger *slog.Logger` al struct `Initializer`.
2. Actualizar `NewInitializer()` para recibir el logger.
3. Crear logger hijo: `logger.With("component", "bootstrap")`.
4. Reemplazar las 6 llamadas:

   | Actual | Nuevo |
   |---|---|
   | `log.Println("INFO: bootstrap skipped...")` | `i.logger.Info("bootstrap skipped, system already initialized")` |
   | `log.Println("INFO: starting system bootstrap")` | `i.logger.Info("starting system bootstrap")` |
   | `log.Printf("INFO: system application created...")` | `i.logger.Info("system application created", "secret_key_hint", secretKey[:8]+"...")` |
   | `log.Printf("INFO: admin user created...")` | `i.logger.Info("admin user created", "username", adminUser.Username)` |
   | `log.Printf("WARN: bootstrap audit log failed...")` | `i.logger.Warn("bootstrap audit log failed", "error", err)` |
   | `log.Println("INFO: system bootstrap completed...")` | `i.logger.Info("system bootstrap completed")` |

5. Actualizar la llamada en `main.go`:
   ```go
   initializer := bootstrap.NewInitializer(..., appLogger)
   ```

**NOTA DE SEGURIDAD:** Nunca loguear la secret key completa. Solo loguear un hint (primeros 8 caracteres + "...").

**Criterios de aceptacion:**
- Cero imports de `"log"` en `initializer.go`.
- El secret key se loguea truncado (max 8 chars + "...").
- Los logs de bootstrap incluyen `"component":"bootstrap"`.
- El constructor `NewInitializer` acepta `*slog.Logger`.

**Estimacion:** 1.5 horas

---

### Tarea 6: Migrar `internal/service/audit_service.go`

**Descripcion:** Reemplazar las 2 llamadas a `log.*` en el servicio de audit por `slog`.

**Archivos a modificar:**
- `internal/service/audit_service.go`

**Cambios:**
1. Agregar `logger *slog.Logger` al struct `AuditService`.
2. Actualizar `NewAuditService()` para recibir el logger.
3. Crear logger hijo: `logger.With("component", "audit")`.
4. Reemplazar las 2 llamadas:

   | Actual | Nuevo |
   |---|---|
   | `log.Printf("AUDIT_ERROR: failed to insert...")` | `s.logger.Error("failed to insert audit log", "event_type", entry.EventType, "error", err)` |
   | `log.Printf("AUDIT_WARN: audit channel full...")` | `s.logger.Warn("audit channel full, dropping event", "event_type", entry.EventType)` |

5. Actualizar la llamada en `main.go`:
   ```go
   auditSvc := service.NewAuditService(auditRepo, appLogger)
   ```

**Criterios de aceptacion:**
- Cero imports de `"log"` en `audit_service.go`.
- Los logs incluyen `"component":"audit"` y `"event_type"` como campo estructurado.
- El comportamiento asincrono del canal no se altera.

**Estimacion:** 1 hora

---

### Tarea 7: Agregar logging a handlers (capa de errores)

**Descripcion:** Agregar logging de errores en los handlers para que los errores internos no se pierdan silenciosamente.

**Archivos a modificar:**
- `internal/handler/auth_handler.go`
- `internal/handler/admin_handler.go`
- `internal/handler/authz_handler.go`

**Enfoque:**
1. Agregar `logger *slog.Logger` a `AuthHandler`, `AdminHandler`, `AuthzHandler`.
2. Actualizar constructores para recibir el logger.
3. En `mapAuthError()` y en cada punto donde se retorna un error 500, loguear el error original:

   ```go
   // Ejemplo en mapAuthError:
   default:
       h.logger.Error("unhandled auth error",
           "error", err,
           "request_id", c.Locals("request_id"),
       )
       return respondError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
   ```

4. No loguear errores de validacion (400) ni de autenticacion (401) como ERROR -- estos son flujo normal. Loguearlos como DEBUG.

**Reglas de nivel por tipo de error:**
| HTTP Status | Nivel slog | Ejemplo |
|---|---|---|
| 400 | DEBUG | Validacion fallida, body invalido |
| 401 | DEBUG | Token invalido, credenciales incorrectas |
| 403 | INFO | Permiso denegado (potencial intento no autorizado) |
| 404 | DEBUG | Recurso no encontrado |
| 409 | INFO | Conflicto (duplicate key, etc.) |
| 500 | ERROR | Error interno no manejado |

**Criterios de aceptacion:**
- Todos los errores 500 se loguean con nivel ERROR incluyendo el error original.
- Los errores 400/401 se loguean con nivel DEBUG.
- Los errores 403 se loguean con nivel INFO.
- El `request_id` se incluye en todos los logs de error.
- Los constructores de handlers aceptan `*slog.Logger`.
- Se actualizan las llamadas en `main.go`.

**Estimacion:** 4 horas

---

### Tarea 8: Agregar logging a middlewares

**Descripcion:** Agregar logging contextual a los middlewares de seguridad.

**Archivos a modificar:**
- `internal/middleware/jwt_middleware.go`
- `internal/middleware/app_key_middleware.go`
- `internal/middleware/permission_middleware.go`

**Cambios:**

**jwt_middleware.go:**
- El middleware pasa a recibir un `*slog.Logger` como parametro adicional.
- Log DEBUG cuando un token es validado exitosamente.
- Log WARN cuando un token es invalido o expirado (potencial ataque).

**app_key_middleware.go:**
- Log WARN cuando un X-App-Key es invalido (potencial acceso no autorizado).
- Log DEBUG para X-App-Key valido.

**permission_middleware.go:**
- Log INFO cuando un permiso es denegado (incluir user_id, permission_code).
- Log DEBUG cuando un permiso es concedido.

**Criterios de aceptacion:**
- Tokens expirados/invalidos se loguean como WARN con IP del cliente.
- App keys invalidas se loguean como WARN.
- Permisos denegados se loguean como INFO con user_id y permission_code.
- Todos los logs incluyen `request_id` si esta disponible en Locals.
- Las funciones middleware actualizadas aceptan `*slog.Logger`.
- Se actualizan las llamadas en `main.go`.

**Estimacion:** 3 horas

---

### Tarea 9: Agregar logging al error handler global de Fiber

**Descripcion:** Mejorar el `ErrorHandler` de Fiber para loguear errores no manejados.

**Archivos a modificar:**
- `cmd/server/main.go` (dentro de `fiber.New(fiber.Config{...})`)

**Cambios:**
El `ErrorHandler` actual (lineas 172-188) retorna JSON pero no loguea nada. Agregar:

```go
ErrorHandler: func(c *fiber.Ctx, err error) error {
    requestID, _ := c.Locals("request_id").(string)
    code := fiber.StatusInternalServerError
    msg := "internal server error"
    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
        msg = e.Message
    }
    if code >= 500 {
        appLogger.Error("unhandled server error",
            "error", err,
            "status", code,
            "path", c.Path(),
            "method", c.Method(),
            "request_id", requestID,
        )
    }
    return c.Status(code).JSON(fiber.Map{...})
}
```

**Criterios de aceptacion:**
- Errores 5xx se loguean como ERROR con request_id, path, method.
- Errores 4xx no se loguean (ya manejados en handlers).
- El formato de la respuesta JSON no cambia.

**Estimacion:** 1 hora

---

### Tarea 10: Agregar logging opcional a repositorios (solo errores)

**Descripcion:** Agregar logging de errores criticos en la capa de repositorios. Esta tarea es de menor prioridad y se puede diferir.

**Archivos a considerar (13 repositorios postgres + 2 redis):**
- `internal/repository/postgres/user_repository.go`
- `internal/repository/postgres/application_repository.go`
- `internal/repository/postgres/audit_repository.go`
- `internal/repository/postgres/refresh_token_repository.go`
- `internal/repository/postgres/role_repository.go`
- `internal/repository/postgres/permission_repository.go`
- `internal/repository/postgres/cost_center_repository.go`
- `internal/repository/postgres/user_role_repository.go`
- `internal/repository/postgres/user_permission_repository.go`
- `internal/repository/postgres/user_cost_center_repository.go`
- `internal/repository/postgres/password_history_repository.go`
- `internal/repository/redis/refresh_token_repository.go`
- `internal/repository/redis/authz_cache.go`

**Enfoque conservador:**
- **NO** agregar logging en cada query (demasiado ruido).
- **SI** agregar un logger al constructor de cada repositorio.
- **SI** loguear errores de conexion o queries que fallen inesperadamente a nivel DEBUG.
- Los errores ya se propagan via `return fmt.Errorf(...)` al servicio, que es donde se loguean a nivel mas alto.

**Criterio principal:** Los repositorios loguean a nivel DEBUG unicamente para depuracion. En produccion con nivel INFO, estos logs no aparecen.

**Criterios de aceptacion:**
- Cada repositorio acepta `*slog.Logger` en su constructor.
- Los errores de query se loguean a nivel DEBUG con el nombre de la operacion.
- No se loguean datos sensibles (passwords, tokens, secret keys).
- Se actualizan todas las llamadas en `main.go`.

**Estimacion:** 5 horas

---

### Tarea 11: Actualizar configuracion YAML y documentacion

**Descripcion:** Documentar las opciones de logging y agregar campo de output al config.

**Archivos a modificar:**
- `config.yaml` -- agregar comentarios explicativos
- `internal/config/config.go` -- agregar campo `Output` a `LoggingConfig` (stdout/stderr/file)
- `CLAUDE.md` -- actualizar seccion de arquitectura y comandos
- `docs/plan/002_logging_system.md` -- marcar como completado

**Cambios en config:**
```go
type LoggingConfig struct {
    Level  string `yaml:"level"`  // debug, info, warn, error
    Format string `yaml:"format"` // json, text
    Output string `yaml:"output"` // stdout, stderr (default: stdout)
}
```

```yaml
logging:
  level: info       # debug | info | warn | error
  format: json      # json | text
  output: stdout    # stdout | stderr
```

**Criterios de aceptacion:**
- `config.yaml` tiene comentarios claros para cada campo de logging.
- El default de `output` es "stdout".
- `CLAUDE.md` documenta el nuevo paquete `internal/logger/` en la tabla de archivos criticos.
- `CLAUDE.md` documenta la convencion de logging (niveles, campos).

**Estimacion:** 1.5 horas

---

### Tarea 12: Tests unitarios del sistema de logging

**Descripcion:** Garantizar cobertura de tests para el nuevo paquete de logging y middlewares.

**Archivos a crear:**
- `internal/logger/logger_test.go`
- `internal/middleware/request_id_middleware_test.go`
- `internal/middleware/request_logger_middleware_test.go`

**Tests minimos:**

**logger_test.go:**
- `TestParseLevel` -- todos los niveles validos + valor invalido -> INFO.
- `TestNewJSON` -- verifica que el output sea JSON valido.
- `TestNewText` -- verifica que el output sea texto.
- `TestLevelFiltering` -- verifica que DEBUG no aparece con nivel INFO.
- `TestWithComponent` -- verifica que el campo `component` se agrega.

**request_id_middleware_test.go:**
- `TestRequestID_Generated` -- sin header, genera UUID.
- `TestRequestID_Propagated` -- con header, reutiliza.
- `TestRequestID_InResponse` -- header en response.

**request_logger_middleware_test.go:**
- `TestRequestLogger_Fields` -- verifica campos estructurados.
- `TestRequestLogger_ErrorLevel` -- status 500 -> ERROR.
- `TestRequestLogger_HealthDebug` -- /health -> DEBUG.

**Criterios de aceptacion:**
- Cobertura >= 90% en `internal/logger/`.
- Cobertura >= 80% en los middlewares nuevos.
- Tests no dependen de I/O externo (usan buffers).

**Estimacion:** 3 horas

---

### Tarea 13: Verificar que no se loguean datos sensibles

**Descripcion:** Revision de seguridad para garantizar que el logging no expone informacion sensible.

**Datos que NUNCA deben aparecer en logs:**
- Passwords (raw o hash)
- Tokens JWT (access o refresh)
- Secret keys de aplicacion
- Contenido de `Authorization` header
- Contenido de `X-App-Key` header

**Verificacion:**
1. Buscar con `grep` en todo el codigo por logging de variables que contengan "password", "token", "secret", "key", "authorization".
2. Verificar que el middleware de request logger NO loguea headers sensibles.
3. Verificar que el bootstrap loguea la secret key truncada (Tarea 5).

**Criterios de aceptacion:**
- Ninguna llamada a `slog` loguea campos con datos sensibles.
- Existe un comentario `// SECURITY: never log sensitive data` en el middleware de request logger.
- Revision manual completada y documentada.

**Estimacion:** 1 hora

---

## 5. Orden de Implementacion y Dependencias

```
Tarea 1 (logger pkg)
  |
  +---> Tarea 2 (request ID middleware)
  |       |
  |       +---> Tarea 3 (request logger middleware)
  |               |
  |               +---> Tarea 4 (migrar main.go) ---+
  |                                                   |
  +---> Tarea 5 (migrar bootstrap) ----+              |
  |                                     |             |
  +---> Tarea 6 (migrar audit_svc) ----+              |
  |                                     |             |
  +---> Tarea 7 (handlers) ------------+--- Tarea 9 (error handler global)
  |                                     |
  +---> Tarea 8 (middlewares) ---------+
  |                                     |
  +---> Tarea 10 (repositorios) -------+
                                        |
                                        +---> Tarea 11 (config + docs)
                                        |
                                        +---> Tarea 12 (tests)
                                        |
                                        +---> Tarea 13 (revision seguridad)
```

**Fases sugeridas:**

| Fase | Tareas | Descripcion | Estimacion |
|---|---|---|---|
| A -- Fundacion | 1, 2, 3 | Paquete logger + middlewares HTTP | 6 horas |
| B -- Migracion core | 4, 5, 6 | Migrar main.go, bootstrap, audit | 5.5 horas |
| C -- Enriquecimiento | 7, 8, 9 | Handlers + middlewares + error handler | 8 horas |
| D -- Repositorios | 10 | Logging en capa de datos (diferible) | 5 horas |
| E -- Cierre | 11, 12, 13 | Docs, tests, revision seguridad | 5.5 horas |
| **Total** | | | **30 horas** |

---

## 6. Resumen de Archivos Afectados

### Archivos nuevos (3)

| Archivo | Descripcion |
|---|---|
| `internal/logger/logger.go` | Paquete central de logging |
| `internal/middleware/request_id_middleware.go` | Middleware de request ID |
| `internal/middleware/request_logger_middleware.go` | Middleware de logging HTTP |

### Archivos de test nuevos (3)

| Archivo | Descripcion |
|---|---|
| `internal/logger/logger_test.go` | Tests del paquete logger |
| `internal/middleware/request_id_middleware_test.go` | Tests del middleware request ID |
| `internal/middleware/request_logger_middleware_test.go` | Tests del middleware request logger |

### Archivos a modificar (20+)

| Archivo | Cambio principal |
|---|---|
| `cmd/server/main.go` | Inicializar logger, reemplazar `log.*`, reemplazar middleware Fiber |
| `internal/config/config.go` | Agregar campo `Output` a `LoggingConfig` |
| `config.yaml` | Agregar comentarios y campo `output` |
| `internal/bootstrap/initializer.go` | Recibir `*slog.Logger`, reemplazar `log.*` |
| `internal/service/audit_service.go` | Recibir `*slog.Logger`, reemplazar `log.*` |
| `internal/handler/auth_handler.go` | Recibir `*slog.Logger`, loguear errores |
| `internal/handler/admin_handler.go` | Recibir `*slog.Logger`, loguear errores |
| `internal/handler/authz_handler.go` | Recibir `*slog.Logger`, loguear errores |
| `internal/middleware/jwt_middleware.go` | Recibir `*slog.Logger`, loguear warns |
| `internal/middleware/app_key_middleware.go` | Recibir `*slog.Logger`, loguear warns |
| `internal/middleware/permission_middleware.go` | Recibir `*slog.Logger`, loguear info |
| `internal/middleware/audit_middleware.go` | Agregar constante `LocalRequestID` |
| `internal/repository/postgres/*.go` (11 archivos) | Recibir `*slog.Logger` en constructor |
| `internal/repository/redis/*.go` (2 archivos) | Recibir `*slog.Logger` en constructor |
| `CLAUDE.md` | Documentar logger y convenciones |

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigacion |
|---|---|---|---|
| Cambio de firma de constructores rompe tests existentes | Alta | Medio | Actualizar tests en la misma tarea. Los 62 unit tests y 11 integration tests deben seguir pasando. |
| Logging excesivo en produccion | Media | Bajo | Nivel default INFO, logs DEBUG solo en desarrollo. Revisar output en ambiente de staging. |
| Performance del logging en hot paths | Baja | Bajo | `slog` no asigna memoria para niveles deshabilitados. Request logging es O(1) por request. |
| Datos sensibles en logs | Media | Alto | Tarea 13 dedicada a revision de seguridad. Regla estricta: nunca loguear passwords, tokens, keys. |
| Conflictos con el middleware Fiber de logger | Baja | Bajo | Se elimina completamente el middleware de Fiber y se reemplaza por el custom. No coexisten. |

---

## 8. Criterios de Aceptacion Globales

1. **Cero imports de `"log"`** en todo el codebase Go (excepto tests si es necesario para setup).
2. **Todos los tests existentes pasan** (62 unit + 11 integration).
3. **Nuevos tests del logger** tienen cobertura >= 90%.
4. **Formato JSON** en produccion con todos los campos de la tabla 3.6.
5. **Request ID** presente en cada response HTTP y en cada log de request.
6. **Ningun dato sensible** aparece en los logs (verificado por Tarea 13).
7. **El audit log de negocio** (`AuditService`) sigue funcionando identicamente (canal asincrono, mismos eventos).
8. **El nivel de log** es configurable via `config.yaml` y variable de entorno `LOG_LEVEL`.
9. **El formato de log** es configurable entre JSON y texto via `config.yaml`.
10. **La documentacion** (`CLAUDE.md`, `config.yaml`) esta actualizada.
