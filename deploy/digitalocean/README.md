# Despliegue de Sentinel en DigitalOcean

Dos opciones de despliegue disponibles, ordenadas de mayor a menor recomendación para esta aplicación:

| Opción | Descripción | Recomendada para |
|---|---|---|
| **Droplet + Docker Compose** | VPS con control total, Traefik para SSL automático | Producción, máximo control |
| **App Platform** | PaaS gestionado, sin gestión de servidor | Prototipos, simplicidad |

> **Por qué se recomienda el Droplet:** Sentinel requiere montar archivos PEM (claves RSA) en tiempo de ejecución. El App Platform tiene limitaciones para bind mounts de archivos arbitrarios. El Droplet elimina esas restricciones.

## Estructura de archivos

```
deploy/digitalocean/
├── docker-compose.yml          # Stack completo (Traefik + frontend + backend + PG + Redis)
├── Dockerfile.frontend         # Build multi-stage React → Nginx
├── nginx.conf                  # Nginx: SPA + proxy /api al backend
├── setup.sh                    # Script de inicialización del Droplet
├── app-spec.yaml               # App Platform spec (opción B)
├── entrypoint-appplatform.sh   # Wrapper RSA keys para App Platform
├── .env.example                # Variables de entorno
└── README.md                   # Esta guía
```

---

## Opción A — Droplet + Docker Compose (Recomendada)

### Arquitectura

```
Tu máquina local
   │  docker build + push → DOCR
   ▼
registry.digitalocean.com/TU_ORG
   │  docker compose pull (desde el Droplet)
   ▼
Droplet: solo ejecuta imágenes pre-construidas

Internet (80/443)
    │
    ▼
Traefik (Let's Encrypt SSL automático)
    │
    ▼
frontend:80 (Nginx)
    ├── /api/*       → proxy → auth-service:8080
    ├── /health      → proxy → auth-service:8080
    ├── /.well-known → proxy → auth-service:8080
    └── /*           → React SPA

Red interna (sentinel-internal):
    auth-service ↔ postgres:5432
    auth-service ↔ redis:6379
```

### Requisitos

- **Droplet:** Ubuntu 22.04 LTS, 2 vCPU / 2 GB RAM / 50 GB SSD (~$18/mes)
  - Mínimo viable: 1 vCPU / 1 GB RAM ($6/mes, sin headroom)
- **DOCR:** Container Registry en tu organización de DigitalOcean
- **DNS:** El dominio debe apuntar a la IP del Droplet antes de activar SSL

### Paso 0 — Crear el Container Registry (DOCR)

En el panel de DigitalOcean → **Container Registry** → **Create Registry**.

Anota la URL del registry: `registry.digitalocean.com/TU_ORG`

### Paso 1 — Build y push de imágenes (desde tu máquina local)

```bash
export REGISTRY=registry.digitalocean.com/TU_ORG
export IMAGE_TAG=v1.0

# Autenticarse con DOCR
docker login registry.digitalocean.com

# Backend
docker build -t ${REGISTRY}/sentinel-auth:${IMAGE_TAG} .
docker push ${REGISTRY}/sentinel-auth:${IMAGE_TAG}

# Frontend (con placeholder para el primer deploy — se actualiza en Paso 6)
docker build -f deploy/digitalocean/Dockerfile.frontend \
  --build-arg VITE_APP_KEY=OBTENER_DEL_PRIMER_DEPLOY \
  -t ${REGISTRY}/sentinel-frontend:${IMAGE_TAG} .
docker push ${REGISTRY}/sentinel-frontend:${IMAGE_TAG}
```

### Paso 2 — Crear y configurar el Droplet

En el panel de DigitalOcean:
1. Crear un Droplet con **Ubuntu 22.04 LTS**
2. Elegir **2 vCPU / 2 GB RAM** (Basic, $18/mes) o superior
3. Agregar tu clave SSH para acceso
4. Opcional: habilitar **Monitoring** y **Backups**

### Paso 3 — Ejecutar el script de setup en el Droplet

