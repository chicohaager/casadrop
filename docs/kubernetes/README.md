# Kubernetes Deployment

Deploy CasaDrop to Kubernetes with these manifests.

## Quick Start

```bash
kubectl apply -f deployment.yaml
```

## Components

- **Deployment**: CasaDrop container with health checks
- **Service**: ClusterIP service on port 8080
- **PersistentVolumeClaim**: Storage for uploads and database
- **Ingress**: Optional ingress configuration

## Files

- `deployment.yaml` - All-in-one manifest
- Customize namespace, storage class, and ingress as needed

## Configuration

### Environment Variables

Edit the ConfigMap in `deployment.yaml`:

```yaml
data:
  TZ: "Europe/Berlin"
  SHARE_ALLOWED_PATHS: "/data"
```

### Secrets

Create a secret for the admin password:

```bash
kubectl create secret generic casadrop-secrets \
  --from-literal=admin-password='your-secure-password'
```

### Storage

Adjust the PVC storage class for your cluster:

```yaml
storageClassName: longhorn  # or: local-path, nfs-client, etc.
```

## Ingress Examples

### Traefik

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: casadrop
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  rules:
    - host: share.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: casadrop
                port:
                  number: 8080
  tls:
    - hosts:
        - share.example.com
      secretName: casadrop-tls
```

### Nginx Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: casadrop
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "10g"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
spec:
  ingressClassName: nginx
  rules:
    - host: share.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: casadrop
                port:
                  number: 8080
  tls:
    - hosts:
        - share.example.com
      secretName: casadrop-tls
```

## Scaling

CasaDrop uses SQLite and local file storage, so it's designed for single-replica deployment. For high availability:

1. Use a shared filesystem (NFS, Longhorn, etc.) for the PVC
2. Consider running multiple instances with a load balancer (stateless requests work, but uploads go to one instance)

For true horizontal scaling, an external database (PostgreSQL) would be needed - this is a potential future feature.

## Monitoring

CasaDrop exposes Prometheus metrics at `/metrics`. Add a ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: casadrop
spec:
  selector:
    matchLabels:
      app: casadrop
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

## Resource Recommendations

| Cluster Size | CPU | Memory |
|--------------|-----|--------|
| Small (home) | 100m | 128Mi |
| Medium | 250m | 256Mi |
| Large | 500m | 512Mi |

Adjust based on expected upload sizes and concurrent users.
