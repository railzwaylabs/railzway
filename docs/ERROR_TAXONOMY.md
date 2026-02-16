# Railzway API Error Taxonomy

[← Back to Documentation Index](index.md)

## Error Response Structure

All API errors follow a consistent JSON structure:

```json
{
  "error": {
    "type": "error_type",
    "message": "Human-readable error message",
    "errors": [
      {
        "field": "field_name",
        "code": "error_code",
        "message": "Detailed error message"
      }
    ]
  }
}
```

- **type**: High-level error category
- **message**: Human-readable description
- **errors**: Array of validation errors (only present for `validation_error` type)

---

## Error Types

### Authentication & Authorization

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `unauthorized` | 401 | Missing or invalid authentication credentials |
| `forbidden` | 403 | Authenticated but lacks permission for the resource |
| `precondition_required` | 428 | Organization context required but not provided |

**Example:**
```json
{
  "error": {
    "type": "unauthorized",
    "message": "unauthorized"
  }
}
```

---

### Resource Errors

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `not_found` | 404 | Requested resource does not exist |
| `conflict` | 409 | Resource already exists or state conflict |

**Example:**
```json
{
  "error": {
    "type": "not_found",
    "message": "not found"
  }
}
```

---

### Validation Errors

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `validation_error` | 400 | Request validation failed |

**Example:**
```json
{
  "error": {
    "type": "validation_error",
    "message": "validation error",
    "errors": [
      {
        "field": "email",
        "code": "invalid_email",
        "message": "invalid value"
      }
    ]
  }
}
```

---

### Rate Limiting & Quotas

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `rate_limited` | 429 | Too many requests, retry after cooldown |
| `quota_exceeded` | 402 | Organization quota limit reached |

**Example:**
```json
{
  "error": {
    "type": "rate_limited",
    "message": "rate limited"
  }
}
```

---

### System Errors

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `internal_error` | 500 | Unexpected server error |
| `service_unavailable` | 503 | Service temporarily unavailable |

**Example:**
```json
{
  "error": {
    "type": "internal_error",
    "message": "internal server error"
  }
}
```

---

## Validation Error Codes by Domain

### Organization
- `invalid_organization` - Invalid or missing organization ID
- `invalid_name` - Invalid organization name
- `invalid_id` - Invalid organization identifier

### Customer
- `invalid_customer` - Invalid customer reference
- `invalid_email` - Invalid email format
- `invalid_name` - Invalid customer name

### Subscription
- `invalid_subscription` - Invalid subscription ID
- `invalid_subscription_item` - Invalid subscription item
- `invalid_product_id` - Invalid product reference
- `invalid_price_id` - Invalid price reference

### Usage
- `invalid_meter` - Invalid meter reference
- `invalid_meter_code` - Invalid meter code
- `invalid_value` - Invalid usage value
- `invalid_recorded_at` - Invalid timestamp
- `invalid_idempotency_key` - Invalid or duplicate idempotency key
- `feature_not_entitled` - Feature not included in subscription

### Invoice
- `invalid_invoice_id` - Invalid invoice identifier
- `invalid_billing_cycle` - Invalid billing cycle
- `billing_cycle_not_closed` - Billing cycle still open
- `currency_mismatch` - Currency mismatch in invoice items

### Payment
- `invalid_provider` - Invalid payment provider
- `invalid_amount` - Invalid payment amount
- `invalid_currency` - Invalid currency code
- `invalid_signature` - Invalid webhook signature

### Product & Pricing
- `invalid_product_id` - Invalid product identifier
- `invalid_price_id` - Invalid price identifier
- `invalid_feature_id` - Invalid feature identifier
- `invalid_meter_id` - Invalid meter identifier
- `invalid_code` - Invalid code (product/feature)
- `invalid_type` - Invalid type value

### API Keys
- `invalid_key_id` - Invalid API key identifier
- `invalid_name` - Invalid API key name

### Audit
- `invalid_page_token` - Invalid pagination token
- `invalid_time_range` - Invalid time range parameters
- `invalid_action` - Invalid audit action type

### Tax
- `invalid_tax_code` - Invalid tax code
- `invalid_tax_mode` - Invalid tax calculation mode
- `invalid_tax_rate` - Invalid tax rate value

---

## Client Handling Recommendations

### Retry Strategy

**Retryable Errors** (with exponential backoff):
- `429 rate_limited` - Wait and retry
- `503 service_unavailable` - Temporary issue, retry
- `500 internal_error` - May be transient, retry with limit

**Non-Retryable Errors**:
- `400 validation_error` - Fix request parameters
- `401 unauthorized` - Refresh authentication
- `403 forbidden` - Check permissions
- `404 not_found` - Resource doesn't exist
- `409 conflict` - Handle conflict (e.g., resource exists)
- `402 quota_exceeded` - Upgrade plan or wait for quota reset

### Example Client Code

**JavaScript/TypeScript:**
```typescript
async function handleApiError(error: any) {
  const errorData = error.response?.data?.error;
  
  switch (errorData?.type) {
    case 'validation_error':
      // Show field-specific errors to user
      errorData.errors.forEach((err: any) => {
        showFieldError(err.field, err.message);
      });
      break;
      
    case 'unauthorized':
      // Redirect to login
      redirectToLogin();
      break;
      
    case 'rate_limited':
      // Retry with backoff
      await retryWithBackoff();
      break;
      
    case 'quota_exceeded':
      // Show upgrade prompt
      showUpgradeDialog();
      break;
      
    default:
      // Show generic error
      showError(errorData?.message || 'An error occurred');
  }
}
```

**Go:**
```go
func handleAPIError(err error, resp *http.Response) {
    var errorResp struct {
        Error struct {
            Type    string `json:"type"`
            Message string `json:"message"`
            Errors  []struct {
                Field   string `json:"field"`
                Code    string `json:"code"`
                Message string `json:"message"`
            } `json:"errors,omitempty"`
        } `json:"error"`
    }
    
    json.NewDecoder(resp.Body).Decode(&errorResp)
    
    switch errorResp.Error.Type {
    case "validation_error":
        for _, e := range errorResp.Error.Errors {
            log.Printf("Validation error on %s: %s", e.Field, e.Message)
        }
    case "rate_limited":
        time.Sleep(time.Second * 5)
        // Retry request
    case "unauthorized":
        // Refresh token
    default:
        log.Printf("API error: %s", errorResp.Error.Message)
    }
}
```

---

## Common Scenarios

### Invalid Request Parameters
```bash
curl -X POST https://api.railzway.com/v1/customers \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email": "invalid-email"}'
```

Response:
```json
{
  "error": {
    "type": "validation_error",
    "message": "validation error",
    "errors": [
      {
        "field": "email",
        "code": "invalid_email",
        "message": "invalid value"
      }
    ]
  }
}
```

### Resource Not Found
```bash
curl https://api.railzway.com/v1/customers/999999999
```

Response:
```json
{
  "error": {
    "type": "not_found",
    "message": "not found"
  }
}
```

### Rate Limited
```bash
# After many rapid requests
curl https://api.railzway.com/v1/usage/events
```

Response:
```json
{
  "error": {
    "type": "rate_limited",
    "message": "rate limited"
  }
}
```
