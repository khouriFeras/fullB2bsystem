# Shopify Custom App Setup Guide

This guide will help you create a Shopify custom app and obtain the Admin API access token needed for the B2B API.

## Prerequisites

- A Shopify store (development store or production store)
- Admin access to the store

## Steps

### 1. Create a Custom App

1. Log in to your Shopify admin panel
2. Navigate to **Settings** → **Apps and sales channels**
3. Click **Develop apps** (or **Develop apps for your store**)
4. Click **Allow custom app development** (if prompted)
5. Click **Create an app**
6. Enter an app name (e.g., "JafarShop B2B Integration")
7. Click **Create app**

### 2. Configure API Scopes

1. In your app settings, click **Configure Admin API scopes**
2. Select the following scopes:
   - `read_products` - To read product and variant information
   - `write_draft_orders` - To create draft orders
   - `read_draft_orders` - To read draft orders (optional, for verification)
3. Click **Save**

### 3. Install the App

1. Click **Install app** (or the app will be installed automatically)
2. Review the permissions and click **Install**

### 4. Get the Admin API Access Token

1. After installation, you'll see the **API credentials** section
2. Click **Reveal token once** or **Show token**
3. **Copy the Admin API access token** - you'll need this for the `SHOPIFY_ACCESS_TOKEN` environment variable
4. **Important**: Store this token securely. You won't be able to see it again after closing the dialog.

### 5. Get Your Shop Domain

Your shop domain is in the format: `your-store-name.myshopify.com`

You can find it:
- In the URL when logged into your Shopify admin
- In **Settings** → **Store details**

Use this for the `SHOPIFY_SHOP_DOMAIN` environment variable.

## Environment Variables

Add these to your `.env` file or environment:

```bash
SHOPIFY_SHOP_DOMAIN=your-store-name.myshopify.com
SHOPIFY_ACCESS_TOKEN=shpat_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Testing the Connection

You can test the connection by running the application and checking the logs. The app will attempt to connect to Shopify when creating draft orders.

## Troubleshooting

### "Invalid API key or access token"
- Verify the access token is correct (starts with `shpat_`)
- Ensure the token hasn't been revoked
- Check that the app is still installed

### "Insufficient permissions"
- Verify all required scopes are granted
- Reinstall the app if you added scopes after initial installation

### "Shop domain not found"
- Ensure the domain is in the format `store-name.myshopify.com` (not `store-name.com`)
- Don't include `https://` in the domain

## Security Notes

- Never commit the access token to version control
- Rotate tokens periodically
- Use environment variables or a secrets management system
- Consider using Shopify's OAuth flow for production apps (more secure, but more complex)

## Next Steps

1. Set up your SKU mappings by syncing products from Shopify
2. Create partner accounts with API keys
3. Test the order submission flow
