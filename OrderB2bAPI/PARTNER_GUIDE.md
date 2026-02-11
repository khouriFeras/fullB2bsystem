# JafarShop B2B API - Partner Integration Guide

**Version:** 1.0  
**Last Updated:** 2026-01-21

Welcome to the JafarShop B2B API integration guide. This comprehensive guide will help you integrate your store with JafarShop's order fulfillment system.

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Authentication](#authentication)
4. [Core Concepts](#core-concepts)
5. [API Endpoints](#api-endpoints)
6. [Field Reference](#field-reference)
7. [Order Statuses](#order-statuses)
8. [Error Handling](#error-handling)
9. [Code Examples](#code-examples)
10. [Best Practices](#best-practices)
11. [FAQ & Troubleshooting](#faq--troubleshooting)
12. [Support & Resources](#support--resources)

---

## Introduction

### What is the JafarShop B2B API?

The JafarShop B2B API allows partner stores to submit orders containing JafarShop products. When a customer orders products from your store that are fulfilled by JafarShop, you can submit the order through this API, and JafarShop will handle fulfillment.

### How It Works

1. **Customer places order** on your store
2. **You submit the cart** to JafarShop B2B API
3. **System checks** if cart contains JafarShop products (mapped SKUs)
4. **If yes:** Order is created in JafarShop's system and appears in Shopify
5. **If no:** API returns 204 (no order created - cart has no JafarShop products)
6. **You track order status** via polling the API

### Benefits

- ✅ Automated order submission
- ✅ Real-time order status tracking
- ✅ Support for mixed carts (JafarShop + your products)
- ✅ Idempotency protection (no duplicate orders)
- ✅ Secure API key authentication

---

## Getting Started

### Prerequisites

Before you begin, you need:

1. **API Key** - Provided by JafarShop during partner setup
2. **Base URL** - API endpoint (production/staging)
3. **Development environment** - Ability to make HTTP requests

### Quick Setup

1. **Receive your API key** from JafarShop
2. **Store it securely** (environment variable, secrets manager)
3. **Test connection** with a simple API call
4. **Start submitting orders**

See [PARTNER_QUICK_START.md](./PARTNER_QUICK_START.md) for a minimal working example.

### Environments

- **Production:** `https://api.jafarshop.com/v1`
- **Staging:** `https://staging-api.jafarshop.com/v1` (if available)
- **Development:** `http://localhost:8080/v1` (for testing)

---

## Authentication

### API Key Authentication

All API requests require authentication using an API key in the `Authorization` header:

```
Authorization: Bearer {your_api_key}
```

### Getting Your API Key

API keys are provided by JafarShop when your partner account is set up. You'll receive:
- Your unique API key
- Base URL for the API
- Integration instructions

**⚠️ Important:** API keys are shown only once during setup. Store them securely!

### Security Best Practices

1. **Never commit API keys to version control**
2. **Use environment variables** or secrets management tools
3. **Rotate keys** if compromised (contact JafarShop)
4. **Use HTTPS only** in production
5. **Never share keys** publicly or in logs

For detailed API key management, see [PARTNER_API_KEY_GUIDE.md](./PARTNER_API_KEY_GUIDE.md).

---

## Core Concepts

### SKU Mapping

**What is SKU Mapping?**

JafarShop maintains a mapping of SKUs (Stock Keeping Units) that identifies which products are fulfilled by JafarShop. This mapping determines:

- **Mapped SKUs** = JafarShop products (fulfilled by JafarShop)
- **Unmapped SKUs** = Your products or other products (not fulfilled by JafarShop)

**How It Works:**

1. When you submit a cart, the system checks each SKU against the mapping
2. If **at least one SKU is mapped** → Order is created
3. If **no SKUs are mapped** → API returns 204 No Content (no order created)

**In the Order Response:**

- `is_supplier_item: true` → Mapped SKU (JafarShop product)
- `is_supplier_item: false` → Unmapped SKU (your product)

### Mixed Carts

You can submit carts containing both JafarShop products and your own products:

**Example Cart:**
- `JDTQ1834` (mapped) → JafarShop product
- `YOUR-SKU-123` (unmapped) → Your product

**Result:**
- Order is created (because at least one SKU is mapped)
- `JDTQ1834` appears as a Shopify variant line item
- `YOUR-SKU-123` appears as a custom line item

Both items are included in the order, but only JafarShop products are fulfilled by JafarShop.

### Order Lifecycle

Orders go through the following statuses:

```
PENDING_CONFIRMATION → CONFIRMED → SHIPPED → DELIVERED
                    ↘ REJECTED
                    ↘ CANCELLED
```

**Status Flow:**
1. **PENDING_CONFIRMATION** - Order received, awaiting manual review
2. **CONFIRMED** - Order confirmed and ready for fulfillment
3. **SHIPPED** - Order shipped with tracking information
4. **DELIVERED** - Order delivered (optional status)
5. **REJECTED** - Order rejected (with reason)
6. **CANCELLED** - Order cancelled

See [Order Statuses](#order-statuses) section for details.

### Idempotency

**What is Idempotency?**

Idempotency prevents duplicate orders if you retry a request. Each order submission should include a unique `Idempotency-Key` header.

**How It Works:**

- **Same key + Same payload** → Returns existing order (no duplicate)
- **Same key + Different payload** → Returns 409 Conflict
- **Different key** → Creates new order

**Best Practice:**

Generate a unique UUID for each order submission:

```javascript
// JavaScript
const idempotencyKey = crypto.randomUUID();

// Python
import uuid
idempotency_key = str(uuid.uuid4())
```

---

## API Endpoints

### 1. Submit Cart

Submit a cart for order processing.

**Endpoint:** `POST /v1/carts/submit`

**Headers:**
- `Authorization: Bearer {api_key}` (required)
- `Content-Type: application/json` (required)
- `Idempotency-Key: {uuid}` (optional, recommended)

**Request Body:**

```json
{
  "partner_order_id": "ORDER-2024-001",
  "items": [
    {
      "sku": "JDTQ1834",
      "title": "JafarShop Product",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://yourstore.com/product/jdtq1834"
    },
    {
      "sku": "YOUR-SKU-123",
      "title": "Your Product",
      "price": 19.99,
      "quantity": 1,
      "product_url": "https://yourstore.com/product/your-sku-123"
    }
  ],
  "customer": {
    "name": "John Doe",
    "phone": "+1234567890"
  },
  "shipping": {
    "street": "123 Main Street",
    "city": "Amman",
    "state": "Khalda",
    "postal_code": "11118",
    "country": "JO"
  },
  "totals": {
    "subtotal": 79.97,
    "tax": 6.40,
    "shipping": 5.00,
    "total": 91.37
  },
  "payment_status": "paid",
  "payment_method": "Cash On Delivery (COD)"
}
```

**Response (200 OK):**

```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING_CONFIRMATION"
}
```

**Response (204 No Content):**

- Cart does not contain any JafarShop products
- No response body
- This is **expected behavior**, not an error

**Response (409 Conflict):**

```json
{
  "error": "idempotency key conflict: same key used with different payload"
}
```

**Response (422 Unprocessable Entity):**

```json
{
  "error": "validation failed",
  "details": "field validation errors..."
}
```

### 2. Get Order Status

Retrieve the current status and details of an order.

**Endpoint:** `GET /v1/orders/{supplier_order_id}`

**Headers:**
- `Authorization: Bearer {api_key}` (required)

**Path Parameters:**
- `supplier_order_id` - UUID of the supplier order (returned from cart submission)

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "partner_order_id": "ORDER-2024-001",
  "status": "CONFIRMED",
  "shopify_order_id": 6349083345108,
  "customer_name": "John Doe",
  "customer_phone": "+1234567890",
  "shipping_address": {
    "street": "123 Main Street",
    "city": "Amman",
    "state": "Khalda",
    "postal_code": "11118",
    "country": "JO"
  },
  "cart_total": 91.37,
  "payment_status": "paid",
  "payment_method": "Cash On Delivery (COD)",
  "items": [
    {
      "sku": "JDTQ1834",
      "title": "JafarShop Product",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://yourstore.com/product/jdtq1834",
      "is_supplier_item": true,
      "shopify_variant_id": 44219312570580
    },
    {
      "sku": "YOUR-SKU-123",
      "title": "Your Product",
      "price": 19.99,
      "quantity": 1,
      "product_url": "https://yourstore.com/product/your-sku-123",
      "is_supplier_item": false
    }
  ],
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:05:00Z"
}
```

**Response (404 Not Found):**

```json
{
  "error": "order not found"
}
```

**Response (403 Forbidden):**

```json
{
  "error": "access denied"
}
```

(Returned when the order belongs to a different partner)

---

## Field Reference

### Cart Submission Fields

#### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `partner_order_id` | string | Your unique order identifier | `"ORDER-2024-001"` |
| `items` | array | Array of cart items (min 1) | See below |
| `customer` | object | Customer information | See below |
| `shipping` | object | Shipping address | See below |
| `totals` | object | Order totals | See below |

#### Optional Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `payment_status` | string | Payment status | `"paid"`, `"pending"` |
| `payment_method` | string | Payment method | `"Cash On Delivery (COD)"` |

### Item Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `sku` | string | ✅ | Product SKU | `"JDTQ1834"` |
| `title` | string | ✅ | Product title | `"JafarShop Product"` |
| `price` | number | ✅ | Unit price (≥ 0) | `29.99` |
| `quantity` | integer | ✅ | Quantity (≥ 1) | `2` |
| `product_url` | string | ❌ | Product URL on your store | `"https://yourstore.com/product"` |

### Customer Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `name` | string | ✅ | Customer full name | `"John Doe"` |
| `phone` | string | ❌ | Customer phone number | `"+1234567890"` |

### Shipping Address Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `street` | string | ✅ | Street address | `"123 Main Street"` |
| `city` | string | ✅ | City | `"Amman"` |
| `state` | string | ❌ | State/Province | `"Khalda"` |
| `postal_code` | string | ✅ | Postal/ZIP code | `"11118"` |
| `country` | string | ✅ | Country code (ISO 3166-1 alpha-2) | `"JO"` |

### Totals Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `subtotal` | number | ✅ | Subtotal (≥ 0) | `79.97` |
| `tax` | number | ✅ | Tax amount (≥ 0) | `6.40` |
| `shipping` | number | ✅ | Shipping cost (≥ 0) | `5.00` |
| `total` | number | ✅ | Total amount (≥ 0) | `91.37` |

**Note:** The `total` should equal `subtotal + tax + shipping` (or your calculation method).

### Payment Methods

Supported payment methods:
- `Cash On Delivery (COD)`
- `Card On Delivery`
- `Credit Card`
- `ZainCash`

---

## Order Statuses

### Status Definitions

| Status | Description | What It Means |
|--------|-------------|---------------|
| `PENDING_CONFIRMATION` | Order received | Order is in queue for manual review |
| `CONFIRMED` | Order confirmed | Order approved and ready for fulfillment |
| `REJECTED` | Order rejected | Order rejected (check `rejection_reason`) |
| `SHIPPED` | Order shipped | Order shipped with tracking information |
| `DELIVERED` | Order delivered | Order delivered to customer (optional) |
| `CANCELLED` | Order cancelled | Order cancelled |

### Status Transitions

```
PENDING_CONFIRMATION
    ├─→ CONFIRMED → SHIPPED → DELIVERED
    ├─→ REJECTED (terminal)
    └─→ CANCELLED (terminal)
```

**Valid Transitions:**
- `PENDING_CONFIRMATION` → `CONFIRMED`, `REJECTED`, or `CANCELLED`
- `CONFIRMED` → `SHIPPED` or `CANCELLED`
- `SHIPPED` → `DELIVERED`
- `REJECTED`, `DELIVERED`, `CANCELLED` → Terminal (no further changes)

### Polling Recommendations

- **Initial:** Poll every 5-10 minutes for `PENDING_CONFIRMATION` orders
- **After CONFIRMED:** Poll every 15-30 minutes
- **After SHIPPED:** Poll once per day (or use webhooks if available)

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| `200` | Success | Process response |
| `204` | No Content | No JafarShop products in cart (expected) |
| `400` | Bad Request | Check request format |
| `401` | Unauthorized | Check API key |
| `403` | Forbidden | Access denied (wrong partner) |
| `404` | Not Found | Order doesn't exist |
| `409` | Conflict | Idempotency key conflict |
| `422` | Unprocessable Entity | Validation error |
| `500` | Internal Server Error | Retry or contact support |

### Error Response Format

All errors follow this format:

```json
{
  "error": "Error message",
  "details": "Additional details (if applicable)"
}
```

### Common Errors

#### 401 Unauthorized

**Cause:** Invalid or missing API key

**Solution:**
- Verify API key is correct
- Check `Authorization` header format: `Bearer {api_key}`
- Ensure API key hasn't been rotated

#### 403 Forbidden

**Cause:** Trying to access another partner's order

**Solution:**
- Verify you're using the correct `supplier_order_id`
- Ensure the order belongs to your partner account

#### 409 Conflict

**Cause:** Same idempotency key used with different payload

**Solution:**
- Generate a new unique idempotency key
- Ensure payload matches if retrying

#### 422 Unprocessable Entity

**Cause:** Validation error (missing required fields, invalid values)

**Solution:**
- Check error details for specific field errors
- Verify all required fields are present
- Check data types and constraints

#### 204 No Content

**Not an error!** This means:
- Cart contains no JafarShop products (all SKUs are unmapped)
- No order was created
- This is expected behavior

### Retry Strategy

**Recommended approach:**

1. **Transient errors (500, 503):**
   - Retry with exponential backoff
   - Max 3 retries
   - Wait: 1s, 2s, 4s

2. **Client errors (400, 401, 403, 422):**
   - Don't retry (fix the request)
   - Log error for debugging

3. **Idempotency conflicts (409):**
   - Generate new idempotency key
   - Retry with new key

---

## Code Examples

### cURL

```bash
curl -X POST https://api.jafarshop.com/v1/carts/submit \
  -H "Authorization: Bearer your-api-key-here" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "partner_order_id": "ORDER-2024-001",
    "items": [
      {
        "sku": "JDTQ1834",
        "title": "JafarShop Product",
        "price": 29.99,
        "quantity": 2
      }
    ],
    "customer": {
      "name": "John Doe",
      "phone": "+1234567890"
    },
    "shipping": {
      "street": "123 Main Street",
      "city": "Amman",
      "postal_code": "11118",
      "country": "JO"
    },
    "totals": {
      "subtotal": 59.98,
      "tax": 4.80,
      "shipping": 5.00,
      "total": 69.78
    },
    "payment_status": "paid",
    "payment_method": "Cash On Delivery (COD)"
  }'
```

### JavaScript/Node.js

```javascript
const axios = require('axios');
const { randomUUID } = require('crypto');

const API_KEY = process.env.JAFARSHOP_API_KEY;
const BASE_URL = 'https://api.jafarshop.com/v1';

async function submitOrder(orderData) {
  try {
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
  } catch (error) {
    if (error.response) {
      // Handle API errors
      console.error('API Error:', error.response.status, error.response.data);
    } else {
      // Handle network errors
      console.error('Network Error:', error.message);
    }
    throw error;
  }
}

// Usage
const orderData = {
  partner_order_id: 'ORDER-2024-001',
  items: [
    {
      sku: 'JDTQ1834',
      title: 'JafarShop Product',
      price: 29.99,
      quantity: 2
    }
  ],
  customer: {
    name: 'John Doe',
    phone: '+1234567890'
  },
  shipping: {
    street: '123 Main Street',
    city: 'Amman',
    postal_code: '11118',
    country: 'JO'
  },
  totals: {
    subtotal: 59.98,
    tax: 4.80,
    shipping: 5.00,
    total: 69.78
  },
  payment_status: 'paid',
  payment_method: 'Cash On Delivery (COD)'
};

submitOrder(orderData)
  .then(result => {
    console.log('Order created:', result.supplier_order_id);
  })
  .catch(error => {
    console.error('Failed to submit order:', error);
  });
```

### Python

```python
import requests
import uuid
import os

API_KEY = os.getenv('JAFARSHOP_API_KEY')
BASE_URL = 'https://api.jafarshop.com/v1'

def submit_order(order_data):
    headers = {
        'Authorization': f'Bearer {API_KEY}',
        'Content-Type': 'application/json',
        'Idempotency-Key': str(uuid.uuid4())
    }
    
    try:
        response = requests.post(
            f'{BASE_URL}/carts/submit',
            json=order_data,
            headers=headers
        )
        
        if response.status_code == 204:
            print('No JafarShop products in cart')
            return None
        
        response.raise_for_status()
        return response.json()
    except requests.exceptions.HTTPError as e:
        print(f'API Error: {e.response.status_code}')
        print(f'Response: {e.response.text}')
        raise
    except requests.exceptions.RequestException as e:
        print(f'Network Error: {e}')
        raise

# Usage
order_data = {
    'partner_order_id': 'ORDER-2024-001',
    'items': [
        {
            'sku': 'JDTQ1834',
            'title': 'JafarShop Product',
            'price': 29.99,
            'quantity': 2
        }
    ],
    'customer': {
        'name': 'John Doe',
        'phone': '+1234567890'
    },
    'shipping': {
        'street': '123 Main Street',
        'city': 'Amman',
        'postal_code': '11118',
        'country': 'JO'
    },
    'totals': {
        'subtotal': 59.98,
        'tax': 4.80,
        'shipping': 5.00,
        'total': 69.78
    },
    'payment_status': 'paid',
    'payment_method': 'Cash On Delivery (COD)'
}

result = submit_order(order_data)
if result:
    print(f'Order created: {result["supplier_order_id"]}')
```

### PHP

```php
<?php

$apiKey = getenv('JAFARSHOP_API_KEY');
$baseUrl = 'https://api.jafarshop.com/v1';

function submitOrder($orderData, $apiKey, $baseUrl) {
    $ch = curl_init($baseUrl . '/carts/submit');
    
    $headers = [
        'Authorization: Bearer ' . $apiKey,
        'Content-Type: application/json',
        'Idempotency-Key: ' . uniqid('', true)
    ];
    
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_POSTFIELDS => json_encode($orderData)
    ]);
    
    $response = curl_exec($ch);
    $statusCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);
    
    if ($statusCode === 204) {
        return null; // No JafarShop products
    }
    
    if ($statusCode !== 200) {
        throw new Exception("API Error: $statusCode - $response");
    }
    
    return json_decode($response, true);
}

// Usage
$orderData = [
    'partner_order_id' => 'ORDER-2024-001',
    'items' => [
        [
            'sku' => 'JDTQ1834',
            'title' => 'JafarShop Product',
            'price' => 29.99,
            'quantity' => 2
        ]
    ],
    'customer' => [
        'name' => 'John Doe',
        'phone' => '+1234567890'
    ],
    'shipping' => [
        'street' => '123 Main Street',
        'city' => 'Amman',
        'postal_code' => '11118',
        'country' => 'JO'
    ],
    'totals' => [
        'subtotal' => 59.98,
        'tax' => 4.80,
        'shipping' => 5.00,
        'total' => 69.78
    ],
    'payment_status' => 'paid',
    'payment_method' => 'Cash On Delivery (COD)'
];

try {
    $result = submitOrder($orderData, $apiKey, $baseUrl);
    if ($result) {
        echo "Order created: " . $result['supplier_order_id'] . "\n";
    }
} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}
?>
```

### Java

```java
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.URI;
import java.util.UUID;
import com.google.gson.Gson;

public class JafarShopClient {
    private final String apiKey;
    private final String baseUrl;
    private final HttpClient httpClient;
    private final Gson gson;
    
    public JafarShopClient(String apiKey, String baseUrl) {
        this.apiKey = apiKey;
        this.baseUrl = baseUrl;
        this.httpClient = HttpClient.newHttpClient();
        this.gson = new Gson();
    }
    
    public OrderResponse submitOrder(OrderRequest orderData) throws Exception {
        String json = gson.toJson(orderData);
        
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(baseUrl + "/carts/submit"))
            .header("Authorization", "Bearer " + apiKey)
            .header("Content-Type", "application/json")
            .header("Idempotency-Key", UUID.randomUUID().toString())
            .POST(HttpRequest.BodyPublishers.ofString(json))
            .build();
        
        HttpResponse<String> response = httpClient.send(
            request,
            HttpResponse.BodyHandlers.ofString()
        );
        
        if (response.statusCode() == 204) {
            return null; // No JafarShop products
        }
        
        if (response.statusCode() != 200) {
            throw new Exception("API Error: " + response.statusCode() + " - " + response.body());
        }
        
        return gson.fromJson(response.body(), OrderResponse.class);
    }
}
```

---

## Best Practices

### 1. Idempotency

**Always use idempotency keys** to prevent duplicate orders:

```javascript
// ✅ Good
const idempotencyKey = crypto.randomUUID();

// ❌ Bad
const idempotencyKey = 'order-123'; // Not unique across retries
```

### 2. Error Handling

**Implement proper error handling:**

```javascript
try {
  const response = await submitOrder(orderData);
  if (response.status === 204) {
    // No JafarShop products - this is expected
    return;
  }
  // Process successful response
} catch (error) {
  if (error.response?.status === 409) {
    // Idempotency conflict - generate new key and retry
  } else if (error.response?.status === 422) {
    // Validation error - fix request
  } else {
    // Log and handle other errors
  }
}
```

### 3. Retry Logic

**Retry transient errors with exponential backoff:**

```javascript
async function submitWithRetry(orderData, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await submitOrder(orderData);
    } catch (error) {
      if (error.response?.status >= 500 && i < maxRetries - 1) {
        await sleep(Math.pow(2, i)); // 1s, 2s, 4s
        continue;
      }
      throw error;
    }
  }
}
```

### 4. Status Polling

**Poll order status efficiently:**

```javascript
async function pollOrderStatus(orderId, maxAttempts = 20) {
  for (let i = 0; i < maxAttempts; i++) {
    const order = await getOrderStatus(orderId);
    
    if (order.status === 'SHIPPED' || order.status === 'DELIVERED') {
      return order; // Final status
    }
    
    if (order.status === 'REJECTED' || order.status === 'CANCELLED') {
      throw new Error(`Order ${order.status}: ${order.rejection_reason || ''}`);
    }
    
    // Wait before next poll
    await sleep(300000); // 5 minutes
  }
  
  throw new Error('Order status polling timeout');
}
```

### 5. Security

- ✅ Store API keys in environment variables
- ✅ Use HTTPS only in production
- ✅ Never log API keys
- ✅ Rotate keys if compromised
- ✅ Use secrets management tools

### 6. Performance

- ✅ Submit orders asynchronously (don't block checkout)
- ✅ Implement proper timeout handling
- ✅ Use connection pooling
- ✅ Batch status checks if possible

---

## FAQ & Troubleshooting

### General Questions

**Q: Do I need to map all my products?**  
A: No. Only map products that JafarShop will fulfill. Unmapped products will appear as custom line items in mixed carts.

**Q: What happens if my cart has no JafarShop products?**  
A: The API returns 204 No Content. This is expected behavior, not an error.

**Q: How do I know which SKUs are mapped?**  
A: Check the order response - items with `is_supplier_item: true` are mapped.

**Q: Can I submit the same order twice?**  
A: Use idempotency keys. Same key + same payload returns the existing order (no duplicate).

**Q: How often should I poll order status?**  
A: Poll every 5-10 minutes for pending orders, less frequently after confirmation.

### Common Issues

**Issue: Getting 401 Unauthorized**

**Solutions:**
- Verify API key is correct
- Check `Authorization` header format: `Bearer {key}`
- Ensure no extra spaces in header
- Contact JafarShop if key was rotated

**Issue: Getting 204 No Content**

**This is not an error!** It means:
- Cart contains no JafarShop products
- No order was created
- This is expected behavior

**Issue: Order status not updating**

**Solutions:**
- Orders start as `PENDING_CONFIRMATION`
- Status changes are manual (admin confirms/rejects)
- Poll more frequently if needed
- Check if order was rejected (see `rejection_reason`)

**Issue: Getting 409 Conflict**

**Solutions:**
- Generate a new unique idempotency key
- Don't reuse keys across different orders
- Ensure payload matches if retrying

**Issue: Getting 422 Validation Error**

**Solutions:**
- Check error details for specific field errors
- Verify all required fields are present
- Check data types (numbers vs strings)
- Ensure values meet constraints (e.g., quantity ≥ 1)

### Debugging Tips

1. **Log all requests/responses** (without API keys)
2. **Check HTTP status codes** first
3. **Read error messages** carefully
4. **Verify field formats** match examples
5. **Test with minimal payload** first
6. **Use staging environment** for testing

---

## Support & Resources

### Contact Information

**API Support:** Feras.jafarShop@gmail.com

**Response Time:** Business hours (Jordan time)

### Documentation

- [API Documentation](./API_DOCUMENTATION.md) - Complete technical reference
- [Quick Start Guide](./PARTNER_QUICK_START.md) - Get started quickly
- [Integration Checklist](./PARTNER_INTEGRATION_CHECKLIST.md) - Step-by-step integration
- [API Key Guide](./PARTNER_API_KEY_GUIDE.md) - API key management

### Additional Resources

- **Base URL:** `https://api.jafarshop.com/v1`
- **API Version:** v1
- **Format:** JSON
- **Authentication:** Bearer token (API key)

### Changelog

**Version 1.0 (2026-01-21)**
- Initial release
- Cart submission endpoint
- Order status tracking
- Idempotency support
- Mixed cart support

---

**Last Updated:** 2026-01-21  
**Document Version:** 1.0