```bash
# En tu máquina local: copiar el script al Droplet
scp deploy/digitalocean/setup.sh root@<DROPLET_IP>:/tmp/setup.sh

# Conectarse al Droplet y ejecutar
ssh root@<DROPLET_IP>
export DO_REGISTRY_TOKEN=<tu_personal_access_token_de_do>
bash /tmp/setup.sh
```

El script instala Docker CE, inicia sesión en DOCR, genera las claves RSA (4096 bits) y configura UFW.

### Paso 4 — Configurar variables de entorno y copiar archivos al Droplet

```bash
# En tu máquina local: preparar el archivo .env
cp deploy/digitalocean/.env.example deploy/digitalocean/.env
nano deploy/digitalocean/.env
```

Completar todos los valores, especialmente:
- `REGISTRY=registry.digitalocean.com/TU_ORG`
- `IMAGE_TAG=v1.0`
- `DOMAIN=sentinel.tudominio.com` (debe estar apuntando al Droplet)
- `ACME_EMAIL=admin@tudominio.com`
- `DB_PASSWORD`, `REDIS_PASSWORD`, `BOOTSTRAP_ADMIN_PASSWORD`

```bash
# Copiar docker-compose.yml y .env al Droplet
scp deploy/digitalocean/docker-compose.yml \
    deploy/digitalocean/.env \
    root@<DROPLET_IP>:/opt/sentinel/
```

### Paso 5 — Primer despliegue

```bash
# En el Droplet
ssh root@<DROPLET_IP>
cd /opt/sentinel

docker compose pull
docker compose up -d
```

Seguir los logs en tiempo real:
```bash
docker compose logs -f
```

Esperar hasta ver:
```
auth-service  | {"level":"info","msg":"server started","port":8080}
traefik       | level=info msg="Configuration loaded from providers"
```

### Paso 6 — Obtener el secret_key del bootstrap

```bash
# En el Droplet
docker compose logs auth-service | grep "secret_key"
```

Copiar el valor de `secret_key`. Guardarlo en un lugar seguro.

### Paso 7 — Actualizar VITE_APP_KEY y redesplegar el frontend

```bash
# En tu máquina local: construir el frontend con el secret_key real
docker build -f deploy/digitalocean/Dockerfile.frontend \
  --build-arg VITE_APP_KEY=<secret_key_del_paso_6> \
  -t ${REGISTRY}/sentinel-frontend:${IMAGE_TAG} .
docker push ${REGISTRY}/sentinel-frontend:${IMAGE_TAG}

# En el Droplet: pull y reiniciar solo el frontend
ssh root@<DROPLET_IP> \
  "cd /opt/sentinel && docker compose pull frontend && docker compose up -d frontend"
```

### Paso 8 — Verificar el despliegue

```bash
# Health check
curl https://sentinel.tudominio.com/health
# Esperado: {"status":"healthy","checks":{"postgresql":"ok","redis":"ok"}}

# JWKS
curl https://sentinel.tudominio.com/.well-known/jwks.json

# Login
curl -X POST https://sentinel.tudominio.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-App-Key: <secret_key>" \
  -d '{"username":"admin","password":"<bootstrap_password>","client_type":"web"}'
```

El dashboard estará disponible en `https://sentinel.tudominio.com`.

---

## Opción B — App Platform (alternativa gestionada)

### Limitaciones importantes

| Limitación | Impacto en Sentinel |
|---|---|
| Sin bind mounts de archivos arbitrarios | Las claves RSA deben codificarse en base64 y pasarse como secretos |
| Build-time env vars en static sites | `VITE_APP_KEY` requiere redeploy para actualizarse |
| Sin acceso a filesystem persistente | Los logs van al panel de DigitalOcean, no a disco |

### Preparar las claves RSA para App Platform

```bash
# Generar claves localmente
openssl genrsa -out private.pem 4096
openssl rsa -in private.pem -pubout -out public.pem

# Codificar en base64 (una sola línea)
base64 -w 0 private.pem   # Linux
base64 -i private.pem     # macOS
```

Copiar la salida y pegarla en `app-spec.yaml` como valor de `RSA_PRIVATE_KEY_B64`.

El `entrypoint-appplatform.sh` decodifica las claves antes de iniciar el servicio.

