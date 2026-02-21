# Guía de Despliegue — Sentinel Auth Service

## Índice

1. [Requisitos previos](#1-requisitos-previos)
2. [Desarrollo local (Docker Compose)](#2-desarrollo-local-docker-compose)
3. [Configuración de variables de entorno](#3-configuración-de-variables-de-entorno)
4. [Generación de claves RSA](#4-generación-de-claves-rsa)
5. [Migraciones de base de datos](#5-migraciones-de-base-de-datos)
6. [Bootstrap del sistema](#6-bootstrap-del-sistema)
7. [Ejecución sin Docker (binario directo)](#7-ejecución-sin-docker-binario-directo)
8. [Frontend — Dashboard de administración](#8-frontend--dashboard-de-administración)
9. [Verificación del despliegue](#9-verificación-del-despliegue)
10. [Despliegue en Azure](#10-despliegue-en-azure)
11. [Operación y mantenimiento](#11-operación-y-mantenimiento)
12. [Resolución de problemas](#12-resolución-de-problemas)

---

## 1. Requisitos previos

### Herramientas necesarias

| Herramienta | Versión mínima | Verificación |
|---|---|---|
| Go | 1.22+ | `go version` |
| Docker | 24+ | `docker --version` |
| Docker Compose | 2.20+ | `docker compose version` |
| OpenSSL | 3.0+ | `openssl version` |
| Node.js (frontend) | 18+ | `node --version` |
| npm (frontend) | 9+ | `npm --version` |
| k6 (tests de carga, opcional) | 0.50+ | `k6 version` |

### Clonar el repositorio

```bash
git clone https://github.com/enunezf/sentinel.git
cd sentinel
```

---

## 2. Desarrollo local (Docker Compose)

El camino más rápido para levantar el stack completo.

### Paso 1 — Generar claves RSA

```bash
make keys
# Genera keys/private.pem y keys/public.pem (2048 bits)
```

> Las claves se montan en el contenedor como volumen de solo lectura. Nunca se incluyen en la imagen Docker.

### Paso 2 — Levantar el stack

```bash
make docker-up
# Equivale a: docker compose up --build -d
```

Esto levanta tres servicios:
- **auth-service** en `http://localhost:8080`
- **PostgreSQL 15** en `localhost:5432`
- **Redis 7** en `localhost:6379`

### Paso 3 — Verificar el estado

```bash
curl http://localhost:8080/health
```

Respuesta esperada:
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

### Detener el stack

```bash
make docker-down
# Para eliminar volúmenes también:
docker compose down -v
```

---

## 3. Configuración de variables de entorno

El servicio lee `config.yaml` y permite sobrescribir cualquier valor con variables de entorno.

### Variables de entorno requeridas

| Variable | Descripción | Ejemplo |
|---|---|---|
| `DB_HOST` | Host de PostgreSQL | `postgres` (docker) / `localhost` |
| `DB_NAME` | Nombre de la base de datos | `sentinel` |
| `DB_USER` | Usuario de PostgreSQL | `sentinel` |
| `DB_PASSWORD` | Contraseña de PostgreSQL | `sentinel_dev` |
| `REDIS_ADDR` | Dirección de Redis (`host:port`) | `redis:6379` |
| `REDIS_PASSWORD` | Contraseña de Redis (puede estar vacía) | `` |
| `JWT_PRIVATE_KEY_PATH` | Ruta a la clave privada RSA | `/app/keys/private.pem` |
| `JWT_PUBLIC_KEY_PATH` | Ruta a la clave pública RSA | `/app/keys/public.pem` |
| `BOOTSTRAP_ADMIN_USER` | Username del administrador inicial | `admin` |
| `BOOTSTRAP_ADMIN_PASSWORD` | Contraseña temporal del administrador | `Admin@Bootstrap!` |

> La contraseña de bootstrap **no valida la política de contraseñas**. El administrador deberá cambiarla en el primer login.

### Variables opcionales

| Variable | Descripción | Default |
|---|---|---|
| `SERVER_PORT` | Puerto HTTP del servicio | `8080` |
| `LOG_LEVEL` | Nivel de logging (`debug`, `info`, `warn`, `error`) | `info` |
| `BCRYPT_COST` | Costo de bcrypt (>= 12) | `12` |

---

## 4. Generación de claves RSA

### Desarrollo local

```bash
make keys
# Crea keys/private.pem y keys/public.pem
```

### Producción (recomendado 4096 bits)

```bash
mkdir -p keys
openssl genrsa -out keys/private.pem 4096
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### Azure Key Vault (producción)

1. Generar el par de claves localmente (4096 bits)
2. Importar la clave privada a Azure Key Vault:
   ```bash
   az keyvault secret set \
     --vault-name <vault-name> \
     --name sentinel-jwt-private-key \
     --file keys/private.pem
   ```
3. Configurar el servicio para leer desde Key Vault montando el secreto como archivo o usando el SDK de Azure.
4. `JWT_PRIVATE_KEY_PATH` apunta al archivo montado en el pod/contenedor.

### Rotación de claves RSA

1. Generar nuevo par de claves con un nuevo `kid` (ej: `2026-04-key-02`)
2. Actualizar `config.yaml` para que el servicio use la nueva clave privada
3. Mantener la clave pública anterior disponible en el JWKS durante al menos 60 minutos (vida del access token)
4. Verificar en `GET /.well-known/jwks.json` que ambas claves aparecen
5. Retirar la clave anterior del JWKS una vez que todos los tokens firmados con ella hayan expirado

---

## 5. Migraciones de base de datos

Las migraciones se ejecutan automáticamente al iniciar el servicio si se configura el runner, o manualmente:

```bash
# Con Go instalado (apunta a la BD configurada en env vars)
make migrate

# Directamente con psql
psql -h localhost -U sentinel -d sentinel \
  -f migrations/001_initial_schema.sql \
  -f migrations/002_seed_permissions.sql
```

### Archivos de migración

| Archivo | Contenido |
|---|---|
| `migrations/001_initial_schema.sql` | 12 tablas: `applications`, `users`, `password_history`, `roles`, `permissions`, `cost_centers`, `user_roles`, `user_permissions`, `user_cost_centers`, `refresh_tokens`, `audit_logs` e índices |
| `migrations/002_seed_permissions.sql` | Permisos base del sistema (idempotente con `ON CONFLICT DO NOTHING`) |

> Las migraciones son **idempotentes**: ejecutarlas múltiples veces no produce errores ni duplicados.

---

## 6. Bootstrap del sistema

El bootstrap se ejecuta **automáticamente** la primera vez que el servicio inicia con una base de datos vacía.

### Qué crea el bootstrap

1. **Aplicación `system`** — con `secret_key` generado aleatoriamente (se muestra una sola vez en los logs)
2. **Rol `admin`** — `is_system = true`, no puede eliminarse
3. **Usuario administrador** — con `must_change_pwd = true`; el operador debe cambiar la contraseña en el primer login

### Recuperar el `secret_key` de la aplicación system

```bash
# Ver logs del servicio al arrancar
docker compose logs auth-service | grep "secret_key"
# O desde PostgreSQL:
psql -h localhost -U sentinel -d sentinel \
  -c "SELECT slug, secret_key FROM applications WHERE slug = 'system';"
```

### El bootstrap es idempotente

Si ya existe al menos una aplicación o usuario en la base de datos, el bootstrap no se ejecuta. Esto garantiza que reinicios del servicio no dupliquen datos.

---

## 7. Ejecución sin Docker (binario directo)

### Compilar

```bash
make build
# Genera bin/auth-service
```

### Ejecutar

```bash
# Asegurarse de que PostgreSQL y Redis estén corriendo
export DB_HOST=localhost
export DB_NAME=sentinel
export DB_USER=sentinel
export DB_PASSWORD=sentinel_dev
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export JWT_PRIVATE_KEY_PATH=./keys/private.pem
export JWT_PUBLIC_KEY_PATH=./keys/public.pem
export BOOTSTRAP_ADMIN_USER=admin
export BOOTSTRAP_ADMIN_PASSWORD=Admin@Bootstrap!

./bin/auth-service
# O con go run para desarrollo:
make run
```

---

## 8. Frontend — Dashboard de administración

### Desarrollo local

```bash
cd web

# 1. Copiar y configurar variables de entorno
cp .env.example .env
# Editar .env:
#   VITE_API_URL=http://localhost:8080/api/v1
#   VITE_APP_KEY=<secret_key de la aplicación system>

# 2. Instalar dependencias
npm install

# 3. Levantar servidor de desarrollo (con proxy al backend)
npm run dev
# Dashboard disponible en http://localhost:5173
```

### Build de producción

```bash
cd web
npm run build
# Genera web/dist/ con los assets estáticos
```

Los archivos de `web/dist/` pueden servirse con cualquier servidor web estático (Nginx, Azure Static Web Apps, etc.).

### Variables de entorno del frontend

| Variable | Descripción |
|---|---|
| `VITE_API_URL` | URL base de la API (`http://localhost:8080/api/v1`) |
| `VITE_APP_KEY` | `secret_key` de la aplicación en Sentinel |

---

## 9. Verificación del despliegue

### Checklist post-despliegue

```bash
# 1. Health check
curl http://localhost:8080/health
# Esperado: {"status":"healthy",...}

# 2. JWKS disponible (sin autenticación)
curl http://localhost:8080/.well-known/jwks.json
# Esperado: {"keys":[{"kty":"RSA","alg":"RS256",...}]}

# 3. Login del administrador
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-App-Key: <secret_key>" \
  -d '{"username":"admin","password":"<bootstrap_password>","client_type":"web"}'
# Esperado: 200 con access_token, refresh_token y must_change_password: true

# 4. Verificar que debe cambiar la contraseña
# El campo user.must_change_password debe ser true en el primer login
```

### Tests automatizados

```bash
# Unitarios
go test ./internal/... -v -cover

# Integración (requiere Docker)
go test ./tests/integration/... -v -timeout 300s
```

---

## 10. Despliegue en Azure

### Opción A — Azure Container Apps

```bash
# 1. Build y push de la imagen
az acr build \
  --registry <acr-name> \
  --image sentinel-auth:latest \
  --file Dockerfile .

# 2. Crear Container App
az containerapp create \
  --name sentinel-auth \
  --resource-group <rg> \
  --environment <env> \
  --image <acr-name>.azurecr.io/sentinel-auth:latest \
  --target-port 8080 \
  --ingress external \
  --env-vars \
    DB_HOST=<pg-host> \
    DB_NAME=sentinel \
    DB_USER=sentinel \
    DB_PASSWORD=secretref:db-password \
    REDIS_ADDR=<redis-host>:6380 \
    JWT_PRIVATE_KEY_PATH=/app/keys/private.pem \
    JWT_PUBLIC_KEY_PATH=/app/keys/public.pem \
    BOOTSTRAP_ADMIN_USER=admin \
    BOOTSTRAP_ADMIN_PASSWORD=secretref:bootstrap-password
```

### Opción B — Azure Kubernetes Service (AKS)

Usar el `Dockerfile` multi-stage incluido. Configurar:
- **Secrets** de Kubernetes para `DB_PASSWORD`, `BOOTSTRAP_ADMIN_PASSWORD`
- **ConfigMap** para variables no sensibles
- **PersistentVolume** o Azure Key Vault para las claves RSA
- **HorizontalPodAutoscaler** con target de CPU 70%

### Infraestructura recomendada en Azure

| Servicio Azure | Uso |
|---|---|
| Azure Container Apps / AKS | Runtime del servicio Go |
| Azure Database for PostgreSQL Flexible Server | Base de datos principal |
| Azure Cache for Redis | Cache y refresh tokens |
| Azure Key Vault | Claves RSA y secrets |
| Azure Container Registry | Imágenes Docker |
| Azure Application Gateway | TLS termination, WAF |
| Azure Monitor + Log Analytics | Observabilidad |

---

## 11. Operación y mantenimiento

### SLAs objetivo

| Métrica | Objetivo |
|---|---|
| Latencia `POST /auth/login` | < 200 ms (p95) |
| Latencia `POST /authz/verify` | < 50 ms (p95) |
| Usuarios concurrentes | 500+ |
| Disponibilidad mensual | 99.5% |
| RTO (Recovery Time Objective) | < 5 minutos |

### Tests de carga (k6)

```bash
# Instalar k6: https://k6.io/docs/get-started/installation/

# Escenario mixto (recomendado en pre-producción)
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env APP_KEY=<secret_key> \
  tests/load/scenarios/mixed_load.js

# Login específico (SLA p95 < 200ms)
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env APP_KEY=<secret_key> \
  tests/load/scenarios/login_load.js

# Authz verify (SLA p95 < 50ms)
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env APP_KEY=<secret_key> \
  tests/load/scenarios/authz_verify_load.js
```

### Rotación periódica de claves RSA

Se recomienda rotar las claves RSA cada 90 días. Ver [Sección 4 — Rotación de claves RSA](#4-generación-de-claves-rsa).

### Limpieza de refresh tokens expirados

Los refresh tokens expirados en PostgreSQL no se eliminan automáticamente. Para limpiar periódicamente:

```sql
DELETE FROM refresh_tokens
WHERE expires_at < NOW() - INTERVAL '7 days'
  AND is_revoked = TRUE;
```

Se recomienda ejecutar este comando como job programado (cron, Azure Function, etc.) una vez al día.

---

## 12. Resolución de problemas

### El servicio no arranca — "missing required field"

Verificar que todas las variables de entorno obligatorias están definidas. El servicio falla con un mensaje claro indicando qué campo falta.

```bash
docker compose logs auth-service | tail -20
```

### Health check retorna 503

Alguna dependencia no está disponible:

```bash
# Verificar PostgreSQL
docker compose ps postgres
docker compose logs postgres | tail -10

# Verificar Redis
docker compose ps redis
docker compose logs redis | tail -10
```

### Error "column does not exist" en PostgreSQL

Las migraciones no se ejecutaron. Ejecutar manualmente:

```bash
make migrate
# O:
psql -h localhost -U sentinel -d sentinel -f migrations/001_initial_schema.sql
```

### Login falla con APPLICATION_NOT_FOUND

El header `X-App-Key` no coincide con ninguna aplicación activa en la base de datos. Verificar el `secret_key` de la aplicación:

```bash
psql -h localhost -U sentinel -d sentinel \
  -c "SELECT name, slug, secret_key, is_active FROM applications;"
```

### Las claves RSA no se encuentran

Verificar que el directorio `keys/` existe y contiene los archivos PEM:

```bash
ls -la keys/
# Si no existen:
make keys
```

---

*Guía de despliegue generada el 2026-02-21 — Sentinel Auth Service v1.0*
