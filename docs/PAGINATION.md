# Railzway API Pagination Guide

[← Back to Documentation Index](index.md)

## Overview

Railzway APIs use **cursor-based pagination** for all list endpoints. This approach provides:
- **Consistent performance** regardless of dataset size
- **Real-time accuracy** - no missing or duplicate items when data changes
- **Scalability** - efficient for large datasets

---

## Query Parameters

All paginated list endpoints accept these parameters:

| Parameter | Type | Required | Default | Max | Description |
|-----------|------|----------|---------|-----|-------------|
| `page_size` | integer | No | 10 | 250 | Number of items per page |
| `page_token` | string | No | - | - | Cursor for next/previous page |

### Example Request

```bash
GET /v1/customers?page_size=20&page_token=eyJpZCI6IjEyMzQ1In0=
```

---

## Response Structure

All paginated responses follow this structure:

```json
{
  "data": [...],
  "page_info": {
    "next_page_token": "eyJpZCI6IjY3ODkwIn0=",
    "previous_page_token": "",
    "has_more": true
  }
}
```

### Fields

- **data**: Array of resources
- **page_info**: Pagination metadata
  - **next_page_token**: Token for the next page (empty if no more pages)
  - **previous_page_token**: Token for the previous page (empty if first page)
  - **has_more**: Boolean indicating if more pages exist

---

## How It Works

### First Page

Request the first page without a `page_token`:

```bash
curl "https://api.railzway.com/v1/customers?page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "data": [
    {"id": "1", "email": "customer1@example.com"},
    {"id": "2", "email": "customer2@example.com"}
  ],
  "page_info": {
    "next_page_token": "eyJpZCI6IjIifQ==",
    "previous_page_token": "",
    "has_more": true
  }
}
```

### Next Page

Use the `next_page_token` from the previous response:

```bash
curl "https://api.railzway.com/v1/customers?page_size=10&page_token=eyJpZCI6IjIifQ==" \
  -H "Authorization: Bearer $TOKEN"
```

### Last Page

When `has_more` is `false`, you've reached the end:

```json
{
  "data": [
    {"id": "99", "email": "customer99@example.com"}
  ],
  "page_info": {
    "next_page_token": "",
    "previous_page_token": "eyJpZCI6Ijg5In0=",
    "has_more": false
  }
}
```

---

## Client Implementation Examples

### JavaScript/TypeScript

```typescript
interface PageInfo {
  next_page_token: string;
  previous_page_token: string;
  has_more: boolean;
}

interface PaginatedResponse<T> {
  data: T[];
  page_info: PageInfo;
}

async function fetchAllCustomers() {
  const allCustomers = [];
  let pageToken = '';
  
  do {
    const params = new URLSearchParams({
      page_size: '50',
      ...(pageToken && { page_token: pageToken })
    });
    
    const response = await fetch(
      `https://api.railzway.com/v1/customers?${params}`,
      {
        headers: { 'Authorization': `Bearer ${token}` }
      }
    );
    
    const result: PaginatedResponse<Customer> = await response.json();
    allCustomers.push(...result.data);
    pageToken = result.page_info.next_page_token;
    
  } while (pageToken);
  
  return allCustomers;
}
```

### Python

```python
import requests

def fetch_all_customers(api_key):
    all_customers = []
    page_token = None
    
    while True:
        params = {'page_size': 50}
        if page_token:
            params['page_token'] = page_token
        
        response = requests.get(
            'https://api.railzway.com/v1/customers',
            headers={'Authorization': f'Bearer {api_key}'},
            params=params
        )
        
        result = response.json()
        all_customers.extend(result['data'])
        
        if not result['page_info']['has_more']:
            break
            
        page_token = result['page_info']['next_page_token']
    
    return all_customers
```

### Go

```go
type PageInfo struct {
    NextPageToken     string `json:"next_page_token"`
    PreviousPageToken string `json:"previous_page_token"`
    HasMore           bool   `json:"has_more"`
}

