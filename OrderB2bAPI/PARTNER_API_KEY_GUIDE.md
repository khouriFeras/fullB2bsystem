# Partner API Key Management Guide

This guide explains how to create partners, generate API keys, and manage partner access to the B2B API.

## Table of Contents

- [Overview](#overview)
- [Creating a Partner](#creating-a-partner)
- [API Key Generation](#api-key-generation)
- [Providing API Keys to Partners](#providing-api-keys-to-partners)
- [How Partners Use API Keys](#how-partners-use-api-keys)
- [Security Best Practices](#security-best-practices)
- [Managing Partners](#managing-partners)
- [Troubleshooting](#troubleshooting)

## Overview

Each partner store that integrates with the B2B API requires:
- A **Partner** record in the database
- A unique **API Key** for authentication
- The API key is hashed using bcrypt before storage (never stored in plain text)

Partners use their API key to:
- Submit orders via `POST /v1/carts/submit`
- Retrieve order status via `GET /v1/orders/:id`
- Access only their own orders (cross-partner access is blocked)

## Creating a Partner

### Step 1: Generate a Secure API Key

**Important:** API keys should be:
- **Unique** for each partner
- **Random and unpredictable** (use a secure random generator)
- **At least 32 characters** long
- **Never reused** across partners

**Recommended approach:**
```bash
# Generate a random API key (Linux/macOS)
openssl rand -hex 32

# Or use a UUID-based approach
uuidgen

# Or use an online secure random generator
# Example format: "partner-zain-2024-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

### Step 2: Create the Partner Record

Use the `create-partner` command-line tool:

```bash
go run cmd/create-partner/main.go "<Partner Name>" "<API Key>"
```

**Example:**
```bash
go run cmd/create-partner/main.go "Zain Shop" "zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

**Output:**
```
✅ Partner created successfully!

Partner ID: 550e8400-e29b-41d4-a716-446655440000
Partner Name: Zain Shop
API Key: zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6

⚠️  IMPORTANT: Save this API key securely! You won't be able to see it again.

Use this API key in the Authorization header:
Authorization: Bearer zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

### Step 3: Save the API Key Securely

**⚠️ CRITICAL:** The API key is shown **only once** during creation. After that, it cannot be retrieved from the database (only the hash is stored).

**What to save:**
- Partner ID (UUID)
- Partner Name
- **API Key** (plain text - this is the only time you'll see it)
- Date created

**Storage recommendations:**
- Use a password manager (1Password, LastPass, etc.)
- Store in a secure, encrypted file
- Never commit API keys to version control
- Share via secure channels only (encrypted email, secure messaging)

## API Key Generation

### Best Practices

1. **Use a consistent naming convention:**
   ```
   Format: <partner-slug>-<year>-<random-hex>
   Example: "zain-shop-2024-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
   ```

2. **Generate cryptographically secure random keys:**
   ```bash
   # Recommended: Use OpenSSL
   openssl rand -hex 32
   
   # Or use Go's crypto/rand
   # Or use a secure online generator
   ```

3. **Document the key generation:**
   - Record when the key was created
   - Note which partner it belongs to
   - Track key rotation dates

### Key Format

API keys can be any string, but recommended format:
- **Length:** 32-64 characters
- **Characters:** Alphanumeric + hyphens/underscores
- **Avoid:** Special characters that might need URL encoding

**Examples:**
- ✅ `zain-shop-2024-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`
- ✅ `partner_abc_xyz_1234567890abcdef`
- ❌ `key with spaces` (spaces not recommended)
- ❌ `key@with#special$chars` (may cause encoding issues)

## Providing API Keys to Partners

### Secure Delivery Methods

1. **Encrypted Email:**
   - Use PGP/GPG encryption
   - Or use a password-protected ZIP file
   - Send password via separate channel (SMS, phone call)

2. **Secure Messaging:**
   - Signal, WhatsApp (with encryption)
   - Secure business messaging platforms

3. **In-Person:**
   - Hand-deliver printed key
   - Use QR code for easy copy-paste

4. **Password Manager:**
   - Share via 1Password, LastPass secure sharing
   - Partner can access via their own account

### What to Include

When providing the API key, include:

1. **API Key:**
   ```
   Your API Key: zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
   ```

2. **API Base URL:**
   ```
   Production: https://api.jafarshop.com
   Staging: https://staging-api.jafarshop.com
   Development: http://localhost:8081
   ```

3. **Usage Instructions:**
   ```
   Include this header in all API requests:
   Authorization: Bearer <your-api-key>
   ```

4. **API Documentation:**
   - Link to `API_DOCUMENTATION.md`
   - Example requests
   - Support contact information

### Example Email Template

```
Subject: Your B2B API Access Credentials

Dear [Partner Name],

Your API access has been set up. Please find your credentials below:

API Key: [API_KEY]
Base URL: [BASE_URL]

⚠️ IMPORTANT SECURITY NOTES:
- Keep this API key secret and never share it publicly
- Do not commit it to version control
- If you suspect it's compromised, contact us immediately to rotate it

Usage:
Include this header in all API requests:
  Authorization: Bearer [API_KEY]

Documentation: [LINK_TO_DOCS]
Support: [SUPPORT_EMAIL]

Best regards,
JafarShop B2B Team
```

## How Partners Use API Keys

### Authentication Header

All API requests must include the `Authorization` header:

```
Authorization: Bearer <api-key>
```

### Example: cURL

```bash
curl -X POST https://api.jafarshop.com/v1/carts/submit \
  -H "Authorization: Bearer zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d @order.json
```

### Example: PowerShell

```powershell
$apiKey = "zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
$headers = @{
    "Authorization" = "Bearer $apiKey"
    "Content-Type" = "application/json"
    "Idempotency-Key" = [guid]::NewGuid().ToString()
}

Invoke-WebRequest -Uri "http://localhost:8081/v1/carts/submit" `
    -Method POST `
    -Headers $headers `
    -Body (Get-Content order.json -Raw)
```

### Example: JavaScript/Node.js

```javascript
const apiKey = 'zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6';

fetch('https://api.jafarshop.com/v1/carts/submit', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
    'Idempotency-Key': crypto.randomUUID()
  },
  body: JSON.stringify(orderData)
});
```

### Example: Python

```python
import requests
import uuid

api_key = "zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"

headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json",
    "Idempotency-Key": str(uuid.uuid4())
}

response = requests.post(
    "https://api.jafarshop.com/v1/carts/submit",
    headers=headers,
    json=order_data
)
```

## Security Best Practices

### For API Administrators

1. **Key Rotation:**
   - Rotate keys annually or when compromised
   - Provide advance notice to partners
   - Support overlapping keys during transition (if needed)

2. **Access Control:**
   - Each partner can only access their own orders
   - Monitor API usage for suspicious activity
   - Implement rate limiting per partner

3. **Key Storage:**
   - Never store plain-text keys in the database
   - Use bcrypt hashing (already implemented)
   - Consider adding SHA256 lookup hash for performance

4. **Audit Logging:**
   - Log all API key usage
   - Track failed authentication attempts
   - Monitor for brute-force attacks

5. **Partner Management:**
   - Deactivate partners immediately if compromised
   - Maintain a list of active partners
   - Document key creation and rotation dates

### For Partners

1. **Key Protection:**
   - Store API keys in environment variables (never hardcode)
   - Use secrets management tools (AWS Secrets Manager, HashiCorp Vault)
   - Never commit keys to version control

2. **Environment Variables:**
   ```bash
   # .env file (add to .gitignore)
   JAFARSHOP_API_KEY=zain-api-key-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
   ```

3. **Secure Transmission:**
   - Always use HTTPS in production
   - Never send API keys in URL parameters
   - Use headers only (Authorization header)

4. **Key Rotation:**
   - Report compromised keys immediately
   - Test new keys in staging before production
   - Update keys in all systems simultaneously

## Managing Partners

### Viewing Partners

Currently, there's no CLI tool to list partners. You can query the database directly:

```sql
SELECT id, name, is_active, created_at, updated_at
FROM partners
ORDER BY created_at DESC;
```

### Activating/Deactivating Partners

To deactivate a partner (blocks all API access):

```sql
UPDATE partners
SET is_active = false, updated_at = NOW()
WHERE id = '<partner-uuid>';
```

To reactivate:

```sql
UPDATE partners
SET is_active = true, updated_at = NOW()
WHERE id = '<partner-uuid>';
```

### Rotating API Keys

**Current limitation:** API keys cannot be updated in-place. To rotate:

1. **Create a new partner record** with the new API key
2. **Migrate existing orders** to the new partner ID (if needed)
3. **Deactivate the old partner** record
4. **Provide new API key** to the partner

**Future improvement:** Add an `UpdateAPIKey` method to allow in-place rotation.

### Deleting Partners

**⚠️ WARNING:** Deleting a partner will orphan their orders. Instead, **deactivate** them:

```sql
-- DO NOT DELETE (orphans orders)
-- DELETE FROM partners WHERE id = '<uuid>';

-- Instead, deactivate:
UPDATE partners SET is_active = false WHERE id = '<uuid>';
```

## Troubleshooting

### Partner Gets "Invalid API Key" Error

**Possible causes:**

1. **Typo in API key:**
   - Verify the exact key (copy-paste recommended)
   - Check for extra spaces or line breaks

2. **Partner is inactive:**
   ```sql
   SELECT is_active FROM partners WHERE id = '<partner-uuid>';
   ```

3. **Wrong Authorization header format:**
   - Must be: `Authorization: Bearer <key>`
   - Not: `Authorization: <key>` (missing "Bearer")
   - Not: `Authorization: bearer <key>` (case-sensitive)

4. **Key was rotated/deleted:**
   - Check if partner was recreated
   - Verify key hasn't expired (if expiration is implemented)

**Solution:**
- Verify key in secure storage
- Check partner status in database
- Test with a known-good key
- Contact support if issue persists

### Partner Can't Access Their Order

**Possible causes:**

1. **Order belongs to different partner:**
   - Each partner can only access their own orders
   - Verify `partner_id` matches in database

2. **Order doesn't exist:**
   - Check order ID is correct
   - Verify order was created successfully

**Solution:**
```sql
-- Check order ownership
SELECT partner_id, partner_order_id, status
FROM supplier_orders
WHERE id = '<order-uuid>';
```

### API Key Compromised

**Immediate actions:**

1. **Deactivate the partner:**
   ```sql
   UPDATE partners SET is_active = false WHERE id = '<uuid>';
   ```

2. **Create new partner with new key:**
   ```bash
   go run cmd/create-partner/main.go "<Partner Name>" "<New API Key>"
   ```

3. **Notify partner:**
   - Explain the security incident
   - Provide new API key via secure channel
   - Advise to update all systems immediately

4. **Audit logs:**
   - Review API access logs for suspicious activity
   - Check for unauthorized order submissions
   - Monitor for unusual patterns

## Additional Resources

- [API Documentation](./API_DOCUMENTATION.md) - Complete API reference
- [Testing Guide](./TESTING_GUIDE.md) - How to test the API
- [README](./README.md) - Project overview and setup

## Support

For API key issues or questions:
- **Email:** Feras.jafarShop@gmail.com
- **Documentation:** See `API_DOCUMENTATION.md`
- **Issues:** Report via your support channel

---

**Last Updated:** 2026-01-21  
**Version:** 1.0
