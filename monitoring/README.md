# Monitoring Stack for Upload Service

This directory contains configurations for setting up a monitoring stack using:
- Loki (for log aggregation)
- Promtail (for log collection)
- Prometheus (for metrics)
- Grafana (for visualization)

## Setup Instructions

1. Create the monitoring namespace:
```bash
kubectl create namespace monitoring
```

2. Apply the configuration files:
```bash
kubectl apply -f loki-config.yaml
kubectl apply -f promtail-config.yaml
kubectl apply -f prometheus-config.yaml
kubectl apply -f deploy.yaml
```

3. Access Grafana:
```bash
kubectl port-forward svc/grafana -n monitoring 3000:3000
```

Then visit: http://localhost:3000

## Configuring Grafana

1. Add Loki as a data source:
   - URL: http://loki:3100
   - Access: Server (default)

2. Add Prometheus as a data source:
   - URL: http://prometheus:9090
   - Access: Server (default)

3. Import dashboards:
   - For logs: Import dashboard ID 12019 for Loki logs
   - For application metrics: Create a custom dashboard

## Log Query Examples

Query logs from the upload service:
```
{app="upload-service"}
```

Query error logs:
```
{app="upload-service", level="error"}
```

Query logs by correlation ID:
```
{app="upload-service"} |= "correlation_id=abc123"
```

Filter by component:
```
{app="upload-service", component="upload_handler"}
```
