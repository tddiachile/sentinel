// =============================================================================
// main.bicepparam — Parameters for Sentinel Azure Container Apps deployment
//
// Usage (interactive — prompts for @secure params):
//   az deployment group create \
//     --resource-group rg-sentinel \
//     --template-file deploy/azure/bicep/main.bicep \
//     --parameters deploy/azure/bicep/main.bicepparam
//
// Usage (non-interactive — ideal for CI/CD):
//   az deployment group create \
//     --resource-group rg-sentinel \
//     --template-file deploy/azure/bicep/main.bicep \
//     --parameters deploy/azure/bicep/main.bicepparam \
//     --parameters dbPassword=$DB_PASSWORD \
//                  bootstrapAdminPassword=$BOOTSTRAP_ADMIN_PASSWORD
// =============================================================================

using './main.bicep'

param appName    = 'sentinel'
param location   = 'eastus'    // Must match resource group location

// ACR login server: az acr show --name <myacr> --query loginServer -o tsv
param acrLoginServer = 'REPLACE_WITH_ACR_LOGIN_SERVER'

param authImageTag     = 'latest'
param frontendImageTag = 'latest'

// @secure params — omit here and provide via --parameters flag or env vars
// param dbPassword             = ''  // DO NOT commit
// param bootstrapAdminPassword = ''  // DO NOT commit

param bootstrapAdminUser = 'admin'

// Custom domain (optional). Leave empty to use default *.azurecontainerapps.io
param customDomain = ''

// Set after first deploy (see README step 7)
// param viteAppKey = ''  // DO NOT commit
