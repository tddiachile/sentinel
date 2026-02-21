#!/usr/bin/env bash
# =============================================================================
# deploy.sh — Script de despliegue completo para Azure Container Apps
#
# Ejecutar desde la raíz del repositorio:
#   bash deploy/azure/deploy.sh
#
# Requisitos: az CLI, Docker, OpenSSL
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuración — editar antes de ejecutar
# ---------------------------------------------------------------------------
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-rg-sentinel}"
LOCATION="${AZURE_LOCATION:-eastus}"
ACR_NAME="${ACR_NAME:-sentinelacr}"
APP_NAME="${APP_NAME:-sentinel}"
AUTH_TAG="${AUTH_IMAGE_TAG:-latest}"
FRONTEND_TAG="${FRONTEND_IMAGE_TAG:-latest}"

# Estos valores se pueden exportar como variables de entorno antes de ejecutar
: "${DB_PASSWORD:?ERROR: Exporta DB_PASSWORD antes de ejecutar}"
: "${BOOTSTRAP_ADMIN_PASSWORD:?ERROR: Exporta BOOTSTRAP_ADMIN_PASSWORD}"

# ---------------------------------------------------------------------------
# Colores
# ---------------------------------------------------------------------------
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
step() { echo -e "\n${GREEN}[$(date +%T)] $1${NC}"; }
warn() { echo -e "${YELLOW}AVISO: $1${NC}"; }

# ---------------------------------------------------------------------------
step "1/10 — Login y suscripción"
# ---------------------------------------------------------------------------
az account show > /dev/null 2>&1 || az login
if [[ -n "${AZURE_SUBSCRIPTION_ID:-}" ]]; then
  az account set --subscription "${AZURE_SUBSCRIPTION_ID}"
fi
echo "  Suscripción: $(az account show --query name -o tsv)"

# ---------------------------------------------------------------------------
step "2/10 — Grupo de recursos"
# ---------------------------------------------------------------------------
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}" --output none
echo "  Grupo: ${RESOURCE_GROUP} (${LOCATION})"

# ---------------------------------------------------------------------------
step "3/10 — Azure Container Registry"
# ---------------------------------------------------------------------------
az acr create \
  --name "${ACR_NAME}" \
  --resource-group "${RESOURCE_GROUP}" \
  --sku Basic \
  --admin-enabled false \
  --output none 2>/dev/null || true

ACR_LOGIN_SERVER=$(az acr show --name "${ACR_NAME}" --query loginServer -o tsv)
echo "  ACR: ${ACR_LOGIN_SERVER}"

# ---------------------------------------------------------------------------
step "4/10 — Generar claves RSA (si no existen)"
# ---------------------------------------------------------------------------
if [[ ! -f keys/private.pem ]]; then
  mkdir -p keys
  openssl genrsa -out keys/private.pem 4096 2>/dev/null
  openssl rsa -in keys/private.pem -pubout -out keys/public.pem 2>/dev/null
  chmod 600 keys/private.pem
  echo "  Claves RSA generadas en keys/"
else
  echo "  Claves RSA ya existen — omitiendo generación"
fi

# ---------------------------------------------------------------------------
step "5/10 — Construir y publicar imágenes en ACR"
# ---------------------------------------------------------------------------
az acr login --name "${ACR_NAME}"

echo "  Construyendo sentinel-auth:${AUTH_TAG}..."
docker build \
  --tag "${ACR_LOGIN_SERVER}/sentinel-auth:${AUTH_TAG}" \
  --file Dockerfile \
  --progress plain \
  .

echo "  Construyendo sentinel-frontend:${FRONTEND_TAG}..."
docker build \
  --tag "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_TAG}" \
  --file deploy/azure/Dockerfile.frontend \
  --build-arg VITE_API_URL="/api/v1" \
  --build-arg VITE_APP_KEY="${VITE_APP_KEY:-placeholder}" \
  --progress plain \
  .

