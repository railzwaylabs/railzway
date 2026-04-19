# Security & Role-Based Access Control (RBAC)

Railzway implements a robust, enterprise-grade Security model designed to support **Segregation of Duties (SoD)** and maintain high financial integrity.

## 1. Governance Model
The security architecture is built on three pillars:
- **Identity**: Managed via the Identity Provider (Auth module).
- **Organization Isolation**: Every resource belongs to an `OrgID`. Cross-org access is strictly forbidden at the database and API levels.
- **Granular RBAC**: Controlled via [Casbin](https://casbin.org/) with a multi-role hierarchy.

---

## 2. Enterprise Business Roles

Railzway defines 7 distinct business roles, each tailored to specific operational responsibilities.

| Role | Persona | Responsibility | Access Level |
| :--- | :--- | :--- | :--- |
| **OWNER** | CEO/Founders | Full system control & legal ownership. | All Access (`.*`) |
| **ADMIN** | IT Manager | Organization settings and team management. | All Access (`.*`) |
| **FINANCE** | CFO / Accountants | Managing invoices, taxes, ledger, and payouts. | Write: Financials / Read: All |
| **OPERATIONS** | Product / Sales Ops | Managing plans, products, and subscriptions. | Write: Catalog / Read: All |
| **DEVELOPER** | Engineers | Integration, API keys, and usage ingestion. | Write: Tech Tooling / Read: All |
| **CUSTOMER_SUPPORT** | Support Agent | Helping customers with billing inquiries. | Read: Customer Data Only |
| **AUDITOR** | Compliance Officer | Verifying financial integrity and audit trails. | Read-Only: Universal |

---

## 3. Policy Enforcement (Casbin)

Authorization is enforced at the API layer using a **Request-Object-Action (r.sub, r.obj, r.act)** model.

### 3.1. Role Mapping Examples
- **FINANCE**: Can `POST` to `/admin/v1/invoices` but will receive `403 Forbidden` if trying to `POST` to `/admin/v1/meters`.
- **OPERATIONS**: Can `PATCH` a `/admin/v1/plans` but cannot `VOID` an invoice.
- **CUSTOMER_SUPPORT**: Can `GET` an invoice link to share with a customer but cannot see the underlying `Ledger` transactions or `Audit Logs`.

---

## 4. Audit Logging

Every state-changing action (CREATE, UPDATE, DELETE, VOID) is recorded in the `audit_logs` table. 
Each entry includes:
- **Actor**: The user ID and Role at the time of the action.
- **Resource**: The type and ID of the affected entity.
- **Payload**: The `Before` and `After` states in JSON format.
- **Context**: Request ID, Timestamp, and IP Address (where applicable).

> [!IMPORTANT]
> Audit logs are **immutable**. Once written, they cannot be modified or deleted through the standard API, providing a reliable trail for financial compliance.

---

## 5. Security Best Practices
- **Least Privilege**: Users should be assigned the most restrictive role that allows them to perform their job.
- **API Key Scoping**: API keys are isolated to an Organization and should be rotated regularly.
- **Token Expiry**: Admin sessions use short-lived JWTs with mandatory CSRF protection.
