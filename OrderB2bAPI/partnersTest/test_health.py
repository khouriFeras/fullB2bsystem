"""Test health check endpoint (no auth required)."""
import json
import requests

url = "http://localhost:8081/health"

response = requests.get(url)

print(f"Status: {response.status_code}")
print()
if response.ok:
    data = response.json()
    print(json.dumps(data, indent=2))
else:
    print(response.text)
