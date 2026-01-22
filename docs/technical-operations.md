# Technical Operations Guide

This document outlines technical interventions and SQL runbooks for managing Railzway's billing engine in exceptional circumstances. 

> [!CAUTION]
> **Database Interventions**
> these operations involve direct database manipulation. They should only be performed by engineers with direct database access and a full understanding of the consequences. Always backup before running update queries.

---

## Force Rating Replay

Railzway's rating engine is **deterministic and idempotent**. This means you can safeley re-run the rating process for a billing cycle as long as the invoice has not been finalized and sent to the customer.

**Use Case:**
- Late-arriving price configuration (you forgot to set a price).
- Late-arriving usage (backfilling data).
- Fixing a bug in the rating logic itself.

### The Algorithm

The scheduler triggers rating only when:
1. `billing_cycles.status` is **Closing (2)**.
2. `billing_cycles.rating_completed_at` is `NULL`.

To force a replay, we must reset these fields to a state where the scheduler picks them up again.

---

## Approval-Based Re-Rating (Recommended)

> [!IMPORTANT]
> **Four-Eyes Principle**
> As of the audit compliance update, re-rating requires approval from a different user than the requester. This prevents unauthorized or accidental billing changes.

### Step 1: Request Re-Rating

Use the Admin API to create an approval request:

```bash
POST /admin/billing/cycles/{cycle_id}/request-rerating
Content-Type: application/json
Authorization: Bearer {your_token}

{
  "reason": "Late-arriving usage data from upstream system"
}
```

**Response:**
```json
{
  "id": "1234567890",
  "status": "PENDING"
}
```

### Step 2: Approve Re-Rating

A **different user** (not the requester) must approve:

```bash
POST /admin/billing/change-requests/{request_id}/approve
Authorization: Bearer {approver_token}
```

**What happens:**
1. System validates that approver ≠ requester (four-eyes principle)
2. Billing cycle is reset:
   - `rating_completed_at = NULL`
   - `closed_at = NULL`
   - `status = 2` (Closing)
   - `last_error = NULL`
3. Audit log records both request and approval
4. Scheduler picks up the cycle on next run

### Step 3: Verify Re-Run

Check the audit logs or change request status:

```bash
GET /admin/billing/change-requests
```

---

## Direct SQL Re-Rating (Emergency Only)

> [!CAUTION]
> **Bypass Approval Workflow**
> Direct SQL updates bypass the approval workflow and audit trail. Only use in emergencies with proper authorization and manual audit logging.

### SQL Runbook

#### 1. Identify the Cycle
Find the `id` of the billing cycle you want to re-rate.

```sql
SELECT id, status, rating_completed_at, period_start, period_end 
FROM billing_cycles 
WHERE subscription_id = 'sub_...' 
ORDER BY period_end DESC 
LIMIT 1;
```

#### 2. Reset Cycle State
Run the following update to "rewind" the cycle state.

> [!WARNING]
> Do not run this on cycles that have `invoice_finalized_at IS NOT NULL` unless you know exactly what you are doing. Modifying finalized invoices breaks financial immutability.

```sql
UPDATE billing_cycles 
SET 
    -- Clear the completion flag so scheduler picks it up
    rating_completed_at = NULL,
    
    -- If the cycle had already moved to Closed (3), pull it back to Closing (2)
    closed_at = NULL,
    status = 2,
    
    -- Clear any previous errors to ensure a clean retry
    last_error = NULL
WHERE id = 'DATA_ID_CYCLE_ANDA';
```

#### 3. Verify Re-run
Check the logs or database. The scheduler will:
1. Detect the cycle is `Closing` and `rating_completed_at` is NULL.
2. DELETE all existing `rating_results` for that cycle.
3. Compute fresh `rating_results` from the current inputs.
4. Update `rating_completed_at` to the new timestamp.
