// =============================================================================
// main.bicep — Sentinel Auth Service on Azure Container Apps
//
// Deploys:
//   - User-Assigned Managed Identity (ACR pull + Key Vault access)
//   - Key Vault (secrets + RSA keys as Key Vault secrets)
//   - Container Apps Environment (with Log Analytics)
//   - Auth Service Container App (internal ingress, RSA keys via Secret volume)
//   - Frontend Container App (external ingress, HTTPS)
//   - PostgreSQL Flexible Server
//   - Azure Cache for Redis
//
// NOTE: No Storage Account required — RSA keys are mounted as Key Vault
//       Secret volumes directly in the Container App. This is simpler and
//       more secure than using Azure Files.
//
// Prerequisites:
//   1. Azure Container Registry with sentinel-auth and sentinel-frontend images
//   2. RSA keys stored in Key Vault (step in README):
//      az keyvault secret set --vault-name <kv> --name jwt-private-key --file keys/private.pem
//      az keyvault secret set --vault-name <kv> --name jwt-public-key  --file keys/public.pem
//
// Deploy:
//   az deployment group create \
//     --resource-group rg-sentinel \
//     --template-file deploy/azure/bicep/main.bicep \
//     --parameters deploy/azure/bicep/main.bicepparam
// =============================================================================

targetScope = 'resourceGroup'

// ---------------------------------------------------------------------------
// Parameters
// ---------------------------------------------------------------------------
@description('Base name for all resources (lowercase, no spaces).')
param appName string = 'sentinel'

@description('Azure region. Defaults to resource group location.')
param location string = resourceGroup().location

@description('Azure Container Registry login server (e.g. myacr.azurecr.io).')
param acrLoginServer string

@description('Tag for the auth-service container image.')
param authImageTag string = 'latest'

@description('Tag for the frontend container image.')
param frontendImageTag string = 'latest'

@description('PostgreSQL admin password.')
@secure()
param dbPassword string

@description('Bootstrap admin password for first startup.')
@secure()
param bootstrapAdminPassword string

@description('Bootstrap admin username.')
param bootstrapAdminUser string = 'admin'

@description('Custom domain (leave empty to use the default .azurecontainerapps.io domain).')
param customDomain string = ''

@description('Frontend VITE_APP_KEY — set after first deploy from bootstrap logs.')
@secure()
param viteAppKey string = 'placeholder'

// ---------------------------------------------------------------------------
// Unique suffix for globally unique resource names
// ---------------------------------------------------------------------------
var uniqueSuffix = uniqueString(resourceGroup().id)
var shortName    = toLower(take(replace(replace(appName, '-', ''), '_', ''), 8))

// Built-in role definition IDs
var keyVaultSecretsUserRoleId   = '4633458b-17de-408a-b874-0445c86b69e6'
var acrPullRoleId               = '7f951dda-4ed3-4680-a7ca-43fe172d538d'

// ---------------------------------------------------------------------------
// User-Assigned Managed Identity
// Used for: ACR image pull + Key Vault Secrets access (no passwords needed)
// ---------------------------------------------------------------------------
resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${appName}-identity'
  location: location
}

// ---------------------------------------------------------------------------
// Log Analytics Workspace
// ---------------------------------------------------------------------------
resource logWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: '${appName}-logs-${uniqueSuffix}'
  location: location
  properties: {
    sku: { name: 'PerGB2018' }
    retentionInDays: 30
  }
}

// ---------------------------------------------------------------------------
// Key Vault
// Stores: DB password, bootstrap password, Redis key, RSA private/public keys
// Access: managed identity via RBAC (Key Vault Secrets User role)
// ---------------------------------------------------------------------------
resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: '${shortName}kv${uniqueSuffix}'
  location: location
  properties: {
    sku: { family: 'A', name: 'standard' }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true
    softDeleteRetentionInDays: 7
    enableSoftDelete: true
    enabledForDeployment: false
    enabledForTemplateDeployment: false
  }
}

// Grant managed identity access to Key Vault secrets
resource kvSecretsUserRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: keyVault
  name: guid(keyVault.id, identity.id, keyVaultSecretsUserRoleId)
  properties: {
    principalId:      identity.properties.principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', keyVaultSecretsUserRoleId)
    principalType:    'ServicePrincipal'
  }
}

// Store DB password in Key Vault
resource kvDbPassword 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = {
  parent: keyVault
  name:   'db-password'
  properties: { value: dbPassword }
}

