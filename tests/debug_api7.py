import requests
import json

BASE_URL = "http://localhost:8080"
s = requests.Session()

resp = s.post(f"{BASE_URL}/api/auth/login", json={"username": "admin", "password": "admin123"})
token = resp.json()["data"]["token"]
s.headers.update({"Authorization": f"Bearer {token}"})

# Create template
resp = s.post(f"{BASE_URL}/api/templates", json={"name": "TestHandwriting"})
data = resp.json()
tid = data["data"]["id"]
print(f"Created template: {tid}")

# Try handwriting update
hw = {"paper_opacity": 0.5, "char_jitter": 3.0}
resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"handwriting": hw})
print(f"Handwriting PUT: {resp.status_code} {resp.json()}")

# Try controls with minimal data
controls = [{"id": "c1", "label": "Test", "type": "textbox", "x": 100.0, "y": 100.0, "width": 200.0, "height": 40.0, "font_size": 20, "font_family": "sans-serif", "required": False, "preview_text": "", "check_size": 24}]
resp = s.put(f"{BASE_URL}/api/templates/{tid}", json={"controls": controls})
print(f"Controls PUT: {resp.status_code} {resp.json()}")

# Verify
resp = s.get(f"{BASE_URL}/api/templates/{tid}")
data = resp.json()["data"]
print(f"Controls: {len(data.get('controls', []))}")
print(f"Handwriting: {data.get('handwriting', {})}")
