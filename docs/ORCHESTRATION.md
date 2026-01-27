# Orchestration Guide (Kubernetes & Nomad)

This guide provides reference configurations for deploying Railzway to container orchestrators.

## ☸️ Kubernetes

Railzway is designed to run natively on Kubernetes.

### 1. Database Migrations (Job)

Run this before deploying or updating the application.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: railzway-migration
spec:
  template:
    spec:
      containers:
      - name: migration
        image: ghcr.io/smallbiznis/railzway/railzway-migration:latest
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: railzway-secrets
              key: db-password
      restartPolicy: OnFailure
```

### 2. Admin Service (Deployment & Service)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: railzway-admin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: railzway-admin
  template:
    metadata:
      labels:
        app: railzway-admin
    spec:
      containers:
      - name: admin
        image: ghcr.io/smallbiznis/railzway/railzway-admin:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: railzway-secrets
              key: db-password
---
apiVersion: v1
kind: Service
metadata:
  name: railzway-admin
spec:
  selector:
    app: railzway-admin
  ports:
  - port: 80
    targetPort: 8080
```

### 3. Invoice Service (Deployment & Service)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: railzway-invoice
spec:
  replicas: 2
  selector:
    matchLabels:
      app: railzway-invoice
  template:
    metadata:
      labels:
        app: railzway-invoice
    spec:
      containers:
      - name: invoice
        image: ghcr.io/smallbiznis/railzway/railzway-invoice:latest
        ports:
        - containerPort: 8080
        env:
        - name: API_URL
          value: "http://railzway-admin" # Cluster DNS
---
apiVersion: v1
kind: Service
metadata:
  name: railzway-invoice
spec:
  selector:
    app: railzway-invoice
  ports:
  - port: 80
    targetPort: 8080
```

### 4. Scheduler (Deployment)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: railzway-scheduler
spec:
  replicas: 1
  selector:
    matchLabels:
      app: railzway-scheduler
  template:
    metadata:
      labels:
        app: railzway-scheduler
    spec:
      containers:
      - name: scheduler
        image: ghcr.io/smallbiznis/railzway/railzway-scheduler:latest
        env:
        - name: ENABLED_JOBS
          value: "billing,usage,invoice"
        - name: DB_HOST
          value: "postgres-service"
```

---

## 🏝️ Nomad

For Nomad users, here is a reference job specification (`railzway.nomad`).

```hcl
job "railzway" {
  datacenters = ["dc1"]
  type        = "service"

  group "admin" {
    count = 1
    network {
      port "http" {
        to = 8080
      }
    }
    task "admin" {
      driver = "docker"
      config {
        image = "ghcr.io/smallbiznis/railzway/railzway-admin:latest"
        ports = ["http"]
      }
      env {
        PORT      = "8080"
        DB_HOST   = "postgres.service.consul"
        DB_NAME   = "railzway"
      }
      service {
        name = "railzway-admin"
        port = "http"
        tags = ["urlprefix-/admin"]
      }
    }
  }

  group "scheduler" {
    count = 1
    task "scheduler" {
      driver = "docker"
      config {
        image = "ghcr.io/smallbiznis/railzway/railzway-scheduler:latest"
      }
      env {
        ENABLED_JOBS = "billing,usage,invoice"
        DB_HOST      = "postgres.service.consul"
      }
    }
  }

  group "invoice" {
    count = 2
    network {
      port "http" {
        to = 8080
      }
    }
    task "invoice" {
      driver = "docker"
      config {
        image = "ghcr.io/smallbiznis/railzway/railzway-invoice:latest"
        ports = ["http"]
      }
      env {
        API_URL = "http://railzway-admin.service.consul:8080"
      }
      service {
        name = "railzway-invoice"
        port = "http"
        tags = ["urlprefix-/invoice"]
      }
    }
  }
}
```

## ⎈ Helm Chart

We assume you are using a standard generic chart or building your own.

**Key Values Structure:**

```yaml
image:
  repository: ghcr.io/smallbiznis/railzway/railzway-admin
  tag: latest

env:
  DB_HOST: "postgres-postgresql"
  DB_USER: "railzway"

# Separate releases recommended for:
# 1. Admin (Ingress enabled)
# 2. Invoice (Ingress enabled, distinct host)
# 3. Scheduler (Ingress disabled)
```
