"""Test POST /v1/carts/submit - submit cart (creates order if SKUs are mapped)."""
import json
import requests
import uuid

url = "http://localhost:8081/v1/carts/submit"
headers = {
    "Authorization": "Bearer test-api-key-123",
    "Content-Type": "application/json",
    "Idempotency-Key": str(uuid.uuid4()),
}

payload = {
    "partner_order_id": f"test-order-{uuid.uuid4().hex[:8]}",
    "items": [
        {"sku": "CO2", "title": "Product CO2", "price": 10, "quantity": 1}
    ],
    "customer": {"name": "Jack Sparrow", "phone": "0781234567"},
    "shipping": {
        "street": "123 Main St",
        "city": "Anytown",
        "state": "Salt",
        "postal_code": "00962",
        "country": "JOR",
    },
    "totals": {"subtotal": 10, "tax": 1.6, "shipping": 3, "total": 11.6},
    "payment_status": "paid",
}

response = requests.post(url, headers=headers, json=payload)

print(f"Status: {response.status_code}")
print()
if response.ok:
    data = response.json()
    print(json.dumps(data, indent=2))
    if data.get("supplier_order_id"):
        print()
        print(f"Order ID: {data['supplier_order_id']}")
else:
    print(response.text)
    if response.status_code == 204:
        print("(204 = no supplier SKUs in cart, nothing to create)")
