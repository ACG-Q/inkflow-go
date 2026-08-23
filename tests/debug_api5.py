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
    controls = data.get("controls", [])
    print(f"Controls: {len(controls)}")
    for c in controls:
        print(f"  - id={c.get('id')}, label={c.get('label')}, type={c.get('type')}, x={c.get('x')}, y={c.get('y')}")
