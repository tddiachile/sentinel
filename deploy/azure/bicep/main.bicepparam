// =============================================================================
// main.bicepparam — Parameters for Sentinel Azure Container Apps deployment
//
// Usage:
//   az deployment group create \
//     --resource-group rg-sentinel \
//     --template-file deploy/azure/bicep/main.bicep \
//     --parameters deploy/azure/bicep/main.bicepparam
// =============================================================================

using './main.bicep'

// Base name used in all resource names
param appName = 'sentinel'

// Azure region (must match the resource group region)
param location = 'eastus'

// Container Registry — replace with your ACR details
// az acr show --name <myacr> --query loginServer -o tsv
param acrLoginServer = 'REPLACE_WITH_ACR_LOGIN_SERVER'
param acrUsername    = 'REPLACE_WITH_ACR_USERNAME'

// Secrets — use az keyvault secret set or provide interactively
// These will prompt during deployment if left as empty strings
param acrPassword           = 'REPLACE_WITH_ACR_PASSWORD'
param dbPassword            = 'REPLACE_WITH_DB_PASSWORD'
param bootstrapAdminPassword = 'REPLACE_WITH_BOOTSTRAP_PASSWORD'

// redisPassword is read automatically from redis.listKeys() in the template
// Leave empty — populated automatically during deployment
param redisPassword = ''

// Custom domain (optional — leave empty to use default .azurecontainerapps.io domain)
// Point CNAME to the frontend FQDN output before adding this
param customDomain = ''

// Frontend app key — set AFTER first deploy (see README step 6)
param viteAppKey = 'placeholder'
