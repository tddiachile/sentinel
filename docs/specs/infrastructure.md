# Especificacion Tecnica: Infraestructura y Configuracion

**Referencia:** `docs/plan/auth-service-spec.md` secciones 2, 10, 11, 12
**Historias relacionadas:** US-001, US-002, US-003, US-038, US-043

---

## 1. Stack Tecnologico

| Componente | Tecnologia | Version |
|---|---|---|
| Lenguaje | Go | 1.22+ |
| Framework HTTP | Fiber v2 o Chi | A decidir por equipo |
| Base de datos principal | PostgreSQL | 15+ |
| Cache / Refresh Tokens | Redis | 7+ |
| Hashing de contrasenas | bcrypt | costo >= 12 |
| Tokens | JWT RS256 | - |
| Configuracion | Env vars + YAML | - |
| Contenedores | Docker + Docker Compose | - |
| Despliegue destino | Azure Container Apps / AKS | - |

---

## 2. Estructura del Proyecto

```
sentinel/
├── cmd/
│   └── server/
│       └── main.go                 # Punto de entrada
├── internal/
│   ├── config/                     # Carga de configuracion (YAML + env)
│   ├── domain/                     # Structs del dominio
│   │   ├── application.go
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── permission.go
│   │   ├── cost_center.go
│   │   ├── token.go
│   │   └── audit.go
│   ├── repository/                 # Acceso a datos
│   │   ├── postgres/
│   │   │   ├── user_repository.go
│   │   │   ├── application_repository.go
│   │   │   ├── permission_repository.go
│   │   │   ├── role_repository.go
│   │   │   ├── user_role_repository.go
│   │   │   ├── user_permission_repository.go
│   │   │   ├── cost_center_repository.go
│   │   │   ├── user_cost_center_repository.go
│   │   │   ├── refresh_token_repository.go
│   │   │   └── audit_repository.go
│   │   └── redis/
│   │       ├── refresh_token_repository.go
│   │       └── authz_cache.go
│   ├── service/                    # Logica de negocio
│   │   ├── auth_service.go
│   │   ├── authz_service.go
│   │   ├── user_service.go
│   │   ├── role_service.go
│   │   ├── permission_service.go
│   │   ├── cost_center_service.go
│   │   └── audit_service.go
│   ├── handler/                    # HTTP handlers
│   │   ├── auth_handler.go
│   │   ├── authz_handler.go
│   │   └── admin_handler.go
│   ├── middleware/                  # Middlewares HTTP
│   │   ├── jwt_middleware.go
│   │   ├── app_key_middleware.go
│   │   ├── permission_middleware.go
│   │   ├── audit_middleware.go
│   │   └── security_headers_middleware.go
│   ├── token/                      # Generacion/validacion JWT
│   │   └── jwt.go
│   └── bootstrap/                  # Inicializacion del sistema
│       └── initializer.go
├── migrations/
│   ├── 001_initial_schema.sql
│   └── 002_seed_permissions.sql
├── docker-compose.yml
├── Dockerfile
├── config.yaml
├── Makefile
└── README.md
```

---

## 3. Configuracion (config.yaml)

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  graceful_shutdown_timeout: 15s

database:
  host: ${DB_HOST}
  port: 5432
  name: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 5m

redis:
  addr: ${REDIS_ADDR}
  password: ${REDIS_PASSWORD}
  db: 0

jwt:
  private_key_path: ${JWT_PRIVATE_KEY_PATH}
  public_key_path: ${JWT_PUBLIC_KEY_PATH}
  access_token_ttl: 60m
  refresh_token_ttl_web: 168h
  refresh_token_ttl_mobile: 720h

security:
  max_failed_attempts: 5
  lockout_duration: 15m
  bcrypt_cost: 12
  password_history: 5

bootstrap:
  admin_user: ${BOOTSTRAP_ADMIN_USER}
  admin_password: ${BOOTSTRAP_ADMIN_PASSWORD}

logging:
  level: info
  format: json
```

### Reglas de Carga

1. Leer `config.yaml` del directorio de trabajo
2. Variables de entorno sobreescriben valores del YAML (notacion `${VAR}`)
3. Validar al arranque: si falta un campo obligatorio, fallo fatal con mensaje claro
4. Campos obligatorios: `database.*`, `redis.addr`, `jwt.private_key_path`, `jwt.public_key_path`, `bootstrap.admin_user`, `bootstrap.admin_password`
5. Secrets (passwords, keys) **solo** via variable de entorno

---

## 4. Docker Compose

```yaml
version: "3.9"
services:
  auth-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_NAME=sentinel
      - DB_USER=sentinel
      - DB_PASSWORD=sentinel_dev
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=
      - JWT_PRIVATE_KEY_PATH=/app/keys/private.pem
      - JWT_PUBLIC_KEY_PATH=/app/keys/public.pem
      - BOOTSTRAP_ADMIN_USER=admin
      - BOOTSTRAP_ADMIN_PASSWORD=Admin@12345!
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./keys:/app/keys:ro

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: sentinel
      POSTGRES_USER: sentinel
      POSTGRES_PASSWORD: sentinel_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sentinel"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
```

---

## 5. Dockerfile (Multi-stage)

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/server/

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /auth-service .
COPY config.yaml .
COPY migrations/ ./migrations/
EXPOSE 8080
ENTRYPOINT ["./auth-service"]
```

---

## 6. Makefile

```makefile
.PHONY: build run test lint migrate

build:
	go build -o bin/auth-service ./cmd/server/

run:
	go run ./cmd/server/

test:
	go test ./... -v -cover

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/migrate/
```

---

## 7. SLAs y Rendimiento

| Metrica | Objetivo |
|---|---|
| Latencia `POST /auth/login` | < 200 ms (p95) |
| Latencia `POST /authz/verify` | < 50 ms (p95) |
| Latencia verificacion JWT local | < 5 ms (en memoria) |
| Usuarios concurrentes | 500+ |
| Usuarios totales | 2.000 |
| Disponibilidad | 99.5% mensual |
| RTO | < 5 minutos |

### Estrategias de Optimizacion

1. Pool de conexiones PostgreSQL (`pgx`): 50 open, 10 idle
2. Cache de permisos en Redis con TTL de 5 min
3. Indices en todas las FK y campos de consulta frecuente
4. Verificacion JWT local en backends (sin round-trip)

---

## 8. Observabilidad

### Logging

- Formato: JSON estructurado
- Nivel default: `info`
- Campos obligatorios por log: `timestamp`, `level`, `message`, `request_id`
- Nunca loguear: passwords, tokens, secret keys

### Metricas (Fase 4)

- Latencia por endpoint (histograma)
- Tasa de errores por endpoint
- Intentos de fuerza bruta (contador)
- Conexiones activas a PostgreSQL y Redis

### Alertas (Fase 4)

- Tasa de errores > 1%
- Latencia p95 > SLA
- Intentos masivos de fuerza bruta (>50 en 1 min)

---

## 9. Guia para el Tester

### Tests Obligatorios

1. **Docker Compose up:** Los tres servicios levantan sin errores
2. **Configuracion YAML:** Se carga correctamente con valores default
3. **Configuracion env override:** Variables de entorno sobreescriben YAML
4. **Configuracion faltante:** Fallo fatal con mensaje claro si falta DB_HOST
5. **Health check:** Retorna 200 cuando PG y Redis estan up
6. **Health check degradado:** Retorna 503 si Redis esta caido
7. **Graceful shutdown:** Requests en vuelo se completan antes de cierre
8. **Pool PostgreSQL:** Conexiones se reciclan correctamente

---

*Fin de especificacion infrastructure.md*
