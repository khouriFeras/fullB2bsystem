import json
import requests

url = "http://localhost:8081/v1/admin/orders"
headers = {"Authorization": "Bearer test-api-key-123"}

response = requests.get(url, headers=headers)

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
