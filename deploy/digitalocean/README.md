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
Internet (80/443)
    │
    ▼
Traefik (Let's Encrypt SSL automático)
    │
    ▼
frontend:80 (Nginx)
    ├── /api/*      → proxy → auth-service:8080
    ├── /health     → proxy → auth-service:8080
    ├── /.well-known → proxy → auth-service:8080
    └── /*          → React SPA

Red interna (sentinel-internal):
    auth-service ↔ postgres:5432
    auth-service ↔ redis:6379
```

### Requisitos

- **Droplet recomendado:** Ubuntu 22.04 LTS, 2 vCPU / 2 GB RAM / 50 GB SSD (~$18/mes)
  - Mínimo viable: 1 vCPU / 1 GB RAM ($6/mes, sin headroom)
- **DNS:** El dominio debe apuntar a la IP del Droplet antes de activar SSL

### Paso 1 — Crear y configurar el Droplet

En el panel de DigitalOcean:
1. Crear un Droplet con **Ubuntu 22.04 LTS**
2. Elegir **2 vCPU / 2 GB RAM** (Basic, $18/mes) o superior
3. Agregar tu clave SSH para acceso
4. Opcional: habilitar **Monitoring** y **Backups**

Conectarse vía SSH:
```bash
ssh root@<DROPLET_IP>
```

### Paso 2 — Ejecutar el script de setup

```bash
# Configurar la URL del repositorio (opcional)
export SENTINEL_REPO_URL=https://github.com/TU_ORG/sentinel.git

# Descargar y ejecutar el script
curl -fsSL https://raw.githubusercontent.com/TU_ORG/sentinel/main/deploy/digitalocean/setup.sh \
  | bash
```

El script instala Docker CE, genera las claves RSA (4096 bits) y configura UFW.

Alternativamente, ejecutar manualmente:
```bash
# En el Droplet
apt-get update && apt-get install -y docker.io docker-compose-plugin git openssl
git clone https://github.com/TU_ORG/sentinel.git /opt/sentinel
cd /opt/sentinel

# Generar claves RSA
mkdir -p deploy/digitalocean/files/keys
openssl genrsa -out deploy/digitalocean/files/keys/private.pem 4096
openssl rsa -in deploy/digitalocean/files/keys/private.pem \
            -pubout -out deploy/digitalocean/files/keys/public.pem
chmod 600 deploy/digitalocean/files/keys/private.pem
```

### Paso 3 — Configurar variables de entorno

```bash
cd /opt/sentinel
cp deploy/digitalocean/.env.example deploy/digitalocean/.env
nano deploy/digitalocean/.env
```

Completar todos los valores, especialmente:
- `DOMAIN=sentinel.tudominio.com` (debe estar apuntando al Droplet)
- `ACME_EMAIL=admin@tudominio.com`
- `DB_PASSWORD`, `REDIS_PASSWORD`, `BOOTSTRAP_ADMIN_PASSWORD`

### Paso 4 — Primer despliegue

```bash
cd /opt/sentinel

docker compose \
  -f deploy/digitalocean/docker-compose.yml \
  --env-file deploy/digitalocean/.env \
  up -d --build
```

Seguir los logs en tiempo real:
```bash
docker compose -f deploy/digitalocean/docker-compose.yml logs -f
```

Esperar hasta ver:
```
auth-service  | {"level":"info","msg":"server started","port":8080}
traefik       | level=info msg="Configuration loaded from providers"
```

### Paso 5 — Obtener el secret_key del bootstrap

```bash
docker compose -f deploy/digitalocean/docker-compose.yml \
  logs auth-service | grep "secret_key"
```

Copiar el valor de `secret_key`. Guardarlo en un lugar seguro.

### Paso 6 — Actualizar VITE_APP_KEY y redesplegar el frontend

```bash
# Editar .env y reemplazar el valor de VITE_APP_KEY
nano deploy/digitalocean/.env

# Reconstruir solo el frontend (el backend NO se reinicia)
docker compose \
  -f deploy/digitalocean/docker-compose.yml \
  --env-file deploy/digitalocean/.env \
  up -d --build frontend
```

### Paso 7 — Verificar el despliegue

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
cd /opt/sentinel
git pull
docker compose \
  -f deploy/digitalocean/docker-compose.yml \
  --env-file deploy/digitalocean/.env \
  up -d --build
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
