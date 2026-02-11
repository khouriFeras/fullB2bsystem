# Scripts Usage Guide

This guide shows how to run all the command-line tools and test scripts in the B2BAPI project.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Server Management](#server-management)
- [Database Management](#database-management)
- [Partner Management](#partner-management)
- [SKU Management](#sku-management)
- [Order Management](#order-management)
- [Shopify Integration](#shopify-integration)
- [Testing Scripts](#testing-scripts)

---

## Prerequisites

### Environment Setup

1. **Load environment variables:**
   ```bash
   # Copy example file
   cp env.example .env
   
   # Edit .env with your configuration
   # Required: DB_*, SHOPIFY_*, API_KEY_HASH_SALT
   ```

2. **Start PostgreSQL (if using Docker):**
   ```bash
   docker-compose up -d
   ```

3. **Verify database connection:**
   ```bash
   # Test connection (all scripts will test this automatically)
   ```

---

## Server Management

### Start the API Server

```bash
# Default port (8080)
go run cmd/server/main.go

# Custom port
PORT=8081 go run cmd/server/main.go

# Windows PowerShell
$env:PORT="8081"; go run cmd/server/main.go
```

**What it does:**
- Loads configuration from `.env`
- Connects to PostgreSQL
- Runs database migrations
- Starts HTTP server on specified port

**Expected output:**
```
2026-01-21T12:00:00.000+0300    INFO    Starting B2B API server
[GIN-debug] POST   /v1/carts/submit
[GIN-debug] GET    /v1/orders/:id
Server started successfully
```

---

## Database Management

### Run Migrations

```bash
# Run all migrations
go run cmd/migrate/main.go migrations/000001_init_schema.up.sql
go run cmd/migrate/main.go migrations/000002_add_payment_method.up.sql
go run cmd/migrate/main.go migrations/000003_add_shopify_order_id.up.sql
```

**Or use golang-migrate CLI:**
```bash
migrate -path ./migrations -database "postgres://postgres:123123@localhost:5432/b2bapi?sslmode=disable" up
```

---

## Partner Management

### Create a Partner

```bash
go run cmd/create-partner/main.go "<Partner Name>" "<API Key>"
```

**Examples:**
```bash
# Basic example
go run cmd/create-partner/main.go "Zain Shop" "zain-api-key-12345"

# With secure key
go run cmd/create-partner/main.go "Test Partner" "test-api-key-$(openssl rand -hex 32)"
```

**Output:**
```
✅ Partner created successfully!

Partner ID: 550e8400-e29b-41d4-a716-446655440000
Partner Name: Zain Shop
API Key: zain-api-key-12345

⚠️  IMPORTANT: Save this API key securely! You won't be able to see it again.
```

**⚠️ Important:** Save the API key immediately - it's shown only once!

---

## SKU Management

### Find SKU in Shopify

```bash
go run cmd/find-sku/main.go "<SKU>"
```

**Examples:**
```bash
# Basic search
go run cmd/find-sku/main.go "JDTQ1834"

# With debug output
go run cmd/find-sku/main.go --debug "Spong daddy"

# With custom limit
go run cmd/find-sku/main.go --limit=50 "JDTQ1834"

# Show hex encoding (for hidden characters)
go run cmd/find-sku/main.go --hex "SKU-123"
```

**Output:**
```
Connected Shopify store:
  Name: جعفر شوب
  Domain: jafarshop.myshopify.com

FOUND (exact match):
  SKU           : "JDTQ1834"
  Product Title : Product Name
  Variant Title : Default Title
  Price         : 10.000

IDs:
  Product ID: 8085607284948
  Variant ID: 44219312570580

To add this to the database, run:
  go run cmd/add-sku/main.go "JDTQ1834" 8085607284948 44219312570580
```

### Add SKU Mapping

```bash
go run cmd/add-sku/main.go "<SKU>" <product-id> <variant-id>
```

**Example:**
```bash
go run cmd/add-sku/main.go "JDTQ1834" 8085607284948 44219312570580
```

**Output:**
```
✅ SKU mapping created successfully!
SKU: JDTQ1834
Shopify Product ID: 8085607284948
Shopify Variant ID: 44219312570580
```

### List SKU Mappings

```bash
go run cmd/list-sku-mappings/main.go
```

**Output:**
```
Active SKU Mappings:
  SKU: JDTQ1834
    Shopify Product ID: 8085607284948
    Shopify Variant ID: 44219312570580
    Created At: 2026-01-21 12:00:00
```

### List All SKUs from Shopify

```bash
go run cmd/list-skus/main.go
```

**Output:**
```
🔍 Fetching all SKUs from Shopify...
✅ Found 150 SKUs with values

First 20 SKUs:
─────────────────────────────────────────────────────────────
1. SKU: JDTQ1834 | Product: Product Name
2. SKU: Spong daddy | Product: Spong Product
...
```

### List Products from Shopify

```bash
go run cmd/list-products/main.go
```

---

## Order Management

### Find Order by Partner Order ID

```bash
go run cmd/find-order/main.go "<partner_order_id>"
```

**Examples:**
```bash
# With hash prefix
go run cmd/find-order/main.go "#45246"

# Without hash
go run cmd/find-order/main.go "ORDER-2024-001"

# If not found, lists recent orders
```

**Output:**
```
✅ Order found!

Supplier Order ID: 550e8400-e29b-41d4-a716-446655440000
Partner Order ID: ORDER-2024-001
Status: PENDING_CONFIRMATION
Customer: John Doe
Total: 91.37
Payment Status: paid
Payment Method: Cash On Delivery (COD)
Shopify Order ID: 6349083345108
Created: 2024-01-01T12:00:00Z
```

### List All Orders

```bash
go run cmd/list-orders/main.go
```

**Output:**
```
📋 Listing all orders in database:
Order #1:
  Supplier Order ID: 550e8400-e29b-41d4-a716-446655440000
  Partner Order ID: ORDER-2024-001
  Status: PENDING_CONFIRMATION
  Customer: John Doe
  Total: 91.37
  Payment Status: paid
  Payment Method: Cash On Delivery (COD)
  Shopify Order ID: 6349083345108
  Created: 2024-01-01T12:00:00Z

✅ Found 1 order(s)
```

### Check Order Items

```bash
go run cmd/check-order-items/main.go <supplier_order_id>
```

**Example:**
```bash
go run cmd/check-order-items/main.go 550e8400-e29b-41d4-a716-446655440000
```

**Output:**
```
Order: ORDER-2024-001
Shopify Order ID: 6349083345108

Order Items (2):
─────────────────────────────────────────────────────────────

Item 1:
  SKU: JDTQ1834
  Title: JafarShop Product
  Price: 29.99
  Quantity: 2
  Is Supplier Item: true
  Shopify Variant ID: 44219312570580
  → This will appear as a VARIANT in Shopify

Item 2:
  SKU: YOUR-SKU-123
  Title: Your Product
  Price: 19.99
  Quantity: 1
  Is Supplier Item: false
  Shopify Variant ID: (none)
  → This will appear as a CUSTOM LINE ITEM in Shopify
```

---

## Shopify Integration

### Get Order from Shopify by Order Number

```bash
go run cmd/get-shopify-order/main.go "<order_number>"
```

**Examples:**
```bash
# With hash
go run cmd/get-shopify-order/main.go "#45246"

# Without hash
go run cmd/get-shopify-order/main.go "45246"

# With name prefix
go run cmd/get-shopify-order/main.go "name:#45246"
```

**Output:**
```
✅ Order found!

Order Information:
  Order Number: #45246
  Order ID: gid://shopify/Order/6349083345108
  Fulfillment Status: Unfulfilled
  Financial Status: Paid
  Total: 111.60 JOD
  Created: 2024-01-01T12:00:00Z
```

### Get Order from Shopify by Order ID

```bash
go run cmd/get-shopify-order-by-id/main.go <shopify_order_id>
```

**Example:**
```bash
go run cmd/get-shopify-order-by-id/main.go 6349083345108
```

**Output:**
```
✅ Order found!

Order Information:
  Order Number: #45246
  Fulfillment Status: Unfulfilled
  Financial Status: Paid
  ...
```

### Check Shopify API Permissions

```bash
go run cmd/check-permissions/main.go
```

**Output:**
```
Checking API permissions...

1. Testing 'read_products' permission...
   ✅ Success! Found product: Product Name

2. Testing 'write_draft_orders' permission...
   ✅ Success! Can create draft orders
```

### Test Shopify Connection

```bash
go run cmd/test-shopify/main.go
```

---

## Testing Scripts

All test scripts are PowerShell scripts (`.ps1`). Run them from PowerShell:

### Test Order Submission

```powershell
.\test-order.ps1
```

**What it does:**
- Submits `test-order.json` to the API
- Shows response status and order ID
- Uses API key from script

**Prerequisites:**
- Server running on `http://localhost:8081` (or update `$BASE_URL` in script)
- Valid API key in script
- `test-order.json` file exists

### Test Idempotency

```powershell
.\test-idempotency.ps1
```

**What it tests:**
1. Same idempotency key + same payload → returns same order
2. Same idempotency key + different payload → returns 409 Conflict

**Output:**
```
Test 1: First submission
✅ Order created: abc-123

Test 2: Same key + same payload
✅ Same order ID returned (no duplicate)

Test 3: Same key + different payload
✅ Got 409 Conflict as expected
```

### Test Mixed Cart Behavior

```powershell
.\test-mixed-cart.ps1
```

**What it tests:**
1. Cart with only unmapped SKUs → returns 204
2. Mixed cart (mapped + unmapped) → creates order with variants and custom items

**Output:**
```
Test 1: Cart with ONLY unmapped SKUs
✅ Got 204 No Content as expected

Test 2: Mixed Cart
✅ Order created: xyz-789
   - Mapped SKU → Shopify variant
   - Unmapped SKU → Custom line item
```

### Test SKU Mapping Lifecycle

```powershell
.\test-sku-lifecycle-simple.ps1
```

**What it tests:**
1. Create order with unmapped SKU → appears as custom item
2. Add SKU mapping
3. Create order again with same SKU → now appears as variant

**Output:**
```
Step 1: Order with unmapped SKU
✅ Order created

Step 2: Adding SKU mapping
✅ SKU mapping added!

Step 3: Order with mapped SKU
✅ Order created
   Is Supplier Item: true
   Shopify Variant ID: 47441132978388
```

### Test Admin Order Lifecycle

```powershell
.\test-admin-lifecycle.ps1
```

**What it tests:**
1. POST `/v1/admin/orders/:id/confirm` → status changes to CONFIRMED
2. POST `/v1/admin/orders/:id/reject` → status changes to REJECTED
3. POST `/v1/admin/orders/:id/ship` → status changes to SHIPPED with tracking

**Output:**
```
Test 1: Confirm Order
✅ Order confirmed! Status: CONFIRMED

Test 2: Reject Order
✅ Order rejected! Status: REJECTED

Test 3: Ship Order
✅ Order shipped! Status: SHIPPED
   Tracking: TRACK123456789
```

### Test Failure Mode Behavior

```powershell
.\test-failure-mode.ps1
```

**What it tests:**
- Verifies that Shopify failures don't break order creation
- Order should still be created in database even if Shopify fails
- API should return 200 (not 500)

---

## Common Workflows

### Complete Setup Workflow

```bash
# 1. Start database
docker-compose up -d

# 2. Run migrations
go run cmd/migrate/main.go migrations/000001_init_schema.up.sql
go run cmd/migrate/main.go migrations/000002_add_payment_method.up.sql
go run cmd/migrate/main.go migrations/000003_add_shopify_order_id.up.sql

# 3. Create a partner
go run cmd/create-partner/main.go "Test Partner" "test-api-key-123"

# 4. Find and map a SKU
go run cmd/find-sku/main.go "JDTQ1834"
go run cmd/add-sku/main.go "JDTQ1834" 8085607284948 44219312570580

# 5. Start server
go run cmd/server/main.go

# 6. Test order submission (in another terminal)
.\test-order.ps1
```

### Daily Operations Workflow

```bash
# Check recent orders
go run cmd/list-orders/main.go

# Find specific order
go run cmd/find-order/main.go "ORDER-2024-001"

# Check order items
go run cmd/check-order-items/main.go <order-id>

# Check Shopify order status
go run cmd/get-shopify-order-by-id/main.go <shopify_order_id>
```

### SKU Management Workflow

```bash
# 1. Search for SKU in Shopify
go run cmd/find-sku/main.go "SKU-NAME"

# 2. Add mapping (use IDs from step 1)
go run cmd/add-sku/main.go "SKU-NAME" <product-id> <variant-id>

# 3. Verify mapping
go run cmd/list-sku-mappings/main.go

# 4. Test with order
.\test-order.ps1
```

---

## Troubleshooting

### Script Fails to Connect to Database

**Error:** `Failed to connect to database`

**Solutions:**
- Check PostgreSQL is running: `docker-compose ps`
- Verify `.env` has correct DB credentials
- Check database exists: `docker-compose exec postgres psql -U postgres -l`

### Script Can't Find Configuration

**Error:** `Failed to load configuration`

**Solutions:**
- Ensure `.env` file exists in project root
- Check `.env` has all required variables
- Verify you're running from project root directory

### Shopify API Errors

**Error:** `Failed to query Shopify`

**Solutions:**
- Verify `SHOPIFY_SHOP_DOMAIN` in `.env` (no `https://`, just domain)
- Verify `SHOPIFY_ACCESS_TOKEN` is correct
- Check API permissions: `go run cmd/check-permissions/main.go`

### PowerShell Script Errors

**Error:** `Unexpected token` or syntax errors

**Solutions:**
- Ensure PowerShell 5.1+ (not PowerShell Core)
- Check file encoding (should be UTF-8)
- Run: `Get-ExecutionPolicy` (should be `RemoteSigned` or `Unrestricted`)

---

## Quick Reference

### Go Commands

| Command | Purpose |
|--------|---------|
| `go run cmd/server/main.go` | Start API server |
| `go run cmd/create-partner/main.go "<name>" "<key>"` | Create partner |
| `go run cmd/find-sku/main.go "<sku>"` | Find SKU in Shopify |
| `go run cmd/add-sku/main.go "<sku>" <pid> <vid>` | Add SKU mapping |
| `go run cmd/list-orders/main.go` | List all orders |
| `go run cmd/find-order/main.go "<id>"` | Find order by partner ID |
| `go run cmd/check-order-items/main.go <id>` | Check order items |

### PowerShell Scripts

| Script | Purpose |
|--------|---------|
| `.\test-order.ps1` | Submit test order |
| `.\test-idempotency.ps1` | Test idempotency |
| `.\test-mixed-cart.ps1` | Test mixed cart behavior |
| `.\test-sku-lifecycle-simple.ps1` | Test SKU mapping lifecycle |
| `.\test-admin-lifecycle.ps1` | Test admin endpoints |
| `.\test-failure-mode.ps1` | Test failure handling |

---

## Environment Variables

All scripts use configuration from `.env` file. Required variables:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=123123
DB_NAME=b2bapi
DB_SSLMODE=disable

# Shopify
SHOPIFY_SHOP_DOMAIN=jafarshop.myshopify.com
SHOPIFY_ACCESS_TOKEN=shpat_xxxxx

# Server
PORT=8080
ENVIRONMENT=development
```

---

**Last Updated:** 2026-01-21
