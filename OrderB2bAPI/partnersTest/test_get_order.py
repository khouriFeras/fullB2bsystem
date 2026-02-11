"""Test GET /v1/orders/:id - get order by ID. Pass order ID as first arg."""
import json
import requests
import sys

order_id = sys.argv[1] if len(sys.argv) > 1 else None
if not order_id:
    print("Usage: python test_get_order.py <order-uuid>")
    print("Get an order ID from test_admin_list_orders.py first.")
    sys.exit(1)

url = f"http://localhost:8081/v1/orders/{order_id}"
headers = {"Authorization": "Bearer test-api-key-123"}

response = requests.get(url, headers=headers)

print(f"Status: {response.status_code}")
print()
if response.ok:
    data = response.json()
    print(json.dumps(data, indent=2))
else:
    print(response.text)
