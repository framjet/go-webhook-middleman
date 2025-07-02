# FramJet Webhook Middleman

🚀 A powerful, flexible webhook router that forwards incoming webhook requests to multiple destinations based on configurable expression-based rules and templates.

## 📋 Table of Contents

- [Features](#-features)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Expression Language](#-expression-language)
- [Examples](#-examples)
- [API Reference](#-api-reference)
- [Monitoring](#-monitoring)
- [Docker](#-docker)
- [Contributing](#-contributing)

## ✨ Features

- **🎯 Expression-Based Routing**: Route webhooks using powerful expression language with access to request data, parameters, and variables
- **📤 Multiple Destinations**: Forward to unlimited destinations simultaneously
- **🔧 Template Engine**: Use Go templates with Sprig functions for dynamic content
- **📊 Prometheus Metrics**: Built-in monitoring and observability
- **🚀 High Performance**: Concurrent forwarding with configurable timeouts
- **🔄 Flexible Configuration**: YAML-based configuration with hot-reload support
- **🛡️ Production Ready**: Structured logging, graceful shutdown, health checks
- **📝 Custom Responses**: Template-based response customization
- **🧮 Rich Expression Context**: Access request headers, body, URL parameters, and custom variables

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
# Global variables available in templates and expressions
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
      - expr: |
          params.service == "frontend" && params.event == "deployment"
        to: ["discord_general", "slack_alerts"]
```

### Configuration Reference

#### Variables
Global variables accessible in all templates via `{{.var.variable_name}}` and in expressions via `var.variable_name`.

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
Route definitions with path patterns and expression-based matching rules.

```yaml
routes:
  - method: "POST"              # Single method
    methods: ["POST", "PUT"]    # Or multiple methods
    path: "/{param}"            # Single path
    paths: ["/{param}", "/alt/{param}"]  # Or multiple paths
    matchers:                   # Expression-based matching rules
      - expr: |
          params.param == "value" && request.method == "POST"
        to: ["destination"]
      - exprs:                  # Multiple expressions (OR logic)
          - params.service == "frontend"
          - params.service == "backend"
        to: ["another_destination"]
    response:                   # Optional custom response
      status:
        success: 200
        failure: 502
      headers:
        "X-Custom": "{{.params.service}}"
      body: |
        {"status": "processed", "service": "{{.params.service}}"}
```

## 🧮 Expression Language

The webhook middleman uses [expr-lang](https://github.com/expr-lang/expr) for powerful expression-based matching. Expressions have access to a rich context including request data, URL parameters, and configuration variables.

### Expression Context

Available variables in expressions:

#### `params` - URL Path Parameters
```yaml
# Route: /{service}/{version}/{action}
# URL: /frontend/v1.2/deploy
params.service   # "frontend"
params.version   # "v1.2" 
params.action    # "deploy"
```

#### `var` - Global Variables
```yaml
# From config variables section
var.environment  # "production"
var.discord_token # "123456789/abcdefghijklmnop"
```

#### `request` - HTTP Request Data
```yaml
request.method          # "POST"
request.host           # "webhook.example.com"
request.body           # Request body as string
request.contentType    # "application/json"
request.userAgent      # User agent string
request.remoteAddr     # Client IP address
request.headers        # Map of headers (map[string][]string)
request.url.full       # Full URL
request.url.scheme     # "https"
request.url.host       # "webhook.example.com"
request.url.path       # "/api/webhook"
request.url.query      # Query parameters (map[string][]string)
request.url.fragment   # URL fragment
request.url.rawQuery   # Raw query string
```

#### `config` - Configuration Access
```yaml
config.variables       # Global variables
config.destinations    # Named destinations
config.routes         # Route definitions
```

#### `route` - Current Route
```yaml
route.method          # Matched route method
route.path           # Matched route path
route.matchers       # Route matchers
```

#### `matcher` - Current Matcher
```yaml
matcher.expr         # Current expression
matcher.to          # Destination list
```

### Expression Examples

#### Simple Parameter Matching
```yaml
matchers:
  - expr: params.service == "frontend"
    to: ["discord"]
```

#### Complex Conditions
```yaml
matchers:
  - expr: |
      params.service == "frontend" && 
      params.environment == "production" && 
      request.method == "POST"
    to: ["critical_alerts"]
```

#### JSON Body Parsing
```yaml
matchers:
  - expr: |
      request.contentType == "application/json" && 
      fromJSON(request.body).status == "success"
    to: ["success_webhook"]
```

#### Header-Based Routing
```yaml
matchers:
  - expr: |
      "X-GitHub-Event" in request.headers && 
      request.headers["X-GitHub-Event"][0] == "push"
    to: ["github_webhook"]
```

#### Regular Expression Matching
```yaml
matchers:
  - expr: |
      params.service matches "^(frontend|backend|api)$" && 
      params.version matches "^v[0-9]+\\.[0-9]+$"
    to: ["version_webhook"]
```

#### Multiple Expressions (OR Logic)
```yaml
matchers:
  - exprs:
      - params.service == "frontend"
      - params.service == "backend"  
      - params.service == "api"
    to: ["service_webhook"]
```

#### Environment and Variable Checks
```yaml
matchers:
  - expr: |
      var.environment == "production" && 
      params.severity == "critical"
    to: ["oncall_alerts"]
```

#### Advanced JSON Processing
```yaml
matchers:
  - expr: |
      request.contentType == "application/json" && 
      has(fromJSON(request.body), "deployment") && 
      fromJSON(request.body).deployment.status == "success"
    to: ["deployment_success"]
```

### Expression Functions

Common functions available in expressions:

- **String functions**: `contains()`, `startsWith()`, `endsWith()`, `matches()`, `len()`
- **JSON functions**: `fromJSON()`, `toJSON()`, `has()`
- **Type functions**: `type()`, `string()`, `int()`, `float()`
- **Collection functions**: `in`, `all()`, `any()`, `filter()`, `map()`
- **Logic functions**: `&&`, `||`, `!`, `==`, `!=`, `<`, `>`, `<=`, `>=`

## 📚 Examples

### 1. Service-Based Routing with Expressions

```yaml
variables:
  discord_webhook: "https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
  environment: "production"

destinations:
  discord:
    url: "{{.var.discord_webhook}}"
    body: |
      {
        "content": "🚀 **{{.params.service}}** deployed to **{{.var.environment}}**!",
        "embeds": [{
          "title": "Deployment Notification",
          "description": "Service: {{.params.service}}\nEvent: {{.params.event}}\nEnvironment: {{.var.environment}}",
          "color": 3066993,
          "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
        }]
      }

routes:
  - path: "/{service}/{event}"
    matchers:
      # Frontend deployments
      - expr: |
          params.service == "frontend" && 
          params.event == "deployment" && 
          var.environment == "production"
        to: ["discord"]
      
      # Backend services with specific events
      - expr: |
          params.service in ["backend", "api", "database"] && 
          params.event in ["deployment", "rollback", "migration"]
        to: ["discord"]
      
      # Critical services (any event)
      - expr: |
          params.service matches "^critical-.*"
        to: ["discord"]
```

### 2. GitHub Webhook Integration

```yaml
variables:
  github_secret: "your-webhook-secret"
  slack_webhook: "https://hooks.slack.com/services/YOUR_SLACK_WEBHOOK"

destinations:
  slack_notifications:
    url: "{{.var.slack_webhook}}"
    body: |
      {
        "text": "GitHub Event: {{.params.event}}",
        "attachments": [{
          "color": {{if eq .params.event "push"}}"good"{{else if eq .params.event "pull_request"}}"warning"{{else}}"#439FE0"{{end}},
          "fields": [
            {"title": "Repository", "value": "{{.params.owner}}/{{.params.repo}}", "short": true},
            {"title": "Event", "value": "{{.params.event}}", "short": true},
            {"title": "Branch", "value": "{{.params.branch}}", "short": true}
          ]
        }]
      }
  
routes:
  - path: "/github/{owner}/{repo}/{event}"
    matchers:
      # GitHub push events
      - expr: |
          params.event == "push" && 
          "X-GitHub-Event" in request.headers && 
          request.headers["X-GitHub-Event"][0] == "push" &&
          request.contentType == "application/json"
        to: ["slack_notifications"]
      
      # Pull request events
      - expr: |
          params.event == "pull_request" && 
          "X-GitHub-Event" in request.headers && 
          request.headers["X-GitHub-Event"][0] == "pull_request"
        to: ["slack_notifications"]
      
      # Repository-specific routing
      - expr: |
          params.owner == "myorg" && 
          params.repo in ["critical-app", "main-service"] && 
          params.event in ["push", "release"]
        to: ["slack_notifications"]
```

### 3. JSON Body-Based Routing

```yaml
variables:
  webhook_url: "https://api.example.com/notifications"

destinations:
  api_webhook:
    url: "{{.var.webhook_url}}"
    body: |
      {
        "source": "webhook-middleman",
        "original_event": {{.body}},
        "processed_at": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
      }

routes:
  - path: "/webhook/{source}"
    matchers:
      # Route based on JSON body content
      - expr: |
          request.contentType == "application/json" && 
          has(fromJSON(request.body), "event_type") && 
          fromJSON(request.body).event_type == "deployment"
        to: ["api_webhook"]

      # Route based on nested JSON properties
      - expr: |
          request.contentType == "application/json" && 
          has(fromJSON(request.body), "deployment") && 
          has(fromJSON(request.body).deployment, "status") && 
          fromJSON(request.body).deployment.status == "success"
        to: ["api_webhook"]

      # Route based on array content
      - expr: |
          request.contentType == "application/json" && 
          has(fromJSON(request.body), "services") && 
          len(fromJSON(request.body).services) > 0 && 
          "frontend" in fromJSON(request.body).services
        to: ["api_webhook"]
```

### 4. Multi-Environment ArgoCD Integration

```yaml
variables:
  discord_dev: "https://discord.com/api/webhooks/DEV_WEBHOOK_URL"
  discord_ops: "https://discord.com/api/webhooks/OPS_WEBHOOK_URL"
  discord_mgmt: "https://discord.com/api/webhooks/MGMT_WEBHOOK_URL"
  argocd_url: "https://argocd.company.com"

destinations:
  dev_notifications:
    url: "{{.var.discord_dev}}"
    body: |
      {
        "embeds": [{
          "title": "{{if eq .params.event "app-deployed"}}🚀 App Deployed{{else if eq .params.event "sync-failed"}}❌ Sync Failed{{else if eq .params.event "health-degraded"}}🔴 Health Degraded{{else}}📢 ArgoCD Event{{end}}",
            "url": "{{.var.argocd_url}}/applications/{{.params.app}}",
          "color": {{if eq .params.event "app-deployed"}}3066993{{else if eq .params.event "sync-failed"}}15158332{{else if eq .params.event "health-degraded"}}16742144{{else}}5793266{{end}},
            "description": "**{{.params.app}}** in **{{.params.project}}** ({{.params.instance}})",
            "fields": [
            {"name": "Instance", "value": "{{.params.instance}}", "inline": true},
            {"name": "Project", "value": "{{.params.project}}", "inline": true},
            {"name": "Application", "value": "{{.params.app}}", "inline": true},
            {"name": "Event", "value": "{{.params.event}}", "inline": false}
            ],
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

  ops_notifications:
    url: "{{.var.discord_ops}}"
    body: |
      {
        "content": {{if or (eq .params.event "sync-failed") (eq .params.event "health-degraded")}}"🚨 <@&ONCALL_ROLE> Action Required!"{{else}}null{{end}},
        "embeds": [{
          "title": "{{if eq .params.event "sync-failed"}}🚨 Sync Failed{{else if eq .params.event "health-degraded"}}⚠️ Health Degraded{{else}}✅ {{.params.event}}{{end}}",
            "url": "{{.var.argocd_url}}/applications/{{.params.app}}",
          "color": {{if eq .params.event "sync-failed"}}15158332{{else if eq .params.event "health-degraded"}}16742144{{else}}3066993{{end}},
            "fields": [
            {"name": "Application", "value": "`{{.params.app}}`", "inline": true},
            {"name": "Project", "value": "`{{.params.project}}`", "inline": true},
            {"name": "Instance", "value": "`{{.params.instance}}`", "inline": true}
            ],
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

  mgmt_notifications:
    url: "{{.var.discord_mgmt}}"
    body: |
      {
        "embeds": [{
          "title": "📊 Production Update",
            "color": {{if eq .params.event "app-deployed"}}3066993{{else if eq .params.event "sync-failed"}}15158332{{else}}5793266{{end}},
          "description": "**{{.params.app}}** has been {{if eq .params.event "app-deployed"}}deployed{{else if eq .params.event "sync-failed"}}failed{{else}}updated{{end}}",
            "fields": [
            {"name": "Application", "value": "{{.params.app}}", "inline": true},
            {"name": "Status", "value": "{{if eq .params.event "app-deployed"}}✅ Success{{else if eq .params.event "sync-failed"}}❌ Failed{{else}}🔄 Updated{{end}}", "inline": true}
            ],
            "timestamp": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
          }
        ]
      }

routes:
  - path: "/{instance}/{project}/{app}/{event}"
    matchers:
      # Dev team gets all notifications
      - expr: "true"  # Always matches
        to: ["dev_notifications"]
      
      # Ops team gets critical events only
      - expr: |
          params.event in ["sync-failed", "health-degraded", "app-deployed"]
        to: ["ops_notifications"]
      
      # Management gets production events only
      - expr: |
          params.project in ["production", "prod"] && 
          params.event in ["app-deployed", "sync-failed"]
        to: ["mgmt_notifications"]

      # Critical applications notify everyone
      - expr: |
          params.app matches "^critical-.*"
        to: ["dev_notifications", "ops_notifications", "mgmt_notifications"]

      # High-priority projects get extra attention
      - expr: |
          params.project in ["platform", "infrastructure", "security"] && 
          params.event in ["sync-failed", "health-degraded"]
        to: ["ops_notifications", "mgmt_notifications"]
```

### 5. Advanced Header and Authentication Routing

```yaml
variables:
  internal_webhook: "https://internal-api.company.com/webhook"
  external_webhook: "https://external-api.company.com/webhook"
  auth_token: "Bearer your-auth-token"

destinations:
  internal_api:
    url: "{{.var.internal_webhook}}"
    headers:
      Authorization: "{{.var.auth_token}}"
    body: |
      {
        "source": "internal",
        "event": "{{.params.event}}",
        "data": {{.body}},
        "timestamp": "{{now | unixEpoch}}"
      }

  external_api:
    url: "{{.var.external_webhook}}"
    body: |
      {
        "source": "external",
        "event": "{{.params.event}}",
        "timestamp": "{{now | unixEpoch}}"
      }

routes:
  - path: "/{source}/{event}"
    matchers:
      # Internal requests with proper authentication
      - expr: |
          params.source == "internal" && 
          "Authorization" in request.headers && 
          startsWith(request.headers["Authorization"][0], "Bearer ") &&
          request.remoteAddr matches "^10\\." # Internal IP range
        to: ["internal_api"]
      
      # External requests with API key
      - expr: |
          params.source == "external" && 
          "X-API-Key" in request.headers && 
          len(request.headers["X-API-Key"][0]) > 0
        to: ["external_api"]

      # GitHub webhooks with signature validation
      - expr: |
          params.source == "github" && 
          "X-GitHub-Event" in request.headers && 
          "X-Hub-Signature-256" in request.headers
        to: ["internal_api"]
      
      # Content-type based routing
      - expr: |
          request.contentType == "application/json" && 
          params.event in ["webhook", "notification"]
        to: ["internal_api"]
```

### 6. Monitoring and Alerting with Expressions

```yaml
variables:
  prometheus_url: "http://alertmanager:9093/api/v1/alerts"
  slack_critical: "https://hooks.slack.com/services/CRITICAL_WEBHOOK"
  slack_warning: "https://hooks.slack.com/services/WARNING_WEBHOOK"

destinations:
  prometheus_alert:
    url: "{{.var.prometheus_url}}"
    method: "POST"
    body: |
      [{
        "labels": {
          "alertname": "{{.params.alert}}",
          "service": "{{.params.service}}",
          "severity": "{{.params.severity}}",
          "environment": "{{.params.env}}"
        },
        "annotations": {
          "summary": "{{.params.alert}} alert for {{.params.service}}",
          "description": "{{.body}}"
        },
        "startsAt": "{{now | date "2006-01-02T15:04:05Z07:00"}}"
      }]

  critical_alerts:
    url: "{{.var.slack_critical}}"
    body: |
      {
        "text": "🚨 CRITICAL ALERT",
        "attachments": [{
          "color": "danger",
          "fields": [
            {"title": "Service", "value": "{{.params.service}}", "short": true},
            {"title": "Alert", "value": "{{.params.alert}}", "short": true},
            {"title": "Environment", "value": "{{.params.env}}", "short": true},
            {"title": "Severity", "value": "{{.params.severity}}", "short": true}
          ]
        }]
      }

  warning_alerts:
    url: "{{.var.slack_warning}}"
    body: |
      {
        "text": "⚠️ Warning Alert",
        "attachments": [{
          "color": "warning",
          "fields": [
            {"title": "Service", "value": "{{.params.service}}", "short": true},
            {"title": "Alert", "value": "{{.params.alert}}", "short": true}
          ]
        }]
      }

routes:
  - path: "/alert/{service}/{alert}/{severity}/{env}"
    matchers:
      # Critical alerts in production
      - expr: |
          params.severity == "critical" && 
          params.env == "production"
        to: ["prometheus_alert", "critical_alerts"]
      
      # Warning alerts in production
      - expr: |
          params.severity == "warning" && 
          params.env == "production"
        to: ["prometheus_alert", "warning_alerts"]
      
      # Any alerts for critical services
      - expr: |
          params.service in ["auth", "payment", "database"] && 
          params.severity in ["critical", "warning"]
        to: ["prometheus_alert", "critical_alerts"]
      
      # JSON body analysis for complex alerts
      - expr: |
          request.contentType == "application/json" && 
          has(fromJSON(request.body), "metrics") && 
          has(fromJSON(request.body).metrics, "cpu_usage") && 
          fromJSON(request.body).metrics.cpu_usage > 90
        to: ["critical_alerts"]
      
      # Time-based routing (business hours)
      - expr: |
          params.severity == "warning" && 
          now().Hour() >= 9 && now().Hour() <= 17
        to: ["warning_alerts"]
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

1. **Expression evaluation errors**
   ```bash
   # Check expression syntax
   curl -v http://localhost:8080/your/path
   # Verify expression syntax and available context variables
   ```

2. **No route matches**
   ```yaml
   # Debug with simple expression
   matchers:
     - expr: "true"  # Always matches for testing
       to: ["test_destination"]
   ```

3. **Template rendering errors**
   ```yaml
   # Check variable names and syntax
   variables:
     test_var: "value"
   destinations:
     test:
       url: "https://example.com/{{.var.test_var}}"  # Correct syntax
   ```

4. **JSON parsing in expressions**
   ```yaml
   # Ensure content-type check before parsing
   matchers:
     - expr: |
         request.contentType == "application/json" && 
         has(fromJSON(request.body), "field_name")
       to: ["destination"]
   ```

### Debug Mode

```bash
# Enable debug logging
./framjet-webhook-middleman --log-level debug

# Check metrics
curl http://localhost:8080/metrics | grep webhook_middleman

# Test expressions with simple always-true matcher
```

### Expression Testing

Test your expressions with a simple configuration:

```yaml
variables:
  test_var: "test_value"

destinations:
  test_dest:
    url: "https://httpbin.org/post"

routes:
  - path: "/{param1}/{param2}"
    matchers:
      # Start with simple expression
      - expr: "params.param1 == 'test'"
        to: ["test_dest"]
      
      # Add complexity gradually
      - expr: |
          params.param1 == "test" && 
          params.param2 in ["value1", "value2"]
        to: ["test_dest"]
      
      # Test JSON parsing
      - expr: |
          request.contentType == "application/json" && 
          fromJSON(request.body).test == "value"
        to: ["test_dest"]
```

### Migration from Old Configuration

If you're migrating from the old parameter-based matching, here are the equivalent expressions:

#### Old Configuration (Parameter-based)
```yaml
matchers:
  - service: "frontend"
    event: "deployment"
    to: ["destination"]
  
  - service: ["frontend", "backend"]
    to: ["destination"]
  
  - service: "/api-.+/"
    to: ["destination"]
```

#### New Configuration (Expression-based)
```yaml
matchers:
  - expr: |
      params.service == "frontend" && 
      params.event == "deployment"
    to: ["destination"]
  
  - expr: |
      params.service in ["frontend", "backend"]
    to: ["destination"]
  
  - expr: |
      params.service matches "^api-.*"
    to: ["destination"]
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

### Expression Testing Framework

For testing expressions during development:

```go
// Test expression evaluation
env := &config.MatcherEnv{
    Params: map[string]string{
        "service": "frontend",
        "event": "deployment",
    },
    Var: map[string]string{
        "environment": "production",
    },
    Request: config.RequestData{
        Method: "POST",
        ContentType: "application/json",
        Body: `{"status": "success"}`,
    },
}

expr := `params.service == "frontend" && params.event == "deployment"`
program, err := expr.Compile(expr, expr.Env(config.MatcherEnv{}))
if err != nil {
    log.Fatal(err)
}

result, err := expr.Run(program, env)
// result should be true
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙋‍♀️ Support

- 📖 [Documentation](https://github.com/framjet/go-webhook-middleman/wiki)
- 🐛 [Issue Tracker](https://github.com/framjet/go-webhook-middleman/issues)
- 💬 [Discussions](https://github.com/framjet/go-webhook-middleman/discussions)
- 📝 [Expression Language Guide](https://expr-lang.org/docs/language-definition)

## 🔄 Changelog

### 2025.7.2 - Expression Language Integration
- **BREAKING**: Migrated from simple parameter matching to powerful expression-based routing
- Added support for `expr-lang` with access to request data, headers, and JSON body parsing
- Enhanced context available in expressions: `params`, `var`, `request`, `config`, `route`, `matcher`
- Improved flexibility for complex routing scenarios
- Backward compatibility maintained through expression equivalents

### Migration Guide
See the [Migration Guide](MIGRATION.md) for detailed instructions on upgrading from v1.x parameter-based configurations to v2.x expression-based configurations.

---

**Made with ❤️ by [FramJet](https://github.com/framjet)**