// Store bootstrap password in Key Vault
resource kvBootstrapPassword 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = {
  parent: keyVault
  name:   'bootstrap-admin-password'
  properties: { value: bootstrapAdminPassword }
}

// Store frontend app key in Key Vault
resource kvViteAppKey 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = {
  parent: keyVault
  name:   'vite-app-key'
  properties: { value: viteAppKey }
}

// NOTE: RSA keys (jwt-private-key, jwt-public-key) must be stored BEFORE deployment:
//   az keyvault secret set --vault-name <kv-name> --name jwt-private-key --file keys/private.pem
//   az keyvault secret set --vault-name <kv-name> --name jwt-public-key  --file keys/public.pem
// The Bicep references them but does not create them (content comes from PEM files).

// ---------------------------------------------------------------------------
// ACR pull role for managed identity
// ---------------------------------------------------------------------------
resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: last(split(acrLoginServer, '.'))!
}

resource acrPullRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: acr
  name: guid(acr.id, identity.id, acrPullRoleId)
  properties: {
    principalId:      identity.properties.principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', acrPullRoleId)
    principalType:    'ServicePrincipal'
  }
}

// ---------------------------------------------------------------------------
// PostgreSQL Flexible Server
// ---------------------------------------------------------------------------
resource postgres 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {
  name: '${appName}-pg-${uniqueSuffix}'
  location: location
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'          // Change to 'GeneralPurpose' for production HA
  }
  properties: {
    administratorLogin:         'sentinel'
    administratorLoginPassword: dbPassword
    version:                    '15'
    storage: { storageSizeGB: 32 }
    backup: {
      backupRetentionDays:  7
      geoRedundantBackup:   'Disabled'
    }
    highAvailability: { mode: 'Disabled' }  // Set 'ZoneRedundant' for production
    authConfig: {
      activeDirectoryAuth: 'Disabled'
      passwordAuth:        'Enabled'
    }
  }
}

resource postgresDatabase 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2023-06-01-preview' = {
  parent: postgres
  name: 'sentinel'
  properties: { charset: 'UTF8', collation: 'en_US.utf8' }
}

resource postgresFirewallRule 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2023-06-01-preview' = {
  parent: postgres
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress:   '0.0.0.0'
  }
}

// ---------------------------------------------------------------------------
// Azure Cache for Redis
// ---------------------------------------------------------------------------
resource redis 'Microsoft.Cache/redis@2023-08-01' = {
  name: '${appName}-redis-${uniqueSuffix}'
  location: location
  properties: {
    sku: { name: 'Basic', family: 'C', capacity: 0 }
    enableNonSslPort: false
    minimumTlsVersion: '1.2'
  }
}

// Store Redis key in Key Vault
resource kvRedisKey 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = {
  parent: keyVault
  name:   'redis-password'
  properties: { value: redis.listKeys().primaryKey }
}

