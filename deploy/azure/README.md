# Despliegue de Sentinel en Azure

Stack completo sobre **Azure Container Apps** (serverless containers) con infraestructura definida como código en **Bicep**.

## Arquitectura

```
Internet (HTTPS)
    │
    ▼
Azure Container Apps (frontend) ← managed certificate automático
    │   Nginx: SPA + proxy /api (NGINX_AUTH_SERVICE_URL via envsubst)
    ▼
Container Apps (auth-service) ← internal ingress only
    │
    ├──→ Azure Database for PostgreSQL Flexible Server (:5432)
    └──→ Azure Cache for Redis (TLS :6380)

Secrets:         Azure Key Vault (con RBAC, acceso via Managed Identity)
RSA keys:        Key Vault Secret Volume Mount → /app/keys/ (sin Storage Account)
ACR pull:        User-Assigned Managed Identity (sin username/password)
Logs:            Log Analytics Workspace
```

**Ventaja del Secret Volume Mount:** Las claves RSA se montan directamente desde
Key Vault como archivos en `/app/keys/private.pem` y `/app/keys/public.pem`.
No se necesita Storage Account ni Azure Files. Auto-rotan en ~30 min al actualizarse en KV.

## Estructura de archivos

```
deploy/azure/
├── bicep/
│   ├── main.bicep              # Template principal (todos los recursos)
│   └── main.bicepparam         # Parámetros del deployment
├── Dockerfile.frontend         # Build React + Nginx con envsubst
├── nginx.conf.template         # Config Nginx (auth-service URL dinámica)
├── deploy.sh                   # Script de despliegue completo (10 pasos)
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

## Despliegue automatizado (recomendado)

El script `deploy.sh` ejecuta todos los pasos de forma automática:

```bash
# Exportar variables obligatorias
export AZURE_RESOURCE_GROUP=rg-sentinel
export AZURE_LOCATION=eastus
export ACR_NAME=mysentinelacr
export DB_PASSWORD=<contraseña_fuerte>
export BOOTSTRAP_ADMIN_PASSWORD=<contraseña_bootstrap>

# Ejecutar desde la raíz del repositorio
bash deploy/azure/deploy.sh
```

El script realiza: login → grupo de recursos → ACR → RSA keys → build imágenes → Bicep → upload keys a Key Vault → reinicio → verificación → instrucciones para VITE_APP_KEY.

---

## Despliegue manual paso a paso

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

## Paso 5 — Subir las claves RSA al Key Vault

Las claves RSA se montan como **Secret Volume** desde Key Vault (no se necesita Storage Account).

```bash
# Obtener el nombre del Key Vault del output de Bicep
KV_NAME=$(az deployment group show \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name main \
  --query properties.outputs.keyVaultName.value -o tsv)

# Conceder acceso temporal para subir las claves (Key Vault Secrets Officer)
MY_ID=$(az ad signed-in-user show --query id -o tsv)
az role assignment create \
  --role "Key Vault Secrets Officer" \
  --assignee "${MY_ID}" \
  --scope "/subscriptions/$(az account show --query id -o tsv)/resourceGroups/${AZURE_RESOURCE_GROUP}/providers/Microsoft.KeyVault/vaults/${KV_NAME}"

# Subir las claves RSA como secretos de Key Vault
az keyvault secret set \
  --vault-name "${KV_NAME}" \
  --name "jwt-private-key" \
  --file /tmp/sentinel-keys/private.pem

az keyvault secret set \
  --vault-name "${KV_NAME}" \
  --name "jwt-public-key" \
  --file /tmp/sentinel-keys/public.pem

echo "Claves RSA almacenadas en Key Vault como secretos."
echo "Se montarán automáticamente en /app/keys/ del Container App."

# Limpiar (mantener backup seguro fuera del repo)
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

Las claves no están en Key Vault o la Managed Identity no tiene acceso. Verificar:
```bash
# 1. Comprobar que los secretos existen en Key Vault
KV_NAME=$(az deployment group show --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name main --query properties.outputs.keyVaultName.value -o tsv)

az keyvault secret list --vault-name "${KV_NAME}" --output table
# Debe mostrar: jwt-private-key, jwt-public-key

# 2. Si faltan, volver al Paso 5 y subirlos.
```

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
