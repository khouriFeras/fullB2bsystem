"""Test auth failure cases: missing header, bad format, invalid key."""
import json
import requests

base = "http://localhost:8081/v1/admin/orders"

print("=" * 50)
print("1. No Authorization header")
print("=" * 50)
r = requests.get(base)
print(f"Status: {r.status_code}")
print(r.text[:200] if r.text else "(empty)")
print()

print("=" * 50)
print("2. Invalid format (no Bearer prefix)")
print("=" * 50)
r = requests.get(base, headers={"Authorization": "test-api-key-123"})
print(f"Status: {r.status_code}")
print(r.text[:200] if r.text else "(empty)")
print()

print("=" * 50)
print("3. Invalid API key")
print("=" * 50)
r = requests.get(base, headers={"Authorization": "Bearer wrong-key-xyz"})
print(f"Status: {r.status_code}")
print(r.text[:200] if r.text else "(empty)")
print()