// ---------------------------------------------------------------------------
// Container Apps Environment
// ---------------------------------------------------------------------------
resource environment 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: '${appName}-env'
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logWorkspace.properties.customerId
        sharedKey:  logWorkspace.listKeys().primarySharedKey
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Auth Service Container App
// - Internal ingress only (frontend Nginx proxies to it)
// - RSA keys mounted as Secret volume from Key Vault (no Storage Account needed)
// - All secrets referenced via Key Vault URLs + managed identity
// ---------------------------------------------------------------------------
resource authService 'Microsoft.App/containerApps@2023-05-01' = {
  name: '${appName}-auth'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identity.id}': {}
    }
  }
  properties: {
    managedEnvironmentId: environment.id
    configuration: {
      secrets: [
        // Key Vault-backed secrets — auto-rotate within 30 minutes on KV change
        {
          name:        'db-password'
          keyVaultUrl: kvDbPassword.properties.secretUri
          identity:    identity.id
        }
        {
          name:        'redis-password'
          keyVaultUrl: kvRedisKey.properties.secretUri
          identity:    identity.id
        }
        {
          name:        'bootstrap-password'
          keyVaultUrl: kvBootstrapPassword.properties.secretUri
          identity:    identity.id
        }
        // RSA keys — referenced from Key Vault, mounted as files in /app/keys
        {
          name:        'jwt-private-key'
          keyVaultUrl: 'https://${keyVault.name}.vault.azure.net/secrets/jwt-private-key'
          identity:    identity.id
        }
        {
          name:        'jwt-public-key'
          keyVaultUrl: 'https://${keyVault.name}.vault.azure.net/secrets/jwt-public-key'
          identity:    identity.id
        }
      ]
      registries: [
        {
          server:   acrLoginServer
          identity: identity.id    // Pull via managed identity — no username/password
        }
      ]
      ingress: {
        external:   false          // Internal only — frontend proxies to this
        targetPort: 8080
        transport:  'http'
      }
    }
    template: {
      volumes: [
        {
          // Secret volume: each secret becomes a file at mountPath/path
          name:        'keys-volume'
          storageType: 'Secret'
          secrets: [
            { secretRef: 'jwt-private-key', path: 'private.pem' }
            { secretRef: 'jwt-public-key',  path: 'public.pem'  }
          ]
        }
      ]
      containers: [
        {
          name:  'auth-service'
          image: '${acrLoginServer}/sentinel-auth:${authImageTag}'
          resources: { cpu: json('0.5'), memory: '1Gi' }
          volumeMounts: [
            { volumeName: 'keys-volume', mountPath: '/app/keys' }
          ]
          env: [
            { name: 'DB_HOST',                  value:     postgres.properties.fullyQualifiedDomainName }
            { name: 'DB_NAME',                  value:     'sentinel' }
            { name: 'DB_USER',                  value:     'sentinel' }
            { name: 'DB_PASSWORD',              secretRef: 'db-password' }
            { name: 'REDIS_ADDR',               value:     '${redis.properties.hostName}:6380' }
            { name: 'REDIS_PASSWORD',           secretRef: 'redis-password' }
            { name: 'JWT_PRIVATE_KEY_PATH',     value:     '/app/keys/private.pem' }
            { name: 'JWT_PUBLIC_KEY_PATH',      value:     '/app/keys/public.pem' }
            { name: 'BOOTSTRAP_ADMIN_USER',     value:     bootstrapAdminUser }
            { name: 'BOOTSTRAP_ADMIN_PASSWORD', secretRef: 'bootstrap-password' }
          ]
          probes: [
            {
              type: 'Liveness'
              httpGet: { path: '/health', port: 8080, scheme: 'HTTP' }
              initialDelaySeconds: 15
              periodSeconds: 10
              failureThreshold: 3
            }
            {
              type: 'Readiness'
              httpGet: { path: '/health', port: 8080, scheme: 'HTTP' }
              initialDelaySeconds: 10
              periodSeconds: 5
            }
          ]
        }
      ]
      scale: {
        minReplicas: 1
        maxReplicas: 5
        rules: [
          {
            name: 'http-scale'
            http: { metadata: { concurrentRequests: '50' } }
          }
        ]
      }
    }
  }
  dependsOn: [kvSecretsUserRole, acrPullRole]
}

// ---------------------------------------------------------------------------
// Frontend Container App (external, HTTPS)
// ---------------------------------------------------------------------------
resource frontend 'Microsoft.App/containerApps@2023-05-01' = {
  name: '${appName}-frontend'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identity.id}': {}
    }
  }
  properties: {
    managedEnvironmentId: environment.id
    configuration: {
      registries: [
        {
          server:   acrLoginServer
          identity: identity.id
        }
      ]
      ingress: {
        external:   true
        targetPort: 80
        transport:  'http'
        customDomains: empty(customDomain) ? [] : [
          {
            name:        customDomain
            bindingType: 'SniEnabled'
          }
        ]
      }
    }
    template: {
      containers: [
        {
          name:  'frontend'
          image: '${acrLoginServer}/sentinel-frontend:${frontendImageTag}'
          resources: { cpu: json('0.25'), memory: '0.5Gi' }
          env: [
            // Internal FQDN of auth-service within the Container Apps environment
            { name: 'NGINX_AUTH_SERVICE_URL', value: 'http://${appName}-auth' }
          ]
          probes: [
            {
              type: 'Liveness'
              httpGet: { path: '/', port: 80, scheme: 'HTTP' }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
          ]
        }
      ]
      scale: {
        minReplicas: 1
        maxReplicas: 3
        rules: [
          {
            name: 'http-scale'
            http: { metadata: { concurrentRequests: '30' } }
          }
        ]
      }
    }
  }
  dependsOn: [authService, acrPullRole]
}

// ---------------------------------------------------------------------------
// Outputs
// ---------------------------------------------------------------------------
output frontendUrl        string = 'https://${frontend.properties.configuration.ingress.fqdn}'
output authServiceFqdn    string = authService.properties.configuration.ingress.fqdn
output postgresHost       string = postgres.properties.fullyQualifiedDomainName
output redisHost          string = redis.properties.hostName
output keyVaultName       string = keyVault.name
output managedIdentityId  string = identity.id