docker push "${ACR_LOGIN_SERVER}/sentinel-auth:${AUTH_TAG}"
docker push "${ACR_LOGIN_SERVER}/sentinel-frontend:${FRONTEND_TAG}"
echo "  Imágenes publicadas."

# ---------------------------------------------------------------------------
step "6/10 — Desplegar infraestructura con Bicep"
# ---------------------------------------------------------------------------
DEPLOY_OUTPUT=$(az deployment group create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "sentinel-$(date +%Y%m%d%H%M%S)" \
  --template-file "deploy/azure/bicep/main.bicep" \
  --parameters "deploy/azure/bicep/main.bicepparam" \
  --parameters \
    acrLoginServer="${ACR_LOGIN_SERVER}" \
    dbPassword="${DB_PASSWORD}" \
    bootstrapAdminPassword="${BOOTSTRAP_ADMIN_PASSWORD}" \
    authImageTag="${AUTH_TAG}" \
    frontendImageTag="${FRONTEND_TAG}" \
  --output json)

KV_NAME=$(echo "${DEPLOY_OUTPUT}" | jq -r '.properties.outputs.keyVaultName.value')
echo "  Key Vault: ${KV_NAME}"
echo "  Deployment completado."

# ---------------------------------------------------------------------------
step "7/10 — Subir claves RSA al Key Vault"
# ---------------------------------------------------------------------------
echo "  Subiendo private.pem → Key Vault secret 'jwt-private-key'..."
az keyvault secret set \
  --vault-name "${KV_NAME}" \
  --name "jwt-private-key" \
  --file keys/private.pem \
  --output none

echo "  Subiendo public.pem → Key Vault secret 'jwt-public-key'..."
az keyvault secret set \
  --vault-name "${KV_NAME}" \
  --name "jwt-public-key" \
  --file keys/public.pem \
  --output none

echo "  Claves RSA almacenadas en Key Vault."

# ---------------------------------------------------------------------------
step "8/10 — Reiniciar auth-service para cargar las claves RSA"
# ---------------------------------------------------------------------------
az containerapp revision copy \
  --name "${APP_NAME}-auth" \
  --resource-group "${RESOURCE_GROUP}" \
  --output none
echo "  Nueva revisión creada."

# ---------------------------------------------------------------------------
step "9/10 — Esperar que auth-service esté healthy"
# ---------------------------------------------------------------------------
echo "  Esperando 30s para que el servicio arranque..."
sleep 30

AUTH_FQDN=$(az containerapp show \
  --name "${APP_NAME}-auth" \
  --resource-group "${RESOURCE_GROUP}" \
  --query properties.configuration.ingress.fqdn -o tsv)

for i in {1..12}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "https://${AUTH_FQDN}/health" 2>/dev/null || echo "000")
  if [[ "${STATUS}" == "200" ]]; then
    echo "  auth-service HEALTHY (${STATUS})"
    break
  fi
  echo "  Intento ${i}/12 — status ${STATUS}, esperando 10s..."
  sleep 10
done

# ---------------------------------------------------------------------------
step "10/10 — Obtener secret_key del bootstrap"
# ---------------------------------------------------------------------------
FRONTEND_URL=$(echo "${DEPLOY_OUTPUT}" | jq -r '.properties.outputs.frontendUrl.value')

echo ""
echo "======================================================"
echo "  Despliegue completado!"
echo "======================================================"
echo ""
echo "  Frontend URL: ${FRONTEND_URL}"
echo ""
echo "  Próximos pasos:"
echo "  1. Obtener el secret_key del bootstrap:"
echo "     az containerapp logs show \\"
echo "       --name ${APP_NAME}-auth \\"
echo "       --resource-group ${RESOURCE_GROUP} \\"
echo "       | grep secret_key"
echo ""
echo "  2. Exportar y redesplegar frontend con la clave real:"
echo "     export VITE_APP_KEY=<secret_key_obtenido>"
echo "     bash deploy/azure/deploy.sh"
echo ""
echo "  3. Verificar:"
echo "     curl ${FRONTEND_URL}/health"
warn "  Guarda un backup seguro de keys/private.pem fuera del repositorio!"
