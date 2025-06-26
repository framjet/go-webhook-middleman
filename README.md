# FramJet Webhook Middleman

🚀 A powerful, flexible webhook router that forwards incoming webhook requests to multiple destinations based on configurable rules and templates.

## 📋 Table of Contents

- [Features](#-features)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Examples](#-examples)
- [API Reference](#-api-reference)
- [Monitoring](#-monitoring)
- [Docker](#-docker)
- [Contributing](#-contributing)

## ✨ Features

- **🎯 Dynamic Routing**: Route webhooks based on URL parameters with regex support
- **📤 Multiple Destinations**: Forward to unlimited destinations simultaneously
- **🔧 Template Engine**: Use Go templates with Sprig functions for dynamic content
- **📊 Prometheus Metrics**: Built-in monitoring and observability
- **🚀 High Performance**: Concurrent forwarding with configurable timeouts
- **🔄 Flexible Configuration**: YAML-based configuration with hot-reload support
- **🛡️ Production Ready**: Structured logging, graceful shutdown, health checks
- **📝 Custom Responses**: Template-based response customization

## 🚀 Quick Start

1. **Download the binary** or use Docker
2. **Create a configuration file** (`config.yaml`)
3. **Run the server**

```bash
# Download and run
wget https://github.com/framjet/go-webhook-middleman/releases/latest/download/framjet-webhook-middleman
chmod +x framjet-webhook-middleman
./framjet-webhook-middleman --config config.yaml
```

## 📦 Installation

### Binary Releases
```bash
# Linux (amd64)
wget https://github.com/framjet/go-webhook-middleman/releases/latest/download/framjet-webhook-middleman-linux-amd64
chmod +x framjet-webhook-middleman-linux-amd64
sudo mv framjet-webhook-middleman-linux-amd64 /usr/local/bin/framjet-webhook-middleman

# macOS (arm64)
wget https://github.com/framjet/go-webhook-middleman/releases/latest/download/framjet-webhook-middleman-darwin-arm64
chmod +x framjet-webhook-middleman-darwin-arm64
sudo mv framjet-webhook-middleman-darwin-arm64 /usr/local/bin/framjet-webhook-middleman
```

### From Source
```bash
git clone https://github.com/framjet/go-webhook-middleman.git
cd go-webhook-middleman
make framjet-webhook-middleman
```

### Docker
```bash
docker pull framjet/webhook-middleman:latest
```

## ⚙️ Configuration

### Basic Structure

```yaml
# Global variables available in templates
variables:
  environment: "production"
  discord_token: "123456789/abcdefghijklmnop"
  slack_token: "T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"

# Named destinations that can be reused
destinations:
  discord_general: 
    url: "https://discord.com/api/webhooks/{{.var.discord_token}}"
  slack_alerts:
    url: "https://hooks.slack.com/services/{{.var.slack_token}}"
    method: "POST"
    body: |
      {
        "text": "🚀 {{.params.service}} deployed to {{.var.environment}}!",
        "username": "DeployBot"
      }

# Route definitions
routes:
  - methods: ["POST"]
    paths: 
      - "/{service}/{event}"
    matchers:
      - service: "frontend"
        event: "deployment"
        to: ["discord_general", "slack_alerts"]
```

### Configuration Reference

#### Variables
Global variables accessible in all templates via `{{.var.variable_name}}`.

#### Destinations
Named destinations with optional method and body templates.

```yaml
destinations:
  my_webhook:
    url: "https://api.example.com/webhook"
    method: "POST"  # Default: POST
    body: |         # Optional custom body template
      {
        "message": "{{.params.service}} event occurred",
        "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
      }
```

#### Routes
Route definitions with path patterns and matching rules.

```yaml
routes:
  - method: "POST"              # Single method
    methods: ["POST", "PUT"]    # Or multiple methods
    path: "/{param}"            # Single path
    paths: ["/{param}", "/alt/{param}"]  # Or multiple paths
    matchers:                   # Matching rules
      - param: "value"
        to: ["destination"]
    response:                   # Optional custom response
      status:
        success: 200
        failure: 502
      headers:
        "X-Custom": "{{.params.service}}"
      body: |
        {"status": "processed", "service": "{{.params.service}}"}
```

## 📚 Examples

### 1. Simple Service Notifications

```yaml
variables:
  discord_webhook: "https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"

destinations:
  discord:
    url: "{{.var.discord_webhook}}"
    body: |
      {
        "content": "🚀 **{{.params.service}}** deployed!",
        "embeds": [{
          "title": "Deployment Notification",
          "description": "Service: {{.params.service}}\nEvent: {{.params.event}}",
          "color": 3066993,
          "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
        }]
      }

routes:
  - path: "/{service}/{event}"
    matchers:
      - service: ["frontend", "backend", "api"]
        event: "deployment"
        to: ["discord"]
```

**Usage:**
```bash
curl -X POST http://localhost:8080/frontend/deployment \
  -H "Content-Type: application/json" \
  -d '{"version": "1.2.3", "author": "john.doe"}'
```

### 2. Multi-Environment Routing

```yaml
variables:
  prod_slack: "https://hooks.slack.com/services/PROD_TOKEN"
  dev_slack: "https://hooks.slack.com/services/DEV_TOKEN"
  environment: "production"

destinations:
  prod_notifications:
    url: "{{.var.prod_slack}}"
    body: |
      {
        "text": "🔴 PRODUCTION: {{.params.service}} {{.params.action}}",
        "username": "ProdBot",
        "icon_emoji": ":rotating_light:"
      }
  
  dev_notifications:
    url: "{{.var.dev_slack}}"
    body: |
      {
        "text": "🟡 DEV: {{.params.service}} {{.params.action}}",
        "username": "DevBot"
      }

routes:
  - path: "/{env}/{service}/{action}"
    matchers:
      - env: ["prod", "production"]
        to: ["prod_notifications"]
      - env: ["dev", "development", "staging"]
        to: ["dev_notifications"]
      - service: "/critical-.+/"
        to: ["prod_notifications", "dev_notifications"]  # Critical services go to both
```

**Usage:**
```bash
# Production deployment
curl -X POST http://localhost:8080/prod/frontend/deploy

# Development deployment  
curl -X POST http://localhost:8080/dev/api/deploy

# Critical service (goes to both channels)
curl -X POST http://localhost:8080/dev/critical-auth/deploy
```

### 3. ArgoCD Multi-Discord Integration

**Problem:** ArgoCD Notifications only supports one webhook URL per service, but you want to notify multiple Discord channels (e.g., dev team, ops team, management).

**Solution:** Use FramJet Webhook Middleman as a proxy to fan out to multiple Discord webhooks.

#### Configuration

```yaml
variables:
  # Multiple Discord webhooks for different teams
  discord_dev_team: "https://discord.com/api/webhooks/123456789/dev-team-webhook-token"
  discord_ops_team: "https://discord.com/api/webhooks/987654321/ops-team-webhook-token"
  discord_management: "https://discord.com/api/webhooks/456789123/management-webhook-token"
  argocd_url: "https://argocd.company.com"

destinations:
  # Dev team gets all notifications with detailed info
  dev_notifications:
    url: "{{.var.discord_dev_team}}"
    body: |
      {
        "embeds": [
          {
            "title": "{{if eq .params.event "app-created"}}🆕 Application Created{{else if eq .params.event "app-deleted"}}🗑️ Application Deleted{{else if eq .params.event "app-deployed"}}🚀 Application Deployed{{else if eq .params.event "sync-succeeded"}}✅ Sync Succeeded{{else if eq .params.event "sync-failed"}}❌ Sync Failed{{else if eq .params.event "health-degraded"}}🔴 Health Degraded{{else}}📢 ArgoCD Event{{end}}",
            "url": "{{.var.argocd_url}}/applications/{{.params.app}}",
            "color": {{if or (eq .params.event "sync-succeeded") (eq .params.event "app-deployed")}}3066993{{else if or (eq .params.event "sync-failed") (eq .params.event "health-degraded")}}15158332{{else if eq .params.event "app-deleted"}}10181046{{else}}16776960{{end}},
            "description": "**{{.params.app}}** in **{{.params.project}}** ({{.params.instance}})",
            "fields": [
              {
                "name": "🏷️ Instance",
                "value": "{{.params.instance}}",
                "inline": true
              },
              {
                "name": "📁 Project", 
                "value": "{{.params.project}}",
                "inline": true
              },
              {
                "name": "📱 Application",
                "value": "{{.params.app}}",
                "inline": true
              },
              {
                "name": "🔄 Event",
                "value": "{{.params.event | title | replace "-" " "}}",
                "inline": false
              },
              {
                "name": "📋 Details",
                "value": "```json\n{{.body | substr 0 800}}{{if gt (len .body) 800}}...{{end}}\n```",
                "inline": false
              }
            ],
            "footer": {
              "text": "ArgoCD • {{.params.instance}}",
              "icon_url": "https://argo-cd.readthedocs.io/en/stable/assets/logo.png"
            },
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

  # Ops team gets critical events only with action items
  ops_notifications:
    url: "{{.var.discord_ops_team}}"
    body: |
      {
        "content": {{if or (eq .params.event "sync-failed") (eq .params.event "health-degraded")}}"🚨 <@&ROLE_ID_OPS_ONCALL> Action Required!"{{else}}null{{end}},
        "embeds": [
          {
            "title": "{{if eq .params.event "sync-failed"}}🚨 Sync Failed{{else if eq .params.event "health-degraded"}}⚠️ Health Degraded{{else if eq .params.event "app-deployed"}}✅ Deployment Success{{else}}📢 {{.params.event | title | replace "-" " "}}{{end}}",
            "url": "{{.var.argocd_url}}/applications/{{.params.app}}",
            "color": {{if eq .params.event "sync-failed"}}15158332{{else if eq .params.event "health-degraded"}}16742144{{else if eq .params.event "app-deployed"}}3066993{{else}}5793266{{end}},
            "fields": [
              {
                "name": "Application",
                "value": "`{{.params.app}}`",
                "inline": true
              },
              {
                "name": "Project",
                "value": "`{{.params.project}}`", 
                "inline": true
              },
              {
                "name": "Instance",
                "value": "`{{.params.instance}}`",
                "inline": true
              }{{if or (eq .params.event "sync-failed") (eq .params.event "health-degraded")}},
              {
                "name": "🔧 Action Items",
                "value": "• Check application logs\n• Review sync status\n• Verify resource health\n• [Open ArgoCD]({{.var.argocd_url}}/applications/{{.params.app}})",
                "inline": false
              }{{end}}
            ],
            "footer": {
              "text": "Ops Team Alert"
            },
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

  # Management gets summary notifications for production only
  management_notifications:
    url: "{{.var.discord_management}}"
    body: |
      {
        "embeds": [
          {
            "title": "📊 Production Deployment Update",
            "color": {{if eq .params.event "app-deployed"}}3066993{{else if eq .params.event "sync-failed"}}15158332{{else}}5793266{{end}},
            "description": "**{{.params.app}}** has been {{if eq .params.event "app-deployed"}}successfully deployed{{else if eq .params.event "sync-failed"}}failed to deploy{{else}}updated{{end}}",
            "fields": [
              {
                "name": "Application",
                "value": "{{.params.app}}",
                "inline": true
              },
              {
                "name": "Status", 
                "value": "{{if eq .params.event "app-deployed"}}✅ Success{{else if eq .params.event "sync-failed"}}❌ Failed{{else}}🔄 Updated{{end}}",
                "inline": true
              },
              {
                "name": "Time",
                "value": "{{now | date "15:04 MST"}}",
                "inline": true
              }
            ],
            "footer": {
              "text": "Production Deployments"
            },
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

routes:
  # ArgoCD webhook route matching the path format from ArgoCD templates
  - path: "/{instance}/{project}/{app}/{event}"
    matchers:
      # Dev team gets all events for all projects
      - to: ["dev_notifications"]
      
      # Ops team gets critical events only
      - event: ["sync-failed", "health-degraded", "app-deployed"]
        to: ["ops_notifications"]
      
      # Management gets production deployment notifications only
      - project: ["production", "prod"]
        event: ["app-deployed", "sync-failed"]
        to: ["management_notifications"]
      
      # Critical applications notify all teams
      - app: "/critical-.+/"
        to: ["dev_notifications", "ops_notifications", "management_notifications"]
```

#### ArgoCD Configuration

In your ArgoCD ConfigMap, configure the webhook service to point to the middleman:

```yaml
# argocd-notifications-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  # Point to your webhook middleman instead of Discord directly
  service.webhook.discord: |
    url: http://webhook-middleman.webhook-middleman.svc.cluster.local:8080
    headers:
    - name: Content-Type
      value: application/json

  # Subscriptions - same as before
  subscriptions: |
    - recipients:
      - discord
      triggers:
        - on-created
        - on-deleted
        - on-deployed
        - on-health-degraded
        - on-sync-failed
        - on-sync-running
        - on-sync-status-unknown
        - on-sync-succeeded

  # Templates for different events
  template.app-created: |
    webhook:
      discord:
        path: /{{.context.instanceName}}/{{.app.spec.project}}/{{.app.metadata.name}}/app-created
        method: POST
        body: |
          {
            "timestamp": "{{.app.metadata.creationTimestamp}}",
            "application": "{{.app.metadata.name}}",
            "project": "{{.app.spec.project}}",
            "namespace": "{{.app.spec.destination.namespace}}",
            "repository": {{- if .app.spec.source }}"{{.app.spec.source.repoURL}}"{{- else if .app.spec.sources }}"{{range $i, $s := .app.spec.sources}}{{if $i}}, {{end}}{{$s.repoURL}}{{end}}"{{- end }},
            "cluster": "{{.app.spec.destination.server}}"
          }

  template.app-deployed: |
    webhook:
      discord:
        path: /{{.context.instanceName}}/{{.app.spec.project}}/{{.app.metadata.name}}/app-deployed
        method: POST
        body: |
          {
            "timestamp": "{{.app.status.operationState.finishedAt}}",
            "application": "{{.app.metadata.name}}",
            "project": "{{.app.spec.project}}",
            "revision": "{{.app.status.sync.revision}}",
            "phase": "{{.app.status.operationState.phase}}",
            "message": "{{.app.status.operationState.message}}"
          }

  template.app-sync-failed: |
    webhook:
      discord:
        path: /{{.context.instanceName}}/{{.app.spec.project}}/{{.app.metadata.name}}/sync-failed
        method: POST
        body: |
          {
            "timestamp": "{{.app.status.operationState.finishedAt}}",
            "application": "{{.app.metadata.name}}",
            "project": "{{.app.spec.project}}",
            "revision": "{{.app.status.sync.revision}}",
            "error": "{{.app.status.operationState.message}}",
            "phase": "{{.app.status.operationState.phase}}"
          }

  template.app-health-degraded: |
    webhook:
      discord:
        path: /{{.context.instanceName}}/{{.app.spec.project}}/{{.app.metadata.name}}/health-degraded
        method: POST
        body: |
          {
            "timestamp": "{{now}}",
            "application": "{{.app.metadata.name}}",
            "project": "{{.app.spec.project}}",
            "health_status": "{{.app.status.health.status}}",
            "sync_status": "{{.app.status.sync.status}}",
            "message": "{{.app.status.health.message}}"
          }

  template.app-sync-succeeded: |
    webhook:
      discord:
        path: /{{.context.instanceName}}/{{.app.spec.project}}/{{.app.metadata.name}}/sync-succeeded
        method: POST
        body: |
          {
            "timestamp": "{{.app.status.operationState.finishedAt}}",
            "application": "{{.app.metadata.name}}",
            "project": "{{.app.spec.project}}",
            "revision": "{{.app.status.sync.revision}}",
            "phase": "{{.app.status.operationState.phase}}"
          }
```

#### Kubernetes Deployment

```yaml
# webhook-middleman deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webhook-middleman
  namespace: webhook-middleman
spec:
  replicas: 2
  selector:
    matchLabels:
      app: webhook-middleman
  template:
    metadata:
      labels:
        app: webhook-middleman
    spec:
      containers:
      - name: webhook-middleman
        image: framjet/webhook-middleman:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_FILE
          value: "/config/config.yaml"
        - name: LOG_LEVEL
          value: "info"
        - name: JSON_LOG
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
      volumes:
      - name: config
        configMap:
          name: webhook-middleman-config
---
apiVersion: v1
kind: Service
metadata:
  name: webhook-middleman
  namespace: webhook-middleman
spec:
  selector:
    app: webhook-middleman
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

#### Benefits of This Approach

1. **Multiple Discord Channels**: Different teams get notifications in their preferred channels
2. **Filtered Content**: Each team gets relevant information (devs get details, management gets summaries)
3. **Custom Formatting**: Each Discord webhook can have different embed styles and content
4. **Easy Configuration**: Add/remove Discord webhooks without touching ArgoCD
5. **Monitoring**: Track which notifications are sent to which teams
6. **Fallback**: If one Discord webhook fails, others still receive notifications

#### Testing

```bash
# Test the webhook manually to verify routing
curl -X POST http://localhost:8080/argocd-prod/production/my-app/app-deployed \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": "2024-01-26T12:00:00Z",
    "application": "my-app",
    "project": "production",
    "revision": "abc123",
    "phase": "Succeeded"
  }'

# Should trigger notifications to:
# - dev_notifications (all events)
# - ops_notifications (deployment events)  
# - management_notifications (production deployments)
```

This setup transforms ArgoCD's single webhook limitation into a powerful multi-channel notification system!

### 4. GitHub Actions Integration

```yaml
variables:
  github_token: "ghp_YOUR_TOKEN"
  notification_channel: "https://discord.com/api/webhooks/YOUR_WEBHOOK"

destinations:
  github_status:
    url: "https://api.github.com/repos/{{.params.owner}}/{{.params.repo}}/statuses/{{.params.sha}}"
    method: "POST"
    headers:
      Authorization: "Bearer {{.var.github_token}}"
      Accept: "application/vnd.github.v3+json"
    body: |
      {
        "state": "{{.params.state}}",
        "target_url": "{{.params.build_url}}",
        "description": "Deployment {{.params.state}}",
        "context": "deployment/{{.params.environment}}"
      }

  discord_notification:
    url: "{{.var.notification_channel}}"
    body: |
      {
        "content": null,
        "embeds": [{
          "title": "🚀 Deployment {{.params.state | title}}",
          "description": "**Repository:** {{.params.owner}}/{{.params.repo}}\n**Environment:** {{.params.environment}}\n**SHA:** `{{.params.sha | substr 0 7}}`",
          "color": {{if eq .params.state "success"}}3066993{{else if eq .params.state "failure"}}15158332{{else}}16776960{{end}},
          "fields": [{
            "name": "Build URL",
            "value": "[View Build]({{.params.build_url}})"
          }],
          "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
        }]
      }

routes:
  - path: "/github/{owner}/{repo}/{environment}/{state}/{sha}"
    matchers:
      - state: ["success", "failure", "pending"]
        to: ["github_status", "discord_notification"]
```

**Usage with GitHub Actions:**
```yaml
# .github/workflows/deploy.yml
- name: Notify Deployment Start
  run: |
    curl -X POST "${{ secrets.WEBHOOK_URL }}/github/${{ github.repository_owner }}/${{ github.event.repository.name }}/production/pending/${{ github.sha }}" \
      -H "Content-Type: application/json" \
      -d '{"build_url": "${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}"}'
```

### 5. Monitoring and Alerting

```yaml
variables:
  prometheus_webhook: "http://alertmanager:9093/api/v1/alerts"
  oncall_webhook: "https://hooks.slack.com/services/ONCALL_TOKEN"

destinations:
  prometheus_alert:
    url: "{{.var.prometheus_webhook}}"
    method: "POST"
    body: |
      [{
        "labels": {
          "alertname": "ServiceDown",
          "service": "{{.params.service}}",
          "severity": "{{.params.severity}}",
          "environment": "{{.params.env}}"
        },
        "annotations": {
          "summary": "Service {{.params.service}} is down",
          "description": "{{.body}}"
        },
        "startsAt": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
      }]

  oncall_notification:
    url: "{{.var.oncall_webhook}}"
    body: |
      {
        "text": "🚨 ALERT: {{.params.service}} ({{.params.severity}})",
        "attachments": [{
          "color": {{if eq .params.severity "critical"}}"danger"{{else if eq .params.severity "warning"}}"warning"{{else}}"good"{{end}},
          "fields": [
            {"title": "Service", "value": "{{.params.service}}", "short": true},
            {"title": "Severity", "value": "{{.params.severity}}", "short": true},
            {"title": "Environment", "value": "{{.params.env}}", "short": true},
            {"title": "Details", "value": "{{.body}}", "short": false}
          ]
        }]
      }

routes:
  - path: "/alert/{service}/{severity}/{env}"
    response:
      body: |
        {
          "status": "alert_processed",
          "service": "{{.params.service}}",
          "severity": "{{.params.severity}}",
          "forwarded_to": {{.forwardedTo}},
          "successful": {{.successful}}
        }
    matchers:
      - severity: "critical"
        to: ["prometheus_alert", "oncall_notification"]
      - severity: ["warning", "info"]
        env: "production"
        to: ["prometheus_alert"]
      - severity: ["warning", "info"]
        env: ["dev", "staging"]
        to: []  # No notifications for dev warnings
```

### 6. Complex Regex Matching

```yaml
destinations:
  api_monitor:
    url: "https://api-monitor.example.com/webhook"
    body: |
      {
        "service": "{{.params.service}}",
        "version": "{{.params.version}}",
        "action": "{{.params.action}}"
      }

  database_monitor:
    url: "https://db-monitor.example.com/webhook"

routes:
  - path: "/{service}/{version}/{action}"
    matchers:
      # Match API services (api-*, service-api-*)
      - service: "/api-.+/"
        version: "/v[0-9]+\\.[0-9]+/"
        to: ["api_monitor"]
      
      # Match database services
      - service: "/(db|database|postgres|mysql)-.+/"
        action: ["backup", "restore", "migrate"]
        to: ["database_monitor"]
      
      # Match version patterns (semantic versioning)
      - version: "/[0-9]+\\.[0-9]+\\.[0-9]+/"
        action: "release"
        to: ["api_monitor", "database_monitor"]
      
      # Match multiple service patterns
      - service: ["frontend", "backend", "/mobile-.+/", "/web-.+/"]
        to: ["api_monitor"]
```

### 7. Template Functions Example

```yaml
variables:
  base_url: "https://api.example.com"
  secret_key: "your-secret-key"

destinations:
  advanced_webhook:
    url: "{{.var.base_url}}/notifications"
    headers:
      Authorization: "Bearer {{.var.secret_key}}"
      X-Timestamp: "{{now | date "2006-01-02T15:04:05Z07:00"}}"
      X-Service: "{{.params.service | upper}}"
    body: |
      {
        "id": "{{uuidv4}}",
        "service": "{{.params.service | title}}",
        "event": "{{.params.event}}",
        "timestamp": "{{now | unixEpoch}}",
        "environment": "{{.var.environment | upper}}",
        "message": "{{.body | b64enc}}",
        "hash": "{{.body | sha256sum}}",
        "random_id": "{{randAlphaNum 10}}",
        "is_production": {{eq .var.environment "production"}},
        "formatted_date": "{{now | date "January 2, 2006"}}",
        "body_length": {{len .body}},
        "params_count": {{len .params}}
      }

routes:
  - path: "/{service}/{event}"
    matchers:
      - to: ["advanced_webhook"]
```

## 🔌 API Reference

### Endpoints

#### `POST /{configured-paths}`
Webhook receiver endpoints as defined in your routes configuration.

**Response Format:**
```json
{
  "forwarded_to": 2,
  "successful": 1,
  "duration_ms": 150,
  "results": [
    {
      "destination": "discord",
      "url": "https://discord.com/api/webhooks/...",
      "method": "POST",
      "success": true,
      "status_code": 200,
      "duration_ms": 120
    }
  ]
}
```

#### `GET /health`
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "destinations": 3,
  "routes": 5,
  "timestamp": "2025-01-26T12:00:00Z"
}
```

#### `GET /metrics`
Prometheus metrics endpoint.

### Template Context

Available variables in templates:

- `{{.params.name}}` - URL path parameters
- `{{.var.name}}` - Global variables from config
- `{{.body}}` - Request body as string
- `{{.request}}` - HTTP request object
- `{{.route}}` - Matched route configuration

### Template Functions

All [Sprig functions](http://masterminds.github.io/sprig/) are available plus:

- `{{now}}` - Current time
- `{{uuidv4}}` - Generate UUID
- `{{randAlphaNum 10}}` - Random alphanumeric string
- `{{.body | b64enc}}` - Base64 encode
- `{{.body | sha256sum}}` - SHA256 hash

## 📊 Monitoring

### Prometheus Metrics

- `webhook_middleman_webhooks_received_total` - Total webhooks received
- `webhook_middleman_webhooks_processed_total` - Total webhooks processed (by status)
- `webhook_middleman_forwarding_duration_seconds` - Forwarding duration histogram
- `webhook_middleman_forwarding_total` - Total forwarding attempts (by destination/status)
- `webhook_middleman_routes_matched_total` - Routes matched (by method/path)

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Webhook Middleman",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(webhook_middleman_webhooks_received_total[5m])"
          }
        ]
      },
      {
        "title": "Success Rate",
        "targets": [
          {
            "expr": "rate(webhook_middleman_webhooks_processed_total{status=\"success\"}[5m]) / rate(webhook_middleman_webhooks_processed_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## 🐳 Docker

### Docker Compose

```yaml
version: '3.8'

services:
  webhook-middleman:
    image: framjet/webhook-middleman:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/config.yaml:ro
    environment:
      - CONFIG_FILE=/config.yaml
      - LOG_LEVEL=info
      - JSON_LOG=true
    restart: unless-stopped
    
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webhook-middleman
spec:
  replicas: 3
  selector:
    matchLabels:
      app: webhook-middleman
  template:
    metadata:
      labels:
        app: webhook-middleman
    spec:
      containers:
      - name: webhook-middleman
        image: framjet/webhook-middleman:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_FILE
          value: "/config/config.yaml"
        - name: LOG_LEVEL
          value: "info"
        - name: JSON_LOG
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: webhook-middleman-config
---
apiVersion: v1
kind: Service
metadata:
  name: webhook-middleman
spec:
  selector:
    app: webhook-middleman
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 🛠️ Command Line Options

```bash
framjet-webhook-middleman [OPTIONS]

OPTIONS:
  --host, -s              Host to bind to (default: "0.0.0.0") [$HTTP_HOST]
  --port, -p              Port to listen on (default: "8080") [$HTTP_PORT]
  --config, -c            Path to configuration file (default: "config.yaml") [$CONFIG_FILE]
  --log-level, -l         Log level: debug, info, warn, error (default: "info") [$LOG_LEVEL]
  --json-log, -j          Enable JSON formatted logging [$JSON_LOG]
  --timeout, -t           HTTP client timeout (default: 30s) [$HTTP_TIMEOUT]
  --help, -h              Show help
  --version               Show version information
```

## 🔧 Troubleshooting

### Common Issues

1. **No route matches**
   ```bash
   # Check route configuration
   curl -v http://localhost:8080/your/path
   # Verify path parameters match route patterns
   ```

2. **Template rendering errors**
   ```yaml
   # Check variable names and syntax
   variables:
     test_var: "value"
   destinations:
     test:
       url: "https://example.com/{{.var.test_var}}"  # Correct syntax
   ```

3. **Destination unreachable**
   ```bash
   # Test destination manually
   curl -X POST https://your-webhook-url
   # Check firewall/network connectivity
   ```

### Debug Mode

```bash
# Enable debug logging
./framjet-webhook-middleman --log-level debug

# Check metrics
curl http://localhost:8080/metrics | grep webhook_middleman
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Add tests: `go test ./...`
5. Submit a pull request

### Development Setup

```bash
git clone https://github.com/framjet/go-webhook-middleman.git
cd go-webhook-middleman
go mod download
make framjet-webhook-middleman
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙋‍♀️ Support

- 📖 [Documentation](https://github.com/framjet/go-webhook-middleman/wiki)
- 🐛 [Issue Tracker](https://github.com/framjet/go-webhook-middleman/issues)
- 💬 [Discussions](https://github.com/framjet/go-webhook-middleman/discussions)

---

**Made with ❤️ by [FramJet](https://github.com/framjet)**