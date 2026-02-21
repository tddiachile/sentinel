// =============================================================================
// main.bicep — Sentinel Auth Service on Azure Container Apps
//
// Deploys:
//   - Container Apps Environment (with Log Analytics)
//   - Auth Service Container App (internal ingress)
//   - Frontend Container App (external ingress, HTTPS)
//   - PostgreSQL Flexible Server
//   - Azure Cache for Redis
//   - Storage Account + File Share (for RSA keys)
//   - Key Vault (for secrets)
//
// Prerequisites:
//   - Azure Container Registry with sentinel-auth and sentinel-frontend images
//   - RSA key files uploaded to the Azure Files share (see README)
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

@description('Container registry login server (e.g. myacr.azurecr.io).')
param acrLoginServer string

@description('Container registry username.')
param acrUsername string

@description('Container registry password.')
@secure()
param acrPassword string

@description('Tag for the auth-service container image.')
param authImageTag string = 'latest'

@description('Tag for the frontend container image.')
param frontendImageTag string = 'latest'

@description('PostgreSQL admin password.')
@secure()
param dbPassword string

@description('Redis cache access key (auto-populated after Redis creation).')
@secure()
param redisPassword string = ''

@description('Bootstrap admin password for first startup.')
@secure()
param bootstrapAdminPassword string

@description('Bootstrap admin username.')
param bootstrapAdminUser string = 'admin'

@description('Custom domain for the frontend (leave empty to use the default Container Apps domain).')
param customDomain string = ''

@description('Frontend VITE_APP_KEY — set after first deploy from bootstrap logs.')
@secure()
param viteAppKey string = 'placeholder'

// ---------------------------------------------------------------------------
// Unique suffix for globally unique resource names
// ---------------------------------------------------------------------------
var uniqueSuffix = uniqueString(resourceGroup().id)
var shortName    = toLower(take(replace(appName, '-', ''), 8))

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
// Key Vault — stores DB password, bootstrap password, RSA key secrets
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
  }
}

// ---------------------------------------------------------------------------
// Storage Account — Azure Files for RSA key persistence
// ---------------------------------------------------------------------------
resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: '${shortName}keys${uniqueSuffix}'
  location: location
  sku: { name: 'Standard_LRS' }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
  }
}

resource fileService 'Microsoft.Storage/storageAccounts/fileServices@2023-01-01' = {
  parent: storageAccount
  name: 'default'
}

resource keysFileShare 'Microsoft.Storage/storageAccounts/fileServices/shares@2023-01-01' = {
  parent: fileService
  name: 'sentinel-keys'
  properties: {
    accessTier: 'TransactionOptimized'
    shareQuota: 1
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
    tier: 'Burstable'
  }
  properties: {
    administratorLogin: 'sentinel'
    administratorLoginPassword: dbPassword
    version: '15'
    storage: { storageSizeGB: 32 }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: { mode: 'Disabled' }
    authConfig: {
      activeDirectoryAuth: 'Disabled'
      passwordAuth: 'Enabled'
    }
  }
}

resource postgresDatabase 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2023-06-01-preview' = {
  parent: postgres
  name: 'sentinel'
  properties: { charset: 'UTF8', collation: 'en_US.utf8' }
}

// Allow Azure services (Container Apps) to connect to PostgreSQL
resource postgresFirewallRule 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2023-06-01-preview' = {
  parent: postgres
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
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
    redisConfiguration: {}
  }
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
        sharedKey: logWorkspace.listKeys().primarySharedKey
      }
    }
  }
}

// Attach Azure Files to the Container Apps Environment
resource envStorage 'Microsoft.App/managedEnvironments/storages@2023-05-01' = {
  parent: environment
  name: 'sentinel-keys'
  properties: {
    azureFile: {
      accountName: storageAccount.name
      accountKey: storageAccount.listKeys().keys[0].value
      shareName: 'sentinel-keys'
      accessMode: 'ReadOnly'
    }
  }
}

// ---------------------------------------------------------------------------
// Auth Service Container App (internal — only accessible within the env)
// ---------------------------------------------------------------------------
resource authService 'Microsoft.App/containerApps@2023-05-01' = {
  name: '${appName}-auth'
  location: location
  properties: {
    managedEnvironmentId: environment.id
    configuration: {
      secrets: [
        { name: 'db-password',        value: dbPassword }
        { name: 'redis-key',          value: redis.listKeys().primaryKey }
        { name: 'bootstrap-password', value: bootstrapAdminPassword }
        { name: 'acr-password',       value: acrPassword }
      ]
      registries: [
        {
          server:            acrLoginServer
          username:          acrUsername
          passwordSecretRef: 'acr-password'
        }
      ]
      ingress: {
        external: false          // Internal only — frontend proxies to this
        targetPort: 8080
        transport: 'http'
      }
    }
    template: {
      volumes: [
        {
          name:        'keys-volume'
          storageType: 'AzureFile'
          storageName: 'sentinel-keys'
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
            { name: 'DB_HOST',                    value: postgres.properties.fullyQualifiedDomainName }
            { name: 'DB_NAME',                    value: 'sentinel' }
            { name: 'DB_USER',                    value: 'sentinel' }
            { name: 'DB_PASSWORD',                secretRef: 'db-password' }
            { name: 'REDIS_ADDR',                 value: '${redis.properties.hostName}:6380' }
            { name: 'REDIS_PASSWORD',             secretRef: 'redis-key' }
            { name: 'JWT_PRIVATE_KEY_PATH',       value: '/app/keys/private.pem' }
            { name: 'JWT_PUBLIC_KEY_PATH',        value: '/app/keys/public.pem' }
            { name: 'BOOTSTRAP_ADMIN_USER',       value: bootstrapAdminUser }
            { name: 'BOOTSTRAP_ADMIN_PASSWORD',   secretRef: 'bootstrap-password' }
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
  dependsOn: [envStorage]
}

// ---------------------------------------------------------------------------
// Frontend Container App (external — HTTPS with managed certificate)
// ---------------------------------------------------------------------------
resource frontend 'Microsoft.App/containerApps@2023-05-01' = {
  name: '${appName}-frontend'
  location: location
  properties: {
    managedEnvironmentId: environment.id
    configuration: {
      secrets: [
        { name: 'acr-password',  value: acrPassword }
        { name: 'vite-app-key', value: viteAppKey }
      ]
      registries: [
        {
          server:            acrLoginServer
          username:          acrUsername
          passwordSecretRef: 'acr-password'
        }
      ]
      ingress: {
        external:   true
        targetPort: 80
        transport:  'http'
        customDomains: empty(customDomain) ? [] : [
          {
            name:          customDomain
            bindingType:   'SniEnabled'
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
            // NGINX_AUTH_SERVICE_URL: internal FQDN of auth-service
            // Used by envsubst to configure nginx.conf at container start
            { name: 'NGINX_AUTH_SERVICE_URL', value: 'http://${appName}-auth' }
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
}

// ---------------------------------------------------------------------------
// Outputs
// ---------------------------------------------------------------------------
output frontendUrl         string = 'https://${frontend.properties.configuration.ingress.fqdn}'
output authServiceFqdn     string = authService.properties.configuration.ingress.fqdn
output postgresHost        string = postgres.properties.fullyQualifiedDomainName
output redisHost           string = redis.properties.hostName
output storageAccountName  string = storageAccount.name
output keysFileShareName   string = keysFileShare.name
output keyVaultName        string = keyVault.name