type PaginatedResponse struct {
    Data     []Customer `json:"data"`
    PageInfo PageInfo   `json:"page_info"`
}

func fetchAllCustomers(client *http.Client, apiKey string) ([]Customer, error) {
    var allCustomers []Customer
    pageToken := ""
    
    for {
        url := fmt.Sprintf(
            "https://api.railzway.com/v1/customers?page_size=50&page_token=%s",
            pageToken,
        )
        
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("Authorization", "Bearer "+apiKey)
        
        resp, err := client.Do(req)
        if err != nil {
            return nil, err
        }
        
        var result PaginatedResponse
        json.NewDecoder(resp.Body).Decode(&result)
        resp.Body.Close()
        
        allCustomers = append(allCustomers, result.Data...)
        
        if !result.PageInfo.HasMore {
            break
        }
        
        pageToken = result.PageInfo.NextPageToken
    }
    
    return allCustomers, nil
}
```

---

## Best Practices

### 1. Use Appropriate Page Sizes

- **Small pages (10-20)**: UI pagination, real-time updates
- **Medium pages (50-100)**: Background processing, exports
- **Large pages (200-250)**: Bulk operations, data migrations

### 2. Handle Errors Gracefully

```typescript
async function fetchPage(pageToken?: string) {
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return await response.json();
  } catch (error) {
    // Retry with exponential backoff
    await retryWithBackoff(fetchPage, pageToken);
  }
}
```

### 3. Cache Page Tokens

For UI pagination, cache page tokens to enable forward/backward navigation:

```typescript
const pageTokens: string[] = [''];  // Start with empty token

function goToNextPage() {
  const currentToken = pageTokens[pageTokens.length - 1];
  const result = await fetchPage(currentToken);
  
  if (result.page_info.next_page_token) {
    pageTokens.push(result.page_info.next_page_token);
  }
  
  return result;
}

function goToPreviousPage() {
  pageTokens.pop();  // Remove current token
  const previousToken = pageTokens[pageTokens.length - 1];
  return fetchPage(previousToken);
}
```

### 4. Don't Store Page Tokens Long-Term

Page tokens are ephemeral and may expire. Always start fresh pagination sessions from the beginning.

### 5. Respect Rate Limits

When fetching all pages, add delays between requests:

```typescript
async function fetchAllWithDelay(delayMs = 100) {
  const results = [];
  let pageToken = '';
  
  do {
    const page = await fetchPage(pageToken);
    results.push(...page.data);
    pageToken = page.page_info.next_page_token;
    
    if (pageToken) {
      await new Promise(resolve => setTimeout(resolve, delayMs));
    }
  } while (pageToken);
  
  return results;
}
```

---

## Paginated Endpoints

The following endpoints support cursor-based pagination:

- `GET /v1/customers`
- `GET /v1/subscriptions`
- `GET /v1/invoices`
- `GET /v1/products`
- `GET /v1/prices`
- `GET /v1/features`
- `GET /v1/meters`
- `GET /v1/usage/events`
- `GET /v1/audit-logs`
- `GET /admin/billing-operations/overdue-invoices`
- `GET /admin/billing-operations/outstanding-customers`
- `GET /admin/billing-operations/payment-issues`
- `GET /admin/billing-operations`

---

## Troubleshooting

### Invalid Page Token

**Error:**
```json
{
  "error": {
    "type": "validation_error",
    "message": "validation error",
    "errors": [{
      "field": "page_token",
      "code": "invalid_page_token",
      "message": "invalid value"
    }]
  }
}
```

**Solution:** Start a new pagination session without a page token.

### Page Size Too Large

**Error:**
```json
{
  "error": {
    "type": "validation_error",
    "message": "validation error",
    "errors": [{
      "field": "page_size",
      "code": "invalid_page_size",
      "message": "page size must be between 1 and 250"
    }]
  }
}
```

**Solution:** Use a page size between 1 and 250.
