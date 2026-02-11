"""Test GET /v1/admin/orders with query params: limit, offset, status."""
import json
import requests

base = "http://localhost:8081/v1/admin/orders"
headers = {"Authorization": "Bearer test-api-key-123"}

# Test with status filter and pagination (valid: INCOMPLETE_CAUTION, UNFULFILLED, FULFILLED, COMPLETE, REJECTED, CANCELED, REFUNDED, ARCHIVED)
params = {"limit": 5, "offset": 0, "status": "INCOMPLETE_CAUTION"}
response = requests.get(base, headers=headers, params=params)

print(f"URL: {response.url}")
print(f"Status: {response.status_code}")
print()
if response.ok:
    data = response.json()
    orders = data.get("orders", [])
    print(f"Orders: {len(orders)} (limit={data.get('limit')}, offset={data.get('offset')})")
    print()
    print(json.dumps(data, indent=2))
else:
    print(response.text)
