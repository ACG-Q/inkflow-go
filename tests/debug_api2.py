import requests

BASE_URL = "http://localhost:8080"
s = requests.Session()

# Login
resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Get all templates
resp = s.get(f"{BASE_URL}/api/templates?page=1&page_size=10")
templates = resp.json()["data"]["items"]
print("Templates:")
for t in templates:
    print(f"  - id={t['id']}: {t['name']}")

# Check latest template
if templates:
    tid = templates[0]["id"]
    resp = s.get(f"{BASE_URL}/api/templates/{tid}")
    data = resp.json()["data"]
    print(f"\nLatest template: {data['name']} (id={tid})")
    print(f"Controls: {len(data.get('controls', []))}")
    for c in data.get("controls", []):
        print(f"  - {c['id']}: {c['label']} at ({c.get('x',0)},{c.get('y',0)})")
