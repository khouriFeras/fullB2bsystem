"""Test POST /v1/carts/submit validation - bad payload returns 422."""
import json
import requests

url = "http://localhost:8081/v1/carts/submit"
headers = {
    "Authorization": "Bearer test-api-key-123",
    "Content-Type": "application/json",
}

# Missing required fields: items, customer, shipping, totals
payload = {"partner_order_id": "bad-order-1"}

response = requests.post(url, headers=headers, json=payload)

print(f"Status: {response.status_code} (expected 422)")
print()
if response.ok:
    print(json.dumps(response.json(), indent=2))
else:
    print(response.text)
