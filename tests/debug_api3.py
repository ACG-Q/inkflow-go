import requests

BASE_URL = "http://localhost:8080"
s = requests.Session()

# Login
resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "TestControls"})
data = resp.json()
tid = data["data"]["id"]
print(f"Created template: {tid}")

# Update with controls
controls = [
    {
        "id": "test_textbox",
        "label": "Test",
        "type": "textbox",
        "x": 100,
        "y": 100,
        "width": 200,
        "height": 40,
        "font_size": 20,
        "font_family": "sans-serif",
        "required": False,
        "preview_text": "Enter",
        "check_size": 24,
    },
]

resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"PUT response: {resp.status_code} {resp.json()}")

# Get template
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls after: {len(data.get('controls', []))}")
for c in data.get("controls", []):
    print(f"  {c['id']}: {c['label']}")
