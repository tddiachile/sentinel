# Despliegue de Sentinel en Azure

Stack completo sobre **Azure Container Apps** (serverless containers) con infraestructura definida como código en **Bicep**.

## Arquitectura

```
Internet (HTTPS)
    │
    ▼
Azure Container Apps (frontend) ← managed certificate (Let's Encrypt)
    │   Nginx: SPA + proxy /api
    ▼
Container Apps (auth-service) ← internal ingress only
    │
    ├──→ Azure Database for PostgreSQL Flexible Server
    └──→ Azure Cache for Redis (TLS :6380)

Secrets:      Azure Key Vault
RSA keys:     Azure Files (Storage Account → File Share → Volume Mount)
Images:       Azure Container Registry (ACR)
Logs:         Log Analytics Workspace
```

## Estructura de archivos

```
deploy/azure/
├── bicep/
│   ├── main.bicep              # Template principal (todos los recursos)
│   └── main.bicepparam         # Parámetros del deployment
├── Dockerfile.frontend         # Build React + Nginx con envsubst
├── nginx.conf.template         # Config Nginx (auth-service URL dinámica)
├── .env.example                # Variables para az CLI
└── README.md                   # Esta guía
```

## Requisitos previos

| Herramienta | Versión mínima | Instalación |
|---|---|---|
| Azure CLI | 2.50+ | [docs.microsoft.com/cli/azure/install](https://docs.microsoft.com/cli/azure/install-azure-cli) |
| Docker | 24+ | Para construir y hacer push de imágenes |
| OpenSSL | 3.0+ | Para generar claves RSA |
| Bicep CLI | Incluido en az CLI 2.20+ | `az bicep install` |

---

## Paso 1 — Preparar el entorno Azure

```bash
# Cargar variables de entorno
source deploy/azure/.env.example   # Editar primero con valores reales

# Login en Azure
az login
az account set --subscription "${AZURE_SUBSCRIPTION_ID}"

# Crear grupo de recursos
az group create \
  --name "${AZURE_RESOURCE_GROUP}" \
  --location "${AZURE_LOCATION}"

# Crear Azure Container Registry
az acr create \
  --name "${ACR_NAME}" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --sku Basic \
  --admin-enabled true

# Guardar las credenciales del ACR
ACR_LOGIN_SERVER=$(az acr show --name "${ACR_NAME}" --query loginServer -o tsv)
ACR_USERNAME=$(az acr credential show --name "${ACR_NAME}" --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name "${ACR_NAME}" --query passwords[0].value -o tsv)

echo "ACR: ${ACR_LOGIN_SERVER}"
echo "Usuario: ${ACR_USERNAME}"
```

---

## Paso 2 — Generar claves RSA

```bash
# Generar par de claves RSA-4096 para JWT RS256
mkdir -p /tmp/sentinel-keys
openssl genrsa -out /tmp/sentinel-keys/private.pem 4096
openssl rsa -in /tmp/sentinel-keys/private.pem \
            -pubout -out /tmp/sentinel-keys/public.pem
chmod 600 /tmp/sentinel-keys/private.pem

echo "Claves generadas en /tmp/sentinel-keys/"
```

Las claves se subirán al Azure Files share en el Paso 5.

---

## Paso 3 — Construir y publicar las imágenes en ACR

```bash
# Login en ACR
az acr login --name "${ACR_NAME}"

# --- Imagen del backend (Go auth-service) ---
docker build \
  --tag "${ACR_LOGIN_SERVER}/sentinel-auth:${AUTH_IMAGE_TAG:-latest}" \
  --file Dockerfile \
  .

docker push "${ACR_LOGIN_SERVER}/sentinel-auth:${AUTH_IMAGE_TAG:-latest}"

# --- Imagen del frontend (React + Nginx) ---
# VITE_APP_KEY: usar 'placeholder' en el primer build (se actualiza en paso 7)
docker build \
  --tag "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_IMAGE_TAG:-latest}" \
  --file deploy/azure/Dockerfile.frontend \
  --build-arg VITE_API_URL="/api/v1" \
  --build-arg VITE_APP_KEY="placeholder" \
  .

docker push "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_IMAGE_TAG:-latest}"
```

---

## Paso 4 — Desplegar la infraestructura con Bicep

Editar los parámetros en `deploy/azure/bicep/main.bicepparam` con los valores reales,
o pasar los parámetros directamente:

```bash
az deployment group create \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --template-file deploy/azure/bicep/main.bicep \
  --parameters deploy/azure/bicep/main.bicepparam \
  --parameters \
    acrLoginServer="${ACR_LOGIN_SERVER}" \
    acrUsername="${ACR_USERNAME}" \
    acrPassword="${ACR_PASSWORD}" \
    dbPassword="${DB_PASSWORD}" \
    bootstrapAdminPassword="${BOOTSTRAP_ADMIN_PASSWORD}"
```

El deployment tarda ~5-10 minutos. Al finalizar, muestra los outputs:

```bash
# Ver outputs del deployment
az deployment group show \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name main \
  --query properties.outputs
```

Guardar `frontendUrl` y `storageAccountName`.

---

## Paso 5 — Subir las claves RSA al Azure Files share

```bash
# Obtener el nombre de la Storage Account del output de Bicep
STORAGE_ACCOUNT=$(az deployment group show \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name main \
  --query properties.outputs.storageAccountName.value -o tsv)

STORAGE_KEY=$(az storage account keys list \
  --account-name "${STORAGE_ACCOUNT}" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --query [0].value -o tsv)

# Subir las claves RSA al file share
az storage file upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --account-key "${STORAGE_KEY}" \
  --share-name "sentinel-keys" \
  --source /tmp/sentinel-keys/private.pem \
  --path private.pem

az storage file upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --account-key "${STORAGE_KEY}" \
  --share-name "sentinel-keys" \
  --source /tmp/sentinel-keys/public.pem \
  --path public.pem

echo "Claves RSA subidas al Azure Files share 'sentinel-keys'"

# Limpiar las claves del sistema local (mantener un backup seguro)
# shred -u /tmp/sentinel-keys/private.pem
```

---

## Paso 6 — Reiniciar auth-service para que tome las claves

```bash
# Forzar un nuevo deployment del Container App
az containerapp revision copy \
  --name "sentinel-auth" \
  --resource-group "${AZURE_RESOURCE_GROUP}"
```

Seguir los logs:
```bash
az containerapp logs show \
  --name "sentinel-auth" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --follow
```

Señal de éxito:
```
{"level":"info","msg":"bootstrap completed"}
{"level":"info","msg":"server started","port":8080}
```

---

## Paso 7 — Obtener el secret_key y actualizar el frontend

```bash
# Buscar el secret_key en los logs del bootstrap
az containerapp logs show \
  --name "sentinel-auth" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  | grep "secret_key"
```

Copiar el valor del `secret_key`. Luego actualizar el frontend:

```bash
export VITE_APP_KEY="<secret_key_obtenido>"

# Reconstruir la imagen del frontend con la clave real
docker build \
  --tag "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_IMAGE_TAG:-latest}" \
  --file deploy/azure/Dockerfile.frontend \
  --build-arg VITE_API_URL="/api/v1" \
  --build-arg VITE_APP_KEY="${VITE_APP_KEY}" \
  .

docker push "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_IMAGE_TAG:-latest}"

# Hacer rollout de la nueva imagen
az containerapp update \
  --name "sentinel-frontend" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --image "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_IMAGE_TAG:-latest}"
```

---

## Paso 8 — Configurar dominio personalizado (opcional)

```bash
# Obtener el FQDN del frontend
FRONTEND_FQDN=$(az containerapp show \
  --name "sentinel-frontend" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --query properties.configuration.ingress.fqdn -o tsv)

echo "URL por defecto: https://${FRONTEND_FQDN}"

# Para dominio personalizado:
# 1. Crear registro CNAME: sentinel.tudominio.com → ${FRONTEND_FQDN}
# 2. Agregar dominio al Container App:
az containerapp hostname add \
  --hostname "${CUSTOM_DOMAIN}" \
  --name "sentinel-frontend" \
  --resource-group "${AZURE_RESOURCE_GROUP}"

# 3. Crear y vincular managed certificate (Let's Encrypt gestionado por Azure)
az containerapp ssl upload \
  --hostname "${CUSTOM_DOMAIN}" \
  --name "sentinel-frontend" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --certificate-type managed
```

---

## Paso 9 — Verificar el despliegue

```bash
FRONTEND_URL=$(az deployment group show \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name main \
  --query properties.outputs.frontendUrl.value -o tsv)

# Health check (vía frontend → proxy → auth-service)
curl "${FRONTEND_URL}/health"
# Esperado: {"status":"healthy","checks":{"postgresql":"ok","redis":"ok"}}

# JWKS
curl "${FRONTEND_URL}/.well-known/jwks.json"

# Login
curl -X POST "${FRONTEND_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-App-Key: <secret_key>" \
  -d '{"username":"admin","password":"<bootstrap_password>","client_type":"web"}'
```

---

## Costos estimados

| Recurso | SKU | Costo aprox./mes |
|---|---|---|
| Container Apps (auth) | 0.5 vCPU / 1 GB, min 1 replica | ~$12 |
| Container Apps (frontend) | 0.25 vCPU / 0.5 GB, min 1 replica | ~$6 |
| PostgreSQL Flexible Server | Standard_B1ms, 32 GB | ~$15 |
| Azure Cache for Redis | Basic C0 (250 MB) | ~$16 |
| Container Registry | Basic | ~$5 |
| Storage Account (keys) | LRS, mínimo | ~$1 |
| Log Analytics | Pay-per-use | ~$2-5 |
| **Total estimado** | | **~$57-60/mes** |

Para reducir costos en staging/dev: usar Redis Basic C0 y PostgreSQL Burstable B1ms.

---

## Actualizaciones continuas (CI/CD)

### GitHub Actions básico

```yaml
# .github/workflows/deploy-azure.yml
name: Deploy to Azure
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v1
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}
      - name: Build and push backend
        run: |
          az acr build \
            --registry ${{ secrets.ACR_NAME }} \
            --image sentinel-auth:${{ github.sha }} \
            --file Dockerfile .
      - name: Update Container App
        run: |
          az containerapp update \
            --name sentinel-auth \
            --resource-group ${{ secrets.AZURE_RESOURCE_GROUP }} \
            --image ${{ secrets.ACR_LOGIN_SERVER }}/sentinel-auth:${{ github.sha }}
```

---

## Resolución de problemas

### auth-service no encuentra las claves RSA

```
Error: open /app/keys/private.pem: no such file or directory
```

Verificar que el file share tiene los archivos:
```bash
az storage file list \
  --account-name "${STORAGE_ACCOUNT}" \
  --share-name "sentinel-keys" \
  --output table
```

Si faltan, volver al Paso 5.

### Error de conexión a PostgreSQL — SSL required

Azure PostgreSQL Flexible Server requiere SSL por defecto. El driver pgx de Go maneja esto automáticamente. Si hay errores, verificar que `DB_HOST` termina en `.postgres.database.azure.com`.

### Redis — conexión rechazada

Azure Cache for Redis solo acepta conexiones TLS en el puerto **6380** (no 6379). El `REDIS_ADDR` debe ser `<host>:6380`. La variable `REDIS_PASSWORD` es la primary access key del Redis.

### Frontend muestra pantalla en blanco

Verificar que `NGINX_AUTH_SERVICE_URL` en el Container App del frontend apunta al nombre correcto del auth-service Container App (e.g. `http://sentinel-auth`).

### Rotación de claves RSA

```bash
# Generar nuevas claves
openssl genrsa -out /tmp/new-private.pem 4096
openssl rsa -in /tmp/new-private.pem -pubout -out /tmp/new-public.pem

# Subir al file share (sobreescribe las anteriores)
az storage file upload --account-name "${STORAGE_ACCOUNT}" \
  --account-key "${STORAGE_KEY}" --share-name "sentinel-keys" \
  --source /tmp/new-private.pem --path private.pem

# Reiniciar auth-service (esperar 60 min para que expiren tokens antiguos)
az containerapp revision copy --name "sentinel-auth" \
  --resource-group "${AZURE_RESOURCE_GROUP}"
```
