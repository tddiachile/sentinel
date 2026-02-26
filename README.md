# Sentinel — Auth Service

Servicio de autenticación y autorización centralizado construido en Go. Gestiona identidades, tokens JWT RS256, control de acceso basado en roles (RBAC) y auditoría completa para aplicaciones internas.

## Stack

| Capa | Tecnología |
|---|---|
| Backend | Go 1.22+, Fiber v2 |
| Base de datos | PostgreSQL 15 |
| Cache / Tokens | Redis 7 |
| Tokens | JWT RS256 (golang-jwt/jwt v5) |
| Frontend | React 18, Vite, TypeScript, Tailwind CSS |
| Contenedores | Docker + Docker Compose |

## Inicio rápido

### Requisitos

- Docker 24+ y Docker Compose 2.20+
- OpenSSL 3.0+

### Levantar en local

```bash
# 1. Generar claves RSA
make keys

# 2. Levantar el stack completo (Go + PostgreSQL + Redis)
make docker-up

# 3. Verificar
curl http://localhost:8080/health
```

El servicio queda disponible en `http://localhost:8080`.

Al arrancar con una base de datos vacía el bootstrap crea automáticamente la aplicación `system`, el rol `admin` y el usuario administrador. El `secret_key` de la aplicación aparece en los logs una sola vez:

```bash
docker compose logs auth-service | grep "secret_key"
```

### Dashboard de administración

```bash
cd web
cp .env.example .env        # Configurar VITE_API_URL y VITE_APP_KEY
npm install
npm run dev                 # http://localhost:5173
```

## Endpoints principales

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/auth/login` | Autenticación con JWT RS256 |
| `POST` | `/auth/refresh` | Renovación de token (rotación automática) |
| `POST` | `/auth/logout` | Cierre de sesión |
| `POST` | `/auth/change-password` | Cambio de contraseña |
| `POST` | `/authz/verify` | Verificación delegada de permisos |
| `GET` | `/authz/me/permissions` | Contexto completo del usuario |
| `GET` | `/authz/permissions-map` | Mapa global firmado (para backends) |
| `GET` | `/authz/permissions-map/version` | Versión (hash) del mapa de permisos |
| `GET` | `/.well-known/jwks.json` | Claves públicas RSA (RFC 7517) |
| `GET` | `/health` | Health check |

La API de administración (`/admin/...`) expone 30 endpoints para gestión de usuarios, roles, permisos, centros de costo, aplicaciones y auditoría.

## Documentación interactiva (Swagger UI)

Con el stack corriendo, la documentación interactiva está disponible en:

```
http://localhost:8080/swagger/
```

Para regenerar los docs tras modificar anotaciones en los handlers:

```bash
~/go/bin/swag init --generalInfo cmd/server/main.go --output docs/api \
  --parseDependency --parseInternal --exclude web/
```

## Comandos disponibles

```bash
make build        # Compilar binario en bin/auth-service
make run          # Ejecutar con go run
make test         # Tests unitarios + integración
make lint         # golangci-lint
make docker-up    # Levantar stack con Docker Compose
make docker-down  # Detener stack
make keys         # Generar par de claves RSA en keys/
make migrate      # Ejecutar migraciones de base de datos
```

## Tests

```bash
# Unitarios (62 tests)
go test ./internal/... -v -cover

# Integración — requiere Docker (11 tests con PostgreSQL + Redis reales)
go test ./tests/integration/... -v -timeout 300s

# Carga — requiere k6 y servicio corriendo
k6 run --env BASE_URL=http://localhost:8080 --env APP_KEY=<key> \
   tests/load/scenarios/mixed_load.js
```

## Documentación

| Documento | Descripción |
|---|---|
| [`docs/api/swagger.json`](docs/api/swagger.json) | Especificación OpenAPI 3.0 — 40 endpoints, 9 grupos |
| [`docs/backlog.md`](docs/backlog.md) | Backlog oficial — 42 historias de usuario / 196 story points |
| [`docs/deployment.md`](docs/deployment.md) | Guía de despliegue completa (local, Docker, Azure) |
| [`docs/specs/auth-api.md`](docs/specs/auth-api.md) | Endpoints de autenticación |
| [`docs/specs/token-management.md`](docs/specs/token-management.md) | JWT RS256, refresh tokens, JWKS |
| [`docs/specs/authorization.md`](docs/specs/authorization.md) | Modelo RBAC + algoritmo HasPermission |
| [`docs/specs/admin-api.md`](docs/specs/admin-api.md) | API de administración (30 endpoints) |
| [`docs/specs/data-model.md`](docs/specs/data-model.md) | Modelo de datos — 12 tablas |
| [`docs/specs/security.md`](docs/specs/security.md) | Política de contraseñas, fuerza bruta, bootstrap |
| [`docs/specs/audit.md`](docs/specs/audit.md) | Sistema de auditoría — 22 tipos de evento |
| [`docs/specs/infrastructure.md`](docs/specs/infrastructure.md) | Docker, configuración, SLAs |
| [`tests/README.md`](tests/README.md) | Instrucciones de ejecución de todos los tests |

## Estructura del proyecto

```
sentinel/
├── cmd/server/          # Punto de entrada (main.go)
├── internal/
│   ├── config/          # Carga de configuración (YAML + env vars)
│   ├── domain/          # Structs del dominio
│   ├── repository/      # Acceso a datos (PostgreSQL + Redis)
│   ├── service/         # Lógica de negocio
│   ├── handler/         # HTTP handlers (Fiber v2)
│   ├── middleware/       # JWT, App-Key, permisos, auditoría, security headers
│   ├── token/           # Generación y validación JWT RS256
│   └── bootstrap/       # Inicialización del sistema
├── migrations/          # DDL de base de datos
├── tests/
│   ├── integration/     # Tests con testcontainers (PG + Redis reales)
│   └── load/            # Escenarios k6
├── web/                 # Dashboard React (Vite + TypeScript + Tailwind)
├── docs/                # Backlog, specs técnicas y guía de despliegue
├── keys/                # Claves RSA (no se commitean — ver .gitignore)
├── docker-compose.yml
├── Dockerfile
├── config.yaml
└── Makefile
```
