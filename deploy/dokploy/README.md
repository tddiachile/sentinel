# Despliegue de Sentinel en Dokploy

Guía paso a paso para desplegar el stack completo (Go backend + React frontend + PostgreSQL + Redis) usando [Dokploy](https://dokploy.com).

## Estructura de archivos

```
deploy/dokploy/
├── docker-compose.yml    # Compose optimizado para Dokploy
├── Dockerfile.frontend   # Build multi-stage React → Nginx
├── nginx.conf            # Nginx: sirve SPA + proxía /api al backend
├── .env.example          # Variables de entorno requeridas
└── README.md             # Esta guía
```

## Arquitectura del despliegue

```
Internet
   │ HTTPS (443)
   ▼
Traefik (Dokploy)
   │
   ▼
frontend (Nginx :80)
   ├── /api/*          → proxy → auth-service:8080
   ├── /health         → proxy → auth-service:8080
   ├── /.well-known/*  → proxy → auth-service:8080
   └── /*              → React SPA (index.html)

Red interna (sentinel-internal):
  auth-service ←→ postgres:5432
  auth-service ←→ redis:6379
```

Solo el contenedor `frontend` está expuesto a Traefik. PostgreSQL y Redis son internos.

---

## Requisitos previos

- Servidor con Dokploy instalado ([instrucciones](https://docs.dokploy.com/docs/core/installation))
- Repositorio Git accesible desde Dokploy (GitHub, GitLab, Gitea, Bitbucket)
- Dominio apuntando a la IP del servidor
- OpenSSL instalado localmente para generar las claves RSA

---

## Paso 1 — Crear el proyecto en Dokploy

1. En el panel de Dokploy, ir a **Projects** → **Create Project**.
2. Crear un proyecto llamado `sentinel`.
3. Dentro del proyecto, hacer clic en **Create Service** → **Compose**.
4. Configurar:
   - **Name:** `sentinel-stack`
   - **Provider:** GitHub (u otro proveedor Git)
   - **Repository:** seleccionar el repositorio de Sentinel
   - **Branch:** `main`
   - **Compose Path:** `deploy/dokploy/docker-compose.yml`
5. Guardar sin desplegar todavía.

---

## Paso 2 — Configurar las variables de entorno

1. En la sección **Environment** del compose, pegar el contenido de `.env.example` y completar los valores reales:

```env
DOMAIN=sentinel.tudominio.com

DB_NAME=sentinel
DB_USER=sentinel
DB_PASSWORD=<contraseña_fuerte>

REDIS_PASSWORD=<contraseña_redis>

BOOTSTRAP_ADMIN_USER=admin
BOOTSTRAP_ADMIN_PASSWORD=<contraseña_bootstrap>

VITE_API_URL=/api/v1
VITE_APP_KEY=placeholder
```

> `VITE_APP_KEY` se establece con `placeholder` en el primer despliegue. Se actualizará en el Paso 5.

---

## Paso 3 — Generar y subir las claves RSA

Las claves RSA **nunca se incluyen en la imagen Docker** ni en el repositorio. Se suben mediante la función **File Mounts** de Dokploy.

### 3a. Generar las claves localmente

```bash
# Producción: 4096 bits
mkdir -p /tmp/sentinel-keys
openssl genrsa -out /tmp/sentinel-keys/private.pem 4096
openssl rsa -in /tmp/sentinel-keys/private.pem -pubout -out /tmp/sentinel-keys/public.pem
```

### 3b. Subir al servidor vía Dokploy File Mounts

El `docker-compose.yml` define este volumen:
```yaml
- ./files/keys:/app/keys:ro
```

Dokploy almacena los archivos de ese bind mount en la carpeta `files/` del directorio del compose. Para subir los archivos:

**Opción A — Panel de Dokploy (recomendada):**
1. En el compose `sentinel-stack`, ir a la pestaña **File Mounts**.
2. Crear la ruta `keys/private.pem` y pegar el contenido del archivo PEM.
3. Crear la ruta `keys/public.pem` y pegar el contenido del archivo PEM.

**Opción B — Acceso SSH al servidor:**
```bash
# Ajustar la ruta según donde Dokploy almacene el proyecto
# Normalmente: /root/dokploy/compose/<project-id>/
ssh usuario@servidor
mkdir -p /ruta/al/proyecto/files/keys
scp /tmp/sentinel-keys/private.pem usuario@servidor:/ruta/al/proyecto/files/keys/
scp /tmp/sentinel-keys/public.pem  usuario@servidor:/ruta/al/proyecto/files/keys/
chmod 600 /ruta/al/proyecto/files/keys/private.pem
```

> Las claves persisten entre redespliegues porque están en un bind mount (`./files/`), no en el filesystem temporal de la imagen.

---

## Paso 4 — Primer despliegue

1. En Dokploy, hacer clic en **Deploy** en el compose `sentinel-stack`.
2. Esperar a que se construyan las imágenes y arranquen los contenedores (~2-4 min la primera vez).
3. Verificar en la sección **Logs** que los tres servicios arrancan correctamente.

Señales de éxito en los logs:
```
auth-service  | {"level":"info","msg":"bootstrap completed"}
auth-service  | {"level":"info","msg":"server started","port":8080}
postgres      | database system is ready to accept connections
redis         | Ready to accept connections
```

---

## Paso 5 — Obtener el secret_key del bootstrap

En el primer arranque, el bootstrap crea la aplicación `system` y muestra su `secret_key` **una sola vez** en los logs:

1. En Dokploy → **Logs** del servicio `auth-service`, buscar:
   ```
   secret_key
   ```
2. Copiar el valor del `secret_key`.

Alternativa si ya no aparece en logs:
```bash
# En el servidor, acceder al contenedor de postgres
docker exec -it <postgres-container> psql -U sentinel -d sentinel \
  -c "SELECT slug, secret_key FROM applications WHERE slug = 'system';"
```

---

## Paso 6 — Configurar VITE_APP_KEY y redesplegar

1. En Dokploy → **Environment** del compose, actualizar:
   ```
   VITE_APP_KEY=<secret_key_obtenido_en_paso_5>
   ```
2. Hacer clic en **Deploy** para reconstruir la imagen del frontend con la clave correcta.

> La reconstrucción del frontend tarda ~1-2 min. El backend NO se reinicia en este redespliegue.

---

## Paso 7 — Configurar dominio y SSL

### Opción A — Usando la UI de Dokploy (recomendada)

1. En el compose, ir a la pestaña **Domains**.
2. Agregar el dominio configurado en `DOMAIN` (ej: `sentinel.tudominio.com`).
3. Seleccionar **HTTPS** con **Let's Encrypt**.
4. Dokploy configurará Traefik automáticamente.

> Asegurarse de que el dominio ya apunta a la IP del servidor antes de activar SSL. De lo contrario, Let's Encrypt no podrá validar el dominio.

### Opción B — Traefik labels (ya incluido en docker-compose.yml)

El `docker-compose.yml` ya tiene las labels de Traefik configuradas en el servicio `frontend`. Dokploy las detecta y configura el router automáticamente si la variable `DOMAIN` está definida.

---

## Paso 8 — Verificación del despliegue

```bash
# Health check del backend
curl https://sentinel.tudominio.com/health
# Esperado: {"status":"healthy","checks":{"postgresql":"ok","redis":"ok"}}

# JWKS (clave pública RSA para backends)
curl https://sentinel.tudominio.com/.well-known/jwks.json
# Esperado: {"keys":[{"kty":"RSA","alg":"RS256",...}]}

# Login del administrador
curl -X POST https://sentinel.tudominio.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-App-Key: <secret_key>" \
  -d '{"username":"admin","password":"<bootstrap_password>","client_type":"web"}'
# Esperado: 200 con access_token y must_change_password: true
```

El dashboard de administración estará disponible en `https://sentinel.tudominio.com`.

---

## Actualizaciones y redespliegue

Dokploy puede configurarse para redesplegar automáticamente cuando se hace push al repositorio:

1. En el compose, ir a **General** → **Auto Deploy**.
2. Copiar el webhook URL y configurarlo en GitHub/GitLab.

Para redespliegues manuales, simplemente hacer clic en **Deploy** en el panel de Dokploy.

---

## Resolución de problemas

### El servicio `auth-service` no arranca

```
Verificar que las variables de entorno DB_PASSWORD y REDIS_PASSWORD estén definidas.
```
Revisar en Dokploy → **Logs** → filtrar por `auth-service`.

### Las claves RSA no se encuentran

```
Error: open /app/keys/private.pem: no such file or directory
```
El bind mount `./files/keys` está vacío. Seguir el Paso 3 para subir los archivos PEM.

### Health check retorna 503

PostgreSQL o Redis no están disponibles. Verificar:
- Logs de `postgres`: `pg_isready` debe responder
- Logs de `redis`: `Ready to accept connections`

### VITE_APP_KEY incorrecto — el dashboard no carga datos

La clave del frontend no coincide con la de la base de datos. Verificar:
```bash
# En el servidor
docker exec -it <postgres-container> psql -U sentinel -d sentinel \
  -c "SELECT secret_key FROM applications WHERE slug = 'system';"
```
Actualizar `VITE_APP_KEY` y redesplegar (Paso 6).

### Cambio de claves RSA (rotación)

1. Generar nuevo par de claves (ver Paso 3a).
2. Reemplazar los archivos en `./files/keys/` vía Dokploy File Mounts o SSH.
3. Redesplegar para que `auth-service` cargue las nuevas claves.
4. Los tokens existentes (firmados con la clave anterior) expirarán en máximo 60 min.

---

## Mantenimiento

### Limpieza de refresh tokens expirados

Ejecutar periódicamente (cron recomendado: diario):
```sql
DELETE FROM refresh_tokens
WHERE expires_at < NOW() - INTERVAL '7 days'
  AND is_revoked = TRUE;
```

Desde Dokploy, se puede ejecutar en el contenedor de postgres:
```bash
docker exec -it <postgres-container> psql -U sentinel -d sentinel \
  -c "DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '7 days' AND is_revoked = TRUE;"
```

### Backups de PostgreSQL

En Dokploy, ir a **Volumes** → `postgres_data` → **Backup** para configurar backups automáticos a S3.

---

*Guía generada para Sentinel Auth Service — Dokploy deployment*
