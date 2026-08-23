import requests
BASE_URL = "http://localhost:8080"
s = requests.Session()
resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

resp = s.get(f"{BASE_URL}/api/templates?page=1&page_size=1")
templates = resp.json()["data"]["items"]
if templates:
    tid = templates[0]["id"]
    resp = s.get(f"{BASE_URL}/api/templates/{tid}")
    data = resp.json()["data"]
    print(f"Template: {data['name']} (id={tid})")
    print(f"Controls count: {len(data.get('controls', []))}")
    for c in data.get("controls", []):
        print(f"  - {c['id']}: {c['label']} at ({c['x']},{c['y']})")
    print(f"Handwriting: {data.get('handwriting', {})}")
else:
    print("No templates found")
