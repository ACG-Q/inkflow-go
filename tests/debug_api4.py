import requests
import json

BASE_URL = "http://localhost:8080"
s = requests.Session()

# Login
resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "TestAPI"})
data = resp.json()
print(f"Create: {data}")
tid = data["data"]["id"]

# Update with just name first
resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"name": "TestAPIUpdated"})
print(f"Update name: {resp.json()}")

# Now try controls
controls = [
    {
        "id": "ctrl_1",
        "label": "Test",
        "type": "textbox",
        "x": 100.0,
        "y": 100.0,
        "width": 200.0,
        "height": 40.0,
        "font_size": 20,
        "font_family": "sans-serif",
        "required": False,
        "preview_text": "Enter",
        "check_size": 24,
    },
]
print(f"Sending controls: {json.dumps(controls, indent=2)}")
resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"Update controls: {resp.status_code} {resp.json()}")

# Verify
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls: {len(data.get('controls', []))}")
