# JafarShop B2B API - Quick Start Guide

Get up and running with the JafarShop B2B API in minutes.

## Prerequisites

- API key from JafarShop
- Ability to make HTTP requests
- Base URL: `https://api.jafarshop.com/v1`

## Step 1: Get Your API Key

Contact JafarShop to receive your API key. Store it securely:

```bash
# Environment variable (recommended)
export JAFARSHOP_API_KEY="your-api-key-here"
```

## Step 2: Submit Your First Order

### Minimal Example (cURL)

```bash
curl -X POST https://api.jafarshop.com/v1/carts/submit \
  -H "Authorization: Bearer your-api-key-here" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "partner_order_id": "ORDER-001",
    "items": [
      {
        "sku": "JDTQ1834",
        "title": "Product Name",
        "price": 10.00,
        "quantity": 1
      }
    ],
    "customer": {
      "name": "John Doe"
    },
    "shipping": {
      "street": "123 Main St",
      "city": "Amman",
      "postal_code": "11118",
      "country": "JO"
    },
    "totals": {
      "subtotal": 10.00,
      "tax": 0.00,
      "shipping": 5.00,
      "total": 15.00
    },
    "payment_status": "paid"
  }'
```

### Response

**Success (200):**
```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING_CONFIRMATION"
}
```

**No JafarShop Products (204):**
- No response body
- This is expected if cart has no mapped SKUs

## Step 3: Check Order Status

```bash
curl -X GET https://api.jafarshop.com/v1/orders/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer your-api-key-here"
```

## Common Use Cases

### JavaScript/Node.js

```javascript
const axios = require('axios');
const { randomUUID } = require('crypto');

const API_KEY = process.env.JAFARSHOP_API_KEY;
const BASE_URL = 'https://api.jafarshop.com/v1';

// Submit order
async function submitOrder(orderData) {
  const response = await axios.post(
    `${BASE_URL}/carts/submit`,
    orderData,
    {
      headers: {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID()
      }
    }
  );
  return response.data;
}

// Check status
async function getOrderStatus(orderId) {
  const response = await axios.get(
    `${BASE_URL}/orders/${orderId}`,
    {
      headers: {
        'Authorization': `Bearer ${API_KEY}`
      }
    }
  );
  return response.data;
}
```

### Python

```python
import requests
import uuid
import os

API_KEY = os.getenv('JAFARSHOP_API_KEY')
BASE_URL = 'https://api.jafarshop.com/v1'

# Submit order
def submit_order(order_data):
    response = requests.post(
        f'{BASE_URL}/carts/submit',
        json=order_data,
        headers={
            'Authorization': f'Bearer {API_KEY}',
            'Content-Type': 'application/json',
            'Idempotency-Key': str(uuid.uuid4())
        }
    )
    if response.status_code == 204:
        return None  # No JafarShop products
    response.raise_for_status()
    return response.json()

# Check status
def get_order_status(order_id):
    response = requests.get(
        f'{BASE_URL}/orders/{order_id}',
        headers={'Authorization': f'Bearer {API_KEY}'}
    )
    response.raise_for_status()
    return response.json()
```

## Required Fields

### Minimum Request

```json
{
  "partner_order_id": "ORDER-001",
  "items": [
    {
      "sku": "SKU-123",
      "title": "Product",
      "price": 10.00,
      "quantity": 1
    }
  ],
  "customer": {
    "name": "Customer Name"
  },
  "shipping": {
    "street": "Address",
    "city": "City",
    "postal_code": "12345",
    "country": "JO"
  },
  "totals": {
    "subtotal": 10.00,
    "tax": 0.00,
    "shipping": 0.00,
    "total": 10.00
  }
}
```

## Quick Reference

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/carts/submit` | Submit order |
| `GET` | `/v1/orders/{id}` | Get order status |

### Headers

```
Authorization: Bearer {api_key}
Content-Type: application/json
Idempotency-Key: {uuid}  (optional but recommended)
```

### Status Codes

| Code | Meaning |
|------|---------|
| `200` | Success |
| `204` | No JafarShop products (expected) |
| `401` | Invalid API key |
| `422` | Validation error |

## Next Steps

1. ✅ Test with a real order
2. ✅ Implement error handling
3. ✅ Add status polling
4. ✅ Read [PARTNER_GUIDE.md](./PARTNER_GUIDE.md) for details
5. ✅ Follow [PARTNER_INTEGRATION_CHECKLIST.md](./PARTNER_INTEGRATION_CHECKLIST.md)

## Need Help?

- **Email:** Feras.jafarShop@gmail.com
- **Full Guide:** [PARTNER_GUIDE.md](./PARTNER_GUIDE.md)
- **API Docs:** [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)
