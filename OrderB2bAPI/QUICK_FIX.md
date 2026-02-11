# Quick Fix for 500 Error

## Issue
Getting 500 Internal Server Error with no error body when submitting orders.

## Solution

**IMPORTANT: Restart the server** - The server needs to be restarted to pick up code changes:

1. **Stop the current server:**
   - Find the terminal/console where `go run cmd/server/main.go` is running
   - Press `Ctrl+C` to stop it

2. **Restart the server:**
   ```powershell
   go run cmd/server/main.go
   ```

3. **Watch the console output** - You should see:
   - Server starting messages
   - Any error logs when you submit an order

4. **Test again:**
   ```powershell
   .\test-order.ps1
   ```

## What Was Fixed

1. ✅ Added custom recovery middleware that logs panics
2. ✅ Improved error messages in cart handler to include error details
3. ✅ Fixed JSON parsing in collection queries

## If Still Failing

Check the server console output - it will show:
- The actual error message
- Stack traces if there's a panic
- Database errors
- Shopify API errors

The server logs will tell you exactly what's failing!
