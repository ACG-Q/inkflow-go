import requests
import json

BASE_URL = "http://localhost:8080"
s = requests.Session()

resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "TestDebug"})
data = resp.json()
tid = data["data"]["id"]
print(f"Created template: {tid}")

# Wait a bit
import time
time.sleep(0.5)

# Update controls
controls = [
    {"id": "test_textbox", "label": "Test", "type": "textbox", "x": 100.0, "y": 100.0, "width": 200.0, "height": 40.0, "font_size": 20, "font_family": "sans-serif", "required": False, "preview_text": "Enter", "check_size": 24},
]
resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"PUT controls: {resp.status_code} {resp.json()}")

# Verify immediately
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls: {len(data.get('controls', []))}")
for c in data.get("controls", []):
    print(f"  JSON keys: {list(c.keys())}")
    print(f"  values: id={c.get('id')}, x={c.get('x')}, fontSize={c.get('fontSize')}, font_size={c.get('font_size')}")