**Nota:** Para que este entrypoint funcione, el `Dockerfile` debe copiarlo y usarlo como `ENTRYPOINT`. Ver `docs/deployment.md` si se modifica la imagen.

### Desplegar con doctl

```bash
# Instalar doctl
# https://docs.digitalocean.com/reference/doctl/how-to/install/

# Autenticar
doctl auth init

# Crear la app
doctl apps create --spec deploy/digitalocean/app-spec.yaml

# Ver estado
doctl apps list
doctl apps logs <APP_ID> --component auth-service

# Actualizar spec
doctl apps update <APP_ID> --spec deploy/digitalocean/app-spec.yaml
```

---

## Mantenimiento (Droplet)

### Actualizar la aplicación

```bash
# En tu máquina local: construir y pushear la nueva versión
export REGISTRY=registry.digitalocean.com/TU_ORG
export IMAGE_TAG=v1.1

docker build -t ${REGISTRY}/sentinel-auth:${IMAGE_TAG} .
docker push ${REGISTRY}/sentinel-auth:${IMAGE_TAG}

# Si el frontend también cambió:
docker build -f deploy/digitalocean/Dockerfile.frontend \
  --build-arg VITE_APP_KEY=<secret_key> \
  -t ${REGISTRY}/sentinel-frontend:${IMAGE_TAG} .
docker push ${REGISTRY}/sentinel-frontend:${IMAGE_TAG}

# En el Droplet: actualizar IMAGE_TAG en .env, pull y reiniciar
ssh root@<DROPLET_IP> "sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=${IMAGE_TAG}/' /opt/sentinel/.env"
ssh root@<DROPLET_IP> "cd /opt/sentinel && docker compose pull && docker compose up -d"
```

### Rotación de claves RSA (cada 90 días recomendado)

```bash
# Hacer backup de las claves actuales
cp deploy/digitalocean/files/keys/private.pem \
   deploy/digitalocean/files/keys/private.pem.bak-$(date +%Y%m%d)

# Generar nuevas claves
openssl genrsa -out deploy/digitalocean/files/keys/private.pem 4096
openssl rsa -in deploy/digitalocean/files/keys/private.pem \
            -pubout -out deploy/digitalocean/files/keys/public.pem

# Reiniciar el auth-service (esperar 60 min antes de eliminar el backup)
docker compose -f deploy/digitalocean/docker-compose.yml restart auth-service
```

### Backup de base de datos

```bash
# Backup manual
docker exec $(docker compose -f deploy/digitalocean/docker-compose.yml \
  ps -q postgres) \
  pg_dump -U sentinel sentinel > backup-$(date +%Y%m%d).sql

# Restaurar
docker exec -i $(docker compose -f deploy/digitalocean/docker-compose.yml \
  ps -q postgres) \
  psql -U sentinel sentinel < backup-YYYYMMDD.sql
```

### Limpieza de refresh tokens expirados (cron diario)

```bash
# Agregar al crontab del Droplet
crontab -e
# Añadir:
# 0 3 * * * docker exec $(docker ps -qf name=postgres) \
#   psql -U sentinel sentinel -c \
#   "DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '7 days' AND is_revoked = TRUE;"
```

---

## Resolución de problemas

### Traefik no genera el certificado SSL

- Verificar que el dominio apunta a la IP del Droplet: `dig +short sentinel.tudominio.com`
- Revisar logs de Traefik: `docker compose -f ... logs traefik | grep -i "error\|cert"`
- El dominio debe estar accesible por HTTP (puerto 80) para el challenge ACME

### auth-service no arranca — RSA key error

```
Error: open /app/keys/private.pem: no such file or directory
```
Las claves no existen o no están montadas. Verificar:
```bash
ls -la /opt/sentinel/deploy/digitalocean/files/keys/
# Debe mostrar private.pem y public.pem
```

### Frontend muestra error de autenticación

`VITE_APP_KEY` incorrecto. Verificar en base de datos:
```bash
docker exec $(docker compose -f deploy/digitalocean/docker-compose.yml ps -q postgres) \
  psql -U sentinel sentinel \
  -c "SELECT slug, secret_key FROM applications WHERE slug='system';"
```
