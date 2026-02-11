# SKU Mapping Setup Guide

## Current Situation

Your Shopify store currently has **0 products**. The B2B API requires **at least one mapped SKU** to process orders.

## How It Works

- **Mapped SKUs**: Products that exist in your Shopify store and are mapped in the database
- **Unmapped SKUs**: Custom products that don't exist in Shopify (will be added as custom line items)

The system requires **at least one mapped SKU** per order. If all SKUs are unmapped, the API returns `204 No Content`.

## Solution: Add Products to Shopify

You have two options:

### Option 1: Add Products via Shopify Admin (Recommended)

1. Go to your Shopify admin: https://jafarshopfetchproducts.myshopify.com/admin
2. Navigate to **Products** → **Add product**
3. Create at least one product with:
   - **Title**: Any name (e.g., "Test Product")
   - **SKU**: Use one from your test-order.json (e.g., "JDTQ1834")
   - **Price**: Set a price
   - **Inventory**: Set quantity if needed
4. **Save** the product

### Option 2: Use Shopify API (Advanced)

You can create products programmatically using Shopify's GraphQL API, but the admin interface is easier for testing.

## After Adding Products

Once you have products in Shopify:

1. **List products to find IDs:**
   ```powershell
   go run cmd/list-skus/main.go
   ```

2. **Find a specific SKU:**
   ```powershell
   go run cmd/find-sku/main.go "JDTQ1834"
   ```

3. **Map the SKU:**
   ```powershell
   go run cmd/add-sku/main.go "JDTQ1834" <product-id> <variant-id>
   ```
   (The find-sku command will show you the exact command to run)

4. **Or use the automated script:**
   ```powershell
   .\map-test-order-skus.ps1
   ```

## Testing with Current Setup

Since your store is empty, you have two paths:

### Path A: Add Products First (Recommended)
1. Add at least one product to Shopify with a SKU from test-order.json
2. Run `.\map-test-order-skus.ps1` to map it
3. Run `.\test-order.ps1` to test

### Path B: Test with Custom Products Only
The current system won't process orders with only unmapped SKUs. You need at least one mapped SKU.

## Quick Test Setup

For a quick test, you can:

1. **Add one simple product in Shopify:**
   - Title: "Test Product"
   - SKU: "JDTQ1834"
   - Price: 10.00

2. **Map it:**
   ```powershell
   go run cmd/find-sku/main.go "JDTQ1834"
   # Copy the command it shows you, then run it
   ```

3. **Test the order:**
   ```powershell
   .\test-order.ps1
   ```

The order will include:
- ✅ "JDTQ1834" - mapped SKU (from Shopify)
- ✅ "Spong daddy" - unmapped (custom line item)
- ✅ "notInOurSotre" - unmapped (custom line item)

## Verification

Check if SKUs are mapped:
```powershell
go run cmd/list-sku-mappings/main.go
```

Check if products exist:
```powershell
go run cmd/list-products/main.go
```
