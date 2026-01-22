# Scheduler Service

The **Scheduler Service** (`apps/scheduler`) is the background worker plane of Railzway. It handles asynchronous jobs such as rating usage, generating invoices, closing billing cycles, and system maintenance.

## Features

- **Single Binary, Multiple Roles**: The same binary can be configured to run all jobs (monolith mode) or specific subsets of jobs (microservice mode).
- **Graceful Shutdown**: Handles SIGTERM/SIGINT to finish in-flight batches before exiting.
- **Prometheus Metrics**: Exports job duration, success/failure counts, and batch sizes.

## Configuration

The service is configured primarily via Environment Variables.

### `ENABLED_JOBS`

Controls which jobs are active in this instance. If empty or unset, **ALL** jobs are enabled.

Accepts a comma-separated list of job names:

| Job Name | Description |
| :--- | :--- |
| `ensure_cycles` | Opens billing cycles for new/renewing subscriptions. |
| `close_cycles` | Closes billing cycles that have reached their period end. |
| `rating` | Computes final costs for closed cycles. |
| `close_after_rating` | Marks cycles as closed after rating is complete. |
| `invoice` | Generates invoices for closed & rated cycles. |
| `rollup_rebuild` | Processes rebuild requests for billing dashboard stats. |
| `rollup_pending` | Updates dashboard stats with new events in real-time. |
| `end_canceled_subs` | Finalizes subscriptions marked for cancellation. |
| `recovery_sweep` | Retries stuck or failed jobs. |
| `sla_evaluation` | Evaluates SLA breaches (if configured). |
| `finops_scoring` | Computes FinOps scores (daily). |

### Other Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for health checks and metrics (`/metrics`). |
| `SCHEDULER_RUN_INTERVAL` | `1m` | How often the main loop triggers. |
| `SCHEDULER_BATCH_SIZE` | `50` | Default batch size for most jobs. |

## Deployment Examples

### 1. Monolith Mode (Default)
Runs every job. Good for development or low-traffic environments.
```bash
./scheduler
```

### 2. Rating Worker
Dedicated worker for heavy rating computations.
```bash
export ENABLED_JOBS="rating,close_after_rating"
./scheduler
```

### 3. Invoice Worker
Dedicated worker for generating invoices.
```bash
export ENABLED_JOBS="invoice"
./scheduler
```

### 4. Lifecycle Manager
Handles opening and closing of cycles (lightweight I/O).
```bash
export ENABLED_JOBS="ensure_cycles,close_cycles,end_canceled_subs"
./scheduler
```

## Build

```bash
go build -o scheduler ./apps/scheduler
```
